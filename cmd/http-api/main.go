package http_api

import (
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/handler"
	"OMEGA3-IOT/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
)

// @title IOT HTTP API
// @version 0.1
// @description IOT device management API
// @host localhost:1222
// @BasePath /api/v1

func Run(mqttService *service.MQTTService, userHandler *handler.UserHandler, deviceHandler *handler.DeviceHandler, config config.Config, deviceService *service.DeviceService, deviceShareService *service.DeviceShareService) error {

	log.Println("[HTTP_API] Run function called")

	r := gin.Default()

	// 正确配置 CORS（生产环境应限制 AllowOrigins）
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"}, // 开发环境可用，生产环境替换为具体域名
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	handler.RegRoutes(r, userHandler, deviceHandler, deviceService, deviceShareService, mqttService)

	log.Println("Starting server on :" + config.Server.Port)
	return r.Run(":" + config.Server.Port)
}
