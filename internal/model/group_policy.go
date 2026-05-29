package model

import (
	"time"
)

// DeviceVisibility constants
const (
	DeviceVisibilityAdminAll   = 0 // admin can see all member devices
	DeviceVisibilitySharedOnly = 1 // only shared devices visible
	DeviceVisibilitySelective  = 2 // admin selects which devices are visible
)

// AdminDeviceAccess constants
const (
	AdminDeviceAccessReadOnly  = 0
	AdminDeviceAccessFull      = 1 // can send actions
)

// ApprovalMode constants
const (
	ApprovalModeAdminOnly  = 0
	ApprovalModeAnyMember  = 1
)

// GroupPolicy stores configurable policies for a group.
type GroupPolicy struct {
	ID                 uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupUUID          string `json:"group_uuid" gorm:"uniqueIndex;not null;type:varchar(36)"`
	DeviceVisibility   int    `json:"device_visibility" gorm:"default:1"`   // default: shared_only
	AdminDeviceAccess  int    `json:"admin_device_access" gorm:"default:0"` // default: read_only
	AllowSearchInvite  bool   `json:"allow_search_invite" gorm:"default:true"`
	AllowInviteLink    bool   `json:"allow_invite_link" gorm:"default:true"`
	AllowMemberInvite  bool   `json:"allow_member_invite" gorm:"default:false"`
	RequireApproval    bool   `json:"require_approval" gorm:"default:false"`
	ApprovalMode       int    `json:"approval_mode" gorm:"default:0"` // default: admin_only
	CreatedAt          int64  `json:"created_at"`
	UpdatedAt          int64  `json:"updated_at"`
}

// UpdatePolicyRequest is the request body for updating group policy.
type UpdatePolicyRequest struct {
	DeviceVisibility  *int  `json:"device_visibility,omitempty"`
	AdminDeviceAccess *int  `json:"admin_device_access,omitempty"`
	AllowSearchInvite *bool `json:"allow_search_invite,omitempty"`
	AllowInviteLink   *bool `json:"allow_invite_link,omitempty"`
	AllowMemberInvite *bool `json:"allow_member_invite,omitempty"`
	RequireApproval   *bool `json:"require_approval,omitempty"`
	ApprovalMode      *int  `json:"approval_mode,omitempty"`
}

// NewGroupPolicy creates a default policy for a group.
func NewGroupPolicy(groupUUID string) *GroupPolicy {
	now := time.Now().Unix()
	return &GroupPolicy{
		GroupUUID:         groupUUID,
		DeviceVisibility:  DeviceVisibilitySharedOnly,
		AdminDeviceAccess: AdminDeviceAccessReadOnly,
		AllowSearchInvite: true,
		AllowInviteLink:   true,
		AllowMemberInvite: false,
		RequireApproval:   false,
		ApprovalMode:      ApprovalModeAdminOnly,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}
