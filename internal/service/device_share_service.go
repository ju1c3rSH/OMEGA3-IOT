package service

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"fmt"
	"gorm.io/gorm"
	"log"
	"time"
)

type DeviceShareService struct {
	instanceRepo     repository.InstanceRepository
	deviceShareRepo  repository.DeviceShareRepository
}

func NewDeviceShareService(db *gorm.DB) *DeviceShareService {
	return &DeviceShareService{
		instanceRepo:    repository.NewInstanceRepository(db),
		deviceShareRepo: repository.NewDeviceShareRepository(db),
	}
}

func (s *DeviceShareService) ShareDevice(instanceUUID string, shareByUUID string, shareWithUUID string, expiredTime int64, permission string) error {
	// Verify the user owns the device
	_, err := s.instanceRepo.FindByUUID(instanceUUID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	// Check ownership by querying with owner UUID
	devices, err := s.instanceRepo.FindByOwnerUUID(shareByUUID)
	if err != nil {
		return err
	}

	found := false
	for _, d := range devices {
		if d.InstanceUUID == instanceUUID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user does not own this device")
	}

	share := &model.DeviceShare{
		InstanceUUID:   instanceUUID,
		SharedByUUID:   shareByUUID,
		SharedWithUUID: shareWithUUID,
		ExpiresAt:      &expiredTime,
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
		Permission:     permission,
		Status:         repository.StatusActive,
	}
	return s.deviceShareRepo.Create(share)
}

func (s *DeviceShareService) UnshareDevice(instanceUUID string, shareWithUUID string) error {
	return s.deviceShareRepo.DeleteByInstanceAndSharedWith(instanceUUID, shareWithUUID)
}

func (s *DeviceShareService) GetSharedDevices(instanceUUID string) ([]model.DeviceShare, error) {
	return s.deviceShareRepo.FindByInstanceUUID(instanceUUID)
}

func (s *DeviceShareService) GetSharedBy(sharedWithUUID string) ([]model.DeviceShare, error) {
	return s.deviceShareRepo.FindBySharedWithUUID(sharedWithUUID)
}

func (s *DeviceShareService) GetSharedWith(instanceUUID string) ([]model.DeviceShare, error) {
	return s.deviceShareRepo.FindByInstanceUUID(instanceUUID)
}

func (s *DeviceShareService) GetAccessibleDevices(userUUID string) (*GetUserAllAccessibleDevicesResponse, error) {
	// Get owned devices
	ownedDevices, err := s.instanceRepo.FindByOwnerUUID(userUUID)
	if err != nil {
		return nil, err
	}

	// Get shared devices
	shares, err := s.deviceShareRepo.FindBySharedWithUUID(userUUID)
	if err != nil {
		return nil, err
	}

	var sharedDevices []model.Instance
	now := time.Now().Unix()
	for _, share := range shares {
		// Check if share is active and not expired
		if share.Status != repository.StatusActive {
			continue
		}
		if share.ExpiresAt != nil && *share.ExpiresAt <= now {
			continue
		}

		instance, err := s.instanceRepo.FindByUUID(share.InstanceUUID)
		if err != nil {
			log.Printf("Warning: failed to find instance %s: %v", share.InstanceUUID, err)
			continue
		}
		sharedDevices = append(sharedDevices, *instance)
	}

	allAccessibleDevices := append(ownedDevices, sharedDevices...)
	count := len(allAccessibleDevices)

	for i := range allAccessibleDevices {
		instanceUUID := allAccessibleDevices[i].InstanceUUID
		shareCount, err := s.deviceShareRepo.CountActiveShares(instanceUUID)
		if err != nil {
			log.Printf("Warning: failed to count shares for device %s: %v", instanceUUID, err)
			shareCount = 0
		}
		allAccessibleDevices[i].SharedCount = int(shareCount)
		allAccessibleDevices[i].IsShared = shareCount > 0
	}

	response := &GetUserAllAccessibleDevicesResponse{InstanceCount: count, Instances: allAccessibleDevices}
	return response, nil
}

func (s *DeviceShareService) CheckDeviceAccess(instanceUUID string, userUUID string, requiredPermission string) (bool, error) {
	// Check if the user owns the instance
	devices, err := s.instanceRepo.FindByOwnerUUID(userUUID)
	if err != nil {
		return false, err
	}

	for _, d := range devices {
		if d.InstanceUUID == instanceUUID {
			return true, nil
		}
	}

	// Check the shared permission
	share, err := s.deviceShareRepo.FindByInstanceAndSharedWith(instanceUUID, userUUID)
	if err != nil {
		return false, nil
	}

	// Check if expired
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

func (s *DeviceShareService) calculateShareInfo(instanceUUID string) (int, bool, error) {
	count, err := s.deviceShareRepo.CountActiveShares(instanceUUID)
	return int(count), count > 0, err
}
