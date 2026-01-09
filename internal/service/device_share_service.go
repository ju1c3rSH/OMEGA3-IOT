package service

import (
	"OMEGA3-IOT/internal/model"
	"fmt"
	"gorm.io/gorm"
	"log"
	"time"
)

type DeviceShareService struct {
	mysqldb *gorm.DB
}

const (
	StatusActive  = "active"
	StatusRevoked = "revoked"
)
const (
	PermissionRead      = "read"
	PermissionWrite     = "write"
	PermissionReadWrite = "read_write" // 或者 "readwrite"
)

func NewDeviceShareService(db *gorm.DB) *DeviceShareService {
	return &DeviceShareService{mysqldb: db}
}

func (s *DeviceShareService) ShareDevice(instanceUUID string, shareByUUID string, shareWithUUID string, expiredTime int64, permission string) error {
	var instance model.Instance

	if err := s.mysqldb.Where("instance_uuid = ? AND owner_uuid = ?", instanceUUID, shareByUUID).First(&instance).Error; err != nil {
		log.Println(err)
		return err
	}

	share := model.DeviceShare{
		InstanceUUID:   instanceUUID,
		SharedByUUID:   shareByUUID,
		SharedWithUUID: shareWithUUID,
		ExpiresAt:      &expiredTime,
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
		Permission:     permission,
		Status:         StatusActive,
	}
	return s.mysqldb.Create(&share).Error
}
func (s *DeviceShareService) UnshareDevice(instanceUUID string, shareWithUUID string) error {
	var share model.DeviceShare
	if err := s.mysqldb.Where("instance_uuid = ? AND shared_with_uuid = ?", instanceUUID, shareWithUUID).First(&share).Error; err != nil {
		log.Println(err)
	}
	return s.mysqldb.Where("instance_uuid = ?", instanceUUID).Delete(&share).Error
}
func (s *DeviceShareService) GetSharedDevices(instanceUUID string) ([]model.DeviceShare, error) {
	var shares []model.DeviceShare
	return shares, s.mysqldb.Where("instance_uuid = ?", instanceUUID).Find(&shares).Error
}
func (s *DeviceShareService) GetSharedBy(instanceUUID string) ([]model.DeviceShare, error) {
	var shares []model.DeviceShare
	return shares, s.mysqldb.Where("shared_with_uuid = ?", instanceUUID).Find(&shares).Error
}
func (s *DeviceShareService) GetSharedWith(instanceUUID string) ([]model.DeviceShare, error) {
	var shares []model.DeviceShare
	return shares, s.mysqldb.Where("instance_uuid = ?", instanceUUID).Find(&shares).Error
}
func (s *DeviceShareService) GetAccessibleDevices(userUUID string) (*GetUserAllAccessibleDevicesResponse, error) {
	var ownedDevices []model.Instance
	s.mysqldb.Where("owner_uuid  = ?", userUUID).Find(&ownedDevices)

	var sharedDevices []model.Instance

	if err := s.mysqldb.
		Table("instances").
		Joins("JOIN device_shares ON device_shares.instance_uuid = instances.instance_uuid").
		Where("device_shares.shared_with_uuid = ?", userUUID).
		Where("device_shares.status = ?", StatusActive).
		Where("(device_shares.expires_at IS NULL OR device_shares.expires_at > ?)", time.Now().Unix()).
		Find(&sharedDevices).Error; err != nil {
		return nil, fmt.Errorf("failed to query shared devices for user %s: %w", userUUID, err)
	}

	allAccessibleDevices := append(ownedDevices, sharedDevices...)
	count := len(allAccessibleDevices)
	for i := range allAccessibleDevices {
		instanceUUID := allAccessibleDevices[i].InstanceUUID

		var shareCount int64
		err := s.mysqldb.Model(&model.DeviceShare{}).
			Where("instance_uuid = ? AND status = ?", instanceUUID, StatusActive).
			Where("(expires_at IS NULL OR expires_at > ?)", time.Now().Unix()).
			Count(&shareCount).Error

		if err != nil {
			log.Printf("Warning: failed to count shares for device %s: %v", instanceUUID, err)
			allAccessibleDevices[i].SharedCount = 0
		} else {
			allAccessibleDevices[i].SharedCount = int(shareCount)
		}
		allAccessibleDevices[i].IsShared = allAccessibleDevices[i].SharedCount > 0
	}
	response := &GetUserAllAccessibleDevicesResponse{InstanceCount: count, Instances: allAccessibleDevices}
	return response, nil
}

func (s *DeviceShareService) CheckDeviceAccess(instanceUUID string, userUUID string, requiredPermission string) (bool, error) {
	var instance model.Instance
	//Check if the owner of the instance
	if err := s.mysqldb.Where("instance_uuid = ? AND owner_uuid = ? AND status = ?", instanceUUID, userUUID, StatusActive).
		First(&instance).Error; err != nil {
		log.Println(err)
	} else {
		return true, nil
	}

	//Check the shared permission
	var share model.DeviceShare
	if err := s.mysqldb.Where("instance_uuid = ? AND shared_with_uuid = ?", instanceUUID, userUUID).
		Where("expires_at IS NULL OR expires_at > ?", time.Now().Unix()).
		First(&share).Error; err != nil {
		log.Println(err)
		return false, nil
	}

	if requiredPermission == PermissionRead && (share.Permission == PermissionRead || share.Permission == PermissionReadWrite) {
		return true, nil

	}
	if requiredPermission == PermissionWrite && (share.Permission == PermissionWrite || share.Permission == PermissionReadWrite) {
		return true, nil
	}
	return false, nil
}
func isValidStatus(status string) bool {
	switch status {
	case StatusActive:
		return true
	case StatusRevoked:
		return true
	default:
		return false

	}
}

func (s *DeviceShareService) calculateShareInfo(instanceUUID string) (int, bool, error) {
	var shareCount int64
	now := time.Now()
	err := s.mysqldb.Model(&model.DeviceShare{}).
		Where("instance_uuid = ? AND status = ?", instanceUUID, StatusActive).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Count(&shareCount).Error
	if err != nil {
		return 0, false, err
	}

	count := int(shareCount)
	isShared := count > 0
	return count, isShared, nil
}
