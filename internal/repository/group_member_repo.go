package repository

import (
	"OMEGA3-IOT/internal/model"

	"gorm.io/gorm"
)

// GroupMemberRepository defines the interface for group member data access.
type GroupMemberRepository interface {
	Create(member *model.GroupMember) error
	FindByGroupAndUser(groupUUID, userUUID string) (*model.GroupMember, error)
	FindActiveByGroupAndUser(groupUUID, userUUID string) (*model.GroupMember, error)
	FindByGroupUUID(groupUUID string) ([]model.GroupMember, error)
	FindActiveByGroupUUID(groupUUID string) ([]model.GroupMember, error)
	FindByUserUUID(userUUID string) ([]model.GroupMember, error)
	FindActiveByUserUUID(userUUID string) ([]model.GroupMember, error)
	Update(member *model.GroupMember) error
	UpdateFields(id uint, fields map[string]interface{}) error
	Delete(id uint) error
	CountActiveByGroup(groupUUID string) (int64, error)
	ExistsActive(groupUUID, userUUID string) (bool, error)
	WithTx(tx *gorm.DB) GroupMemberRepository
}

type gormGroupMemberRepository struct {
	db *gorm.DB
}

// NewGroupMemberRepository creates a new GroupMemberRepository.
func NewGroupMemberRepository(db *gorm.DB) GroupMemberRepository {
	return &gormGroupMemberRepository{db: db}
}

func (r *gormGroupMemberRepository) Create(member *model.GroupMember) error {
	return r.db.Create(member).Error
}

func (r *gormGroupMemberRepository) FindByGroupAndUser(groupUUID, userUUID string) (*model.GroupMember, error) {
	var member model.GroupMember
	err := r.db.Where("group_uuid = ? AND user_uuid = ?", groupUUID, userUUID).First(&member).Error
	return &member, err
}

func (r *gormGroupMemberRepository) FindActiveByGroupAndUser(groupUUID, userUUID string) (*model.GroupMember, error) {
	var member model.GroupMember
	err := r.db.Where("group_uuid = ? AND user_uuid = ? AND status = ?", groupUUID, userUUID, model.GroupMemberStatusActive).First(&member).Error
	return &member, err
}

func (r *gormGroupMemberRepository) FindByGroupUUID(groupUUID string) ([]model.GroupMember, error) {
	var members []model.GroupMember
	err := r.db.Where("group_uuid = ?", groupUUID).Find(&members).Error
	return members, err
}

func (r *gormGroupMemberRepository) FindActiveByGroupUUID(groupUUID string) ([]model.GroupMember, error) {
	var members []model.GroupMember
	err := r.db.Where("group_uuid = ? AND status = ?", groupUUID, model.GroupMemberStatusActive).Find(&members).Error
	return members, err
}

func (r *gormGroupMemberRepository) FindByUserUUID(userUUID string) ([]model.GroupMember, error) {
	var members []model.GroupMember
	err := r.db.Where("user_uuid = ?", userUUID).Find(&members).Error
	return members, err
}

func (r *gormGroupMemberRepository) FindActiveByUserUUID(userUUID string) ([]model.GroupMember, error) {
	var members []model.GroupMember
	err := r.db.Where("user_uuid = ? AND status = ?", userUUID, model.GroupMemberStatusActive).Find(&members).Error
	return members, err
}

func (r *gormGroupMemberRepository) Update(member *model.GroupMember) error {
	return r.db.Save(member).Error
}

func (r *gormGroupMemberRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.GroupMember{}).Where("id = ?", id).Updates(fields).Error
}

func (r *gormGroupMemberRepository) Delete(id uint) error {
	return r.db.Delete(&model.GroupMember{}, id).Error
}

func (r *gormGroupMemberRepository) CountActiveByGroup(groupUUID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.GroupMember{}).Where("group_uuid = ? AND status = ?", groupUUID, model.GroupMemberStatusActive).Count(&count).Error
	return count, err
}

func (r *gormGroupMemberRepository) ExistsActive(groupUUID, userUUID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.GroupMember{}).Where("group_uuid = ? AND user_uuid = ? AND status = ?", groupUUID, userUUID, model.GroupMemberStatusActive).Count(&count).Error
	return count > 0, err
}

func (r *gormGroupMemberRepository) WithTx(tx *gorm.DB) GroupMemberRepository {
	return &gormGroupMemberRepository{db: tx}
}
