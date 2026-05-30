package handler

import (
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/types"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// DeviceFolderHandler handles HTTP requests for device folders.
type DeviceFolderHandler struct {
	folderService *service.DeviceFolderService
}

// NewDeviceFolderHandler creates a new DeviceFolderHandler.
func NewDeviceFolderHandler(folderService *service.DeviceFolderService) *DeviceFolderHandler {
	return &DeviceFolderHandler{
		folderService: folderService,
	}
}

// CreateFolder handles POST /devices/folders
func (h *DeviceFolderHandler) CreateFolder(c *gin.Context) {
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

	ownerUUID := userUUID.(string)

	folder, err := h.folderService.CreateFolder(input.Name, input.Description, ownerUUID)
	if err != nil {
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to create folder", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(folder, http.StatusOK, "Folder created successfully")
	c.JSON(http.StatusOK, response)
}

// AddDeviceToFolder handles POST /devices/:instance_uuid/folders
func (h *DeviceFolderHandler) AddDeviceToFolder(c *gin.Context) {
	deviceUUID := c.Param("instance_uuid")
	if deviceUUID == "" {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid device_uuid", "device uuid is required")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var input struct {
		FolderUUID string `json:"folder_uuid" binding:"required"`
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

	folderUUID := input.FolderUUID
	if err := h.folderService.AddDeviceToFolder(folderUUID, deviceUUID, userUUID.(string)); err != nil {
		errMsg := err.Error()
		if errMsg == "invalid folder" || errMsg == "device access denied" || errMsg == "permission denied" {
			response := types.NewErrorResponse(http.StatusForbidden, "Access denied", err.Error())
			c.JSON(http.StatusForbidden, response)
			return
		}
		if errMsg == "device not found" {
			response := types.NewErrorResponse(http.StatusNotFound, "Device not found", err.Error())
			c.JSON(http.StatusNotFound, response)
			return
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to add device to folder", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(gin.H{
		"folder_uuid": input.FolderUUID,
		"device_uuid": deviceUUID,
	}, http.StatusOK, "Device added to folder successfully")
	c.JSON(http.StatusOK, response)
}

// RemoveDeviceFromFolder handles DELETE /devices/:instance_uuid/folders/:folder_uuid
func (h *DeviceFolderHandler) RemoveDeviceFromFolder(c *gin.Context) {
	deviceUUID := c.Param("instance_uuid")
	if deviceUUID == "" {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid device_uuid", "device uuid is required")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	folderUUID := c.Param("folder_uuid")
	if folderUUID == "" {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid folder_uuid", "folder_uuid is required")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated")
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	if err := h.folderService.RemoveDeviceFromFolder(folderUUID, deviceUUID, userUUID.(string)); err != nil {
		errMsg := err.Error()
		if errMsg == "invalid folder" || errMsg == "device access denied" || errMsg == "permission denied" {
			response := types.NewErrorResponse(http.StatusForbidden, "Access denied", err.Error())
			c.JSON(http.StatusForbidden, response)
			return
		}
		if errMsg == "device not found" {
			response := types.NewErrorResponse(http.StatusNotFound, "Device not found", err.Error())
			c.JSON(http.StatusNotFound, response)
			return
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to remove device from folder", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(gin.H{
		"folder_uuid": folderUUID,
		"device_uuid": deviceUUID,
	}, http.StatusOK, "Device removed from folder successfully")
	c.JSON(http.StatusOK, response)
}

// GetFolders handles GET /users/me/device_folders
func (h *DeviceFolderHandler) GetFolders(c *gin.Context) {
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

	ownerUUID := userUUID.(string)

	folders, total, err := h.folderService.GetFolders(ownerUUID, page, pageSize)
	if err != nil {
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to get folders", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(gin.H{
		"folders":   folders,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, http.StatusOK, "Folders retrieved successfully")
	c.JSON(http.StatusOK, response)
}

// GetFolderDevices handles GET /devices/folders/:folder_uuid/devices
func (h *DeviceFolderHandler) GetFolderDevices(c *gin.Context) {
	folderUUID := c.Param("folder_uuid")
	if folderUUID == "" {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid folder_uuid", "folder_uuid is required")
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

	devices, total, err := h.folderService.GetFolderDevices(folderUUID, userUUID.(string), page, pageSize)
	if err != nil {
		if err.Error() == "folder not found" {
			response := types.NewErrorResponse(http.StatusNotFound, "Folder not found")
			c.JSON(http.StatusNotFound, response)
			return
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to get folder devices", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(gin.H{
		"devices":   devices,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, http.StatusOK, "Folder devices retrieved successfully")
	c.JSON(http.StatusOK, response)
}

// DeleteFolder handles DELETE /devices/folders/:folder_uuid
func (h *DeviceFolderHandler) DeleteFolder(c *gin.Context) {
	folderUUID := c.Param("folder_uuid")
	if folderUUID == "" {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid folder_uuid", "folder_uuid is required")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	userUUID, exists := c.Get("user_uuid")
	if !exists {
		response := types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated")
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	if err := h.folderService.DeleteFolder(folderUUID, userUUID.(string)); err != nil {
		if err.Error() == "folder not found" {
			response := types.NewErrorResponse(http.StatusNotFound, "Folder not found")
			c.JSON(http.StatusNotFound, response)
			return
		}
		if err.Error() == "permission denied" {
			response := types.NewErrorResponse(http.StatusForbidden, "Access denied", err.Error())
			c.JSON(http.StatusForbidden, response)
			return
		}
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to delete folder", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponseWithCode(gin.H{
		"folder_uuid": folderUUID,
	}, http.StatusOK, "Folder deleted successfully")
	c.JSON(http.StatusOK, response)
}
