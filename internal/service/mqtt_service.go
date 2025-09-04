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
	VerifyCode string                 `json:"verify_code"`
	TimeStamp  int64                  `json:"timestamp"`
	Data       map[string]interface{} `json:"data"`
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

func (m *MQTTService) setupSubscription() {
	log.Printf("MQTT Service setup subscription")
	if token := m.broker.Subscribe("data/device/+/property", 1, m.handlePropertyData); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to subscribe to data topic : %s", token.Error())
	} else {
		log.Printf("Successfully subscribed to topic [data/device/+/property]")
	}
}
func (m *MQTTService) handlePropertyData(c mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()
	log.Printf("Received property data from MQTT topic [%s] (QOS %d): %s", topic, msg.Qos(), string(payload))
	deviceUUID, _ := extractDeviceUUIDFromTopic(topic)
	var instance model.Instance
	dbSession := m.db.Session(&gorm.Session{})
	if err := dbSession.Where("instance_uuid = ?", deviceUUID).First(&instance).Error; err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			fmt.Errorf("device with uuid %s was not found in the database for: %s", deviceUUID)
		}
		_ = fmt.Errorf("device with uuid %s was not found in the database for: %s", deviceUUID, err)
	}
	var message DeviceMessage
	if err := json.Unmarshal(msg.Payload(), &message); err != nil {
		fmt.Errorf("error unmarshalling device message: %v", err)
	}
	hashedVerifyCode := utils.HashVerifyCode(message.VerifyCode)
	if err := dbSession.Where("verify_hash = ?", hashedVerifyCode).First(&instance).Error; err != nil {
		fmt.Errorf("device with uuid %s was not found in the database for: %s", deviceUUID, err)
	}
	rawPropsData, ok := message.Data["properties"]
	if !ok {
		log.Printf("Warning: 'properties' field missing in message data from device %s", deviceUUID)
	}

	// 类型断言为 map[string]interface{}
	_, ok = rawPropsData.(map[string]interface{})
	if !ok {
		log.Printf("Error: 'properties' field in message data is not a map for device %s", deviceUUID)

	}

	// 确保 instance.Properties.Items 已初始化
	if instance.Properties.Items == nil {
		instance.Properties.Items = make(map[string]*model.PropertyItem)
	}
	/*
		if err := m.updateDeviceProperties(instance, message.Data); err != nil {
		}

	*/

}
func (m *MQTTService) updateDeviceProperties(instance model.Instance, data map[string]interface{}) error {

	if instance.Properties.Items == nil {
		instance.Properties.Items = make(map[string]*model.PropertyItem)

	}
	return nil
}
func extractDeviceUUIDFromTopic(topic string) (string, error) {
	// 简单的字符串分割方法
	parts := strings.Split(topic, "/")
	if len(parts) >= 4 && parts[0] == "data" && parts[1] == "device" && parts[3] == "property" {
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
