package model

import "time"

// AdminLog records admin operations for audit purposes.
type AdminLog struct {
	ID         uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	AdminUUID  string `json:"admin_uuid" gorm:"type:varchar(36);not null;index"`
	Action     string `json:"action" gorm:"type:varchar(100);not null"`  // e.g. "user.delete", "device.transfer"
	TargetType string `json:"target_type" gorm:"type:varchar(50)"`      // "user", "device", "group"
	TargetUUID string `json:"target_uuid" gorm:"type:varchar(36)"`      // target entity UUID
	Detail     string `json:"detail" gorm:"type:text"`                  // JSON detail
	IP         string `json:"ip" gorm:"type:varchar(45)"`
	CreatedAt  int64  `json:"created_at"`
}

// NewAdminLog creates a new admin log entry.
func NewAdminLog(adminUUID, action, targetType, targetUUID, detail, ip string) *AdminLog {
	return &AdminLog{
		AdminUUID:  adminUUID,
		Action:     action,
		TargetType: targetType,
		TargetUUID: targetUUID,
		Detail:     detail,
		IP:         ip,
		CreatedAt:  time.Now().Unix(),
	}
}
