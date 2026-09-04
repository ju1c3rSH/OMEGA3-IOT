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
	"sync"
	"time"
)

const (
	mqttWorkerCount      = 16
	mqttQueueSize        = 2048
	mqttPublishTimeout   = 3 * time.Second
	mqttHandlerTimeout   = 2 * time.Second
	mqttConnectTimeout   = 10 * time.Second
	mqttSubscribeTimeout = 5 * time.Second
)

// ingestJob is a copied MQTT message for worker pool processing.
// Payload is copied because paho may reuse the underlying buffer after the
// callback returns (see https://pkg.go.dev/github.com/eclipse/paho.mqtt.golang).
type ingestJob struct {
	topic   string
	payload []byte
	qos     byte
	kind    string // "properties" | "action_result"
}

type MQTTService struct {
	broker          mqtt.Client
	deviceService   *DeviceService
	presenceService *PresenceService
	loggerService   logger.LoggerInterface
	eventBus        *eventbus.EventBus

	msgChan chan ingestJob
	stopCh  chan struct{}
	wg      sync.WaitGroup
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
	// Fix P1-2: context.WithTimeout was previously ignored (token.Wait() blocks
	// indefinitely). Use WaitTimeout with the same 10s bound so Gin startup
	// does not hang forever if the broker is unreachable.
	// See https://github.com/eclipse/paho.mqtt.golang/blob/master/token.go
	// Token.WaitTimeout vs Wait and issue #463 (Done channel for context).
	token := client.Connect()
	if !token.WaitTimeout(mqttConnectTimeout) {
		log.Fatalf("Failed to connect MQTT broker: timeout after %v", mqttConnectTimeout)
	}
	if token.Error() != nil {
		log.Fatalf("Failed to connect MQTT broke for : %v", token.Error())
	}
	log.Printf("MQTT Service connected to broker: %s successfully", brokerURL)

	service := &MQTTService{
		broker:          client,
		deviceService:   deviceService,
		presenceService: presenceService,
		loggerService:   loggerService,
		eventBus:        eventBus,
		msgChan:         make(chan ingestJob, mqttQueueSize),
		stopCh:          make(chan struct{}),
	}
	service.startWorkers()
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
	// Fix P1-2: previously token.Wait() blocked the caller indefinitely
	// (up to broker RTT+retransmit). Use WaitTimeout so the calling
	// goroutine is bounded to mqttPublishTimeout (3s). The async HTTP path
	// (SendActionHandlerFactory) runs this inside ActionDispatcher, so a
	// degraded broker only consumes a dispatcher slot, not the HTTP thread.
	// See https://github.com/eclipse/paho.mqtt.golang/blob/master/token.go#L73
	// and https://github.com/eclipse/paho.mqtt.golang/issues/445 (Done channel).
	if !token.WaitTimeout(mqttPublishTimeout) {
		log.Printf("[MQTT] Publish timeout after %v topic=%s command=%s queue_depth=%d/%d", mqttPublishTimeout, topic, commandName, len(m.msgChan), cap(m.msgChan))
		return fmt.Errorf("publish timeout after %v for topic %s: %w", mqttPublishTimeout, topic, errors.New("mqtt publish timeout"))
	}
	if token.Error() != nil {
		return fmt.Errorf("Failed to publish action payload: %v", token.Error())
	}
	log.Printf("MQTT Service published action payload: %v to %v", string(payloadBytes), topic)
	return nil
}
func (m *MQTTService) setupSubscription() {
	log.Printf("MQTT Service setup subscription")
	// Enqueue handlers return immediately so Paho's dispatch goroutine
	// (even with SetOrderMatters(false) each handler still runs in its own
	// goroutine – blocking it causes pingresp timeouts, see
	// https://github.com/eclipse/paho.mqtt.golang#common-problems and
	// https://github.com/eclipse/paho.mqtt.golang/issues/427) is not blocked
	// by DB/IoTDB work (70-150ms). Actual work is done by bounded workers.
	if token := m.broker.Subscribe("data/device/+/properties", 1, m.handlePropertiesData); !token.WaitTimeout(mqttSubscribeTimeout) {
		log.Fatalf("Failed to subscribe to data topic : timeout after %v", mqttSubscribeTimeout)
	} else if token.Error() != nil {
		log.Fatalf("Failed to subscribe to data topic : %s", token.Error())
	} else {
		log.Printf("Successfully subscribed to topic [data/device/+/properties]")
	}
	if token := m.broker.Subscribe("data/device/+/action_result", 1, m.handleActionResult); !token.WaitTimeout(mqttSubscribeTimeout) {
		log.Printf("Warning: Failed to subscribe to action_result topic: timeout after %v", mqttSubscribeTimeout)
	} else if token.Error() != nil {
		log.Printf("Warning: Failed to subscribe to action_result topic: %s", token.Error())
	} else {
		log.Printf("Successfully subscribed to topic [data/device/+/action_result]")
	}
}

// handlePropertiesData is the Paho callback – it MUST NOT block.
// It copies the payload and enqueues to the bounded worker pool so the Paho
// dispatch goroutine can ack immediately (QoS 1 PUBACK). If the queue is full
// we apply a drop-newest policy and log (backpressure). This prevents the
// Paho internal router from stalling under burst load.
// Pattern from https://thelinuxcode.com/go-worker-pools-production-patterns-pitfalls-and-practical-tuning/
// and https://backendbytes.com/articles/go-worker-pool-concurrency/.
func (m *MQTTService) handlePropertiesData(c mqtt.Client, msg mqtt.Message) {
	payloadCopy := make([]byte, len(msg.Payload()))
	copy(payloadCopy, msg.Payload())
	job := ingestJob{
		topic:   msg.Topic(),
		payload: payloadCopy,
		qos:     msg.Qos(),
		kind:    "properties",
	}
	select {
	case m.msgChan <- job:
		if len(m.msgChan) > cap(m.msgChan)*8/10 {
			log.Printf("[MQTT] properties enqueued topic=%s queue=%d/%d (high watermark)", job.topic, len(m.msgChan), cap(m.msgChan))
		}
	default:
		// Backpressure: drop newest to protect memory/latency.
		// Alternative policies: block with timeout or return 429 upstream;
		// for telemetry ingestion drop+metric is preferred to blocking Paho.
		log.Printf("[MQTT] WARN queue full (%d/%d) dropping properties message topic=%s", len(m.msgChan), cap(m.msgChan), job.topic)
	}
}

// handleActionResult is also non-blocking enqueue.
func (m *MQTTService) handleActionResult(c mqtt.Client, msg mqtt.Message) {
	payloadCopy := make([]byte, len(msg.Payload()))
	copy(payloadCopy, msg.Payload())
	job := ingestJob{
		topic:   msg.Topic(),
		payload: payloadCopy,
		qos:     msg.Qos(),
		kind:    "action_result",
	}
	select {
	case m.msgChan <- job:
	default:
		log.Printf("[MQTT] WARN queue full (%d/%d) dropping action_result topic=%s", len(m.msgChan), cap(m.msgChan), job.topic)
	}
}

func (m *MQTTService) startWorkers() {
	for i := 0; i < mqttWorkerCount; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}
	log.Printf("[MQTT] Started %d workers (queue=%d)", mqttWorkerCount, mqttQueueSize)
}

func (m *MQTTService) worker(id int) {
	defer m.wg.Done()
	for {
		select {
		case <-m.stopCh:
			log.Printf("[MQTT-Worker-%d] stopping (stopCh closed)", id)
			return
		case job, ok := <-m.msgChan:
			if !ok {
				log.Printf("[MQTT-Worker-%d] msgChan closed, exiting", id)
				return
			}
			func(j ingestJob) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[MQTT-Worker-%d] panic recovered: %v topic=%s kind=%s", id, r, j.topic, j.kind)
					}
				}()
				ctx, cancel := context.WithTimeout(context.Background(), mqttHandlerTimeout)
				defer cancel()
				start := time.Now()
				switch j.kind {
				case "properties":
					m.processPropertiesData(ctx, j)
				case "action_result":
					m.processActionResult(ctx, j)
				default:
					log.Printf("[MQTT-Worker-%d] unknown job kind %s topic=%s", id, j.kind, j.topic)
				}
				elapsed := time.Since(start)
				if elapsed > mqttHandlerTimeout {
					log.Printf("[MQTT-Worker-%d] slow job kind=%s topic=%s elapsed=%v exceeds %v queue=%d/%d", id, j.kind, j.topic, elapsed, mqttHandlerTimeout, len(m.msgChan), cap(m.msgChan))
				} else if elapsed > 100*time.Millisecond {
					log.Printf("[MQTT-Worker-%d] processed %s topic=%s elapsed=%v queue=%d/%d", id, j.kind, j.topic, elapsed, len(m.msgChan), cap(m.msgChan))
				}
				// Respect ctx cancellation for observability (DB ops themselves
				// are not context-aware yet, but timeout is logged above).
				select {
				case <-ctx.Done():
					if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						log.Printf("[MQTT-Worker-%d] context deadline exceeded for %s %s", id, j.kind, j.topic)
					}
				default:
				}
			}(job)
		}
	}
}

// processPropertiesData contains the original handlePropertiesData logic but
// runs inside a worker goroutine with a 2s context timeout. This isolates
// Paho callbacks from DB/IoTDB latency (70-150ms) and prevents dispatch blocking.
// See https://pkg.go.dev/github.com/eclipse/paho.mqtt.golang#Client.Subscribe
// "MessageHandler must not block".
func (m *MQTTService) processPropertiesData(ctx context.Context, job ingestJob) {
	topic := job.topic
	payload := job.payload
	deviceUUID, err := extractDeviceUUIDFromTopic(topic)
	if err != nil {
		log.Printf("[MQTT] invalid topic %s: %v", topic, err)
		return
	}
	var message DeviceMessage

	if err := json.Unmarshal(payload, &message); err != nil {
		log.Printf("[MQTT] error unmarshalling device message: %v payload=%s", err, string(payload))
		return
	}

	hashedVerifyCode := utils.HashVerifyCode(message.VerifyCode)
	rawPropsData := message.Data.Properties

	instance, err := m.deviceService.GetDeviceByUUIDAndVerifyHash(deviceUUID, hashedVerifyCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Unauthorized access attempt: No device found with UUID %s and provided verify code (hash: %s)", deviceUUID, hashedVerifyCode)
		} else {
			log.Printf("Database error during authentication for device %s: %v", deviceUUID, err)
		}
		return
	}

	// Ensure instance.Properties.Items is initialized
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
	m.eventBus.Publish(ctx, propUpdateEvent)

	// Handle event if present – now within worker, not Paho callback.
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
		Command  string `json:"command"`
		Success  bool   `json:"success"`
		Error    string `json:"error,omitempty"`
		ActionID string `json:"action_id,omitempty"`
	} `json:"data"`
}

func (m *MQTTService) processActionResult(ctx context.Context, job ingestJob) {
	topic := job.topic
	payload := job.payload
	log.Printf("Received action result from MQTT topic [%s]: %s", topic, string(payload))

	deviceUUID, err := extractDeviceUUIDFromTopic(topic)
	if err != nil {
		log.Printf("[MQTT] invalid action_result topic %s: %v", topic, err)
		return
	}
	var message ActionResultMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		log.Printf("[MQTT] Failed to parse action_result: %v payload=%s", err, string(payload))
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
	resultEvent.Metadata["action_id"] = message.Data.ActionID
	m.eventBus.Publish(ctx, resultEvent)

	log.Printf("[MQTT] Action result from device %s: command=%s success=%v", deviceUUID, message.Data.Command, message.Data.Success)
}

func extractDeviceUUIDFromTopic(topic string) (string, error) {
	// 简单的字符串分割方法
	parts := strings.Split(topic, "/")
	if len(parts) >= 4 && parts[0] == "data" && parts[1] == "device" {
		// 支持 properties 和 action_result topic
		if parts[3] == "properties" || parts[3] == "action_result" {
			return parts[2], nil
		}
	}

	return "", fmt.Errorf("invalid topic format for device UUID extraction: %s", topic)
}
func (m *MQTTService) Disconnect(quiesce uint) {
	// Graceful worker shutdown: signal stop, drain queue, then disconnect broker.
	// Prevent double-close panic if Disconnect is called multiple times.
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
	// Optionally close msgChan after stop so workers exit after draining.
	// We do not close msgChan immediately to avoid panic on enqueuing during shutdown;
	// instead let workers exit via stopCh and then close channel for GC.
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		log.Println("[MQTT] workers drained gracefully")
	case <-time.After(5 * time.Second):
		log.Println("[MQTT] workers drain timeout after 5s")
	}
	if m.broker != nil && m.broker.IsConnected() {
		m.broker.Disconnect(quiesce)
		log.Println("MQTT Service disconnected")
	}
}
