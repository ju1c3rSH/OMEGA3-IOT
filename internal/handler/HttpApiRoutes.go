package handler

import (
	"OMEGA3-IOT/internal/handler/MiddleWares"
	"OMEGA3-IOT/internal/service"

	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ✅ 允许所有来源（调试用）
		c.Header("Access-Control-Allow-Origin", "*")

		// ✅ 允许的请求方法
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// ✅ 允许的请求头
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		// ✅ 暴露给前端的响应头（关键！）
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type, Access-Control-Allow-Origin")

		// ✅ 预检请求缓存 12 小时
		c.Header("Access-Control-Max-Age", "86400")

		// 如果是 OPTIONS 预检请求，直接返回 200
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}

		c.Next()
	}
}
func RegRoutes(router *gin.Engine, userHandler *UserHandler, deviceService *service.DeviceService) {
	apiGroup := router.Group("/api/v1")

	apiGroup.GET("/GetTest", func(c *gin.Context) {
		msg := c.DefaultQuery("msg", "hello world")
		c.JSON(200, gin.H{
			"message": msg,
		})
	})
	apiGroup.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	apiGroup.POST("Register", userHandler.Register)

	apiGroup.POST("/Login", userHandler.Login)
	apiGroup.POST("/DeviceReg", DeviceRegisterAnonymouslyHandlerFactory(deviceService))
	protected := apiGroup.Group("")
	protected.Use(MiddleWares.JwtAuthMiddleWare())
	{
		protected.GET("/GetUserInfo", userHandler.GetUserInfo)
		protected.POST("/AddDevice", AddDeviceHandlerFactory(deviceService))
		protected.POST("/BindDeviceByRegCode", userHandler.BindDeviceByRegCode)
	}
}
