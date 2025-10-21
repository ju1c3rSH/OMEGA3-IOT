package service

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/utils"
	"context"
	"fmt"
	"gorm.io/gorm"
	"log"
	"time"
)

type DeviceService struct {
	mysqlDB     *gorm.DB
	iotdbClient *db.IOTDBClient
}

func NewDeviceService(mysqlDB *gorm.DB, iotDBClient *db.IOTDBClient) *DeviceService {
	return &DeviceService{
		mysqlDB:     mysqlDB,
		iotdbClient: iotDBClient,
	}
}
func (s *DeviceService) updateDeviceProperties(instance model.Instance, data map[string]model.PropertyItem) error {

	if instance.Properties.Items == nil {
		instance.Properties.Items = make(map[string]*model.PropertyItem)
	}
	for key, value := range data {
		//1创建一个新的 PropertyItem 指针，并复制值
		//不担心复用，可以直接取地址 &value，但要注意循环变量作用域问题
		//更安全的方式是创建副本：
		valueCopy := value
		va := valueCopy.Value
		instance.Properties.Items[key].Value = va
	}
	instance.LastSeen = time.Now().Unix()
	instance.UpdatedAt = time.Now()
	instance.Online = true
	dbSession := s.mysqlDB.Session(&gorm.Session{})
	if err := dbSession.Save(instance).Error; err != nil {
		return fmt.Errorf("failed to save updated instance %s to database: %w", instance.InstanceUUID, err)
	}

	log.Printf("Database record for device %s updated with new properties.", instance.InstanceUUID)
	return nil
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

	if err := s.mysqlDB.WithContext(ctx).Create(record).Error; err != nil {
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
	//验证设备类型
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

	if err := s.mysqlDB.WithContext(ctx).Create(instance).Error; err != nil {
		// 判断是否为唯一键冲突（如设备名重复）
		if err != nil && len(err.Error()) > 0 && (err.Error() == "UNIQUE constraint failed" || err.Error() == "duplicate key value violates unique constraint") {
			return nil, gorm.ErrDuplicatedKey
		}
		return nil, err
	}

	return instance, nil
}
