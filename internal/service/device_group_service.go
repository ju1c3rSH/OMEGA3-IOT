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

type DeviceGroupService struct {
	groupRepo     repository.DeviceGroupRepository
	instanceRepo  repository.InstanceRepository
	deviceShareRepo repository.DeviceShareRepository
	loggerService logger.LoggerInterface
	iotdbClient   *db.IOTDBClient
	db            *gorm.DB
}

func NewDeviceGroupService(db *gorm.DB, iotdbClient *db.IOTDBClient, loggerService logger.LoggerInterface) *DeviceGroupService {
	return &DeviceGroupService{
		groupRepo:       repository.NewDeviceGroupRepository(db),
		instanceRepo:    repository.NewInstanceRepository(db),
		deviceShareRepo: repository.NewDeviceShareRepository(db),
		loggerService:   loggerService,
		iotdbClient:     iotdbClient,
		db:              db,
	}
}

func (s *DeviceGroupService) CreateGroup(name, description string, ownerID int64) (*model.DeviceGroup, error) {
	group := &model.DeviceGroup{
		ID:          utils.GenerateSnowflakeID(),
		Name:        name,
		OwnerID:     ownerID,
		Description: description,
		Valid:       1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.groupRepo.CreateGroup(group); err != nil {
		log.Printf("[DeviceGroupService] Failed to create group: owner_id=%d, error=%v", ownerID, err)
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	logEvent := logger.NewUserLogEvent(fmt.Sprintf("%d", ownerID), logger.LogLevelInfo,
		fmt.Sprintf("Group created: %s", name),
		logger.LogEventGroupCreated)
	logEvent.Metadata = map[string]interface{}{
		"group_id":    group.ID,
		"group_name":  name,
		"owner_id":    ownerID,
	}
	s.loggerService.EmitUserLog(logEvent)

	s.reportIotdbMetric("GROUP_CREATED", ownerID)

	return group, nil
}

func (s *DeviceGroupService) JoinGroup(groupID int64, deviceUUID string, userUUID string) error {
	// 验证 UUID 格式
	if deviceUUID == "" {
		return fmt.Errorf("device uuid is required")
	}

	group, err := s.groupRepo.GetGroupByID(groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[DeviceGroupService] Group not found: group_id=%d, user_uuid=%s", groupID, userUUID)
			return fmt.Errorf("invalid group")
		}
		log.Printf("[DeviceGroupService] Failed to get group: group_id=%d, error=%v", groupID, err)
		return fmt.Errorf("failed to get group: %w", err)
	}

	instance, err := s.instanceRepo.FindByUUID(deviceUUID)
	if err != nil {
		log.Printf("[DeviceGroupService] Device not found: device_uuid=%s, user_uuid=%s", deviceUUID, userUUID)
		return fmt.Errorf("device not found")
	}

	hasAccess, err := s.checkDeviceAccessByUUID(instance.InstanceUUID, userUUID, "write")
	if err != nil || !hasAccess {
		log.Printf("[DeviceGroupService] Device access denied: device_uuid=%s, user_uuid=%s", deviceUUID, userUUID)
		return fmt.Errorf("device access denied")
	}

	if group.OwnerID != utils.ParseUserIDFromUUID(userUUID) {
		log.Printf("[DeviceGroupService] Permission denied: group_id=%d, user_uuid=%s, owner_id=%d", groupID, userUUID, group.OwnerID)
		return fmt.Errorf("permission denied")
	}

	existingRelation, err := s.groupRepo.GetRelationByGroupAndDevice(groupID, deviceUUID)
	if err == nil && existingRelation != nil {
		if existingRelation.Valid == 1 {
			return nil
		}
		existingRelation.Valid = 1
		existingRelation.JoinedAt = time.Now()
		if updateErr := s.groupRepo.UpdateDeviceGroupRelation(existingRelation); updateErr != nil {
			log.Printf("[DeviceGroupService] Failed to update relation: group_id=%d, device_uuid=%s, error=%v", groupID, deviceUUID, updateErr)
			return fmt.Errorf("failed to join group: %w", updateErr)
		}
	} else {
		relation := &model.DeviceGroupRelation{
			GroupID:    groupID,
			DeviceUUID: deviceUUID,
			JoinedAt:   time.Now(),
			Valid:      1,
		}
		if createErr := s.groupRepo.AddDeviceToGroup(relation); createErr != nil {
			if isDuplicateKeyError(createErr) {
				existingRelation, getErr := s.groupRepo.GetRelationByGroupAndDevice(groupID, deviceUUID)
				if getErr == nil && existingRelation != nil {
					existingRelation.Valid = 1
					existingRelation.JoinedAt = time.Now()
					s.groupRepo.UpdateDeviceGroupRelation(existingRelation)
					return nil
				}
			}
			log.Printf("[DeviceGroupService] Failed to add device to group: group_id=%d, device_uuid=%s, error=%v", groupID, deviceUUID, createErr)
			return fmt.Errorf("failed to join group: %w", createErr)
		}
	}

	logEvent := logger.NewUserLogEvent(userUUID, logger.LogLevelInfo,
		fmt.Sprintf("Device joined group: device_uuid=%s, group_id=%d", deviceUUID, groupID),
		logger.LogEventDeviceJoinedGroup)
	logEvent.Metadata = map[string]interface{}{
		"group_id":    groupID,
		"device_uuid": deviceUUID,
		"group_name":  group.Name,
	}
	s.loggerService.EmitUserLog(logEvent)

	s.reportIotdbMetric("DEVICE_JOINED_GROUP", utils.ParseUserIDFromUUID(userUUID))

	return nil
}

func (s *DeviceGroupService) QuitGroup(groupID int64, deviceUUID string, userUUID string) error {
	// 验证 UUID 格式
	if deviceUUID == "" {
		return fmt.Errorf("device uuid is required")
	}

	group, err := s.groupRepo.GetGroupByID(groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[DeviceGroupService] Group not found: group_id=%d, user_uuid=%s", groupID, userUUID)
			return fmt.Errorf("invalid group")
		}
		log.Printf("[DeviceGroupService] Failed to get group: group_id=%d, error=%v", groupID, err)
		return fmt.Errorf("failed to get group: %w", err)
	}

	instance, err := s.instanceRepo.FindByUUID(deviceUUID)
	if err != nil {
		log.Printf("[DeviceGroupService] Device not found: device_uuid=%s, user_uuid=%s", deviceUUID, userUUID)
		return fmt.Errorf("device not found")
	}

	hasAccess, err := s.checkDeviceAccessByUUID(instance.InstanceUUID, userUUID, "write")
	if err != nil || !hasAccess {
		log.Printf("[DeviceGroupService] Device access denied: device_uuid=%s, user_uuid=%s", deviceUUID, userUUID)
		return fmt.Errorf("device access denied")
	}

	if group.OwnerID != utils.ParseUserIDFromUUID(userUUID) {
		log.Printf("[DeviceGroupService] Permission denied: group_id=%d, user_uuid=%s, owner_id=%d", groupID, userUUID, group.OwnerID)
		return fmt.Errorf("permission denied")
	}

	if err := s.groupRepo.RemoveDeviceFromGroup(groupID, deviceUUID); err != nil {
		log.Printf("[DeviceGroupService] Failed to remove device from group: group_id=%d, device_uuid=%s, error=%v", groupID, deviceUUID, err)
		return fmt.Errorf("failed to quit group: %w", err)
	}

	logEvent := logger.NewUserLogEvent(userUUID, logger.LogLevelInfo,
		fmt.Sprintf("Device quit group: device_uuid=%s, group_id=%d", deviceUUID, groupID),
		logger.LogEventDeviceQuitGroup)
	logEvent.Metadata = map[string]interface{}{
		"group_id":    groupID,
		"device_uuid": deviceUUID,
		"group_name":  group.Name,
	}
	s.loggerService.EmitUserLog(logEvent)

	s.reportIotdbMetric("DEVICE_QUIT_GROUP", utils.ParseUserIDFromUUID(userUUID))

	return nil
}

func (s *DeviceGroupService) GetGroups(ownerID int64, page, pageSize int) ([]model.DeviceGroup, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	return s.groupRepo.GetGroupsByOwner(ownerID, page, pageSize)
}

func (s *DeviceGroupService) GetGroupMembers(groupID int64, userUUID string, page, pageSize int) ([]model.GroupMemberDevice, int64, error) {
	group, err := s.groupRepo.GetGroupByID(groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[DeviceGroupService] Group not found for members query: group_id=%d, user_uuid=%s", groupID, userUUID)
			return nil, 0, fmt.Errorf("Could not find a match group")
		}
		log.Printf("[DeviceGroupService] Failed to get group for members query: group_id=%d, error=%v", groupID, err)
		return nil, 0, fmt.Errorf("Could not find a match group")
	}

	if group.OwnerID != utils.ParseUserIDFromUUID(userUUID) {
		log.Printf("[DeviceGroupService] Permission denied for members query: group_id=%d, user_uuid=%s, owner_id=%d", groupID, userUUID, group.OwnerID)
		return nil, 0, fmt.Errorf("Could not find a match group")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	return s.groupRepo.GetGroupMembers(groupID, page, pageSize)
}

func (s *DeviceGroupService) DismissGroup(groupID int64, userUUID string) error {
	group, err := s.groupRepo.GetGroupByID(groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[DeviceGroupService] Group not found for dismiss: group_id=%d, user_uuid=%s", groupID, userUUID)
			return fmt.Errorf("Could not find a match group")
		}
		log.Printf("[DeviceGroupService] Failed to get group for dismiss: group_id=%d, error=%v", groupID, err)
		return fmt.Errorf("Could not find a match group")
	}

	if group.OwnerID != utils.ParseUserIDFromUUID(userUUID) {
		log.Printf("[DeviceGroupService] Permission denied for dismiss: group_id=%d, user_uuid=%s, owner_id=%d", groupID, userUUID, group.OwnerID)
		return fmt.Errorf("Could not find a match group")
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		log.Printf("[DeviceGroupService] Failed to begin transaction: group_id=%d, error=%v", groupID, tx.Error)
		return fmt.Errorf("failed to dismiss group: %w", tx.Error)
	}

	groupRepoWithTx := s.groupRepo.WithTx(tx)

	if err := groupRepoWithTx.DismissGroupWithTx(tx, groupID); err != nil {
		tx.Rollback()
		log.Printf("[DeviceGroupService] Failed to dismiss group: group_id=%d, error=%v", groupID, err)
		return fmt.Errorf("failed to dismiss group: %w", err)
	}

	if err := groupRepoWithTx.RemoveAllDevicesFromGroupWithTx(tx, groupID); err != nil {
		tx.Rollback()
		log.Printf("[DeviceGroupService] Failed to remove devices from group: group_id=%d, error=%v", groupID, err)
		return fmt.Errorf("failed to dismiss group: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("[DeviceGroupService] Failed to commit transaction: group_id=%d, error=%v", groupID, err)
		return fmt.Errorf("failed to dismiss group: %w", err)
	}

	logEvent := logger.NewUserLogEvent(userUUID, logger.LogLevelInfo,
		fmt.Sprintf("Group dismissed: %s", group.Name),
		logger.LogEventGroupDismissed)
	logEvent.Metadata = map[string]interface{}{
		"group_id":   groupID,
		"group_name": group.Name,
	}
	s.loggerService.EmitUserLog(logEvent)

	s.reportIotdbMetric("GROUP_DISMISSED", utils.ParseUserIDFromUUID(userUUID))

	return nil
}

func (s *DeviceGroupService) checkDeviceAccessByUUID(instanceUUID, userUUID, requiredPermission string) (bool, error) {
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

func (s *DeviceGroupService) reportIotdbMetric(operationType string, userID int64) {
	if s.iotdbClient == nil {
		return
	}

	path := "root.mm1.metrics.group_operations"
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	measurements := []string{"operation_type", "user_id", "count"}
	dataTypes := []client.TSDataType{client.STRING, client.INT64, client.INT64}
	values := []interface{}{operationType, userID, int64(1)}

	session, err := s.iotdbClient.SessionPool.GetSession()
	if err != nil {
		log.Printf("[DeviceGroupService] Failed to get IoTDB session for metrics: %v", err)
		return
	}
	defer s.iotdbClient.SessionPool.PutBack(session)

	_, err = session.InsertRecord(path, measurements, dataTypes, values, timestamp)
	if err != nil {
		log.Printf("[DeviceGroupService] Failed to write metrics to IoTDB: %v", err)
	}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Duplicate entry") || strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "1062")
}
