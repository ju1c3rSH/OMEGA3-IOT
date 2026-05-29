package repository

import (
	"OMEGA3-IOT/internal/model"

	"gorm.io/gorm"
)

// UserListFilter defines filter criteria for listing users.
type UserListFilter struct {
	Search    string // search username, nickname, or UUID
	Role      *int   // filter by role (nil = all)
	Status    *int   // filter by status (nil = all)
	SortBy    string // created_at, last_seen, username (default: created_at)
	SortOrder string // asc, desc (default: desc)
}

// UserListItem is a projection of User for list responses (excludes sensitive fields).
type UserListItem struct {
	UserUUID    string `json:"user_uuid"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	Avatar      string `json:"avatar"`
	Role        int    `json:"role"`
	Status      int    `json:"status"`
	Online      bool   `json:"online"`
	IP          string `json:"ip"`
	LastSeen    int64  `json:"last_seen"`
	CreatedAt   int64  `json:"created_at"`
	DeviceCount int64  `json:"device_count"`
}

// AdminUserRepository extends user queries for admin operations.
type AdminUserRepository interface {
	ListUsers(filter UserListFilter, page, pageSize int) ([]UserListItem, int64, error)
	CountByRole(role int) (int64, error)
	CountAll() (int64, error)
	CountActiveToday() (int64, error)
	FindAdmins() ([]model.User, error)
	WithTx(tx *gorm.DB) AdminUserRepository
}

type gormAdminUserRepository struct {
	db *gorm.DB
}

// NewAdminUserRepository creates a new AdminUserRepository.
func NewAdminUserRepository(db *gorm.DB) AdminUserRepository {
	return &gormAdminUserRepository{db: db}
}

func (r *gormAdminUserRepository) ListUsers(filter UserListFilter, page, pageSize int) ([]UserListItem, int64, error) {
	query := r.db.Model(&model.User{})

	// Apply filters
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("user_name LIKE ? OR nickname LIKE ? OR user_uuid LIKE ?", like, like, like)
	}
	if filter.Role != nil {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sort
	sortBy := "created_at"
	if filter.SortBy != "" {
		allowed := map[string]bool{"created_at": true, "last_seen": true, "user_name": true}
		if allowed[filter.SortBy] {
			sortBy = filter.SortBy
		}
	}
	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	query = query.Order(sortBy + " " + sortOrder)

	// Paginate
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Select only non-sensitive fields + device count subquery
	var users []UserListItem
	err := query.
		Select(`user_uuid, user_name as username, nickname, avatar, role, status, online, ip, last_seen, created_at,
			(SELECT COUNT(*) FROM instances WHERE instances.owner_uuid = users.user_uuid) as device_count`).
		Limit(pageSize).Offset(offset).
		Scan(&users).Error

	return users, total, err
}

func (r *gormAdminUserRepository) CountByRole(role int) (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("role = ?", role).Count(&count).Error
	return count, err
}

func (r *gormAdminUserRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Count(&count).Error
	return count, err
}

func (r *gormAdminUserRepository) CountActiveToday() (int64, error) {
	var count int64
	// Users who were seen in the last 24 hours
	err := r.db.Model(&model.User{}).Where("online = ? OR last_seen > ?", true, 0).Count(&count).Error
	return count, err
}

func (r *gormAdminUserRepository) FindAdmins() ([]model.User, error) {
	var users []model.User
	err := r.db.Where("role >= ?", model.RoleModerator).Find(&users).Error
	return users, err
}

func (r *gormAdminUserRepository) WithTx(tx *gorm.DB) AdminUserRepository {
	return &gormAdminUserRepository{db: tx}
}
