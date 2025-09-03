package service

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/utils"
	"context"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type DeviceService struct{}

func NewDeviceService() *DeviceService {
	return &DeviceService{}
}

func (s *DeviceService) RegisterDeviceAnonymously(deviceTypeID int, verifyCode string) (*model.DeviceRegistrationRecord, error) {
	_, valid := model.GlobalDeviceTypeManager.GetById(deviceTypeID)
	if !valid {
		return nil, gorm.ErrInvalidData
	}
	//NewRegistrationRecord
	hashedVerifyCode := utils.HashVerifyCode(verifyCode)
	record, err := model.NewRegistrationRecord(deviceTypeID, hashedVerifyCode)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.DB.WithContext(ctx).Create(record).Error; err != nil {
		// 判断是否为唯一键冲突（如设备名重复）
		if err != nil && len(err.Error()) > 0 && (err.Error() == "UNIQUE constraint failed" || err.Error() == "duplicate key value violates unique constraint") {
			return nil, gorm.ErrDuplicatedKey
		}
		return nil, err
	}
	//TODO加上防刷
	return record, nil
}

func (s *DeviceService) AddDevice(name string, deviceTypeID int, remark string, ownerUUID string) (*model.Instance, error) {
	// 验证设备类型
	//TODO这个不能直接用！！！！
	deviceType, valid := model.GlobalDeviceTypeManager.GetById(deviceTypeID)
	if !valid {
		return nil, gorm.ErrInvalidData
	}

	instance, err := model.NewInstanceFromConfig(name, ownerUUID, deviceType, "", remark, utils.GenerateUUID().String())
	if err != nil {
		return nil, fmt.Errorf("failed to create device instance: %w", err)
	}
	// instance.Remark = remark

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.DB.WithContext(ctx).Create(instance).Error; err != nil {
		// 判断是否为唯一键冲突（如设备名重复）
		if err != nil && len(err.Error()) > 0 && (err.Error() == "UNIQUE constraint failed" || err.Error() == "duplicate key value violates unique constraint") {
			return nil, gorm.ErrDuplicatedKey
		}
		return nil, err
	}

	return instance, nil
}
