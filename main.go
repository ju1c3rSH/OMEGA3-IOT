package main

import (
	http_api "OMEGA3-IOT/cmd/http-api"
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/handler"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/service"
	"fmt"
	"log"
)

// TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

//var globalMQTTService *service.MQTTService // 全局 MQTT 服务变量 不用了 用依赖注入

var userService *service.UserService

func main() {
	//TIP <p>Press <shortcut actionId="ShowIntentionActions"/> when your caret is at the underlined text
	// to see how GoLand suggests fixing the warning.</p><p>Alternatively, if available, click the lightbulb to view possible fixes.</p>
	s := "gopher"
	fmt.Printf("Hello and welcome, %s!\n", s)
	//cfg, err := config.LoadConfig(".")
	cfg, _ := config.LoadConfig("./internal/config")
	if err := model.GlobalDeviceTypeManager.LoadDeviceTypeFromYAML("./internal/config/device_type_list.yaml"); err != nil {
		log.Fatalf("Failed to load device types: %v", err)
	}

	log.Println("Device types loaded successfully")
	db.InitDB(cfg)

	//brokerURL := cfg.MQTT.Broker.Address()
	brokerURL := "tcp://yuyuko.food:1883"
	globalMQTTService, err := service.NewMQTTService(brokerURL, db.DB)
	if err != nil {
		log.Fatalf("Failed to initialize MQTT service: %v", err)
	}
	userService = service.NewUserService(globalMQTTService)
	userHandler := handler.NewUserHandler(userService)
	httpApiErr := http_api.Run(userHandler)
	if httpApiErr != nil {
		log.Panicf("Error loading config: %v", httpApiErr)
	}

	defer func() {
		if globalMQTTService != nil {
			globalMQTTService.Disconnect(250) // 250ms 是断开前等待的时间
			log.Println("MQTT Service disconnected on exit.")
		}
	}()
}
