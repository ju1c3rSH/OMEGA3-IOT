package handler

import (
	"fmt"
	"gorm.io/gorm"
)

// 数据库初始化之后调用
func InitMQTTHandler(brokerURL string, db *gorm.DB) error {
	var err error
	//mqttService, err = service.NewMQTTService(brokerURL, db)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	// mqttService.SomePublicSetupSubscriptionMethod() // 如果 service 包提供了公共方法
	return nil
}
