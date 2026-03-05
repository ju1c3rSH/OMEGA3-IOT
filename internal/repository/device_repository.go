package repository

import (
	"OMEGA3-IOT/internal/model"
	"gorm.io/gorm"
)

type InstanceRepository interface {
	Create(instance *model.Instance) error
	FindByID(id uint) (*model.Instance, error)
	FindByUUID(instanceUUID string) (*model.Instance, error)
	FindByOwnerUUID(ownerUUID string) ([]model.Instance, error)
	Update(instance *model.Instance) error
	UpdateFields(instanceUUID string, fields map[string]interface{}) error
	Delete(id uint) error
	DeleteByUUID(instanceUUID string) error
	UpdateProperties(instanceUUID string, properties model.Properties) error
	UpdateOnlineStatus(instanceUUID string, online bool, lastSeen int64) error
	Exists(instanceUUID string) (bool, error)

	// Transaction support
	WithTx(tx *gorm.DB) InstanceRepository
}

type gormInstanceRepository struct {
	db *gorm.DB
}

func NewInstanceRepository(db *gorm.DB) InstanceRepository {
	return &gormInstanceRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *gormInstanceRepository) WithTx(tx *gorm.DB) InstanceRepository {
	return &gormInstanceRepository{db: tx}
}

func (r *gormInstanceRepository) Create(instance *model.Instance) error {
	return r.db.Create(instance).Error
}

func (r *gormInstanceRepository) FindByID(id uint) (*model.Instance, error) {
	var instance model.Instance
	err := r.db.First(&instance, id).Error
	return &instance, err
}

func (r *gormInstanceRepository) FindByUUID(instanceUUID string) (*model.Instance, error) {
	var instance model.Instance
	err := r.db.Where("instance_uuid = ?", instanceUUID).First(&instance).Error
	return &instance, err
}

func (r *gormInstanceRepository) FindByOwnerUUID(ownerUUID string) ([]model.Instance, error) {
	var instances []model.Instance
	err := r.db.Where("owner_uuid = ?", ownerUUID).Find(&instances).Error
	return instances, err
}

func (r *gormInstanceRepository) Update(instance *model.Instance) error {
	return r.db.Save(instance).Error
}

func (r *gormInstanceRepository) UpdateFields(instanceUUID string, fields map[string]interface{}) error {
	return r.db.Model(&model.Instance{}).Where("instance_uuid = ?", instanceUUID).Updates(fields).Error
}

func (r *gormInstanceRepository) Delete(id uint) error {
	return r.db.Delete(&model.Instance{}, id).Error
}

func (r *gormInstanceRepository) DeleteByUUID(instanceUUID string) error {
	return r.db.Where("instance_uuid = ?", instanceUUID).Delete(&model.Instance{}).Error
}

func (r *gormInstanceRepository) Exists(instanceUUID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Instance{}).Where("instance_uuid = ?", instanceUUID).Count(&count).Error
	return count > 0, err
}

func (r *gormInstanceRepository) UpdateProperties(instanceUUID string, properties model.Properties) error {
	return r.db.Model(&model.Instance{}).
		Where("instance_uuid = ?", instanceUUID).
		Update("properties", properties).Error
}

func (r *gormInstanceRepository) UpdateOnlineStatus(instanceUUID string, online bool, lastSeen int64) error {
	return r.db.Model(&model.Instance{}).
		Where("instance_uuid = ?", instanceUUID).
		Updates(map[string]interface{}{
			"online":    online,
			"last_seen": lastSeen,
		}).Error
}
