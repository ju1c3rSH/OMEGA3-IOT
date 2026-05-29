package model

import (
	"time"
)

// GroupMemberStatus constants
const (
	GroupMemberStatusActive  = 0
	GroupMemberStatusLeft    = 1
	GroupMemberStatusKicked  = 2
	GroupMemberStatusPending = 3 // pending approval
)

// GroupMember represents the relationship between a user and a group.
type GroupMember struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupUUID string `json:"group_uuid" gorm:"type:varchar(36);not null;index:idx_group_user,unique"`
	UserUUID  string `json:"user_uuid" gorm:"type:varchar(36);not null;index:idx_group_user,unique"`
	Role      int    `json:"role" gorm:"default:0"` // 0=member, 1=group_admin, 2=group_owner
	Status    int    `json:"status" gorm:"default:0;index"`
	JoinedAt  int64  `json:"joined_at"`
	InvitedBy string `json:"invited_by,omitempty" gorm:"type:varchar(36)"`
}

// NewGroupMember creates a new GroupMember record.
func NewGroupMember(groupUUID, userUUID string, role int, invitedBy string) *GroupMember {
	return &GroupMember{
		GroupUUID: groupUUID,
		UserUUID:  userUUID,
		Role:      role,
		Status:    GroupMemberStatusActive,
		JoinedAt:  time.Now().Unix(),
		InvitedBy: invitedBy,
	}
}
