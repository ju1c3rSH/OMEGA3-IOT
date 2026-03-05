package repository

import (
	"OMEGA3-IOT/internal/model"
	"gorm.io/gorm"
	"time"
)

type DeviceRegistrationRecordRepository interface {
	Create(record *model.DeviceRegistrationRecord) error
	FindByRegCode(regCode string) (*model.DeviceRegistrationRecord, error)
	FindByDeviceUUID(deviceUUID string) (*model.DeviceRegistrationRecord, error)
	FindUnexpiredAndUnbound() ([]model.DeviceRegistrationRecord, error)
	MarkAsBound(deviceUUID string) error
	DeleteExpired() error
	DeleteByDeviceUUID(deviceUUID string) error

	// Transaction support
	WithTx(tx *gorm.DB) DeviceRegistrationRecordRepository
}
type gormDeviceRegistrationRecordRepository struct {
	db *gorm.DB
}

func NewDeviceRegistrationRecordRepository(db *gorm.DB) DeviceRegistrationRecordRepository {
	return &gormDeviceRegistrationRecordRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *gormDeviceRegistrationRecordRepository) WithTx(tx *gorm.DB) DeviceRegistrationRecordRepository {
	return &gormDeviceRegistrationRecordRepository{db: tx}
}

func (r *gormDeviceRegistrationRecordRepository) Create(record *model.DeviceRegistrationRecord) error {
	return r.db.Create(record).Error
}

func (r *gormDeviceRegistrationRecordRepository) FindByRegCode(regCode string) (*model.DeviceRegistrationRecord, error) {
	var record model.DeviceRegistrationRecord
	err := r.db.Where("reg_code = ?", regCode).First(&record).Error
	return &record, err
}

func (r *gormDeviceRegistrationRecordRepository) FindByDeviceUUID(deviceUUID string) (*model.DeviceRegistrationRecord, error) {
	var record model.DeviceRegistrationRecord
	err := r.db.Where("device_uuid = ?", deviceUUID).First(&record).Error
	return &record, err
}

func (r *gormDeviceRegistrationRecordRepository) FindUnexpiredAndUnbound() ([]model.DeviceRegistrationRecord, error) {
	var records []model.DeviceRegistrationRecord
	err := r.db.Where("expires_at > ? AND is_bound = ?", time.Now().Unix(), false).
		Find(&records).Error
	return records, err
}

func (r *gormDeviceRegistrationRecordRepository) MarkAsBound(deviceUUID string) error {
	return r.db.Model(&model.DeviceRegistrationRecord{}).
		Where("device_uuid = ?", deviceUUID).
		Update("is_bound", true).Error
}

func (r *gormDeviceRegistrationRecordRepository) DeleteExpired() error {
	return r.db.Where("expires_at <= ?", time.Now().Unix()).
		Delete(&model.DeviceRegistrationRecord{}).Error
}

func (r *gormDeviceRegistrationRecordRepository) DeleteByDeviceUUID(deviceUUID string) error {
	return r.db.Where("device_uuid = ?", deviceUUID).
		Delete(&model.DeviceRegistrationRecord{}).Error
}
