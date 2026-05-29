package repository

import (
	"OMEGA3-IOT/internal/model"
	"time"

	"gorm.io/gorm"
)

// GroupInviteRepository defines the interface for group invite data access.
type GroupInviteRepository interface {
	Create(invite *model.GroupInvite) error
	FindByCode(inviteCode string) (*model.GroupInvite, error)
	FindByGroupUUID(groupUUID string) ([]model.GroupInvite, error)
	FindPendingByGroupAndInvitee(groupUUID, inviteeUUID string) (*model.GroupInvite, error)
	Update(invite *model.GroupInvite) error
	UpdateFields(id uint, fields map[string]interface{}) error
	Delete(id uint) error
	DeleteExpired() error
	WithTx(tx *gorm.DB) GroupInviteRepository
}

type gormGroupInviteRepository struct {
	db *gorm.DB
}

// NewGroupInviteRepository creates a new GroupInviteRepository.
func NewGroupInviteRepository(db *gorm.DB) GroupInviteRepository {
	return &gormGroupInviteRepository{db: db}
}

func (r *gormGroupInviteRepository) Create(invite *model.GroupInvite) error {
	return r.db.Create(invite).Error
}

func (r *gormGroupInviteRepository) FindByCode(inviteCode string) (*model.GroupInvite, error) {
	var invite model.GroupInvite
	err := r.db.Where("invite_code = ?", inviteCode).First(&invite).Error
	return &invite, err
}

func (r *gormGroupInviteRepository) FindByGroupUUID(groupUUID string) ([]model.GroupInvite, error) {
	var invites []model.GroupInvite
	err := r.db.Where("group_uuid = ?", groupUUID).Find(&invites).Error
	return invites, err
}

func (r *gormGroupInviteRepository) FindPendingByGroupAndInvitee(groupUUID, inviteeUUID string) (*model.GroupInvite, error) {
	var invite model.GroupInvite
	err := r.db.Where("group_uuid = ? AND invitee_uuid = ? AND status = ?", groupUUID, inviteeUUID, model.InviteStatusPending).First(&invite).Error
	return &invite, err
}

func (r *gormGroupInviteRepository) Update(invite *model.GroupInvite) error {
	return r.db.Save(invite).Error
}

func (r *gormGroupInviteRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.GroupInvite{}).Where("id = ?", id).Updates(fields).Error
}

func (r *gormGroupInviteRepository) Delete(id uint) error {
	return r.db.Delete(&model.GroupInvite{}, id).Error
}

func (r *gormGroupInviteRepository) DeleteExpired() error {
	return r.db.Where("expires_at <= ? AND status = ?", time.Now().Unix(), model.InviteStatusPending).Delete(&model.GroupInvite{}).Error
}

func (r *gormGroupInviteRepository) WithTx(tx *gorm.DB) GroupInviteRepository {
	return &gormGroupInviteRepository{db: tx}
}
