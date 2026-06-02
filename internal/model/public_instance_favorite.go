package model

// PublicInstanceFavorite stores a user's favorite public instances.
type PublicInstanceFavorite struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	UserUUID     string `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_instance" json:"user_uuid"`
	InstanceUUID string `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_instance" json:"instance_uuid"`
	CreatedAt    int64  `gorm:"not null" json:"created_at"`
}