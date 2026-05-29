package repository

import (
	"OMEGA3-IOT/internal/model"
	"time"

	"gorm.io/gorm"
)

// GroupDeviceShareRepository defines the interface for group device share data access.
type GroupDeviceShareRepository interface {
	Create(share *model.GroupDeviceShare) error
	FindByGroupAndDevice(groupUUID, instanceUUID string) (*model.GroupDeviceShare, error)
	FindActiveByGroupAndDevice(groupUUID, instanceUUID string) (*model.GroupDeviceShare, error)
	FindActiveByGroup(groupUUID string) ([]model.GroupDeviceShare, error)
	FindActiveByDevice(instanceUUID string) ([]model.GroupDeviceShare, error)
	FindActiveByOwner(ownerUUID string) ([]model.GroupDeviceShare, error)
	Update(share *model.GroupDeviceShare) error
	UpdateFields(id uint, fields map[string]interface{}) error
	Revoke(id uint) error
	RevokeByGroupAndDevice(groupUUID, instanceUUID string) error
	RevokeAllByGroup(groupUUID string) error
	RevokeAllByDevice(instanceUUID string) error
	Delete(id uint) error
	WithTx(tx *gorm.DB) GroupDeviceShareRepository
}

type gormGroupDeviceShareRepository struct {
	db *gorm.DB
}

// NewGroupDeviceShareRepository creates a new GroupDeviceShareRepository.
func NewGroupDeviceShareRepository(db *gorm.DB) GroupDeviceShareRepository {
	return &gormGroupDeviceShareRepository{db: db}
}

func (r *gormGroupDeviceShareRepository) Create(share *model.GroupDeviceShare) error {
	return r.db.Create(share).Error
}

func (r *gormGroupDeviceShareRepository) FindByGroupAndDevice(groupUUID, instanceUUID string) (*model.GroupDeviceShare, error) {
	var share model.GroupDeviceShare
	err := r.db.Where("group_uuid = ? AND instance_uuid = ?", groupUUID, instanceUUID).First(&share).Error
	return &share, err
}

func (r *gormGroupDeviceShareRepository) FindActiveByGroupAndDevice(groupUUID, instanceUUID string) (*model.GroupDeviceShare, error) {
	var share model.GroupDeviceShare
	err := r.db.Where("group_uuid = ? AND instance_uuid = ? AND status = ?", groupUUID, instanceUUID, model.GroupDeviceShareStatusActive).First(&share).Error
	return &share, err
}

func (r *gormGroupDeviceShareRepository) FindActiveByGroup(groupUUID string) ([]model.GroupDeviceShare, error) {
	var shares []model.GroupDeviceShare
	err := r.db.Where("group_uuid = ? AND status = ?", groupUUID, model.GroupDeviceShareStatusActive).Find(&shares).Error
	return shares, err
}

func (r *gormGroupDeviceShareRepository) FindActiveByDevice(instanceUUID string) ([]model.GroupDeviceShare, error) {
	var shares []model.GroupDeviceShare
	err := r.db.Where("instance_uuid = ? AND status = ?", instanceUUID, model.GroupDeviceShareStatusActive).Find(&shares).Error
	return shares, err
}

func (r *gormGroupDeviceShareRepository) FindActiveByOwner(ownerUUID string) ([]model.GroupDeviceShare, error) {
	var shares []model.GroupDeviceShare
	err := r.db.Where("owner_uuid = ? AND status = ?", ownerUUID, model.GroupDeviceShareStatusActive).Find(&shares).Error
	return shares, err
}

func (r *gormGroupDeviceShareRepository) Update(share *model.GroupDeviceShare) error {
	return r.db.Save(share).Error
}

func (r *gormGroupDeviceShareRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.GroupDeviceShare{}).Where("id = ?", id).Updates(fields).Error
}

func (r *gormGroupDeviceShareRepository) Revoke(id uint) error {
	now := time.Now().Unix()
	return r.db.Model(&model.GroupDeviceShare{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     model.GroupDeviceShareStatusRevoked,
		"revoked_at": now,
	}).Error
}

func (r *gormGroupDeviceShareRepository) RevokeByGroupAndDevice(groupUUID, instanceUUID string) error {
	now := time.Now().Unix()
	return r.db.Model(&model.GroupDeviceShare{}).Where("group_uuid = ? AND instance_uuid = ? AND status = ?", groupUUID, instanceUUID, model.GroupDeviceShareStatusActive).Updates(map[string]interface{}{
		"status":     model.GroupDeviceShareStatusRevoked,
		"revoked_at": now,
	}).Error
}

func (r *gormGroupDeviceShareRepository) RevokeAllByGroup(groupUUID string) error {
	now := time.Now().Unix()
	return r.db.Model(&model.GroupDeviceShare{}).Where("group_uuid = ? AND status = ?", groupUUID, model.GroupDeviceShareStatusActive).Updates(map[string]interface{}{
		"status":     model.GroupDeviceShareStatusRevoked,
		"revoked_at": now,
	}).Error
}

func (r *gormGroupDeviceShareRepository) RevokeAllByDevice(instanceUUID string) error {
	now := time.Now().Unix()
	return r.db.Model(&model.GroupDeviceShare{}).Where("instance_uuid = ? AND status = ?", instanceUUID, model.GroupDeviceShareStatusActive).Updates(map[string]interface{}{
		"status":     model.GroupDeviceShareStatusRevoked,
		"revoked_at": now,
	}).Error
}

func (r *gormGroupDeviceShareRepository) Delete(id uint) error {
	return r.db.Delete(&model.GroupDeviceShare{}, id).Error
}

func (r *gormGroupDeviceShareRepository) WithTx(tx *gorm.DB) GroupDeviceShareRepository {
	return &gormGroupDeviceShareRepository{db: tx}
}
