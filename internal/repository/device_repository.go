package repository

import (
	"OMEGA3-IOT/internal/model"
	"gorm.io/gorm"
)

type InstanceRepository interface {
	Create(instance *model.Instance) error
	FindByID(id uint) (*model.Instance, error)
	FindByUUID(instanceUUID string) (*model.Instance, error)
	FindByOwnerUUID(ownerUUID string) ([]model.Instance, error)
	FindOwnerUUIDByUUID(instanceUUID string) (string, error)
	FindByUUIDs(instanceUUIDs []string) ([]model.Instance, error)
	Update(instance *model.Instance) error
	UpdateFields(instanceUUID string, fields map[string]interface{}) error
	Delete(id uint) error
	DeleteByUUID(instanceUUID string) error
	UpdateProperties(instanceUUID string, properties model.Properties) error
	UpdateOnlineStatus(instanceUUID string, online bool, lastSeen int64) error
	Exists(instanceUUID string) (bool, error)
	// ExistsByOwner performs indexed point lookup SELECT 1 FROM instances
	// WHERE instance_uuid=? AND owner_uuid=? LIMIT 1.
	// It replaces the anti-pattern FindByOwnerUUID(userUUID) + linear scan.
	ExistsByOwner(ownerUUID, instanceUUID string) (bool, error)

	// Transaction support
	WithTx(tx *gorm.DB) InstanceRepository
}

type gormInstanceRepository struct {
	db *gorm.DB
}

func NewInstanceRepository(db *gorm.DB) InstanceRepository {
	return &gormInstanceRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *gormInstanceRepository) WithTx(tx *gorm.DB) InstanceRepository {
	return &gormInstanceRepository{db: tx}
}

func (r *gormInstanceRepository) Create(instance *model.Instance) error {
	return r.db.Create(instance).Error
}

func (r *gormInstanceRepository) FindByID(id uint) (*model.Instance, error) {
	var instance model.Instance
	err := r.db.First(&instance, id).Error
	return &instance, err
}

func (r *gormInstanceRepository) FindByUUID(instanceUUID string) (*model.Instance, error) {
	var instance model.Instance
	err := r.db.Where("instance_uuid = ?", instanceUUID).First(&instance).Error
	return &instance, err
}

func (r *gormInstanceRepository) FindOwnerUUIDByUUID(instanceUUID string) (string, error) {
	var row struct {
		OwnerUUID string
	}
	err := r.db.Select("owner_uuid").Where("instance_uuid = ?", instanceUUID).First(&row).Error
	return row.OwnerUUID, err
}

func (r *gormInstanceRepository) FindByOwnerUUID(ownerUUID string) ([]model.Instance, error) {
	var instances []model.Instance
	err := r.db.Where("owner_uuid = ?", ownerUUID).Find(&instances).Error
	return instances, err
}

func (r *gormInstanceRepository) FindByUUIDs(instanceUUIDs []string) ([]model.Instance, error) {
	if len(instanceUUIDs) == 0 {
		return []model.Instance{}, nil
	}
	// Dedup to shrink IN list and avoid duplicate placeholders.
	// See https://www.softwarejutsu.com/articles/n-plus-one-what-every-go-engineer-should-know (batch load with IN + dedup + map join)
	seen := make(map[string]struct{}, len(instanceUUIDs))
	uniq := make([]string, 0, len(instanceUUIDs))
	for _, id := range instanceUUIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			uniq = append(uniq, id)
		}
	}
	if len(uniq) == 0 {
		return []model.Instance{}, nil
	}
	// Chunk IN clause to avoid DB limits:
	// - Oracle 1000 expression limit (https://stackoverflow.com/questions/400255/how-to-put-more-than-1000-values-into-an-oracle-in-clause)
	// - MySQL max_allowed_packet (https://stackoverflow.com/questions/1532366/mysql-number-of-items-within-in-clause)
	// - PostgreSQL 65535 parameters (https://github.com/go-gorm/gorm/issues/6849, https://github.com/go-gorm/gorm/issues/6989)
	// - GORM preload batch splitting (https://github.com/go-gorm/gorm/issues/7792)
	// Batch of 1000 keeps plan efficient and avoids degraded index handling.
	const batchSize = 1000
	var result []model.Instance
	for i := 0; i < len(uniq); i += batchSize {
		end := i + batchSize
		if end > len(uniq) {
			end = len(uniq)
		}
		batch := uniq[i:end]
		var batchResult []model.Instance
		// GORM translates `IN ?` to `IN (?)` with bound args; single batched query
		// replaces N per-row FindByUUID calls (see https://gorm.io/docs/query.html
		// and https://blog.stackademic.com/the-n-1-query-problem-in-gorm-how-to-avoid-silent-performance-killers-856e028d4b15)
		if err := r.db.Where("instance_uuid IN ?", batch).Find(&batchResult).Error; err != nil {
			return nil, err
		}
		result = append(result, batchResult...)
	}
	return result, nil
}

func (r *gormInstanceRepository) Update(instance *model.Instance) error {
	return r.db.Save(instance).Error
}

func (r *gormInstanceRepository) UpdateFields(instanceUUID string, fields map[string]interface{}) error {
	return r.db.Model(&model.Instance{}).Where("instance_uuid = ?", instanceUUID).Updates(fields).Error
}

func (r *gormInstanceRepository) Delete(id uint) error {
	return r.db.Delete(&model.Instance{}, id).Error
}

func (r *gormInstanceRepository) DeleteByUUID(instanceUUID string) error {
	return r.db.Where("instance_uuid = ?", instanceUUID).Delete(&model.Instance{}).Error
}

func (r *gormInstanceRepository) Exists(instanceUUID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Instance{}).Where("instance_uuid = ?", instanceUUID).Count(&count).Error
	return count > 0, err
}

func (r *gormInstanceRepository) ExistsByOwner(ownerUUID, instanceUUID string) (bool, error) {
	// Indexed point lookup — uses uniqueIndex on instance_uuid plus index on owner_uuid.
	// Avoids the full-scan anti-pattern FindByOwnerUUID(user) then linear scan.
	// Pitfalls avoided:
	// - COUNT(*) without LIMIT scans all matches vs Limit 1 stops after first row
	//   (see https://cdn.jsdelivr.net/npm/miokit@2.0.13/skills/external/golang/gorm/performance.md and https://stackoverflow.com/questions/66392372/select-exists-with-gorm)
	// - SELECT * for existence fetches Properties JSON; SELECT 1 (or PK) is lighter.
	// - Raw `SELECT EXISTS(...)` requires hard-coded table name; RowsAffected approach
	//   keeps table name derived from model via GORM, safe for plugins/prefix.
	// Query plan: MySQL uses uniqueIndex on instance_uuid to locate 1 row, then
	// filters owner_uuid equality; composite (owner_uuid,instance_uuid) would be
	// covering but not required — verification: instances already has index on owner_uuid
	// and uniqueIndex on instance_uuid, so point lookup is O(log n).
	// See https://github.com/go-gorm/gorm/discussions/6000 for EXISTS vs Limit 1 discussion.
	var dummy []model.Instance
	result := r.db.Model(&model.Instance{}).
		Select("instance_uuid").
		Where("instance_uuid = ? AND owner_uuid = ?", instanceUUID, ownerUUID).
		Limit(1).
		Find(&dummy)
	if result.Error != nil {
		return false, result.Error
	}
	return len(dummy) > 0, nil
}

func (r *gormInstanceRepository) UpdateProperties(instanceUUID string, properties model.Properties) error {
	return r.db.Model(&model.Instance{}).
		Where("instance_uuid = ?", instanceUUID).
		Update("properties", properties).Error
}

func (r *gormInstanceRepository) UpdateOnlineStatus(instanceUUID string, online bool, lastSeen int64) error {
	return r.db.Model(&model.Instance{}).
		Where("instance_uuid = ?", instanceUUID).
		Updates(map[string]interface{}{
			"online":    online,
			"last_seen": lastSeen,
		}).Error
}
