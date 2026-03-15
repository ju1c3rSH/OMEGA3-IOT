package handler

import (
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/types"
	"OMEGA3-IOT/internal/utils"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DeviceGroupHandler struct {
	groupService *service.DeviceGroupService
}

func NewDeviceGroupHandler(groupService *service.DeviceGroupService) *DeviceGroupHandler {
	return &DeviceGroupHandler{
		groupService: groupService,
	}
}

func (h *DeviceGroupHandler) CreateGroup(c *gin.Context) {
	var input struct {
		Name        string `json:"name" binding:"required,max=128"`
		Description string `json:"description,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error())
		c.JSON(http.StatusBadRequest, response)
		return
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated")
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	ownerID := utils.ParseUserIDFromUUID(userUUID.(string))

	group, err := h.groupService.CreateGroup(input.Name, input.Description, ownerID)
	if err != nil {
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to create group", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(group, http.StatusOK, "Group created successfully")
	c.JSON(http.StatusOK, response)
}

func (h *DeviceGroupHandler) JoinGroup(c *gin.Context) {
	deviceUUID := c.Param("instance_uuid")
	if deviceUUID == "" {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid device_uuid", "device uuid is required")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var input struct {
		GroupID string `json:"group_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error())
		c.JSON(http.StatusBadRequest, response)
		return
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated")
		c.JSON(http.StatusUnauthorized, response)
		return
	}
	groupID, err := strconv.ParseInt(input.GroupID, 10, 64)
	if err != nil {
		log.Fatal("转换失败：", err)
	}
	if err := h.groupService.JoinGroup(groupID, deviceUUID, userUUID.(string)); err != nil {
		errMsg := err.Error()
		if errMsg == "invalid group" || errMsg == "device access denied" || errMsg == "permission denied" {
			response := types.NewErrorResponse(http.StatusForbidden, "Access denied", err.Error())
			c.JSON(http.StatusForbidden, response)
			return
		}
		if errMsg == "device not found" {
			response := types.NewErrorResponse(http.StatusNotFound, "Device not found", err.Error())
			c.JSON(http.StatusNotFound, response)
			return
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to join group", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(gin.H{
		"group_id":    input.GroupID,
		"device_uuid": deviceUUID,
	}, http.StatusOK, "Device joined group successfully")
	c.JSON(http.StatusOK, response)
}

func (h *DeviceGroupHandler) QuitGroup(c *gin.Context) {
	deviceUUID := c.Param("instance_uuid")
	if deviceUUID == "" {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid device_uuid", "device uuid is required")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var input struct {
		GroupID int64 `json:"group_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error())
		c.JSON(http.StatusBadRequest, response)
		return
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated")
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	if err := h.groupService.QuitGroup(input.GroupID, deviceUUID, userUUID.(string)); err != nil {
		errMsg := err.Error()
		if errMsg == "invalid group" || errMsg == "device access denied" || errMsg == "permission denied" {
			response := types.NewErrorResponse(http.StatusForbidden, "Access denied", err.Error())
			c.JSON(http.StatusForbidden, response)
			return
		}
		if errMsg == "device not found" {
			response := types.NewErrorResponse(http.StatusNotFound, "Device not found", err.Error())
			c.JSON(http.StatusNotFound, response)
			return
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to quit group", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(gin.H{
		"group_id":    input.GroupID,
		"device_uuid": deviceUUID,
	}, http.StatusOK, "Device quit group successfully")
	c.JSON(http.StatusOK, response)
}

func (h *DeviceGroupHandler) GetGroups(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated")
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	ownerID := utils.ParseUserIDFromUUID(userUUID.(string))

	groups, total, err := h.groupService.GetGroups(ownerID, page, pageSize)
	if err != nil {
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to get groups", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(gin.H{
		"groups":    groups,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, http.StatusOK, "Groups retrieved successfully")
	c.JSON(http.StatusOK, response)
}

func (h *DeviceGroupHandler) GetGroupMembers(c *gin.Context) {
	groupIDStr := c.Param("group_id")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid group_id", err.Error())
		c.JSON(http.StatusBadRequest, response)
		return
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated")
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	members, total, err := h.groupService.GetGroupMembers(groupID, userUUID.(string), page, pageSize)
	if err != nil {
		if err.Error() == "Could not find a match group" {
			response := types.NewErrorResponse(http.StatusNotFound, "Could not find a match group")
			c.JSON(http.StatusNotFound, response)
			return
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to get group members", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(gin.H{
		"members":   members,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, http.StatusOK, "Group members retrieved successfully")
	c.JSON(http.StatusOK, response)
}

func (h *DeviceGroupHandler) DismissGroup(c *gin.Context) {
	groupIDStr := c.Param("group_id")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid group_id", err.Error())
		c.JSON(http.StatusBadRequest, response)
		return
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated")
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	if err := h.groupService.DismissGroup(groupID, userUUID.(string)); err != nil {
		if err.Error() == "Could not find a match group" {
			response := types.NewErrorResponse(http.StatusNotFound, "Could not find a match group")
			c.JSON(http.StatusNotFound, response)
			return
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to dismiss group", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(gin.H{
		"group_id": groupID,
	}, http.StatusOK, "Group dismissed successfully")
	c.JSON(http.StatusOK, response)
}
