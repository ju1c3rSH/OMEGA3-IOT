package repository

import (
	"OMEGA3-IOT/internal/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *model.User) error
	FindByID(id uint) (*model.User, error)
	FindByUUID(userUUID string) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	Update(user *model.User) error
	UpdateFields(userUUID string, fields map[string]interface{}) error
	Delete(id uint) error

	// Transaction support
	WithTx(tx *gorm.DB) UserRepository
}
type gormUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *gormUserRepository) WithTx(tx *gorm.DB) UserRepository {
	return &gormUserRepository{db: tx}
}

func (r *gormUserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *gormUserRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	return &user, err
}

func (r *gormUserRepository) FindByUUID(userUUID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("user_uuid = ?", userUUID).First(&user).Error
	return &user, err
}

func (r *gormUserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("user_name = ?", username).First(&user).Error
	return &user, err
}

func (r *gormUserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *gormUserRepository) UpdateFields(userUUID string, fields map[string]interface{}) error {
	return r.db.Model(&model.User{}).Where("user_uuid = ?", userUUID).Updates(fields).Error
}

func (r *gormUserRepository) Delete(id uint) error {
	return r.db.Delete(&model.User{}, id).Error
}
