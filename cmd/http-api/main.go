package http_api

import (
	"OMEGA3-IOT/internal/handler"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
)

// @title IOT HTTP API
// @version 0.1
// @description IOT device management API
// @host localhost:1222
// @BasePath /api/v1

func Run() error {
	r := gin.Default()

	// 正确配置 CORS（生产环境应限制 AllowOrigins）
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"}, // 开发环境可用，生产环境替换为具体域名
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	handler.RegRoutes(r)

	log.Println("Starting server on :27015")
	return r.Run(":27015")
}
