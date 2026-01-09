package handler

import (
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"net/http"
)

type DeviceHandler struct {
	deviceService      *service.DeviceService
	deviceShareService *service.DeviceShareService
}

func NewDeviceHandler(deviceService service.DeviceService, deviceShareService service.DeviceShareService) *DeviceHandler {
	return &DeviceHandler{deviceService: &deviceService,
		deviceShareService: &deviceShareService,
	}
}

func (d *DeviceHandler) AddDevice(c *gin.Context) {

	var input struct {
		Name        string `form:"name" binding:"required"`
		DeviceType  int    `form:"device_type" binding:"required"`
		Description string `form:"description,omitempty"`
	}
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
	}

	device, err := d.deviceService.AddDevice(input.Name, input.DeviceType, input.Description, userUUID.(string))
	if err != nil {
		if err == gorm.ErrDuplicatedKey {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Device name already exists"})
		}
		if err == gorm.ErrInvalidData {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported device type"})
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create device: " + err.Error()})

	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Device created successfully",
		"device": gin.H{
			"id":          device.ID,
			"uuid":        device.InstanceUUID,
			"name":        device.Name,
			"type":        device.Type,
			"online":      device.Online,
			"description": device.Description,
			"created_at":  device.AddTime,
			"last_seen":   device.LastSeen,
			"properties":  device.Properties.Items,
		},
	})
}

func (d *DeviceHandler) DeviceRegisterAnonymously(c *gin.Context) {
	var input struct {
		DeviceTypeID int `form:"device_type_id" binding:"required"`
	}

	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing query parameter", "details": err.Error()})
		return
	}
	verifyCode, err := utils.GenerateVerifyCode()
	if err != nil {
		log.Printf("Failed to generate verify code: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate verification code"})
		return
	}

	//verifyHash := utils.HashVerifyCode(verifyCode)

	record, err := d.deviceService.RegisterDeviceAnonymously(input.DeviceTypeID, verifyCode)
	{
		if err != nil {
			if err == gorm.ErrDuplicatedKey {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Device name already exists"})
				return
			}
			if err == gorm.ErrInvalidData {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported device type"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create device: " + err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Device Registered successfully",
		"device": gin.H{
			"id":          record.ID,
			"uuid":        record.DeviceUUID,
			"reg_code":    record.RegCode,
			"type":        record.DeviceTypeID,
			"expires_at":  record.ExpiresAt,
			"verify_code": verifyCode,
		},
	})
}

func AddDeviceHandlerFactory(deviceService *service.DeviceService) gin.HandlerFunc {
	return func(c *gin.Context) {

		var input struct {
			Name        string `form:"name" binding:"required"`
			DeviceType  int    `form:"device_type" binding:"required"`
			Description string `form:"description,omitempty"`
		}
		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userUUID, exists := c.Get("user_uuid")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		device, err := deviceService.AddDevice(input.Name, input.DeviceType, input.Description, userUUID.(string))
		if err != nil {
			if err == gorm.ErrDuplicatedKey {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Device name already exists"})
				return
			}
			if err == gorm.ErrInvalidData {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported device type"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create device: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Device created successfully",
			"device": gin.H{
				"id":          device.ID,
				"uuid":        device.InstanceUUID,
				"name":        device.Name,
				"type":        device.Type,
				"online":      device.Online,
				"description": device.Description,
				"created_at":  device.AddTime,
				"last_seen":   device.LastSeen,
				"properties":  device.Properties.Items,
			},
		})
	}
}
func DeviceRegisterAnonymouslyHandlerFactory(deviceService *service.DeviceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			DeviceTypeID int `form:"device_type_id" binding:"required"`
		}

		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing query parameter", "details": err.Error()})
			return
		}
		verifyCode, err := utils.GenerateVerifyCode()
		if err != nil {
			log.Printf("Failed to generate verify code: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate verification code"})
			return
		}

		//verifyHash := utils.HashVerifyCode(verifyCode)

		record, err := deviceService.RegisterDeviceAnonymously(input.DeviceTypeID, verifyCode)
		{
			if err != nil {
				if err == gorm.ErrDuplicatedKey {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Device name already exists"})
					return
				}
				if err == gorm.ErrInvalidData {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported device type"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create device: " + err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": "Device Registered successfully",
			"device": gin.H{
				"id":          record.ID,
				"uuid":        record.DeviceUUID,
				"reg_code":    record.RegCode,
				"type":        record.DeviceTypeID,
				"expires_at":  record.ExpiresAt,
				"verify_code": verifyCode,
			},
		})
	}
}

// POST /api/v1/devices/{instance_uuid}/share
func ShareDeviceHandlerFactory(deviceShareService *service.DeviceShareService) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceUUID := c.Param("instance_uuid")
		userUUID, exists := c.Get("user_uuid")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		var input struct {
			ShareWithUUID string `json:"shared_with_uuid" binding:"required"`
			Permission    string `json:"permission" binding:"required,oneof=read write read_write"` // 使用 binding 验证权限
			ExpiresAt     *int64 `json:"expires_at"`                                                //使用*int64永不过期 (nil)
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 调用 Service 方法 (假设 Service 方法签名已调整为接收 *time.Time)
		// err := deviceShareService.ShareDevice(instanceUUID, input.ShareWithUUID, input.Permission, expiresAt, userUUID.(string))
		// if err != nil {
		//     c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to share device: " + err.Error()})
		//     return
		// }

		// 临时：使用 ShareDeviceService 中假设的方法 (需要你根据实际情况调整)
		// 例如，如果 ShareDevice 方法接收 *time.Time
		err := deviceShareService.ShareDevice(instanceUUID, userUUID.(string), input.ShareWithUUID, *input.ExpiresAt, input.Permission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to share device: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Device shared successfully"})
	}
}

//GET /api/v1/devices/accessible

func GetAccessibleDevicesHandlerFactory(deviceShareService *service.DeviceShareService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID, exists := c.Get("user_uuid")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		response, err := deviceShareService.GetAccessibleDevices(userUUID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get devices: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, response)
	}
}
