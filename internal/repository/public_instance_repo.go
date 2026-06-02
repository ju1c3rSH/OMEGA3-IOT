package repository

import (
	"OMEGA3-IOT/internal/model"

	"gorm.io/gorm"
)

type PublicInstanceRepository interface {
	// Public device queries
	FindAllPublic() ([]model.Instance, error)
	FindPublicByType(deviceType string) ([]model.Instance, error)
	FindPublicByUUID(uuid string) (*model.Instance, error)

	// Favorite operations
	AddFavorite(favorite *model.PublicInstanceFavorite) error
	RemoveFavorite(userUUID, instanceUUID string) error
	FindFavoritesByUser(userUUID string) ([]model.PublicInstanceFavorite, error)
	IsFavorited(userUUID, instanceUUID string) (bool, error)

	WithTx(tx *gorm.DB) PublicInstanceRepository
}

type gormPublicInstanceRepository struct {
	db *gorm.DB
}

func NewPublicInstanceRepository(db *gorm.DB) PublicInstanceRepository {
	return &gormPublicInstanceRepository{db: db}
}

func (r *gormPublicInstanceRepository) WithTx(tx *gorm.DB) PublicInstanceRepository {
	return &gormPublicInstanceRepository{db: tx}
}

func (r *gormPublicInstanceRepository) FindAllPublic() ([]model.Instance, error) {
	var instances []model.Instance
	err := r.db.Where("is_public = ? AND status = ?", true, "active").Find(&instances).Error
	return instances, err
}

func (r *gormPublicInstanceRepository) FindPublicByType(deviceType string) ([]model.Instance, error) {
	var instances []model.Instance
	err := r.db.Where("is_public = ? AND status = ? AND type = ?", true, "active", deviceType).Find(&instances).Error
	return instances, err
}

func (r *gormPublicInstanceRepository) FindPublicByUUID(uuid string) (*model.Instance, error) {
	var instance model.Instance
	err := r.db.Where("instance_uuid = ? AND is_public = ? AND status = ?", uuid, true, "active").First(&instance).Error
	return &instance, err
}

func (r *gormPublicInstanceRepository) AddFavorite(favorite *model.PublicInstanceFavorite) error {
	return r.db.Create(favorite).Error
}

func (r *gormPublicInstanceRepository) RemoveFavorite(userUUID, instanceUUID string) error {
	return r.db.Where("user_uuid = ? AND instance_uuid = ?", userUUID, instanceUUID).
		Delete(&model.PublicInstanceFavorite{}).Error
}

func (r *gormPublicInstanceRepository) FindFavoritesByUser(userUUID string) ([]model.PublicInstanceFavorite, error) {
	var favorites []model.PublicInstanceFavorite
	err := r.db.Where("user_uuid = ?", userUUID).Find(&favorites).Error
	return favorites, err
}

func (r *gormPublicInstanceRepository) IsFavorited(userUUID, instanceUUID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.PublicInstanceFavorite{}).
		Where("user_uuid = ? AND instance_uuid = ?", userUUID, instanceUUID).
		Count(&count).Error
	return count > 0, err
}