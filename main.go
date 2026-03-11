package main

import (
	http_api "OMEGA3-IOT/cmd/http-api"
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/eventbus"
	"OMEGA3-IOT/internal/handler"
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"OMEGA3-IOT/internal/service"
	"fmt"

	"log"
)

// var globalMQTTService *service.MQTTService // 全局 MQTT 服务变量 不用了 用依赖注入
var iotdbClient *db.IOTDBClient

var userService *service.UserService

func main() {
	s := "gopher"
	fmt.Printf("Hello and welcome, %s!\n", s)
	//cfg, err := config.DeLoadConfig(".")
	cfg, _ := config.LoadConfig("./internal/config")
	if err := model.GlobalDeviceTypeManager.LoadDeviceTypeFromYAML("./internal/config/device_type_list.yaml"); err != nil {
		log.Fatalf("Failed to load device types: %v", err)
	}

	if cfg.Server.Port == "" {
		log.Fatalf("[Main] Server port not configured in config file")
	}

	log.Println("[Main] Device types loaded successfully")
	db.InitDB(cfg)
	iotdbClient, err := db.NewIotDBFromConfig(cfg)
	if err != nil {
		log.Fatalf("[Main] Failed to create IoTDB client: %v", err)
	}
	if err := iotdbClient.InitializeSchema(); err != nil {
		log.Fatalf("[Main] Failed to initialize IoTDB schema: %v", err)
	}
	defer iotdbClient.Close()

	// Initialize EventBus
	eventBus := eventbus.New()
	log.Println("[Main] EventBus initialized")

	// Initialize LoggerService
	loggerService := logger.NewLoggerService(iotdbClient, eventBus)
	if err := loggerService.InitializeLogSchema(); err != nil {
		log.Printf("[Main] Warning: Failed to initialize log schema: %v", err)
	}
	loggerService.Start()
	log.Println("[Main] LoggerService started")

	deviceService := service.NewDeviceService(db.DB, iotdbClient)
	newURL := fmt.Sprintf("%s://%s:%d", cfg.MQTT.Broker.Protocol, cfg.MQTT.Broker.Host, cfg.MQTT.Broker.Port)
	mqttService, err := service.NewMQTTService(newURL, deviceService, loggerService)
	if err != nil {
		log.Fatalf("[Main] Failed to initialize MQTT service: %v", err)
	}
	defer mqttService.Disconnect(250)

	// Create repositories
	userRepo := repository.NewUserRepository(db.DB)
	instanceRepo := repository.NewInstanceRepository(db.DB)
	deviceRegistrationRepo := repository.NewDeviceRegistrationRecordRepository(db.DB)

	userService = service.NewUserService(mqttService, userRepo, instanceRepo, deviceRegistrationRepo, iotdbClient, loggerService)
	log.Println("[Main] UserService created")
	userHandler := handler.NewUserHandler(userService)
	log.Println("[Main] UserHandler created")
	deviceShareService := service.NewDeviceShareService(db.DB, loggerService)
	log.Println("[Main] DeviceShareService created")
	deviceHandler := handler.NewDeviceHandler(db.DB, mqttService)

	// Create LogHandler
	logHandler := logger.NewLogHandler(loggerService)
	log.Println("[Main] LogHandler created")

	deviceGroupService := service.NewDeviceGroupService(db.DB, iotdbClient, loggerService)
	log.Println("[Main] DeviceGroupService created")
	deviceGroupHandler := handler.NewDeviceGroupHandler(deviceGroupService)
	log.Println("[Main] DeviceGroupHandler created")

	httpApiErr := http_api.Run(mqttService, userHandler, deviceHandler, logHandler, cfg, deviceService, deviceShareService, deviceGroupHandler)
	log.Println("[Main] After calling http_api.Run")
	if httpApiErr != nil {
		log.Panicf("[Main] Error starting HTTP server: %v", httpApiErr)
	}
}
