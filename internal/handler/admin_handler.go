package handler

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/types"
	"OMEGA3-IOT/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AdminHandler handles HTTP requests for admin management.
type AdminHandler struct {
	adminService *service.AdminService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

// ==================== Admin Login ====================

// Challenge handles POST /admin/challenge
func (h *AdminHandler) Challenge(c *gin.Context) {
	var input model.ChallengeRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid input", err.Error()))
		return
	}

	resp, err := h.adminService.AdminChallenge(input.Username)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "account is not an admin" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Account is not an admin"))
			return
		}
		if errMsg == "account is disabled" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Account is disabled"))
			return
		}
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Challenge failed", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(resp))
}

// Login handles POST /admin/login
func (h *AdminHandler) Login(c *gin.Context) {
	var input model.LoginUser
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid input", err.Error()))
		return
	}

	user, err := h.adminService.AdminLogin(input.Username, input.Proof)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "invalid credentials" {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "Invalid credentials"))
			return
		}
		if errMsg == "account is not an admin" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Account is not an admin"))
			return
		}
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, errMsg))
		return
	}

	token, err := utils.GenerateToken(user.UserName, user.UserUUID, user.Role, utils.GenerateUUID().String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to generate token"))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
		"access_token": token,
		"user": gin.H{
			"user_uuid": user.UserUUID,
			"username":  user.UserName,
			"nickname":  user.Nickname,
			"role":      user.Role,
		},
	}, http.StatusOK, "Login successful"))
}

// Logout handles POST /admin/logout
func (h *AdminHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(nil, http.StatusOK, "Logged out"))
}

// ==================== Admin Management (super_admin only) ====================

// GetAdmins handles GET /admin/admins
func (h *AdminHandler) GetAdmins(c *gin.Context) {
	admins, err := h.adminService.GetAdmins()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get admins", err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"admins": admins}, http.StatusOK, "OK"))
}

// PromoteUser handles POST /admin/admins
func (h *AdminHandler) PromoteUser(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")

	var req struct {
		UserUUID string `json:"user_uuid" binding:"required"`
		Role     int    `json:"role" binding:"required,min=2,max=3"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request", err.Error()))
		return
	}

	if err := h.adminService.PromoteUser(req.UserUUID, model.Role(req.Role), adminUUID, c.ClientIP()); err != nil {
		errMsg := err.Error()
		if errMsg == "cannot change your own role" || errMsg == "invalid target role: must be moderator or admin" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to promote user", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"user_uuid": req.UserUUID, "role": req.Role}, http.StatusOK, "User promoted"))
}

// UpdateAdminRole handles PUT /admin/admins/:user_uuid
func (h *AdminHandler) UpdateAdminRole(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")
	targetUUID := c.Param("user_uuid")

	var req struct {
		Role int `json:"role" binding:"required,min=2,max=3"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request", err.Error()))
		return
	}

	if err := h.adminService.UpdateAdminRole(targetUUID, model.Role(req.Role), adminUUID, c.ClientIP()); err != nil {
		errMsg := err.Error()
		if errMsg == "cannot assign super_admin via API" || errMsg == "cannot modify super_admin role" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to update role", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"user_uuid": targetUUID, "role": req.Role}, http.StatusOK, "Role updated"))
}

// DemoteAdmin handles DELETE /admin/admins/:user_uuid
func (h *AdminHandler) DemoteAdmin(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")
	targetUUID := c.Param("user_uuid")

	if err := h.adminService.DemoteAdmin(targetUUID, adminUUID, c.ClientIP()); err != nil {
		errMsg := err.Error()
		if errMsg == "cannot demote yourself" || errMsg == "cannot demote super_admin" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to demote admin", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"user_uuid": targetUUID}, http.StatusOK, "Admin demoted"))
}

// ==================== User Management ====================

// ListUsers handles GET /admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.UserListFilter{
		Search:    c.Query("search"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}
	if roleStr := c.Query("role"); roleStr != "" {
		if r, err := strconv.Atoi(roleStr); err == nil {
			filter.Role = &r
		}
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			filter.Status = &s
		}
	}

	users, total, err := h.adminService.ListUsers(filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to list users", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, http.StatusOK, "OK"))
}

// GetUser handles GET /admin/users/:user_uuid
func (h *AdminHandler) GetUser(c *gin.Context) {
	user, err := h.adminService.GetUser(c.Param("user_uuid"))
	if err != nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "User not found", err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(user, http.StatusOK, "OK"))
}

// EditUser handles PUT /admin/users/:user_uuid
func (h *AdminHandler) EditUser(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")
	targetUUID := c.Param("user_uuid")

	var req struct {
		Nickname    *string `json:"nickname,omitempty"`
		Description *string `json:"description,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request", err.Error()))
		return
	}

	if err := h.adminService.EditUser(targetUUID, req.Nickname, req.Description, adminUUID, c.ClientIP()); err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to edit user", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"user_uuid": targetUUID}, http.StatusOK, "User updated"))
}

// UpdateUserStatus handles PUT /admin/users/:user_uuid/status
func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")
	targetUUID := c.Param("user_uuid")

	var req struct {
		Status int `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request", err.Error()))
		return
	}

	if err := h.adminService.UpdateUserStatus(targetUUID, req.Status, adminUUID, c.ClientIP()); err != nil {
		errMsg := err.Error()
		if errMsg == "cannot change admin status via this endpoint" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to update status", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"user_uuid": targetUUID, "status": req.Status}, http.StatusOK, "Status updated"))
}

// DeleteUser handles DELETE /admin/users/:user_uuid
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")
	targetUUID := c.Param("user_uuid")

	if err := h.adminService.DeleteUser(targetUUID, adminUUID, c.ClientIP()); err != nil {
		errMsg := err.Error()
		if errMsg == "cannot delete admin users" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to delete user", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"user_uuid": targetUUID}, http.StatusOK, "User deleted"))
}

// ResetPassword handles POST /admin/users/:user_uuid/reset-password
func (h *AdminHandler) ResetPassword(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")
	targetUUID := c.Param("user_uuid")

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request", err.Error()))
		return
	}

	if err := h.adminService.ResetUserPassword(targetUUID, req.NewPassword, adminUUID, c.ClientIP()); err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to reset password", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"user_uuid": targetUUID}, http.StatusOK, "Password reset successful"))
}

// ==================== Device Management ====================

// ListDevices handles GET /admin/devices
func (h *AdminHandler) ListDevices(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.DeviceListFilter{
		Search:    c.Query("search"),
		Type:      c.Query("type"),
		Status:    c.Query("status"),
		OwnerUUID: c.Query("owner"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}
	if onlineStr := c.Query("online"); onlineStr != "" {
		if onlineStr == "true" {
			t := true
			filter.Online = &t
		} else if onlineStr == "false" {
			f := false
			filter.Online = &f
		}
	}

	devices, total, err := h.adminService.ListDevices(filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to list devices", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
		"devices":   devices,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, http.StatusOK, "OK"))
}

// GetDevice handles GET /admin/devices/:instance_uuid
func (h *AdminHandler) GetDevice(c *gin.Context) {
	device, err := h.adminService.GetDevice(c.Param("instance_uuid"))
	if err != nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "Device not found", err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(device, http.StatusOK, "OK"))
}

// EditDevice handles PUT /admin/devices/:instance_uuid
func (h *AdminHandler) EditDevice(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")
	instanceUUID := c.Param("instance_uuid")

	var req struct {
		Name        *string `json:"name,omitempty"`
		Description *string `json:"description,omitempty"`
		Remark      *string `json:"remark,omitempty"`
		SN          *string `json:"sn,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request", err.Error()))
		return
	}

	if err := h.adminService.EditDevice(instanceUUID, req.Name, req.Description, req.Remark, req.SN, adminUUID, c.ClientIP()); err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to edit device", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"instance_uuid": instanceUUID}, http.StatusOK, "Device updated"))
}

// DeleteDevice handles DELETE /admin/devices/:instance_uuid
func (h *AdminHandler) DeleteDevice(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")
	instanceUUID := c.Param("instance_uuid")

	if err := h.adminService.DeleteDevice(instanceUUID, adminUUID, c.ClientIP()); err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to delete device", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"instance_uuid": instanceUUID}, http.StatusOK, "Device deleted"))
}

// TransferDevice handles POST /admin/devices/:instance_uuid/transfer
func (h *AdminHandler) TransferDevice(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")
	instanceUUID := c.Param("instance_uuid")

	var req struct {
		NewOwnerUUID       string `json:"new_owner_uuid" binding:"required"`
		KeepOriginalAccess bool   `json:"keep_original_access"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request", err.Error()))
		return
	}

	if err := h.adminService.TransferDevice(instanceUUID, req.NewOwnerUUID, req.KeepOriginalAccess, adminUUID, c.ClientIP()); err != nil {
		errMsg := err.Error()
		if errMsg == "device already belongs to this user" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to transfer device", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
		"instance_uuid":  instanceUUID,
		"new_owner":      req.NewOwnerUUID,
		"keep_original":  req.KeepOriginalAccess,
	}, http.StatusOK, "Device transferred"))
}

// ==================== Group Management ====================

// ListGroups handles GET /admin/groups
func (h *AdminHandler) ListGroups(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	groups, total, err := h.adminService.ListGroups(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to list groups", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
		"groups":    groups,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, http.StatusOK, "OK"))
}

// GetGroup handles GET /admin/groups/:group_uuid
func (h *AdminHandler) GetGroup(c *gin.Context) {
	group, err := h.adminService.GetGroup(c.Param("group_uuid"))
	if err != nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "Group not found", err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(group, http.StatusOK, "OK"))
}

// GetGroupMembers handles GET /admin/groups/:group_uuid/members
func (h *AdminHandler) GetGroupMembers(c *gin.Context) {
	members, err := h.adminService.GetGroupMembers(c.Param("group_uuid"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get members", err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"members": members}, http.StatusOK, "OK"))
}

// DissolveGroup handles DELETE /admin/groups/:group_uuid
func (h *AdminHandler) DissolveGroup(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")

	if err := h.adminService.DissolveGroup(c.Param("group_uuid"), adminUUID, c.ClientIP()); err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to dissolve group", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"group_uuid": c.Param("group_uuid")}, http.StatusOK, "Group dissolved"))
}

// RemoveGroupMember handles DELETE /admin/groups/:group_uuid/members/:user_uuid
func (h *AdminHandler) RemoveGroupMember(c *gin.Context) {
	adminUUID := c.GetString("user_uuid")

	if err := h.adminService.RemoveGroupMember(c.Param("group_uuid"), c.Param("user_uuid"), adminUUID, c.ClientIP()); err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to remove member", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
		"group_uuid": c.Param("group_uuid"),
		"user_uuid":  c.Param("user_uuid"),
	}, http.StatusOK, "Member removed"))
}

// ==================== System Stats ====================

// GetStats handles GET /admin/stats/overview
func (h *AdminHandler) GetStats(c *gin.Context) {
	stats, err := h.adminService.GetSystemStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get stats", err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(stats, http.StatusOK, "OK"))
}

// ==================== Admin Logs ====================

// GetLogs handles GET /admin/logs
func (h *AdminHandler) GetLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, total, err := h.adminService.ListAdminLogs(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get logs", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, http.StatusOK, "OK"))
}
