package model

import (
	"time"
)

// DeviceFolder represents a named collection of devices owned by a single user.
// It is an organizational tool (like a folder/tag), not a collaboration group.
type DeviceFolder struct {
	FolderUUID  string    `gorm:"primaryKey;column:folder_uuid;type:char(36);not null" json:"folder_uuid"`
	Name        string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	OwnerUUID   string    `gorm:"column:owner_uuid;type:char(36);not null;index:idx_owner_uuid" json:"owner_uuid"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Description string    `gorm:"column:description;type:text" json:"description,omitempty"`
	Valid       int8      `gorm:"column:valid;type:tinyint(1);default:1" json:"valid"`
}

func (DeviceFolder) TableName() string {
	return "device_folder"
}

// DeviceFolderItem represents a device's membership in a DeviceFolder.
type DeviceFolderItem struct {
	FolderUUID string    `gorm:"primaryKey;column:folder_uuid;type:char(36);not null;index:idx_folder_uuid" json:"folder_uuid"`
	DeviceUUID string    `gorm:"primaryKey;column:device_uuid;type:varchar(36);not null;index:idx_device_uuid" json:"device_uuid"`
	JoinedAt   time.Time `gorm:"column:joined_at;autoCreateTime" json:"joined_at"`
	Valid      int8      `gorm:"column:valid;type:tinyint(1);default:1" json:"valid"`
}

func (DeviceFolderItem) TableName() string {
	return "device_folder_item"
}

// FolderDeviceItem is a read-only DTO for listing devices within a folder.
type FolderDeviceItem struct {
	InstanceUUID string     `json:"instance_uuid"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	Online       bool       `json:"online"`
	OwnerUUID    string     `json:"owner_uuid"`
	Description  string     `json:"description,omitempty"`
	Properties   Properties `json:"properties"`
	Status       string     `json:"status"`
	JoinedAt     time.Time  `json:"joined_at"`
}
