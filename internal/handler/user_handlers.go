package handler

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/types"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"net/http"
)

/*
	func init() {
		userService = service.NewUserService()
	}
*/
type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userSvc *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userSvc,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var input model.RegUser
	if err := c.ShouldBind(&input); err != nil {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid input", err.Error())
		c.JSON(http.StatusBadRequest, response)
		return
	}

	user, err := h.userService.Register(input.Username, input.Password, c.ClientIP())
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			response := types.NewErrorResponse(http.StatusBadRequest, "Username already taken", err.Error())
			c.JSON(http.StatusBadRequest, response)
			return
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to register user", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	userInfo := gin.H{
		"id":         user.ID,
		"username":   user.UserName,
		"role":       user.Role,
		"created_at": user.CreatedAt,
		"last_seen":  user.LastSeen,
		"last_ip":    user.IP,
	}

	response := types.NewSuccessResponseWithCode(userInfo, http.StatusCreated, "User created successfully")
	c.JSON(http.StatusCreated, response)
}

func (h *UserHandler) Login(c *gin.Context) {
	var input model.LoginUser
	if err := c.ShouldBind(&input); err != nil {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid input", err.Error())
		c.JSON(http.StatusBadRequest, response)
		return
	}

	token, user, err := h.userService.Login(input.Username, input.Password, c.ClientIP())
	if err != nil {
		response := types.NewErrorResponse(http.StatusUnauthorized, "Invalid username or password", err.Error())
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	loginInfo := gin.H{
		"access_token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.UserName,
			"role":     user.Role,
		},
	}

	response := types.NewSuccessResponseWithCode(loginInfo, http.StatusOK, "Login successful")
	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) GetUserInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "Authentication required", "")
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	user, err := h.userService.GetUserInfoByID(userID.(uint))
	if err != nil {
		response := types.NewErrorResponse(http.StatusNotFound, "User not found", err.Error())
		c.JSON(http.StatusNotFound, response)
		return
	}

	userInfo := gin.H{
		"id":         user.ID,
		"username":   user.UserName,
		"role":       user.Role,
		"created_at": user.CreatedAt,
		"last_seen":  user.LastSeen,
		"last_ip":    user.IP,
	}

	response := types.NewSuccessResponseWithCode(userInfo, http.StatusOK, "User info retrieved successfully")
	c.JSON(http.StatusOK, response)
}
func (h *UserHandler) GetUserAllDevices(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "Authentication required", "")
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	userUUIDStr, ok := userUUID.(string)
	if !ok {
		response := types.NewErrorResponse(http.StatusInternalServerError, "Invalid user UUID type in context", "")
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	responseData, err := h.userService.GetUserAllDevices(userUUIDStr)
	if err != nil {
		log.Printf("Error getting devices for user %s: %v\n", userUUIDStr, err)
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to retrieve devices", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	c.JSON(http.StatusOK, responseData)

}
func (h *UserHandler) BindDeviceByRegCode(c *gin.Context) {
	var input model.BindDeviceByRegCode
	if err := c.ShouldBind(&input); err != nil {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid input", err.Error())
		c.JSON(http.StatusBadRequest, response)
		return
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated", "")
		c.JSON(http.StatusUnauthorized, response)
		return
	}
	device, err := h.userService.BindDeviceByRegCode(userUUID.(string), input.RegCode, input.DeviceNick, input.DeviceRemark)
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

	payload := gin.H{"device": gin.H{
		"id":          device.ID,
		"uuid":        device.InstanceUUID,
		"name":        device.Name,
		"type":        device.Type,
		"online":      device.Online,
		"description": device.Description,
		"created_at":  device.AddTime,
		"last_seen":   device.LastSeen,
		"properties":  device.Properties.Items,
		"remark":      device.Remark,
		"verify_hash": device.VerifyHash,
	}}

	response := types.NewSuccessResponseWithCode(payload, http.StatusCreated, "Device created successfully")
	c.JSON(http.StatusCreated, response)
}
