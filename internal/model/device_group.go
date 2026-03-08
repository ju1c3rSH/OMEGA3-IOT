package model

import (
	"time"
)

type DeviceGroup struct {
	ID          int64     `gorm:"primaryKey;column:id" json:"id"`
	Name        string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	OwnerID     int64     `gorm:"column:owner_id;type:bigint;not null;index:idx_owner_id" json:"owner_id"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Description string    `gorm:"column:description;type:text" json:"description,omitempty"`
	Valid       int8      `gorm:"column:valid;type:tinyint(1);default:1" json:"valid"`
}

func (DeviceGroup) TableName() string {
	return "device_group"
}

type DeviceGroupRelation struct {
	GroupID  int64     `gorm:"primaryKey;column:group_id;type:bigint;not null" json:"group_id"`
	DeviceID int64     `gorm:"primaryKey;column:device_id;type:bigint;not null;index:idx_device_id" json:"device_id"`
	JoinedAt time.Time `gorm:"column:joined_at;autoCreateTime" json:"joined_at"`
	Valid    int8      `gorm:"column:valid;type:tinyint(1);default:1" json:"valid"`
}

func (DeviceGroupRelation) TableName() string {
	return "device_group_relation"
}

type GroupMemberDevice struct {
	InstanceID   int64      `json:"instance_id"`
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
