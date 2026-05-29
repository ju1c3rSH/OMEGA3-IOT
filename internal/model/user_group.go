package model

import (
	"OMEGA3-IOT/internal/utils"
	"fmt"
	"time"
)

// GroupRole constants
const (
	GroupRoleMember = 0
	GroupRoleAdmin  = 1
	GroupRoleOwner  = 2
)

// GroupStatus constants
const (
	GroupStatusActive   = 0
	GroupStatusDissolved = 1
)

// UserGroup represents a user group in the system.
type UserGroup struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupUUID   string `json:"group_uuid" gorm:"uniqueIndex;not null;type:varchar(36)"`
	Name        string `json:"name" gorm:"type:varchar(128);not null" validate:"required,min=1,max=128"`
	Description string `json:"description,omitempty" gorm:"type:text"`
	OwnerUUID   string `json:"owner_uuid" gorm:"type:varchar(36);not null;index"`
	MaxMembers  int    `json:"max_members" gorm:"default:0"` // 0 means no limit
	Status      int    `json:"status" gorm:"default:0;index"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// CreateGroupRequest is the request body for creating a group.
type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=128"`
	Description string `json:"description"`
	MaxMembers  int    `json:"max_members"`
}

// UpdateGroupRequest is the request body for updating a group.
type UpdateGroupRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	MaxMembers  *int    `json:"max_members,omitempty"`
}

// NewUserGroup creates a new UserGroup with generated UUID and timestamps.
func NewUserGroup(name, description, ownerUUID string, maxMembers int) (*UserGroup, error) {
	if name == "" {
		return nil, fmt.Errorf("group name is required")
	}
	if ownerUUID == "" {
		return nil, fmt.Errorf("owner UUID is required")
	}
	now := time.Now().Unix()
	return &UserGroup{
		GroupUUID:   utils.GenerateUUID().String(),
		Name:        name,
		Description: description,
		OwnerUUID:   ownerUUID,
		MaxMembers:  maxMembers,
		Status:      GroupStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}
