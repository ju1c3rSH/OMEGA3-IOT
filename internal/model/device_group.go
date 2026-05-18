package model

import (
	"time"
)

type DeviceGroup struct {
	GroupUUID   string    `gorm:"primaryKey;column:group_uuid;type:char(36);not null" json:"group_uuid"`
	Name        string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	OwnerUUID   string    `gorm:"column:owner_uuid;type:char(36);not null;index:idx_owner_uuid" json:"owner_uuid"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Description string    `gorm:"column:description;type:text" json:"description,omitempty"`
	Valid       int8      `gorm:"column:valid;type:tinyint(1);default:1" json:"valid"`
}

func (DeviceGroup) TableName() string {
	return "device_group"
}

type DeviceGroupRelation struct {
	GroupUUID   string    `gorm:"primaryKey;column:group_uuid;type:char(36);not null;index:idx_group_uuid" json:"group_uuid"`
	DeviceUUID  string    `gorm:"primaryKey;column:device_uuid;type:varchar(36);not null;index:idx_device_uuid" json:"device_uuid"`
	JoinedAt    time.Time `gorm:"column:joined_at;autoCreateTime" json:"joined_at"`
	Valid       int8      `gorm:"column:valid;type:tinyint(1);default:1" json:"valid"`
}

func (DeviceGroupRelation) TableName() string {
	return "device_group_relation"
}

type GroupMemberDevice struct {
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
