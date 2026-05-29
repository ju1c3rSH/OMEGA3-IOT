package repository

import (
	"OMEGA3-IOT/internal/model"

	"gorm.io/gorm"
)

// DeviceListFilter defines filter criteria for listing devices.
type DeviceListFilter struct {
	Search    string // search name, UUID, or SN
	Type      string // filter by device type
	Status    string // filter by status (active/inactive)
	OwnerUUID string // filter by owner
	Online    *bool  // filter by online status (nil = all)
	SortBy    string // created_at, last_seen, name (default: created_at)
	SortOrder string // asc, desc (default: desc)
}

// DeviceListItem is a projection of Instance for admin list responses.
type DeviceListItem struct {
	InstanceUUID string `json:"instance_uuid"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Online       bool   `json:"online"`
	OwnerUUID    string `json:"owner_uuid"`
	OwnerName    string `json:"owner_name"`
	Status       string `json:"status"`
	SN           string `json:"sn"`
	LastSeen     int64  `json:"last_seen"`
	CreatedAt    int64  `json:"created_at"`
	IsShared     bool   `json:"is_shared"`
	SharedCount  int    `json:"shared_count"`
}

// AdminDeviceRepository extends device queries for admin operations.
type AdminDeviceRepository interface {
	ListDevices(filter DeviceListFilter, page, pageSize int) ([]DeviceListItem, int64, error)
	CountAll() (int64, error)
	CountOnline() (int64, error)
	WithTx(tx *gorm.DB) AdminDeviceRepository
}

type gormAdminDeviceRepository struct {
	db *gorm.DB
}

// NewAdminDeviceRepository creates a new AdminDeviceRepository.
func NewAdminDeviceRepository(db *gorm.DB) AdminDeviceRepository {
	return &gormAdminDeviceRepository{db: db}
}

func (r *gormAdminDeviceRepository) ListDevices(filter DeviceListFilter, page, pageSize int) ([]DeviceListItem, int64, error) {
	query := r.db.Model(&model.Instance{})

	// Apply filters
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("name LIKE ? OR instance_uuid LIKE ? OR sn LIKE ?", like, like, like)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.OwnerUUID != "" {
		query = query.Where("owner_uuid = ?", filter.OwnerUUID)
	}
	if filter.Online != nil {
		query = query.Where("online = ?", *filter.Online)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sort
	sortBy := "created_at"
	if filter.SortBy != "" {
		allowed := map[string]bool{"created_at": true, "last_seen": true, "name": true}
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

	// Join with users to get owner name
	var devices []DeviceListItem
	err := query.
		Select(`instances.instance_uuid, instances.name, instances.type, instances.online,
			instances.owner_uuid, instances.status, instances.sn, instances.last_seen,
			instances.created_at, instances.is_shared, instances.shared_count,
			COALESCE(users.user_name, '') as owner_name`).
		Joins("LEFT JOIN users ON users.user_uuid = instances.owner_uuid").
		Limit(pageSize).Offset(offset).
		Scan(&devices).Error

	return devices, total, err
}

func (r *gormAdminDeviceRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&model.Instance{}).Count(&count).Error
	return count, err
}

func (r *gormAdminDeviceRepository) CountOnline() (int64, error) {
	var count int64
	err := r.db.Model(&model.Instance{}).Where("online = ?", true).Count(&count).Error
	return count, err
}

func (r *gormAdminDeviceRepository) WithTx(tx *gorm.DB) AdminDeviceRepository {
	return &gormAdminDeviceRepository{db: tx}
}
