package service

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"OMEGA3-IOT/internal/utils"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/apache/iotdb-client-go/client"
	"gorm.io/gorm"
)

// DeviceFolderService handles device folder (organizational grouping) business logic.
type DeviceFolderService struct {
	folderRepo      repository.DeviceFolderRepository
	instanceRepo    repository.InstanceRepository
	deviceShareRepo repository.DeviceShareRepository
	loggerService   logger.LoggerInterface
	iotdbClient     *db.IOTDBClient
	db              *gorm.DB
}

// NewDeviceFolderService creates a new DeviceFolderService.
func NewDeviceFolderService(db *gorm.DB, iotdbClient *db.IOTDBClient, loggerService logger.LoggerInterface) *DeviceFolderService {
	return &DeviceFolderService{
		folderRepo:      repository.NewDeviceFolderRepository(db),
		instanceRepo:    repository.NewInstanceRepository(db),
		deviceShareRepo: repository.NewDeviceShareRepository(db),
		loggerService:   loggerService,
		iotdbClient:     iotdbClient,
		db:              db,
	}
}

// CreateFolder creates a new device folder.
func (s *DeviceFolderService) CreateFolder(name, description string, ownerUUID string) (*model.DeviceFolder, error) {
	folder := &model.DeviceFolder{
		FolderUUID:  utils.GenerateUUID().String(),
		Name:        name,
		OwnerUUID:   ownerUUID,
		Description: description,
		Valid:       1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.folderRepo.CreateFolder(folder); err != nil {
		log.Printf("[DeviceFolderService] Failed to create folder: owner_uuid=%s, error=%v", ownerUUID, err)
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	logEvent := logger.NewUserLogEvent(ownerUUID, logger.LogLevelInfo,
		fmt.Sprintf("Folder created: %s", name),
		logger.LogEventFolderCreated)
	logEvent.Metadata = map[string]interface{}{
		"folder_uuid": folder.FolderUUID,
		"folder_name": name,
		"owner_uuid":  ownerUUID,
	}
	s.loggerService.EmitUserLog(logEvent)

	s.reportIotdbMetric("FOLDER_CREATED", utils.ParseUserIDFromUUID(ownerUUID))

	return folder, nil
}

// AddDeviceToFolder adds a device to a folder.
func (s *DeviceFolderService) AddDeviceToFolder(folderUUID string, deviceUUID string, userUUID string) error {
	if deviceUUID == "" {
		return fmt.Errorf("device uuid is required")
	}

	folder, err := s.folderRepo.GetFolderByUUID(folderUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[DeviceFolderService] Folder not found: folder_uuid=%s, user_uuid=%s", folderUUID, userUUID)
			return fmt.Errorf("invalid folder")
		}
		log.Printf("[DeviceFolderService] Failed to get folder: folder_uuid=%s, error=%v", folderUUID, err)
		return fmt.Errorf("failed to get folder: %w", err)
	}

	instance, err := s.instanceRepo.FindByUUID(deviceUUID)
	if err != nil {
		log.Printf("[DeviceFolderService] Device not found: device_uuid=%s, user_uuid=%s", deviceUUID, userUUID)
		return fmt.Errorf("device not found")
	}

	hasAccess, err := s.checkDeviceAccessByUUID(instance.InstanceUUID, userUUID, "write")
	if err != nil || !hasAccess {
		log.Printf("[DeviceFolderService] Device access denied: device_uuid=%s, user_uuid=%s", deviceUUID, userUUID)
		return fmt.Errorf("device access denied")
	}

	if folder.OwnerUUID != userUUID {
		log.Printf("[DeviceFolderService] Permission denied: folder_uuid=%s, user_uuid=%s, owner_uuid=%s", folderUUID, userUUID, folder.OwnerUUID)
		return fmt.Errorf("permission denied")
	}

	existingItem, err := s.folderRepo.GetItemByFolderAndDevice(folderUUID, deviceUUID)
	if err == nil && existingItem != nil {
		if existingItem.Valid == 1 {
			return nil
		}
		existingItem.Valid = 1
		existingItem.JoinedAt = time.Now()
		if updateErr := s.folderRepo.UpdateItem(existingItem); updateErr != nil {
			log.Printf("[DeviceFolderService] Failed to update item: folder_uuid=%s, device_uuid=%s, error=%v", folderUUID, deviceUUID, updateErr)
			return fmt.Errorf("failed to add to folder: %w", updateErr)
		}
	} else {
		item := &model.DeviceFolderItem{
			FolderUUID: folderUUID,
			DeviceUUID: deviceUUID,
			JoinedAt:   time.Now(),
			Valid:      1,
		}
		if createErr := s.folderRepo.AddItem(item); createErr != nil {
			if isDuplicateKeyError(createErr) {
				existingItem, getErr := s.folderRepo.GetItemByFolderAndDevice(folderUUID, deviceUUID)
				if getErr == nil && existingItem != nil {
					existingItem.Valid = 1
					existingItem.JoinedAt = time.Now()
					s.folderRepo.UpdateItem(existingItem)
					return nil
				}
			}
			log.Printf("[DeviceFolderService] Failed to add device to folder: folder_uuid=%s, device_uuid=%s, error=%v", folderUUID, deviceUUID, createErr)
			return fmt.Errorf("failed to add to folder: %w", createErr)
		}
	}

	logEvent := logger.NewUserLogEvent(userUUID, logger.LogLevelInfo,
		fmt.Sprintf("Device added to folder: device_uuid=%s, folder_uuid=%s", deviceUUID, folderUUID),
		logger.LogEventDeviceAddedToFolder)
	logEvent.Metadata = map[string]interface{}{
		"folder_uuid": folderUUID,
		"device_uuid": deviceUUID,
		"folder_name": folder.Name,
	}
	s.loggerService.EmitUserLog(logEvent)

	s.reportIotdbMetric("DEVICE_ADDED_TO_FOLDER", utils.ParseUserIDFromUUID(userUUID))

	return nil
}

// RemoveDeviceFromFolder removes a device from a folder.
func (s *DeviceFolderService) RemoveDeviceFromFolder(folderUUID string, deviceUUID string, userUUID string) error {
	if deviceUUID == "" {
		return fmt.Errorf("device uuid is required")
	}

	folder, err := s.folderRepo.GetFolderByUUID(folderUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[DeviceFolderService] Folder not found: folder_uuid=%s, user_uuid=%s", folderUUID, userUUID)
			return fmt.Errorf("invalid folder")
		}
		log.Printf("[DeviceFolderService] Failed to get folder: folder_uuid=%s, error=%v", folderUUID, err)
		return fmt.Errorf("failed to get folder: %w", err)
	}

	instance, err := s.instanceRepo.FindByUUID(deviceUUID)
	if err != nil {
		log.Printf("[DeviceFolderService] Device not found: device_uuid=%s, user_uuid=%s", deviceUUID, userUUID)
		return fmt.Errorf("device not found")
	}

	hasAccess, err := s.checkDeviceAccessByUUID(instance.InstanceUUID, userUUID, "write")
	if err != nil || !hasAccess {
		log.Printf("[DeviceFolderService] Device access denied: device_uuid=%s, user_uuid=%s", deviceUUID, userUUID)
		return fmt.Errorf("device access denied")
	}

	if folder.OwnerUUID != userUUID {
		log.Printf("[DeviceFolderService] Permission denied: folder_uuid=%s, user_uuid=%s, owner_uuid=%s", folderUUID, userUUID, folder.OwnerUUID)
		return fmt.Errorf("permission denied")
	}

	if err := s.folderRepo.RemoveItem(folderUUID, deviceUUID); err != nil {
		log.Printf("[DeviceFolderService] Failed to remove device from folder: folder_uuid=%s, device_uuid=%s, error=%v", folderUUID, deviceUUID, err)
		return fmt.Errorf("failed to remove from folder: %w", err)
	}

	logEvent := logger.NewUserLogEvent(userUUID, logger.LogLevelInfo,
		fmt.Sprintf("Device removed from folder: device_uuid=%s, folder_uuid=%s", deviceUUID, folderUUID),
		logger.LogEventDeviceRemovedFromFolder)
	logEvent.Metadata = map[string]interface{}{
		"folder_uuid": folderUUID,
		"device_uuid": deviceUUID,
		"folder_name": folder.Name,
	}
	s.loggerService.EmitUserLog(logEvent)

	s.reportIotdbMetric("DEVICE_REMOVED_FROM_FOLDER", utils.ParseUserIDFromUUID(userUUID))

	return nil
}

// GetFolders returns folders owned by a user.
func (s *DeviceFolderService) GetFolders(ownerUUID string, page, pageSize int) ([]model.DeviceFolder, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	return s.folderRepo.GetFoldersByOwner(ownerUUID, page, pageSize)
}

// GetFolderDevices returns devices in a folder.
func (s *DeviceFolderService) GetFolderDevices(folderUUID string, userUUID string, page, pageSize int) ([]model.FolderDeviceItem, int64, error) {
	folder, err := s.folderRepo.GetFolderByUUID(folderUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[DeviceFolderService] Folder not found for devices query: folder_uuid=%s, user_uuid=%s", folderUUID, userUUID)
			return nil, 0, fmt.Errorf("folder not found")
		}
		log.Printf("[DeviceFolderService] Failed to get folder for devices query: folder_uuid=%s, error=%v", folderUUID, err)
		return nil, 0, fmt.Errorf("folder not found")
	}

	if folder.OwnerUUID != userUUID {
		log.Printf("[DeviceFolderService] Permission denied for devices query: folder_uuid=%s, user_uuid=%s, owner_uuid=%s", folderUUID, userUUID, folder.OwnerUUID)
		return nil, 0, fmt.Errorf("folder not found")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	return s.folderRepo.GetFolderDevices(folderUUID, page, pageSize)
}

// DeleteFolder deletes a folder and removes all its items.
func (s *DeviceFolderService) DeleteFolder(folderUUID string, userUUID string) error {
	folder, err := s.folderRepo.GetFolderByUUID(folderUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[DeviceFolderService] Folder not found for delete: folder_uuid=%s, user_uuid=%s", folderUUID, userUUID)
			return fmt.Errorf("folder not found")
		}
		log.Printf("[DeviceFolderService] Failed to get folder for delete: folder_uuid=%s, error=%v", folderUUID, err)
		return fmt.Errorf("folder not found")
	}

	if folder.OwnerUUID != userUUID {
		log.Printf("[DeviceFolderService] Permission denied for delete: folder_uuid=%s, user_uuid=%s, owner_uuid=%s", folderUUID, userUUID, folder.OwnerUUID)
		return fmt.Errorf("permission denied")
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		log.Printf("[DeviceFolderService] Failed to begin transaction: folder_uuid=%s, error=%v", folderUUID, tx.Error)
		return fmt.Errorf("failed to delete folder: %w", tx.Error)
	}

	folderRepoWithTx := s.folderRepo.WithTx(tx)

	if err := folderRepoWithTx.DeleteFolderWithTx(tx, folderUUID); err != nil {
		tx.Rollback()
		log.Printf("[DeviceFolderService] Failed to delete folder: folder_uuid=%s, error=%v", folderUUID, err)
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	if err := folderRepoWithTx.RemoveAllItemsWithTx(tx, folderUUID); err != nil {
		tx.Rollback()
		log.Printf("[DeviceFolderService] Failed to remove items from folder: folder_uuid=%s, error=%v", folderUUID, err)
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("[DeviceFolderService] Failed to commit transaction: folder_uuid=%s, error=%v", folderUUID, err)
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	logEvent := logger.NewUserLogEvent(userUUID, logger.LogLevelInfo,
		fmt.Sprintf("Folder deleted: %s", folder.Name),
		logger.LogEventFolderDeleted)
	logEvent.Metadata = map[string]interface{}{
		"folder_uuid": folderUUID,
		"folder_name": folder.Name,
	}
	s.loggerService.EmitUserLog(logEvent)

	s.reportIotdbMetric("FOLDER_DELETED", utils.ParseUserIDFromUUID(userUUID))

	return nil
}

func (s *DeviceFolderService) checkDeviceAccessByUUID(instanceUUID, userUUID, requiredPermission string) (bool, error) {
	devices, err := s.instanceRepo.FindByOwnerUUID(userUUID)
	if err != nil {
		return false, err
	}

	for _, d := range devices {
		if d.InstanceUUID == instanceUUID {
			return true, nil
		}
	}

	share, err := s.deviceShareRepo.FindByInstanceAndSharedWith(instanceUUID, userUUID)
	if err != nil {
		return false, nil
	}

	if share.ExpiresAt != nil && *share.ExpiresAt <= time.Now().Unix() {
		return false, nil
	}

	if requiredPermission == repository.PermissionRead &&
		(share.Permission == repository.PermissionRead || share.Permission == repository.PermissionReadWrite) {
		return true, nil
	}
	if requiredPermission == repository.PermissionWrite &&
		(share.Permission == repository.PermissionWrite || share.Permission == repository.PermissionReadWrite) {
		return true, nil
	}
	return false, nil
}

func (s *DeviceFolderService) reportIotdbMetric(operationType string, userID int64) {
	if s.iotdbClient == nil {
		return
	}

	path := "root.mm1.metrics.folder_operations"
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	measurements := []string{"operation_type", "user_id", "count"}
	dataTypes := []client.TSDataType{client.STRING, client.INT64, client.INT64}
	values := []interface{}{operationType, userID, int64(1)}

	session, err := s.iotdbClient.SessionPool.GetSession()
	if err != nil {
		log.Printf("[DeviceFolderService] Failed to get IoTDB session for metrics: %v", err)
		return
	}
	defer s.iotdbClient.SessionPool.PutBack(session)

	_, err = session.InsertRecord(path, measurements, dataTypes, values, timestamp)
	if err != nil {
		log.Printf("[DeviceFolderService] Failed to write metrics to IoTDB: %v", err)
	}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Duplicate entry") || strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "1062")
}
