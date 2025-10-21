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

// var globalMQTTService *service.MQTTService // 全局 MQTT 服务变量 不用了 用依赖注入
var iotdbClient *db.IOTDBClient

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

	if cfg.Server.Port == "" {
		log.Fatalf("[Main] Server port not configured in config file")
	}

	log.Println("[Main] Device types loaded successfully")
	db.InitDB(cfg)
	//iotdbClient, _ = db.NewIotDBFromConfig(cfg)
	iotdbClient, err := db.NewIotDBFromConfig(cfg)
	if err != nil {
		log.Fatalf("[Main] Failed to create IoTDB client: %v", err)
	}
	if err := iotdbClient.InitializeSchema(); err != nil {
		log.Fatalf("[Main] Failed to initialize IoTDB schema: %v", err)
	}
	defer iotdbClient.Close()

	deviceService := service.NewDeviceService(db.DB, iotdbClient)

	brokerURL := "tcp://yuyuko.food:1883"
	mqttService, err := service.NewMQTTService(brokerURL, deviceService)
	if err != nil {
		log.Fatalf("[Main] Failed to initialize MQTT service: %v", err)
	}
	defer mqttService.Disconnect(250)

	log.Println("[Main] Before creating UserService")
	userService = service.NewUserService(mqttService, db.DB, iotdbClient)
	log.Println("[Main] UserService created")
	log.Println("[Main] Before creating UserHandler")
	userHandler := handler.NewUserHandler(userService)
	log.Println("[Main] UserHandler created")

	log.Println("[Main] Before calling http_api.Run")
	httpApiErr := http_api.Run(userHandler, cfg, deviceService)
	log.Println("[Main] After calling http_api.Run")

	if httpApiErr != nil {
		log.Panicf("[Main] Error starting HTTP server: %v", httpApiErr)
	}
}
