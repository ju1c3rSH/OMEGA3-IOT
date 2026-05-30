package repository

import (
	"OMEGA3-IOT/internal/model"

	"gorm.io/gorm"
)

// DeviceFolderRepository defines the interface for device folder data access.
type DeviceFolderRepository interface {
	CreateFolder(folder *model.DeviceFolder) error
	GetFolderByUUID(folderUUID string) (*model.DeviceFolder, error)
	GetFoldersByOwner(ownerUUID string, page, pageSize int) ([]model.DeviceFolder, int64, error)
	UpdateFolder(folder *model.DeviceFolder) error
	DeleteFolder(folderUUID string) error
	DeleteFolderWithTx(tx *gorm.DB, folderUUID string) error

	AddItem(item *model.DeviceFolderItem) error
	UpdateItem(item *model.DeviceFolderItem) error
	RemoveItem(folderUUID string, deviceUUID string) error
	RemoveItemWithTx(tx *gorm.DB, folderUUID string, deviceUUID string) error
	RemoveAllItemsWithTx(tx *gorm.DB, folderUUID string) error
	GetFolderDevices(folderUUID string, page, pageSize int) ([]model.FolderDeviceItem, int64, error)
	GetItemByFolderAndDevice(folderUUID string, deviceUUID string) (*model.DeviceFolderItem, error)

	WithTx(tx *gorm.DB) DeviceFolderRepository
}

type gormDeviceFolderRepository struct {
	db *gorm.DB
}

// NewDeviceFolderRepository creates a new DeviceFolderRepository.
func NewDeviceFolderRepository(db *gorm.DB) DeviceFolderRepository {
	return &gormDeviceFolderRepository{db: db}
}

func (r *gormDeviceFolderRepository) WithTx(tx *gorm.DB) DeviceFolderRepository {
	return &gormDeviceFolderRepository{db: tx}
}

func (r *gormDeviceFolderRepository) CreateFolder(folder *model.DeviceFolder) error {
	return r.db.Create(folder).Error
}

func (r *gormDeviceFolderRepository) GetFolderByUUID(folderUUID string) (*model.DeviceFolder, error) {
	var folder model.DeviceFolder
	err := r.db.Where("folder_uuid = ? AND valid = 1", folderUUID).First(&folder).Error
	return &folder, err
}

func (r *gormDeviceFolderRepository) GetFoldersByOwner(ownerUUID string, page, pageSize int) ([]model.DeviceFolder, int64, error) {
	var folders []model.DeviceFolder
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	err := r.db.Model(&model.DeviceFolder{}).
		Where("owner_uuid = ? AND valid = 1", ownerUUID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.Where("owner_uuid = ? AND valid = 1", ownerUUID).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&folders).Error

	return folders, total, err
}

func (r *gormDeviceFolderRepository) UpdateFolder(folder *model.DeviceFolder) error {
	return r.db.Save(folder).Error
}

func (r *gormDeviceFolderRepository) DeleteFolder(folderUUID string) error {
	return r.db.Model(&model.DeviceFolder{}).
		Where("folder_uuid = ?", folderUUID).
		Update("valid", 0).Error
}

func (r *gormDeviceFolderRepository) DeleteFolderWithTx(tx *gorm.DB, folderUUID string) error {
	return tx.Model(&model.DeviceFolder{}).
		Where("folder_uuid = ?", folderUUID).
		Update("valid", 0).Error
}

func (r *gormDeviceFolderRepository) AddItem(item *model.DeviceFolderItem) error {
	return r.db.Create(item).Error
}

func (r *gormDeviceFolderRepository) UpdateItem(item *model.DeviceFolderItem) error {
	return r.db.Model(&model.DeviceFolderItem{}).
		Where("folder_uuid = ? AND device_uuid = ?", item.FolderUUID, item.DeviceUUID).
		Updates(map[string]interface{}{
			"valid":     item.Valid,
			"joined_at": item.JoinedAt,
		}).Error
}

func (r *gormDeviceFolderRepository) RemoveItem(folderUUID string, deviceUUID string) error {
	return r.db.Model(&model.DeviceFolderItem{}).
		Where("folder_uuid = ? AND device_uuid = ?", folderUUID, deviceUUID).
		Update("valid", 0).Error
}

func (r *gormDeviceFolderRepository) RemoveItemWithTx(tx *gorm.DB, folderUUID string, deviceUUID string) error {
	return tx.Model(&model.DeviceFolderItem{}).
		Where("folder_uuid = ? AND device_uuid = ?", folderUUID, deviceUUID).
		Update("valid", 0).Error
}

func (r *gormDeviceFolderRepository) RemoveAllItemsWithTx(tx *gorm.DB, folderUUID string) error {
	return tx.Model(&model.DeviceFolderItem{}).
		Where("folder_uuid = ?", folderUUID).
		Update("valid", 0).Error
}

func (r *gormDeviceFolderRepository) GetFolderDevices(folderUUID string, page, pageSize int) ([]model.FolderDeviceItem, int64, error) {
	var devices []model.FolderDeviceItem
	var total int64

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	countErr := r.db.Model(&model.DeviceFolderItem{}).
		Where("folder_uuid = ? AND valid = 1", folderUUID).
		Count(&total).Error
	if countErr != nil {
		return nil, 0, countErr
	}

	query := `
		SELECT
			i.instance_uuid as instance_uuid,
			i.name as name,
			i.type as type,
			i.online as online,
			i.owner_uuid as owner_uuid,
			i.description as description,
			i.properties as properties,
			i.status as status,
			dfi.joined_at as joined_at
		FROM device_folder_item dfi
		JOIN instances i ON dfi.device_uuid = i.instance_uuid
		WHERE dfi.folder_uuid = ? AND dfi.valid = 1 AND i.status = 'active'
		ORDER BY dfi.joined_at DESC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(query, folderUUID, pageSize, offset).Scan(&devices).Error

	return devices, total, err
}

func (r *gormDeviceFolderRepository) GetItemByFolderAndDevice(folderUUID string, deviceUUID string) (*model.DeviceFolderItem, error) {
	var item model.DeviceFolderItem
	err := r.db.Where("folder_uuid = ? AND device_uuid = ?", folderUUID, deviceUUID).First(&item).Error
	return &item, err
}
