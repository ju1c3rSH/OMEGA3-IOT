package handler

import (
	"OMEGA3-IOT/internal/handler/MiddleWares"
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type, Access-Control-Allow-Origin")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func RegRoutes(router *gin.Engine, userHandler *UserHandler, deviceHandler *DeviceHandler, logHandler *logger.LogHandler, deviceService *service.DeviceService, deviceShareService *service.DeviceShareService, mqttService *service.MQTTService) {
	rateLimiter := MiddleWares.NewRateLimiter(15, time.Minute)

	v1 := router.Group("/api/v1", Cors(), rateLimiter.RateLimitMiddleware())

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
			userProtected.POST("/addDevice", deviceHandler.AddDevice)
			userProtected.POST("/bindDeviceByRegCode", userHandler.BindDeviceByRegCode)
		}
	}

	protected := v1.Group("/")
	protected.Use(MiddleWares.JwtAuthMiddleWare())
	{
		protected.POST("/devices/:instance_uuid/getHistoryData", MiddleWares.DeviceAccessMiddleware(*deviceShareService, "read"), GetDeviceHistoryHandlerFactory(deviceService))
		protected.POST("/devices/:instance_uuid/actions", MiddleWares.DeviceAccessMiddleware(*deviceShareService, "write"), SendActionHandlerFactory(mqttService))
		protected.GET("/devices/accessible", GetAccessibleDevicesHandlerFactory(deviceShareService))
		protected.POST("/devices/:instance_uuid/share", MiddleWares.DeviceAccessMiddleware(*deviceShareService, "write"), ShareDeviceHandlerFactory(deviceShareService))
	}

	deviceGroup := v1.Group("/device")
	{
		deviceGroup.POST("/deviceRegisterAnon", deviceHandler.DeviceRegisterAnonymously)
	}

	logGroup := v1.Group("/logs")
	logGroup.Use(MiddleWares.JwtAuthMiddleWare())
	{
		logGroup.GET("/device", logHandler.QueryDeviceLogs)
		logGroup.POST("/device/upload", logHandler.UploadDeviceLog)
		logGroup.GET("/user", logHandler.QueryUserLogs)
	}

}
