package repository

import (
	"OMEGA3-IOT/internal/model"

	"gorm.io/gorm"
)

type DeviceGroupRepository interface {
	CreateGroup(group *model.DeviceGroup) error
	GetGroupByID(groupID int64) (*model.DeviceGroup, error)
	GetGroupsByOwner(ownerID int64, page, pageSize int) ([]model.DeviceGroup, int64, error)
	UpdateGroup(group *model.DeviceGroup) error
	DismissGroup(groupID int64) error
	DismissGroupWithTx(tx *gorm.DB, groupID int64) error

	AddDeviceToGroup(relation *model.DeviceGroupRelation) error
	UpdateDeviceGroupRelation(relation *model.DeviceGroupRelation) error
	RemoveDeviceFromGroup(groupID, deviceID int64) error
	RemoveDeviceFromGroupWithTx(tx *gorm.DB, groupID, deviceID int64) error
	RemoveAllDevicesFromGroupWithTx(tx *gorm.DB, groupID int64) error
	GetGroupMembers(groupID int64, page, pageSize int) ([]model.GroupMemberDevice, int64, error)
	GetRelationByGroupAndDevice(groupID, deviceID int64) (*model.DeviceGroupRelation, error)

	WithTx(tx *gorm.DB) DeviceGroupRepository
}

type gormDeviceGroupRepository struct {
	db *gorm.DB
}

func NewDeviceGroupRepository(db *gorm.DB) DeviceGroupRepository {
	return &gormDeviceGroupRepository{db: db}
}

func (r *gormDeviceGroupRepository) WithTx(tx *gorm.DB) DeviceGroupRepository {
	return &gormDeviceGroupRepository{db: tx}
}

func (r *gormDeviceGroupRepository) CreateGroup(group *model.DeviceGroup) error {
	return r.db.Create(group).Error
}

func (r *gormDeviceGroupRepository) GetGroupByID(groupID int64) (*model.DeviceGroup, error) {
	var group model.DeviceGroup
	err := r.db.Where("id = ? AND valid = 1", groupID).First(&group).Error
	return &group, err
}

func (r *gormDeviceGroupRepository) GetGroupsByOwner(ownerID int64, page, pageSize int) ([]model.DeviceGroup, int64, error) {
	var groups []model.DeviceGroup
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	err := r.db.Model(&model.DeviceGroup{}).
		Where("owner_id = ? AND valid = 1", ownerID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.Where("owner_id = ? AND valid = 1", ownerID).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&groups).Error

	return groups, total, err
}

func (r *gormDeviceGroupRepository) UpdateGroup(group *model.DeviceGroup) error {
	return r.db.Save(group).Error
}

func (r *gormDeviceGroupRepository) DismissGroup(groupID int64) error {
	return r.db.Model(&model.DeviceGroup{}).
		Where("id = ?", groupID).
		Update("valid", 0).Error
}

func (r *gormDeviceGroupRepository) DismissGroupWithTx(tx *gorm.DB, groupID int64) error {
	return tx.Model(&model.DeviceGroup{}).
		Where("id = ?", groupID).
		Update("valid", 0).Error
}

func (r *gormDeviceGroupRepository) AddDeviceToGroup(relation *model.DeviceGroupRelation) error {
	return r.db.Create(relation).Error
}

func (r *gormDeviceGroupRepository) UpdateDeviceGroupRelation(relation *model.DeviceGroupRelation) error {
	return r.db.Model(&model.DeviceGroupRelation{}).
		Where("group_id = ? AND device_id = ?", relation.GroupID, relation.DeviceID).
		Updates(map[string]interface{}{
			"valid":     relation.Valid,
			"joined_at": relation.JoinedAt,
		}).Error
}

func (r *gormDeviceGroupRepository) RemoveDeviceFromGroup(groupID, deviceID int64) error {
	return r.db.Model(&model.DeviceGroupRelation{}).
		Where("group_id = ? AND device_id = ?", groupID, deviceID).
		Update("valid", 0).Error
}

func (r *gormDeviceGroupRepository) RemoveDeviceFromGroupWithTx(tx *gorm.DB, groupID, deviceID int64) error {
	return tx.Model(&model.DeviceGroupRelation{}).
		Where("group_id = ? AND device_id = ?", groupID, deviceID).
		Update("valid", 0).Error
}

func (r *gormDeviceGroupRepository) RemoveAllDevicesFromGroupWithTx(tx *gorm.DB, groupID int64) error {
	return tx.Model(&model.DeviceGroupRelation{}).
		Where("group_id = ?", groupID).
		Update("valid", 0).Error
}

func (r *gormDeviceGroupRepository) GetGroupMembers(groupID int64, page, pageSize int) ([]model.GroupMemberDevice, int64, error) {
	var members []model.GroupMemberDevice
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	countErr := r.db.Model(&model.DeviceGroupRelation{}).
		Where("group_id = ? AND valid = 1", groupID).
		Count(&total).Error
	if countErr != nil {
		return nil, 0, countErr
	}

	query := `
		SELECT
			i.id as instance_id,
			i.instance_uuid as instance_uuid,
			i.name as name,
			i.type as type,
			i.online as online,
			i.owner_uuid as owner_uuid,
			i.description as description,
			i.properties as properties,
			i.status as status,
			dgr.joined_at as joined_at
		FROM device_group_relation dgr
		JOIN instances i ON dgr.device_id = i.id
		WHERE dgr.group_id = ? AND dgr.valid = 1 AND i.status = 'active'
		ORDER BY dgr.joined_at DESC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(query, groupID, pageSize, offset).Scan(&members).Error

	return members, total, err
}

func (r *gormDeviceGroupRepository) GetRelationByGroupAndDevice(groupID, deviceID int64) (*model.DeviceGroupRelation, error) {
	var relation model.DeviceGroupRelation
	err := r.db.Where("group_id = ? AND device_id = ?", groupID, deviceID).First(&relation).Error
	return &relation, err
}
