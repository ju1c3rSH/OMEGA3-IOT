package repository

import (
	"OMEGA3-IOT/internal/model"
	"time"

	"gorm.io/gorm"
)

const (
	StatusActive  = "active"
	StatusRevoked = "revoked"
)

const (
	PermissionRead      = "read"
	PermissionWrite     = "write"
	PermissionReadWrite = "read_write"
)

type DeviceShareRepository interface {
	Create(share *model.DeviceShare) error
	FindByID(id uint) (*model.DeviceShare, error)
	FindByInstanceAndSharedWith(instanceUUID, sharedWithUUID string) (*model.DeviceShare, error)
	FindByInstanceUUID(instanceUUID string) ([]model.DeviceShare, error)
	FindBySharedWithUUID(sharedWithUUID string) ([]model.DeviceShare, error)
	FindActiveSharesByInstance(instanceUUID string) ([]model.DeviceShare, error)
	Update(share *model.DeviceShare) error
	UpdateStatus(instanceUUID, sharedWithUUID, status string) error
	Delete(id uint) error
	DeleteByInstanceAndSharedWith(instanceUUID, sharedWithUUID string) error
	CountActiveShares(instanceUUID string) (int64, error)
	CountActiveSharesBatch(instanceUUIDs []string) (map[string]int64, error)

	// Transaction support
	WithTx(tx *gorm.DB) DeviceShareRepository
}

type gormDeviceShareRepository struct {
	db *gorm.DB
}

func NewDeviceShareRepository(db *gorm.DB) DeviceShareRepository {
	return &gormDeviceShareRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *gormDeviceShareRepository) WithTx(tx *gorm.DB) DeviceShareRepository {
	return &gormDeviceShareRepository{db: tx}
}

func (r *gormDeviceShareRepository) Create(share *model.DeviceShare) error {
	return r.db.Create(share).Error
}

func (r *gormDeviceShareRepository) FindByID(id uint) (*model.DeviceShare, error) {
	var share model.DeviceShare
	err := r.db.First(&share, id).Error
	return &share, err
}

func (r *gormDeviceShareRepository) FindByInstanceAndSharedWith(instanceUUID, sharedWithUUID string) (*model.DeviceShare, error) {
	var share model.DeviceShare
	err := r.db.Where("instance_uuid = ? AND shared_with_uuid = ?", instanceUUID, sharedWithUUID).
		First(&share).Error
	return &share, err
}

func (r *gormDeviceShareRepository) FindByInstanceUUID(instanceUUID string) ([]model.DeviceShare, error) {
	var shares []model.DeviceShare
	err := r.db.Where("instance_uuid = ?", instanceUUID).Find(&shares).Error
	return shares, err
}

func (r *gormDeviceShareRepository) FindBySharedWithUUID(sharedWithUUID string) ([]model.DeviceShare, error) {
	var shares []model.DeviceShare
	err := r.db.Where("shared_with_uuid = ?", sharedWithUUID).Find(&shares).Error
	return shares, err
}

func (r *gormDeviceShareRepository) FindActiveSharesByInstance(instanceUUID string) ([]model.DeviceShare, error) {
	var shares []model.DeviceShare
	now := time.Now().Unix()
	err := r.db.Where("instance_uuid = ? AND status = ?", instanceUUID, StatusActive).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Find(&shares).Error
	return shares, err
}

func (r *gormDeviceShareRepository) Update(share *model.DeviceShare) error {
	return r.db.Save(share).Error
}

func (r *gormDeviceShareRepository) UpdateStatus(instanceUUID, sharedWithUUID, status string) error {
	return r.db.Model(&model.DeviceShare{}).
		Where("instance_uuid = ? AND shared_with_uuid = ?", instanceUUID, sharedWithUUID).
		Update("status", status).Error
}

func (r *gormDeviceShareRepository) Delete(id uint) error {
	return r.db.Delete(&model.DeviceShare{}, id).Error
}

func (r *gormDeviceShareRepository) DeleteByInstanceAndSharedWith(instanceUUID, sharedWithUUID string) error {
	return r.db.Where("instance_uuid = ? AND shared_with_uuid = ?", instanceUUID, sharedWithUUID).
		Delete(&model.DeviceShare{}).Error
}

func (r *gormDeviceShareRepository) CountActiveShares(instanceUUID string) (int64, error) {
	var count int64
	now := time.Now().Unix()
	err := r.db.Model(&model.DeviceShare{}).
		Where("instance_uuid = ? AND status = ?", instanceUUID, StatusActive).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Count(&count).Error
	return count, err
}

func (r *gormDeviceShareRepository) CountActiveSharesBatch(instanceUUIDs []string) (map[string]int64, error) {
	if len(instanceUUIDs) == 0 {
		return map[string]int64{}, nil
	}
	// Dedup to avoid duplicate IN placeholders and duplicate GROUP BY work.
	seen := make(map[string]struct{}, len(instanceUUIDs))
	uniq := make([]string, 0, len(instanceUUIDs))
	for _, id := range instanceUUIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			uniq = append(uniq, id)
		}
	}
	if len(uniq) == 0 {
		return map[string]int64{}, nil
	}
	// Batch GROUP BY COUNT vs per-row COUNT:
	// Single `SELECT instance_uuid, COUNT(*) ... WHERE instance_uuid IN (?) AND status='active' AND (expires_at IS NULL OR expires_at>now) GROUP BY instance_uuid`
	// replaces M per-device CountActiveShares queries (M = len(allAccessibleDevices)).
	// MySQL can use index on instance_uuid to avoid temp table for GROUP BY (see https://dev.mysql.com/doc/refman/26.7/en/group-by-optimization.html
	// and https://oneuptime.com/blog/post/2026-03-31-mysql-optimize-group-by/view).
	// Chunk IN at 1000 to stay under Oracle 1000 limit (https://stackoverflow.com/questions/400255/how-to-put-more-than-1000-values-into-an-oracle-in-clause),
	// MySQL max_allowed_packet (https://stackoverflow.com/questions/1532366/mysql-number-of-items-within-in-clause),
	// PG 65535 (https://github.com/go-gorm/gorm/issues/6849), and GORM preload splitting guidance (https://github.com/go-gorm/gorm/issues/7792).
	const batchSize = 1000
	now := time.Now().Unix()
	type row struct {
		InstanceUUID string `gorm:"column:instance_uuid"`
		Cnt          int64  `gorm:"column:cnt"`
	}
	result := make(map[string]int64, len(uniq))
	for i := 0; i < len(uniq); i += batchSize {
		end := i + batchSize
		if end > len(uniq) {
			end = len(uniq)
		}
		batch := uniq[i:end]
		var rows []row
		err := r.db.Model(&model.DeviceShare{}).
			Select("instance_uuid, COUNT(*) as cnt").
			Where("instance_uuid IN ?", batch).
			Where("status = ?", StatusActive).
			Where("(expires_at IS NULL OR expires_at > ?)", now).
			Group("instance_uuid").
			Find(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			result[r.InstanceUUID] = r.Cnt
		}
	}
	return result, nil
}
