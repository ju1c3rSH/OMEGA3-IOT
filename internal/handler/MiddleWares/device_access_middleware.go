package MiddleWares

import (
	"OMEGA3-IOT/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

func DeviceAccessMiddleware(deviceShareService service.DeviceShareService, requiedPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceUUID := c.Param("instance_uuid")
		userUUID, exists := c.Get("user_uuid")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "You have no access to this device"})
			c.AbortWithStatus(403)
			return
		}

		hasAccess, err := deviceShareService.CheckDeviceAccess(instanceUUID, userUUID.(string), requiedPermission)
		if err != nil || !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "You have no access to this device"})
			c.AbortWithStatus(403)
			return
		}
		c.Next()
	}

}
