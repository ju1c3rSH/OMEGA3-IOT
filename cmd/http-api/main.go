package http_api

import (
	"OMEGA3-IOT/internal/handler"
	"github.com/gin-gonic/gin"
)

// @title IOT HTTP API
// @ver 0.1

func Run() error {
	r := gin.Default()
	r.Use(handler.Cors())
	handler.RegRoutes(r)
	return r.Run(":1222")
}
