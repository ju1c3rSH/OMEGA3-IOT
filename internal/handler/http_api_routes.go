package handler

import (
	"OMEGA3-IOT/internal/handler/MiddleWares"
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/push"
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/types"
	"captcha"

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

func RegRoutes(router *gin.Engine, userHandler *UserHandler, deviceHandler *DeviceHandler, logHandler *logger.LogHandler, deviceService *service.DeviceService, deviceShareService *service.DeviceShareService, deviceFolderHandler *DeviceFolderHandler, mqttService *service.MQTTService, jwtAuth *MiddleWares.JWTAuth, pushHandler *push.PushHandler, userGroupHandler *UserGroupHandler, adminHandler *AdminHandler, captchaService *captcha.CaptchaService) {
	router.Static("/uploads", "./uploads")
	router.StaticFile("/debugger", "./debugger/index.html")
	router.Static("/debugger/assets", "./debugger/assets")

	v1 := router.Group("/api/v1", Cors(), MiddleWares.NewRateLimiter(15, 60).RateLimitMiddleware())

	v1.GET("/test", func(c *gin.Context) {
		msg := c.DefaultQuery("msg", "hello world")
		c.JSON(200, types.NewSuccessResponseWithCode(gin.H{"message": msg}, 200, "OK"))
	})
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, types.NewSuccessResponseWithCode(gin.H{"status": "ok"}, 200, "OK"))
	})

	// EinkiCaptcha routes
	RegisterCaptchaRoutes(v1, captchaService)

	userGroup := v1.Group("/users")
	{
		userGroup.POST("/challenge", userHandler.Challenge)
		userGroup.POST("/register", userHandler.Register)
		userGroup.POST("/login", userHandler.Login)

		userProtected := userGroup.Group("")
		userProtected.Use(jwtAuth.JwtAuthMiddleWare())
		{
			userProtected.POST("/logout", userHandler.Logout)
			userProtected.GET("/getUserAllDevices", userHandler.GetUserAllDevices)
			userProtected.GET("/info", userHandler.GetUserInfo)
			userProtected.PUT("/profile", userHandler.UpdateProfile)
			userProtected.POST("/avatar", userHandler.UploadAvatar)
			userProtected.DELETE("/avatar", userHandler.ResetAvatar)
			userProtected.POST("/addDevice", deviceHandler.AddDevice)
			userProtected.POST("/bindDeviceByRegCode", userHandler.BindDeviceByRegCode)
		}
	}

	protected := v1.Group("/")
	protected.Use(jwtAuth.JwtAuthMiddleWare())
	{
		protected.POST("/devices/:instance_uuid/getHistoryData", MiddleWares.DeviceAccessMiddleware(*deviceShareService, "read"), GetDeviceHistoryHandlerFactory(deviceService))
		protected.POST("/devices/:instance_uuid/actions", MiddleWares.DeviceAccessMiddleware(*deviceShareService, "write"), SendActionHandlerFactory(mqttService, deviceService))
		protected.GET("/devices/accessible", GetAccessibleDevicesHandlerFactory(deviceShareService))
		protected.POST("/devices/:instance_uuid/share", MiddleWares.DeviceAccessMiddleware(*deviceShareService, "write"), ShareDeviceHandlerFactory(deviceShareService))

		// Device Folder routes (organizational grouping of devices)
		protected.POST("/devices/folders", deviceFolderHandler.CreateFolder)
		protected.POST("/devices/:instance_uuid/folders", deviceFolderHandler.AddDeviceToFolder)
		protected.DELETE("/devices/:instance_uuid/folders/:folder_uuid", deviceFolderHandler.RemoveDeviceFromFolder)
		protected.GET("/devices/folders/:folder_uuid/devices", deviceFolderHandler.GetFolderDevices)
		protected.DELETE("/devices/folders/:folder_uuid", deviceFolderHandler.DeleteFolder)
	}

	usersMe := v1.Group("/users/me")
	usersMe.Use(jwtAuth.JwtAuthMiddleWare())
	{
		usersMe.GET("/device_folders", deviceFolderHandler.GetFolders)
	}

	deviceGroup := v1.Group("/device")
	{
		deviceGroup.POST("/deviceRegisterAnon", deviceHandler.DeviceRegisterAnonymously)
	}

	// WebSocket push channel
	wsGroup := v1.Group("/ws")
	wsGroup.Use(jwtAuth.JwtAuthMiddleWare())
	{
		wsGroup.GET("", pushHandler.HandleWebSocket)
	}

	// User Group routes
	groupRoutes := v1.Group("/groups")
	groupRoutes.Use(jwtAuth.JwtAuthMiddleWare())
	{
		// Group CRUD
		groupRoutes.POST("", userGroupHandler.CreateGroup)
		groupRoutes.GET("", userGroupHandler.GetMyGroups)
		groupRoutes.GET("/:group_uuid", userGroupHandler.GetGroup)
		groupRoutes.PUT("/:group_uuid", userGroupHandler.UpdateGroup)
		groupRoutes.DELETE("/:group_uuid", userGroupHandler.DissolveGroup)

		// Member management
		groupRoutes.GET("/:group_uuid/members", userGroupHandler.GetMembers)
		groupRoutes.POST("/:group_uuid/invite/search", userGroupHandler.SearchInvite)
		groupRoutes.POST("/:group_uuid/invite/link", userGroupHandler.CreateLinkInvite)
		groupRoutes.POST("/:group_uuid/members/:user_uuid/approve", userGroupHandler.ApproveMember)
		groupRoutes.POST("/:group_uuid/members/:user_uuid/reject", userGroupHandler.RejectMember)
		groupRoutes.DELETE("/:group_uuid/members/:user_uuid", userGroupHandler.RemoveMember)
		groupRoutes.POST("/:group_uuid/leave", userGroupHandler.LeaveGroup)
		groupRoutes.PUT("/:group_uuid/members/:user_uuid/role", userGroupHandler.UpdateMemberRole)

		// Device management
		groupRoutes.GET("/:group_uuid/devices", userGroupHandler.GetGroupDevices)
		groupRoutes.POST("/:group_uuid/devices/share", userGroupHandler.ShareDeviceToGroup)
		groupRoutes.DELETE("/:group_uuid/devices/:instance_uuid", userGroupHandler.RevokeGroupDeviceShare)

		// Policy management
		groupRoutes.GET("/:group_uuid/policy", userGroupHandler.GetPolicy)
		groupRoutes.PUT("/:group_uuid/policy", userGroupHandler.UpdatePolicy)

		// Invites
		groupRoutes.GET("/:group_uuid/invites", userGroupHandler.GetPendingInvites)
	}
	// Accept invite (no group_uuid in path)
	groupRoutes.POST("/invite/:invite_code/accept", userGroupHandler.AcceptInvite)

	// Admin routes
	adminGroup := v1.Group("/admin")
	{
		// Public: admin challenge and login
		adminGroup.POST("/challenge", adminHandler.Challenge)
		adminGroup.POST("/login", adminHandler.Login)

		// Protected: all admin endpoints require JWT + admin role
		adminProtected := adminGroup.Group("")
		adminProtected.Use(jwtAuth.JwtAuthMiddleWare(), MiddleWares.AdminAuthMiddleware())
		{
			adminProtected.POST("/logout", adminHandler.Logout)

			// Admin management (super_admin only)
			adminProtected.GET("/admins", MiddleWares.RequirePermission(model.PermAdminView), adminHandler.GetAdmins)
			adminProtected.POST("/admins", MiddleWares.RequirePermission(model.PermAdminManage), adminHandler.PromoteUser)
			adminProtected.PUT("/admins/:user_uuid", MiddleWares.RequirePermission(model.PermAdminManage), adminHandler.UpdateAdminRole)
			adminProtected.DELETE("/admins/:user_uuid", MiddleWares.RequirePermission(model.PermAdminManage), adminHandler.DemoteAdmin)

			// User management
			adminProtected.GET("/users", MiddleWares.RequirePermission(model.PermUserView), adminHandler.ListUsers)
			adminProtected.GET("/users/:user_uuid", MiddleWares.RequirePermission(model.PermUserView), adminHandler.GetUser)
			adminProtected.PUT("/users/:user_uuid", MiddleWares.RequirePermission(model.PermUserEdit), adminHandler.EditUser)
			adminProtected.PUT("/users/:user_uuid/status", MiddleWares.RequirePermission(model.PermUserStatus), adminHandler.UpdateUserStatus)
			adminProtected.DELETE("/users/:user_uuid", MiddleWares.RequirePermission(model.PermUserDelete), adminHandler.DeleteUser)
			adminProtected.POST("/users/:user_uuid/reset-password", MiddleWares.RequirePermission(model.PermUserReset), adminHandler.ResetPassword)

			// Device management
			adminProtected.GET("/devices", MiddleWares.RequirePermission(model.PermDeviceView), adminHandler.ListDevices)
			adminProtected.GET("/devices/:instance_uuid", MiddleWares.RequirePermission(model.PermDeviceView), adminHandler.GetDevice)
			adminProtected.PUT("/devices/:instance_uuid", MiddleWares.RequirePermission(model.PermDeviceEdit), adminHandler.EditDevice)
			adminProtected.DELETE("/devices/:instance_uuid", MiddleWares.RequirePermission(model.PermDeviceDelete), adminHandler.DeleteDevice)
			adminProtected.POST("/devices/:instance_uuid/transfer", MiddleWares.RequirePermission(model.PermDeviceTransfer), adminHandler.TransferDevice)

			// Group management
			adminProtected.GET("/groups", MiddleWares.RequirePermission(model.PermGroupView), adminHandler.ListGroups)
			adminProtected.GET("/groups/:group_uuid", MiddleWares.RequirePermission(model.PermGroupView), adminHandler.GetGroup)
			adminProtected.GET("/groups/:group_uuid/members", MiddleWares.RequirePermission(model.PermGroupView), adminHandler.GetGroupMembers)
			adminProtected.DELETE("/groups/:group_uuid", MiddleWares.RequirePermission(model.PermGroupManage), adminHandler.DissolveGroup)
			adminProtected.DELETE("/groups/:group_uuid/members/:user_uuid", MiddleWares.RequirePermission(model.PermGroupManage), adminHandler.RemoveGroupMember)

			// System
			adminProtected.GET("/stats/overview", MiddleWares.RequirePermission(model.PermSystemStats), adminHandler.GetStats)
			adminProtected.GET("/logs", MiddleWares.RequirePermission(model.PermSystemLogs), adminHandler.GetLogs)
		}
	}

	logGroup := v1.Group("/logs")
	logGroup.Use(jwtAuth.JwtAuthMiddleWare())
	{
		logGroup.GET("/device", logHandler.QueryDeviceLogs)
		logGroup.POST("/device/upload", logHandler.UploadDeviceLog)
		logGroup.GET("/user", logHandler.QueryUserLogs)
	}

}
