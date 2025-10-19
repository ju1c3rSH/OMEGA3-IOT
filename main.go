package main

import (
	http_api "OMEGA3-IOT/cmd/http-api"
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/service"
	"fmt"
	"log"
)

// TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

var globalMQTTService *service.MQTTService // 全局 MQTT 服务变量

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
	httpApiErr := http_api.Run()
	if httpApiErr != nil {
		log.Panicf("Error loading config: %v", httpApiErr)
	}
	for i := 1; i <= 5; i++ {
		//TIP <p>To start your debugging session, right-click your code in the editor and select the Debug option.</p> <p>We have set one <icon src="AllIcons.Debugger.Db_set_breakpoint"/> breakpoint
		// for you, but you can always add more by pressing <shortcut actionId="ToggleLineBreakpoint"/>.</p>
		fmt.Println("i =", 100/i)
	}
	//brokerURL := cfg.MQTT.Broker.Address()
	brokerURL := "tcp://yuyuko.food:1883"
	globalMQTTService, err := service.NewMQTTService(brokerURL, db.DB)
	if err != nil {
		log.Fatalf("Failed to initialize MQTT service: %v", err)
	}

	// 确保程序退出时断开 MQTT 连接
	defer func() {
		if globalMQTTService != nil {
			globalMQTTService.Disconnect(250) // 250ms 是断开前等待的时间
			log.Println("MQTT Service disconnected on exit.")
		}
	}()
}
