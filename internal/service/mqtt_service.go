package service

import (
	"OMEGA3-IOT/internal/eventbus"
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/spec"
	"OMEGA3-IOT/internal/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gorm.io/gorm"
	"log"
	"strings"
	"time"
)

type MQTTService struct {
	broker          mqtt.Client
	deviceService   *DeviceService
	presenceService *PresenceService
	loggerService   logger.LoggerInterface
	eventBus        *eventbus.EventBus
}

type DeviceMessage struct {
	VerifyCode string `json:"verify_code"`
	TimeStamp  int64  `json:"timestamp"`
	Data       Data   `json:"data"`
}

type Data struct {
	Properties map[string]model.TypedInstancePropertyItem `json:"properties"`
	Event      model.DeviceEvent                          `json:"event"`
	Action     model.Action                               `json:"action"`
}
type Publisher interface {
	PublishActionToDevice(deviceUUID string, actionName string, payload interface{}) error
}

func NewMQTTService(brokerURL string, deviceService *DeviceService, loggerService logger.LoggerInterface, presenceService *PresenceService, eventBus *eventbus.EventBus) (*MQTTService, error) {
	options := mqtt.NewClientOptions()
	options.AddBroker(brokerURL)

	options.SetClientID("omega3-iot-server")
	options.SetAutoReconnect(true)
	options.SetConnectRetry(true)
	options.SetOrderMatters(false) //无序处理消息

	options.SetCleanSession(true)

	options.SetOnConnectHandler(func(client mqtt.Client) {
		log.Printf("MQTT Service connected to broker: %s", brokerURL)
	})
	options.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("MQTT Service disconnected from broker: %s for : %s", brokerURL, err)
	})
	client := mqtt.NewClient(options)
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect MQTT broke for : %v", token.Error())
	}
	log.Printf("MQTT Service connected to broker: %s successfully", brokerURL)

	//deviceSvc := NewDeviceService()
	service := &MQTTService{
		broker:          client,
		deviceService:   deviceService,
		presenceService: presenceService,
		loggerService:   loggerService,
		eventBus:        eventBus,
	}
	service.setupSubscription()
	return service, nil
}
func (m *MQTTService) PublishActionToDevice(deviceUUID string, commandName string, payload model.Action) error {
	topic := fmt.Sprintf("data/device/%s/action", deviceUUID)
	payloadBytes, err := json.Marshal(payload)
	//TODO 解耦
	if err != nil {
		return fmt.Errorf("failed to marshal action payload: %w", err)
	}
	token := m.broker.Publish(topic, 1, false, payloadBytes)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("Failed to publish action payload: %v", token.Error())
	}
	log.Printf("MQTT Service published action payload: %v to %v", string(payloadBytes), topic)
	return nil
}
func (m *MQTTService) setupSubscription() {
	log.Printf("MQTT Service setup subscription")
	if token := m.broker.Subscribe("data/device/+/properties", 1, m.handlePropertiesData); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to subscribe to data topic : %s", token.Error())
	} else {
		log.Printf("Successfully subscribed to topic [data/device/+/properties]")
	}
	if token := m.broker.Subscribe("data/device/+/action_result", 1, m.handleActionResult); token.Wait() && token.Error() != nil {
		log.Printf("Warning: Failed to subscribe to action_result topic: %s", token.Error())
	} else {
		log.Printf("Successfully subscribed to topic [data/device/+/action_result]")
	}
}
func (m *MQTTService) handlePropertiesData(c mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()
	log.Printf("Received property data from MQTT topic [%s] (QOS %d): %s", topic, msg.Qos(), string(payload))
	deviceUUID, _ := extractDeviceUUIDFromTopic(topic)
	var message DeviceMessage

	if err := json.Unmarshal(payload, &message); err != nil {
		fmt.Errorf("error unmarshalling device message: %v", err)
	}

	hashedVerifyCode := utils.HashVerifyCode(message.VerifyCode)
	rawPropsData := message.Data.Properties
	fmt.Printf("Properties Object: %+v\n", rawPropsData)

	instance, err := m.deviceService.GetDeviceByUUIDAndVerifyHash(deviceUUID, hashedVerifyCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Unauthorized access attempt: No device found with UUID %s and provided verify code (hash: %s)", deviceUUID, hashedVerifyCode)
		} else {
			log.Printf("Database error during authentication for device %s: %v", deviceUUID, err)
		}
		return
	}

	// 确保 instance.Properties.Items 已初始化
	if instance.Properties.Items == nil {
		instance.Properties.Items = make(map[string]*model.TypedInstancePropertyItem)
	}

	if err := m.deviceService.UpdateDeviceProperties(*instance, rawPropsData); err != nil {
		log.Printf("Failed to update device properties: %v", err)
		return
	}

	// Mark device online via PresenceService
	m.presenceService.MarkOnline(deviceUUID)

	// Publish property.update event to EventBus for WebSocket push
	propUpdateEvent := logger.NewDeviceLogEvent(deviceUUID, logger.LogLevelInfo, "Properties updated", logger.LogEventDevicePropertyUpdate)
	propsMap := make(map[string]interface{})
	for k, v := range rawPropsData {
		propsMap[k] = v.Value.V
	}
	propUpdateEvent.Metadata["properties"] = propsMap
	m.eventBus.Publish(context.Background(), propUpdateEvent)

	// Handle event if present
	if message.Data.Event.EventKey != "" {
		m.handleEvent(instance, message.Data.Event)
	}
}

func (m *MQTTService) handleEvent(instance *model.Instance, event model.DeviceEvent) {
	// Check for shutdown/offline events — these take priority
	if event.EventKey == "shutdown" || event.EventKey == "offline" {
		m.presenceService.HandleShutdownEvent(instance.InstanceUUID)
		log.Printf("[MQTT] Device %s sent '%s' event — marked OFFLINE", instance.InstanceUUID, event.EventKey)
		return
	}

	typeDef, ok := model.GlobalDeviceTypeManager.GetByName(instance.Type)
	if !ok {
		log.Printf("[MQTT] Unknown device type '%s' for event validation", instance.Type)
		return
	}

	// Validate event against spec
	if _, exists := typeDef.Events[event.EventKey]; exists {
		// Event key is defined in the spec — validate if there's a payload
		if event.Content != "" {
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(event.Content), &payload); err == nil {
				if err := spec.ValidateEvent(typeDef, event.EventKey, payload); err != nil {
					log.Printf("[MQTT] Event validation failed for device %s, event '%s': %v", instance.InstanceUUID, event.EventKey, err)
					return
				}
			}
		}
		severity := typeDef.Events[event.EventKey].Severity
		log.Printf("[MQTT] Received validated event '%s' from device %s (severity: %s)", event.EventKey, instance.InstanceUUID, severity)

		// Publish to EventBus for WebSocket push (only warning/critical)
		if severity == "warning" || severity == "critical" {
			var eventData interface{}
			if event.Content != "" {
				json.Unmarshal([]byte(event.Content), &eventData)
			}
			pushEvent := logger.NewDeviceLogEvent(instance.InstanceUUID, logger.LogLevelInfo, fmt.Sprintf("Event: %s", event.EventKey), logger.LogEventDeviceError)
			pushEvent.Metadata["event_key"] = event.EventKey
			pushEvent.Metadata["severity"] = severity
			pushEvent.Metadata["data"] = eventData
			// Use a custom event type for event push
			pushEvent.BaseEvent.Type = eventbus.EventType("device.event.received")
			m.eventBus.Publish(context.Background(), pushEvent)
		}
	} else {
		log.Printf("[MQTT] Received unknown event '%s' from device %s", event.EventKey, instance.InstanceUUID)
	}
}

type ActionResultMessage struct {
	VerifyCode string `json:"verify_code"`
	TimeStamp  int64  `json:"timestamp"`
	Data       struct {
		Command string `json:"command"`
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	} `json:"data"`
}

func (m *MQTTService) handleActionResult(c mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()
	log.Printf("Received action result from MQTT topic [%s]: %s", topic, string(payload))

	deviceUUID, _ := extractDeviceUUIDFromTopic(topic)
	var message ActionResultMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		log.Printf("[MQTT] Failed to parse action_result: %v", err)
		return
	}

	// Authenticate device
	hashedVerifyCode := utils.HashVerifyCode(message.VerifyCode)
	instance, err := m.deviceService.GetDeviceByUUIDAndVerifyHash(deviceUUID, hashedVerifyCode)
	if err != nil {
		log.Printf("[MQTT] Action result auth failed for device %s: %v", deviceUUID, err)
		return
	}

	// Publish to EventBus for WebSocket push
	resultEvent := logger.NewDeviceLogEvent(instance.InstanceUUID, logger.LogLevelInfo, fmt.Sprintf("Action result: %s", message.Data.Command), logger.LogEventDeviceActionResult)
	resultEvent.Metadata["command"] = message.Data.Command
	resultEvent.Metadata["success"] = message.Data.Success
	resultEvent.Metadata["error"] = message.Data.Error
	m.eventBus.Publish(context.Background(), resultEvent)

	log.Printf("[MQTT] Action result from device %s: command=%s success=%v", deviceUUID, message.Data.Command, message.Data.Success)
}

func extractDeviceUUIDFromTopic(topic string) (string, error) {
	// 简单的字符串分割方法
	parts := strings.Split(topic, "/")
	if len(parts) >= 4 && parts[0] == "data" && parts[1] == "device" && parts[3] == "properties" {
		return parts[2], nil
	}

	// 或者使用正则表达式 (更灵活，但稍慢)
	// re := regexp.MustCompile(`^data/device/([^/]+)/property$`)
	// matches := re.FindStringSubmatch(topic)
	// if len(matches) == 2 {
	// 	return matches[1], nil
	// }

	return "", fmt.Errorf("invalid topic format for device UUID extraction: %s", topic)
}
func (m *MQTTService) Disconnect(quiesce uint) {
	if m.broker != nil && m.broker.IsConnected() {
		m.broker.Disconnect(quiesce)
		log.Println("MQTT Service disconnected")
	}
}
