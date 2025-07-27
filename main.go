package main

import (
	"OMEGA3-IOT/cmd/http-api"
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/model"
	"fmt"
	"log"
)

// TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>
// TODO HTTP API
// TODO MQTT
// TODO
func main() {
	//TIP <p>Press <shortcut actionId="ShowIntentionActions"/> when your caret is at the underlined text
	// to see how GoLand suggests fixing the warning.</p><p>Alternatively, if available, click the lightbulb to view possible fixes.</p>
	s := "gopher"
	fmt.Printf("Hello and welcome, %s!\n", s)
	//cfg, err := config.LoadConfig(".")
	cfg := config.LoadConfig("./internal/config")
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
}
