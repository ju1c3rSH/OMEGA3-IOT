package model

import (
	"time"
)

// GroupDeviceShareStatus constants
const (
	GroupDeviceShareStatusActive  = 0
	GroupDeviceShareStatusRevoked = 1
)

// GroupDeviceShare represents a device shared to a group.
type GroupDeviceShare struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupUUID    string `json:"group_uuid" gorm:"type:varchar(36);not null;index:idx_group_device"`
	InstanceUUID string `json:"instance_uuid" gorm:"type:varchar(36);not null;index:idx_group_device"`
	OwnerUUID    string `json:"owner_uuid" gorm:"type:varchar(36);not null;index"`
	Permission   string `json:"permission" gorm:"type:varchar(20);not null;default:'read'" validate:"oneof=read write read_write"`
	SharedBy     string `json:"shared_by" gorm:"type:varchar(36);not null"`
	Status       int    `json:"status" gorm:"default:0;index"`
	CreatedAt    int64  `json:"created_at"`
	RevokedAt    *int64 `json:"revoked_at,omitempty"`
}

// NewGroupDeviceShare creates a new device share record for a group.
func NewGroupDeviceShare(groupUUID, instanceUUID, ownerUUID, permission, sharedBy string) *GroupDeviceShare {
	return &GroupDeviceShare{
		GroupUUID:    groupUUID,
		InstanceUUID: instanceUUID,
		OwnerUUID:    ownerUUID,
		Permission:   permission,
		SharedBy:     sharedBy,
		Status:       GroupDeviceShareStatusActive,
		CreatedAt:    time.Now().Unix(),
	}
}
