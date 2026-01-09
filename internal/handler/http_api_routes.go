package handler

import (
	"OMEGA3-IOT/internal/handler/MiddleWares"
	"OMEGA3-IOT/internal/service"
	"time"

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
func RegRoutes(router *gin.Engine, userHandler *UserHandler, deviceHandler *DeviceHandler, deviceService *service.DeviceService, deviceShareService *service.DeviceShareService, mqttService *service.MQTTService) {
	rateLimiter := MiddleWares.NewRateLimiter(15, time.Minute)

	v1 := router.Group("/api/v1", rateLimiter.RateLimitMiddleware())

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
			userProtected.POST("/bindDeviceByRegCode", userHandler.BindDeviceByRegCode)
		}
	}

	protected := v1.Group("/")
	protected.Use(MiddleWares.JwtAuthMiddleWare())
	{
		protected.POST("/devices/:instance_uuid/actions", MiddleWares.DeviceAccessMiddleware(*deviceShareService, "write"), SendActionHandlerFactory(mqttService))
		protected.GET("/devices/accessible", GetAccessibleDevicesHandlerFactory(deviceShareService))
		protected.POST("/devices/:instance_uuid/share", MiddleWares.DeviceAccessMiddleware(*deviceShareService, "write"), ShareDeviceHandlerFactory(deviceShareService))
	}

	/*
		protected := apiGroup.Group("")
		protected.Use(MiddleWares.JwtAuthMiddleWare())
		{
		    // 设备所有者操作
		    protected.POST("/devices/:instance_uuid/share", DeviceAccessMiddleware("write"), deviceHandler.ShareDevice)
		    protected.DELETE("/devices/:instance_uuid/share/:shared_with_uuid", DeviceAccessMiddleware("write"), deviceHandler.RevokeShare)

		    // 设备访问（需要read权限）
		    protected.GET("/devices/:instance_uuid", DeviceAccessMiddleware("read"), deviceHandler.GetDevice)
		    protected.GET("/devices/:instance_uuid/properties", DeviceAccessMiddleware("read"), deviceHandler.GetDeviceProperties)


		    // 获取所有可访问设备
		    protected.GET("/devices/accessible", deviceHandler.GetAccessibleDevices)
		}
	*/
	deviceGroup := v1.Group("/device")
	{

		deviceGroup.POST("/deviceRegisterAnon", DeviceRegisterAnonymouslyHandlerFactory(deviceService))

	}

}
