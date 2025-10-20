package handler

import (
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"net/http"
)

var deviceService *service.DeviceService

func init() {
	deviceService = service.NewDeviceService()
}

func AddDevice(c *gin.Context) {
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

func DeviceRegisterAnonymously(c *gin.Context) {
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
