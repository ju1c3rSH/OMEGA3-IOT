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
			ExpiresAt     *int64 `json:"expires_at"`                                                //使用*int64永不过期 (nil)
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			response := types.NewErrorResponse(http.StatusBadRequest, "Invalid or missing query parameter", err.Error())
			c.JSON(http.StatusBadRequest, response)
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
