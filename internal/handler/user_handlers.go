package handler

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.Register(input.Username, input.Password, c.ClientIP())
	if err != nil {
		if err == gorm.ErrDuplicatedKey {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username already taken"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User created successfully",
		"user":    user,
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var input model.LoginUser
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, user, err := h.userService.Login(input.Username, input.Password, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Login successful",
		"access_token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.UserName,
			"role":     user.Role,
		},
	})
}

func (h *UserHandler) GetUserInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	user, err := h.userService.GetUserInfoByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"username":   user.UserName,
			"role":       user.Role,
			"created_at": user.CreatedAt,
			"last_seen":  user.LastSeen,
			"last_ip":    user.IP,
		},
	})
}

func (h *UserHandler) BindDeviceByRegCode(c *gin.Context) {
	var input model.BindDeviceByRegCode
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	device, err := h.userService.BindDeviceByRegCode(userUUID.(string), input.RegCode, input.DeviceNick, input.DeviceRemark)
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
			"remark":      device.Remark,
			"verify_hash": device.VerifyHash,
		},
	})
}
