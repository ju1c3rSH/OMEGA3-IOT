package MiddleWares

import (
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/types"
	"github.com/gin-gonic/gin"
	"net/http"
)

// DeviceAccessMiddleware checks device access via DeviceShareService.
// NOTE: deviceShareService is *service.DeviceShareService (pointer), not value.
// Pass-by-value would copy the entire struct (including sync.Map cache, mutexes, etc.)
// on every request where the middleware is instantiated, causing extra allocation
// and potential race if the struct ever contains non-copy-safe fields.
// See https://gin-gonic.com/en/docs/middleware/goroutines-inside-a-middleware/
// and https://stackoverflow.com/questions/75913434/how-to-inject-a-repo-or-service-in-a-middleware-in-a-clean-way
func DeviceAccessMiddleware(deviceShareService *service.DeviceShareService, requiedPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceUUID := c.Param("instance_uuid")
		userUUID, exists := c.Get("user_uuid")
		if !exists {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "You have no access to this device"))
			c.Abort()
			return
		}

		hasAccess, err := deviceShareService.CheckDeviceAccess(instanceUUID, userUUID.(string), requiedPermission)
		if err != nil || !hasAccess {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "You have no access to this device"))
			c.Abort()
			return
		}
		c.Next()
	}

}
