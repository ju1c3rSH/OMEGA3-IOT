package repository

import (
	"OMEGA3-IOT/internal/model"

	"gorm.io/gorm"
)

// GroupPolicyRepository defines the interface for group policy data access.
type GroupPolicyRepository interface {
	Create(policy *model.GroupPolicy) error
	FindByGroupUUID(groupUUID string) (*model.GroupPolicy, error)
	Update(policy *model.GroupPolicy) error
	UpdateFields(groupUUID string, fields map[string]interface{}) error
	Delete(id uint) error
	WithTx(tx *gorm.DB) GroupPolicyRepository
}

type gormGroupPolicyRepository struct {
	db *gorm.DB
}

// NewGroupPolicyRepository creates a new GroupPolicyRepository.
func NewGroupPolicyRepository(db *gorm.DB) GroupPolicyRepository {
	return &gormGroupPolicyRepository{db: db}
}

func (r *gormGroupPolicyRepository) Create(policy *model.GroupPolicy) error {
	return r.db.Create(policy).Error
}

func (r *gormGroupPolicyRepository) FindByGroupUUID(groupUUID string) (*model.GroupPolicy, error) {
	var policy model.GroupPolicy
	err := r.db.Where("group_uuid = ?", groupUUID).First(&policy).Error
	return &policy, err
}

func (r *gormGroupPolicyRepository) Update(policy *model.GroupPolicy) error {
	return r.db.Save(policy).Error
}

func (r *gormGroupPolicyRepository) UpdateFields(groupUUID string, fields map[string]interface{}) error {
	return r.db.Model(&model.GroupPolicy{}).Where("group_uuid = ?", groupUUID).Updates(fields).Error
}

func (r *gormGroupPolicyRepository) Delete(id uint) error {
	return r.db.Delete(&model.GroupPolicy{}, id).Error
}

func (r *gormGroupPolicyRepository) WithTx(tx *gorm.DB) GroupPolicyRepository {
	return &gormGroupPolicyRepository{db: tx}
}
