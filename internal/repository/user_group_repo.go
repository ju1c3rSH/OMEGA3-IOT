package repository

import (
	"OMEGA3-IOT/internal/model"

	"gorm.io/gorm"
)

// UserGroupRepository defines the interface for group data access.
type UserGroupRepository interface {
	Create(group *model.UserGroup) error
	FindByID(id uint) (*model.UserGroup, error)
	FindByUUID(groupUUID string) (*model.UserGroup, error)
	FindByOwnerUUID(ownerUUID string) ([]model.UserGroup, error)
	Update(group *model.UserGroup) error
	UpdateFields(groupUUID string, fields map[string]interface{}) error
	Delete(id uint) error
	DeleteByUUID(groupUUID string) error
	Exists(groupUUID string) (bool, error)
	WithTx(tx *gorm.DB) UserGroupRepository
}

type gormUserGroupRepository struct {
	db *gorm.DB
}

// NewUserGroupRepository creates a new UserGroupRepository.
func NewUserGroupRepository(db *gorm.DB) UserGroupRepository {
	return &gormUserGroupRepository{db: db}
}

func (r *gormUserGroupRepository) Create(group *model.UserGroup) error {
	return r.db.Create(group).Error
}

func (r *gormUserGroupRepository) FindByID(id uint) (*model.UserGroup, error) {
	var group model.UserGroup
	err := r.db.First(&group, id).Error
	return &group, err
}

func (r *gormUserGroupRepository) FindByUUID(groupUUID string) (*model.UserGroup, error) {
	var group model.UserGroup
	err := r.db.Where("group_uuid = ?", groupUUID).First(&group).Error
	return &group, err
}

func (r *gormUserGroupRepository) FindByOwnerUUID(ownerUUID string) ([]model.UserGroup, error) {
	var groups []model.UserGroup
	err := r.db.Where("owner_uuid = ?", ownerUUID).Find(&groups).Error
	return groups, err
}

func (r *gormUserGroupRepository) Update(group *model.UserGroup) error {
	return r.db.Save(group).Error
}

func (r *gormUserGroupRepository) UpdateFields(groupUUID string, fields map[string]interface{}) error {
	return r.db.Model(&model.UserGroup{}).Where("group_uuid = ?", groupUUID).Updates(fields).Error
}

func (r *gormUserGroupRepository) Delete(id uint) error {
	return r.db.Delete(&model.UserGroup{}, id).Error
}

func (r *gormUserGroupRepository) DeleteByUUID(groupUUID string) error {
	return r.db.Where("group_uuid = ?", groupUUID).Delete(&model.UserGroup{}).Error
}

func (r *gormUserGroupRepository) Exists(groupUUID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.UserGroup{}).Where("group_uuid = ?", groupUUID).Count(&count).Error
	return count > 0, err
}

func (r *gormUserGroupRepository) WithTx(tx *gorm.DB) UserGroupRepository {
	return &gormUserGroupRepository{db: tx}
}
