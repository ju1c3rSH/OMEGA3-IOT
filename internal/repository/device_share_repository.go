package repository

import (
	"OMEGA3-IOT/internal/model"
	"time"

	"gorm.io/gorm"
)

const (
	StatusActive  = "active"
	StatusRevoked = "revoked"
)

const (
	PermissionRead      = "read"
	PermissionWrite     = "write"
	PermissionReadWrite = "read_write"
)

type DeviceShareRepository interface {
	Create(share *model.DeviceShare) error
	FindByID(id uint) (*model.DeviceShare, error)
	FindByInstanceAndSharedWith(instanceUUID, sharedWithUUID string) (*model.DeviceShare, error)
	FindByInstanceUUID(instanceUUID string) ([]model.DeviceShare, error)
	FindBySharedWithUUID(sharedWithUUID string) ([]model.DeviceShare, error)
	FindActiveSharesByInstance(instanceUUID string) ([]model.DeviceShare, error)
	Update(share *model.DeviceShare) error
	UpdateStatus(instanceUUID, sharedWithUUID, status string) error
	Delete(id uint) error
	DeleteByInstanceAndSharedWith(instanceUUID, sharedWithUUID string) error
	CountActiveShares(instanceUUID string) (int64, error)

	// Transaction support
	WithTx(tx *gorm.DB) DeviceShareRepository
}

type gormDeviceShareRepository struct {
	db *gorm.DB
}

func NewDeviceShareRepository(db *gorm.DB) DeviceShareRepository {
	return &gormDeviceShareRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *gormDeviceShareRepository) WithTx(tx *gorm.DB) DeviceShareRepository {
	return &gormDeviceShareRepository{db: tx}
}

func (r *gormDeviceShareRepository) Create(share *model.DeviceShare) error {
	return r.db.Create(share).Error
}

func (r *gormDeviceShareRepository) FindByID(id uint) (*model.DeviceShare, error) {
	var share model.DeviceShare
	err := r.db.First(&share, id).Error
	return &share, err
}

func (r *gormDeviceShareRepository) FindByInstanceAndSharedWith(instanceUUID, sharedWithUUID string) (*model.DeviceShare, error) {
	var share model.DeviceShare
	err := r.db.Where("instance_uuid = ? AND shared_with_uuid = ?", instanceUUID, sharedWithUUID).
		First(&share).Error
	return &share, err
}

func (r *gormDeviceShareRepository) FindByInstanceUUID(instanceUUID string) ([]model.DeviceShare, error) {
	var shares []model.DeviceShare
	err := r.db.Where("instance_uuid = ?", instanceUUID).Find(&shares).Error
	return shares, err
}

func (r *gormDeviceShareRepository) FindBySharedWithUUID(sharedWithUUID string) ([]model.DeviceShare, error) {
	var shares []model.DeviceShare
	err := r.db.Where("shared_with_uuid = ?", sharedWithUUID).Find(&shares).Error
	return shares, err
}

func (r *gormDeviceShareRepository) FindActiveSharesByInstance(instanceUUID string) ([]model.DeviceShare, error) {
	var shares []model.DeviceShare
	now := time.Now().Unix()
	err := r.db.Where("instance_uuid = ? AND status = ?", instanceUUID, StatusActive).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Find(&shares).Error
	return shares, err
}

func (r *gormDeviceShareRepository) Update(share *model.DeviceShare) error {
	return r.db.Save(share).Error
}

func (r *gormDeviceShareRepository) UpdateStatus(instanceUUID, sharedWithUUID, status string) error {
	return r.db.Model(&model.DeviceShare{}).
		Where("instance_uuid = ? AND shared_with_uuid = ?", instanceUUID, sharedWithUUID).
		Update("status", status).Error
}

func (r *gormDeviceShareRepository) Delete(id uint) error {
	return r.db.Delete(&model.DeviceShare{}, id).Error
}

func (r *gormDeviceShareRepository) DeleteByInstanceAndSharedWith(instanceUUID, sharedWithUUID string) error {
	return r.db.Where("instance_uuid = ? AND shared_with_uuid = ?", instanceUUID, sharedWithUUID).
		Delete(&model.DeviceShare{}).Error
}

func (r *gormDeviceShareRepository) CountActiveShares(instanceUUID string) (int64, error) {
	var count int64
	now := time.Now().Unix()
	err := r.db.Model(&model.DeviceShare{}).
		Where("instance_uuid = ? AND status = ?", instanceUUID, StatusActive).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Count(&count).Error
	return count, err
}
