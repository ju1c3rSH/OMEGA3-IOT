package service

import (
	"OMEGA3-IOT/internal/model"
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
	broker    mqtt.Client
	db        *gorm.DB
	deviceSvc *DeviceService
}

type DeviceMessage struct {
	VerifyCode string `json:"verify_code"`
	TimeStamp  int64  `json:"timestamp"`
	Data       Data   `json:"data"`
}

type Data struct {
	Properties map[string]model.PropertyItem `json:"properties"`
	Event      model.Event                   `json:"event"`
	Action     model.Action                  `json:"action"`
}
type Publisher interface {
	PublishActionToDevice(deviceUUID string, actionName string, payload interface{}) error
}

func NewMQTTService(brokerURL string, db *gorm.DB) (*MQTTService, error) {
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

	deviceSvc := NewDeviceService()
	service := &MQTTService{
		broker:    client,
		db:        db,
		deviceSvc: deviceSvc,
	}
	service.setupSubscription()
	return service, nil
}
func (m *MQTTService) PublishActionToDevice(deviceUUID string, commandName string, payload interface{}) error {
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
		log.Printf("Successfully subscribed to topic [data/device/+/property]")
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

	var instance model.Instance
	dbSession := m.db.Session(&gorm.Session{})
	if err := dbSession.Where("instance_uuid = ? AND verify_hash = ?", deviceUUID, hashedVerifyCode).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Unauthorized access attempt: No device found with UUID %s and provided verify code (hash: %s)", deviceUUID, hashedVerifyCode)
		} else {
			log.Printf("Database error during authentication for device %s: %v", deviceUUID, err)
		}
		return
	}

	// 确保 instance.Properties.Items 已初始化
	if instance.Properties.Items == nil {
		instance.Properties.Items = make(map[string]*model.PropertyItem)
	}

	if err := m.updateDeviceProperties(instance, rawPropsData); err != nil {

	}

}
func (m *MQTTService) updateDeviceProperties(instance model.Instance, data map[string]model.PropertyItem) error {

	if instance.Properties.Items == nil {
		instance.Properties.Items = make(map[string]*model.PropertyItem)
	}
	for key, value := range data {
		//1创建一个新的 PropertyItem 指针，并复制值
		//不担心复用，可以直接取地址 &value，但要注意循环变量作用域问题
		//更安全的方式是创建副本：
		valueCopy := value
		va := valueCopy.Value
		instance.Properties.Items[key].Value = va
	}
	instance.LastSeen = time.Now().Unix()
	instance.UpdatedAt = time.Now()
	instance.Online = true
	dbSession := m.db.Session(&gorm.Session{})
	if err := dbSession.Save(instance).Error; err != nil {
		return fmt.Errorf("failed to save updated instance %s to database: %w", instance.InstanceUUID, err)
	}

	log.Printf("Database record for device %s updated with new properties.", instance.InstanceUUID)
	return nil
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
