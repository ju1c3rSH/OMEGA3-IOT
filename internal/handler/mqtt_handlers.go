package handler

import (
	"gorm.io/gorm"
)

// 数据库初始化之后调用
func InitMQTTHandler(brokerURL string, db *gorm.DB) error {

	// mqttService.SomePublicSetupSubscriptionMethod() // 如果 service 包提供了公共方法
	return nil
}
