package handler

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/types"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserGroupHandler handles HTTP requests for group management.
type UserGroupHandler struct {
	groupService  *service.UserGroupService
	inviteService *service.GroupInviteService
}

// NewUserGroupHandler creates a new UserGroupHandler.
func NewUserGroupHandler(groupService *service.UserGroupService, inviteService *service.GroupInviteService) *UserGroupHandler {
	return &UserGroupHandler{
		groupService:  groupService,
		inviteService: inviteService,
	}
}

// ==================== Group CRUD ====================

// CreateGroup handles POST /groups
func (h *UserGroupHandler) CreateGroup(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	var req model.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error()))
		return
	}

	group, err := h.groupService.CreateGroup(req.Name, req.Description, userUUID.(string), req.MaxMembers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to create group", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponseWithCode(group, http.StatusCreated, "Group created"))
}

// GetMyGroups handles GET /groups
func (h *UserGroupHandler) GetMyGroups(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groups, err := h.groupService.GetUserGroups(userUUID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get groups", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"groups": groups}, http.StatusOK, "OK"))
}

// GetGroup handles GET /groups/:group_uuid
func (h *UserGroupHandler) GetGroup(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	group, err := h.groupService.GetGroup(groupUUID)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "group not found" || errMsg == "group is dissolved" {
			c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "Group not found", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get group", errMsg))
		return
	}

	// Check membership
	if err := h.groupService.CheckMembership(groupUUID, userUUID.(string)); err != nil {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(group, http.StatusOK, "OK"))
}

// UpdateGroup handles PUT /groups/:group_uuid
func (h *UserGroupHandler) UpdateGroup(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	var req model.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error()))
		return
	}

	if err := h.groupService.UpdateGroup(groupUUID, userUUID.(string), &req); err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: insufficient role" || errMsg == "permission denied: not a group member" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to update group", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"group_uuid": groupUUID}, http.StatusOK, "Group updated"))
}

// DissolveGroup handles DELETE /groups/:group_uuid
func (h *UserGroupHandler) DissolveGroup(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	if err := h.groupService.DissolveGroup(groupUUID, userUUID.(string)); err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to dissolve group", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"group_uuid": groupUUID}, http.StatusOK, "Group dissolved"))
}

// ==================== Member Management ====================

// GetMembers handles GET /groups/:group_uuid/members
func (h *UserGroupHandler) GetMembers(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	members, err := h.groupService.GetMembers(groupUUID, userUUID.(string))
	if err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: not a group member" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get members", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"members": members}, http.StatusOK, "OK"))
}

// SearchInvite handles POST /groups/:group_uuid/invite/search
func (h *UserGroupHandler) SearchInvite(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	var req service.SearchInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error()))
		return
	}

	invite, err := h.inviteService.CreateSearchInvite(groupUUID, userUUID.(string), &req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: not a group member" || errMsg == "permission denied: members cannot invite" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		if errMsg == "user not found" {
			c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "User not found", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to create invite", errMsg))
		return
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponseWithCode(invite, http.StatusCreated, "Invite created"))
}

// CreateLinkInvite handles POST /groups/:group_uuid/invite/link
func (h *UserGroupHandler) CreateLinkInvite(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	var req service.CreateLinkInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error()))
		return
	}

	invite, err := h.inviteService.CreateLinkInvite(groupUUID, userUUID.(string), &req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: not a group member" || errMsg == "permission denied: members cannot create invite links" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to create invite link", errMsg))
		return
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponseWithCode(invite, http.StatusCreated, "Invite link created"))
}

// AcceptInvite handles POST /groups/invite/:invite_code/accept
func (h *UserGroupHandler) AcceptInvite(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	inviteCode := c.Param("invite_code")

	member, err := h.inviteService.AcceptInvite(inviteCode, userUUID.(string))
	if err != nil {
		errMsg := err.Error()
		if errMsg == "invite not found" {
			c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "Invite not found", errMsg))
			return
		}
		if errMsg == "invite is no longer valid" || errMsg == "invite has expired" || errMsg == "this invite is not for you" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid invite", errMsg))
			return
		}
		if errMsg == "user is already a member" {
			c.JSON(http.StatusConflict, types.NewErrorResponse(http.StatusConflict, "Already a member", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to accept invite", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(member, http.StatusOK, "Invite accepted"))
}

// ApproveMember handles POST /groups/:group_uuid/members/:user_uuid/approve
func (h *UserGroupHandler) ApproveMember(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")
	targetUUID := c.Param("user_uuid")

	if err := h.groupService.ApproveMember(groupUUID, targetUUID, userUUID.(string)); err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: insufficient role" || errMsg == "permission denied: not a group member" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		if errMsg == "member not found" {
			c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "Member not found", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to approve member", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"group_uuid": groupUUID, "user_uuid": targetUUID}, http.StatusOK, "Member approved"))
}

// RejectMember handles POST /groups/:group_uuid/members/:user_uuid/reject
func (h *UserGroupHandler) RejectMember(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")
	targetUUID := c.Param("user_uuid")

	if err := h.groupService.RejectMember(groupUUID, targetUUID, userUUID.(string)); err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: insufficient role" || errMsg == "permission denied: not a group member" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		if errMsg == "member not found" {
			c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "Member not found", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to reject member", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"group_uuid": groupUUID, "user_uuid": targetUUID}, http.StatusOK, "Member rejected"))
}

// RemoveMember handles DELETE /groups/:group_uuid/members/:user_uuid
func (h *UserGroupHandler) RemoveMember(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")
	targetUUID := c.Param("user_uuid")

	if err := h.groupService.RemoveMember(groupUUID, targetUUID, userUUID.(string)); err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: insufficient role" || errMsg == "permission denied: not a group member" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		if errMsg == "member not found" || errMsg == "cannot remove the group owner" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Cannot remove member", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to remove member", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"group_uuid": groupUUID, "user_uuid": targetUUID}, http.StatusOK, "Member removed"))
}

// LeaveGroup handles POST /groups/:group_uuid/leave
func (h *UserGroupHandler) LeaveGroup(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	if err := h.groupService.LeaveGroup(groupUUID, userUUID.(string)); err != nil {
		errMsg := err.Error()
		if errMsg == "owner cannot leave the group" || errMsg == "not a member of this group" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Cannot leave group", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to leave group", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"group_uuid": groupUUID}, http.StatusOK, "Left group"))
}

// UpdateMemberRole handles PUT /groups/:group_uuid/members/:user_uuid/role
func (h *UserGroupHandler) UpdateMemberRole(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")
	targetUUID := c.Param("user_uuid")

	var req struct {
		Role int `json:"role" binding:"required,min=0,max=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error()))
		return
	}

	if err := h.groupService.UpdateMemberRole(groupUUID, targetUUID, userUUID.(string), req.Role); err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		if errMsg == "member not found" || errMsg == "cannot change your own role" || errMsg == "invalid role" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid operation", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to update role", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"group_uuid": groupUUID, "user_uuid": targetUUID, "role": req.Role}, http.StatusOK, "Role updated"))
}

// ==================== Device Management ====================

// GetGroupDevices handles GET /groups/:group_uuid/devices
func (h *UserGroupHandler) GetGroupDevices(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	shares, err := h.groupService.GetGroupDevices(groupUUID, userUUID.(string))
	if err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: not a group member" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get devices", errMsg))
		return
	}

	// Paginate
	total := len(shares)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
		"devices":   shares[start:end],
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, http.StatusOK, "OK"))
}

// ShareDeviceToGroup handles POST /groups/:group_uuid/devices/share
func (h *UserGroupHandler) ShareDeviceToGroup(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	var req struct {
		InstanceUUID string `json:"instance_uuid" binding:"required"`
		Permission   string `json:"permission" binding:"required,oneof=read write read_write"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error()))
		return
	}

	if err := h.groupService.ShareDeviceToGroup(groupUUID, req.InstanceUUID, userUUID.(string), req.Permission); err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: not a group member" || errMsg == "permission denied: you do not own this device" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		if errMsg == "device not found" {
			c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "Device not found", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to share device", errMsg))
		return
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponseWithCode(gin.H{"group_uuid": groupUUID, "instance_uuid": req.InstanceUUID}, http.StatusCreated, "Device shared"))
}

// RevokeGroupDeviceShare handles DELETE /groups/:group_uuid/devices/:instance_uuid
func (h *UserGroupHandler) RevokeGroupDeviceShare(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")
	instanceUUID := c.Param("instance_uuid")

	if err := h.groupService.RevokeGroupDeviceShare(groupUUID, instanceUUID, userUUID.(string)); err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		if errMsg == "share not found" {
			c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "Share not found", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to revoke share", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"group_uuid": groupUUID, "instance_uuid": instanceUUID}, http.StatusOK, "Share revoked"))
}

// ==================== Policy Management ====================

// GetPolicy handles GET /groups/:group_uuid/policy
func (h *UserGroupHandler) GetPolicy(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	policy, err := h.groupService.GetPolicy(groupUUID, userUUID.(string))
	if err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: not a group member" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get policy", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(policy, http.StatusOK, "OK"))
}

// UpdatePolicy handles PUT /groups/:group_uuid/policy
func (h *UserGroupHandler) UpdatePolicy(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	var req model.UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request parameters", err.Error()))
		return
	}

	if err := h.groupService.UpdatePolicy(groupUUID, userUUID.(string), &req); err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: insufficient role" || errMsg == "permission denied: not a group member" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to update policy", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"group_uuid": groupUUID}, http.StatusOK, "Policy updated"))
}

// GetPendingInvites handles GET /groups/:group_uuid/invites
func (h *UserGroupHandler) GetPendingInvites(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}

	groupUUID := c.Param("group_uuid")

	invites, err := h.inviteService.GetPendingInvites(groupUUID, userUUID.(string))
	if err != nil {
		errMsg := err.Error()
		if errMsg == "permission denied: not a group member" || errMsg == "permission denied: insufficient role" {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", errMsg))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get invites", errMsg))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{"invites": invites}, http.StatusOK, "OK"))
}
