package handler

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/types"
	"OMEGA3-IOT/internal/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"net/http"
	"time"
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
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid or missing query parameter", err.Error())
		c.JSON(http.StatusBadRequest, response)
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated")
		c.JSON(http.StatusUnauthorized, response)
	}

	device, err := d.deviceService.AddDevice(input.Name, input.DeviceType, input.Description, userUUID.(string))
	if err != nil {
		if err == gorm.ErrDuplicatedKey {
			response := types.NewErrorResponse(http.StatusBadRequest, "Device name already exists", err.Error())
			c.JSON(http.StatusBadRequest, response)
		}
		if err == gorm.ErrInvalidData {
			response := types.NewErrorResponse(http.StatusBadRequest, "Unsupported device type", err.Error())
			c.JSON(http.StatusBadRequest, response)
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to create device", err.Error())
		c.JSON(http.StatusInternalServerError, response)

	}

	response := types.NewSuccessResponseWithCode(device, http.StatusOK, "Device Created Successfully.")
	c.JSON(http.StatusOK, response)
}

func (d *DeviceHandler) DeviceRegisterAnonymously(c *gin.Context) {
	var input struct {
		DeviceTypeID int `form:"device_type_id" binding:"required"`
	}

	if err := c.ShouldBind(&input); err != nil {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid or missing query parameter", err.Error())
		c.JSON(http.StatusBadRequest, response)
		return
	}
	verifyCode, err := utils.GenerateVerifyCode()
	if err != nil {
		log.Printf("Failed to generate verify code: %v", err)
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to generate verification code", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	//verifyHash := utils.HashVerifyCode(verifyCode)

	record, err := d.deviceService.RegisterDeviceAnonymously(input.DeviceTypeID, verifyCode)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			response := types.NewErrorResponse(http.StatusBadRequest, "Device name already exists", err.Error())
			c.JSON(http.StatusBadRequest, response)
			return
		}
		if errors.Is(err, gorm.ErrInvalidData) {
			response := types.NewErrorResponse(http.StatusBadRequest, "Unsupported device type", err.Error())
			c.JSON(http.StatusBadRequest, response)
			return
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to create device", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	response := types.NewSuccessResponseWithCode(gin.H{
		"device": gin.H{
			"id":          record.ID,
			"uuid":        record.DeviceUUID,
			"reg_code":    record.RegCode,
			"type":        record.DeviceTypeID,
			"expires_at":  record.ExpiresAt,
			"verify_code": verifyCode,
		},
	}, http.StatusOK, "Device Registered successfully")
	c.JSON(http.StatusOK, response)
}

func AddDeviceHandlerFactory(deviceService *service.DeviceService) gin.HandlerFunc {
	return func(c *gin.Context) {

		var input struct {
			Name        string `form:"name" binding:"required"`
			DeviceType  int    `form:"device_type" binding:"required"`
			Description string `form:"description,omitempty"`
		}
		if err := c.ShouldBind(&input); err != nil {
			response := types.NewErrorResponse(http.StatusBadRequest, "Invalid or missing query parameter", err.Error())
			c.JSON(http.StatusBadRequest, response)
			return
		}

		userUUID, exists := c.Get("user_uuid")
		if !exists {
			response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated")
			c.JSON(http.StatusUnauthorized, response)
			return
		}

		device, err := deviceService.AddDevice(input.Name, input.DeviceType, input.Description, userUUID.(string))
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				response := types.NewErrorResponse(http.StatusBadRequest, "Device name already exists", err.Error())
				c.JSON(http.StatusBadRequest, response)
				return
			}
			if errors.Is(err, gorm.ErrInvalidData) {
				response := types.NewErrorResponse(http.StatusBadRequest, "Unsupported device type", err.Error())
				c.JSON(http.StatusBadRequest, response)
				return
			}
			response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to create device", err.Error())
			c.JSON(http.StatusInternalServerError, response)
			return
		}
		response := types.NewSuccessResponseWithCode(device, http.StatusOK, "Device created successfully")
		c.JSON(http.StatusOK, response)

	}
}
func DeviceRegisterAnonymouslyHandlerFactory(deviceService *service.DeviceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			DeviceTypeID int `form:"device_type_id" binding:"required"`
		}

		if err := c.ShouldBind(&input); err != nil {
			response := types.NewErrorResponse(http.StatusBadRequest, "Invalid or missing query parameter", err.Error())
			c.JSON(http.StatusBadRequest, response)
			return
		}
		verifyCode, err := utils.GenerateVerifyCode()
		if err != nil {
			log.Printf("Failed to generate verify code: %v", err)
			response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to generate verification code", err.Error())
			c.JSON(http.StatusInternalServerError, response)
			return
		}

		//verifyHash := utils.HashVerifyCode(verifyCode)

		record, err := deviceService.RegisterDeviceAnonymously(input.DeviceTypeID, verifyCode)
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				response := types.NewErrorResponse(http.StatusBadRequest, "Device name already exists", err.Error())
				c.JSON(http.StatusBadRequest, response)
				return
			}
			if errors.Is(err, gorm.ErrInvalidData) {
				response := types.NewErrorResponse(http.StatusBadRequest, "Unsupported device type", err.Error())
				c.JSON(http.StatusBadRequest, response)
				return
			}
			response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to create device", err.Error())
			c.JSON(http.StatusInternalServerError, response)
			return
		}
		response := types.NewSuccessResponseWithCode(gin.H{
			"device": gin.H{
				"id":          record.ID,
				"uuid":        record.DeviceUUID,
				"reg_code":    record.RegCode,
				"type":        record.DeviceTypeID,
				"expires_at":  record.ExpiresAt,
				"verify_code": verifyCode,
			},
		}, http.StatusOK, "Device Registered successfully")
		c.JSON(http.StatusOK, response)
	}
}

// POST /devices/:instance_uuid/actions
func SendActionHandlerFactory(mqttService *service.MQTTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceUUID := c.Param("instance_uuid")
		if instanceUUID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing query parameter"})
			return
		}

		var input struct {
			Command string                 `json:"command" binding:"required"`
			Params  map[string]interface{} `json:"params,omitempty"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			response := types.NewErrorResponse(http.StatusBadRequest, "Invalid or missing query parameter", err.Error())
			c.JSON(http.StatusBadRequest, response)
			return
		}

		actionPayload := model.Action{
			Command:   input.Command,
			Params:    input.Params,
			Timestamp: time.Now().Unix(),
		}

		err := mqttService.PublishActionToDevice(instanceUUID, actionPayload.Command, actionPayload)
		if err != nil {
			response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to send action", err.Error())
			c.JSON(http.StatusInternalServerError, response)
			return
		}
		response := types.NewSuccessResponseWithCode(gin.H{
			"instance_uuid": instanceUUID,
			"command":       input.Command,
		}, http.StatusOK, "Action sent successfully")
		c.JSON(http.StatusOK, response)
	}
}

// POST /devices/:instance_uuid/getHistoryData
func GetDeviceHistoryHandlerFactory(deviceService *service.DeviceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceUUID := c.Param("instance_uuid")
		if instanceUUID == "" {
			response := types.NewErrorResponse(http.StatusBadRequest, "Missing instance_uuid", "")
			c.JSON(http.StatusBadRequest, response)
			return
		}

		userUUID, exists := c.Get("user_uuid")
		if !exists {
			response := types.NewErrorResponse(http.StatusUnauthorized, "Unauthorized: missing user_uuid", "")
			c.JSON(http.StatusUnauthorized, response)
			return
		}
		_, ok := userUUID.(string)
		if !ok {
			response := types.NewErrorResponse(http.StatusUnauthorized, "Unauthorized: invalid user_uuid format", "")
			c.JSON(http.StatusUnauthorized, response)
			return
		}

		var input struct {
			StartTimestamp int64    `json:"start_timestamp" binding:"required"`
			EndTimestamp   int64    `json:"end_timestamp" binding:"required"`
			Properties     []string `json:"properties,omitempty"`
			Limit          int      `json:"limit,omitempty" binding:"max=5000,min=1"`
			Offset         int      `json:"offset,omitempty" binding:"min=0"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			response := types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error())
			c.JSON(http.StatusBadRequest, response)
			return
		}

		if input.StartTimestamp >= input.EndTimestamp {
			response := types.NewErrorResponse(http.StatusBadRequest, "start_timestamp must be less than end_timestamp", "")
			c.JSON(http.StatusBadRequest, response)
			return
		}

		if input.EndTimestamp-input.StartTimestamp > 15552000 {
			response := types.NewErrorResponse(http.StatusUnprocessableEntity, "Time range exceeds maximum allowed (30 days)", "")
			c.JSON(http.StatusUnprocessableEntity, response)
			return
		}

		if input.Limit <= 0 {
			input.Limit = 1000
		}
		if input.Offset < 0 {
			input.Offset = 0
		}

		historyData, err := deviceService.GetDeviceHistoryData(
			instanceUUID,
			input.StartTimestamp,
			input.EndTimestamp,
			input.Limit,
			input.Offset,
			input.Properties,
		)
		if err != nil {
			if err.Error() == "device not found" {
				response := types.NewErrorResponse(http.StatusNotFound, "Device not found", "")
				c.JSON(http.StatusNotFound, response)
				return
			}
			if err.Error() == "permission denied" {
				response := types.NewErrorResponse(http.StatusForbidden, "Access denied: insufficient permissions", "")
				c.JSON(http.StatusForbidden, response)
				return
			}
			response := types.NewErrorResponse(http.StatusInternalServerError, "Internal server error", err.Error())
			c.JSON(http.StatusInternalServerError, response)
			return
		}

		returnedCount := len(*historyData)
		hasMore := returnedCount >= input.Limit

		response := types.NewSuccessResponseWithCode(gin.H{
			"instance_uuid":  instanceUUID,
			"total_count":    returnedCount, // TODO: 需要服务层返回真实总数
			"returned_count": returnedCount,
			"has_more":       hasMore,
			"records":        historyData,
		}, http.StatusOK, "Device shared successfully")
		c.JSON(http.StatusOK, response)
	}
}

// POST /api/v1/devices/{instance_uuid}/share
func ShareDeviceHandlerFactory(deviceShareService *service.DeviceShareService) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceUUID := c.Param("instance_uuid")
		userUUID, exists := c.Get("user_uuid")
		if !exists {
			response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated", "")
			c.JSON(http.StatusUnauthorized, response)
			return
		}

		var input struct {
			ShareWithUUID string `json:"shared_with_uuid" binding:"required"`
			Permission    string `json:"permission" binding:"required,oneof=read write read_write"` // 使用 binding 验证权限
			ExpiresAt     int64  `json:"expires_at"`                                                //使用*int64永不过期 (nil)
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			response := types.NewErrorResponse(http.StatusBadRequest, "Invalid or missing query parameter", err.Error())
			c.JSON(http.StatusBadRequest, response)
			return
		}

		err := deviceShareService.ShareDevice(instanceUUID, userUUID.(string), input.ShareWithUUID, input.ExpiresAt, input.Permission)
		if err != nil {
			response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to share device", err.Error())
			c.JSON(http.StatusInternalServerError, response)
			return
		}

		response := types.NewSuccessResponseWithCode("[]", http.StatusOK, "Device shared successfully")
		c.JSON(http.StatusCreated, response)
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
