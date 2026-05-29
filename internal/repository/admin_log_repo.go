package repository

import (
	"OMEGA3-IOT/internal/model"

	"gorm.io/gorm"
)

// AdminLogRepository defines the interface for admin operation log access.
type AdminLogRepository interface {
	Create(log *model.AdminLog) error
	FindByAdminUUID(adminUUID string, limit, offset int) ([]model.AdminLog, error)
	FindByAction(action string, limit, offset int) ([]model.AdminLog, error)
	FindAll(limit, offset int) ([]model.AdminLog, error)
	Count() (int64, error)
	CountByAdmin(adminUUID string) (int64, error)
	WithTx(tx *gorm.DB) AdminLogRepository
}

type gormAdminLogRepository struct {
	db *gorm.DB
}

// NewAdminLogRepository creates a new AdminLogRepository.
func NewAdminLogRepository(db *gorm.DB) AdminLogRepository {
	return &gormAdminLogRepository{db: db}
}

func (r *gormAdminLogRepository) Create(log *model.AdminLog) error {
	return r.db.Create(log).Error
}

func (r *gormAdminLogRepository) FindByAdminUUID(adminUUID string, limit, offset int) ([]model.AdminLog, error) {
	var logs []model.AdminLog
	err := r.db.Where("admin_uuid = ?", adminUUID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, err
}

func (r *gormAdminLogRepository) FindByAction(action string, limit, offset int) ([]model.AdminLog, error) {
	var logs []model.AdminLog
	err := r.db.Where("action = ?", action).Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, err
}

func (r *gormAdminLogRepository) FindAll(limit, offset int) ([]model.AdminLog, error) {
	var logs []model.AdminLog
	err := r.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, err
}

func (r *gormAdminLogRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.AdminLog{}).Count(&count).Error
	return count, err
}

func (r *gormAdminLogRepository) CountByAdmin(adminUUID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.AdminLog{}).Where("admin_uuid = ?", adminUUID).Count(&count).Error
	return count, err
}

func (r *gormAdminLogRepository) WithTx(tx *gorm.DB) AdminLogRepository {
	return &gormAdminLogRepository{db: tx}
}
