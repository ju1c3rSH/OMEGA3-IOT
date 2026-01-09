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
	v1 := router.Group("/api/v1")

	v1.GET("/test", func(c *gin.Context) {
		msg := c.DefaultQuery("msg", "hello world")
		c.JSON(200, gin.H{
			"message": msg,
		})
	})
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	userGroup := v1.Group("/users")
	{
		userGroup.POST("/register", userHandler.Register)
		userGroup.POST("/login", userHandler.Login)

		userProtected := userGroup.Group("")
		userProtected.Use(MiddleWares.JwtAuthMiddleWare())
		{
			userProtected.GET("/getUserAllDevices", userHandler.GetUserAllDevices)
			userProtected.GET("/info", userHandler.GetUserInfo)
			userProtected.POST("/addDevice", AddDeviceHandlerFactory(deviceService))
			userProtected.POST("bindDeviceByRegCode", userHandler.BindDeviceByRegCode)
		}
	}

	deviceGroup := v1.Group("/device")
	{
		deviceGroup.POST("/deviceRegisterAnon", DeviceRegisterAnonymouslyHandlerFactory(deviceService))

	}

}
