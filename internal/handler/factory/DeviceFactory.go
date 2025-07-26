package factory

import (
	"OMEGA3-IOT/internal/model"
	"github.com/google/uuid"
	"time"
)

type DeviceFactory interface {
	CreateInstance(name, deviceType, description, ownerUUID string) (model.Instance, error)
	GetSupportedTypes() []string
}

type deviceFactory struct {
	propertyMetaMap map[string]model.PropertyMeta
}

func (d deviceFactory) CreateInstance(name, deviceType, description, ownerUUID string) (model.Instance, error) {
	instance := &model.Instance{
		InstanceUUID: uuid.New().String(),
		Name:         name,
		Type:         deviceType,
		Description:  description,
		OwnerUUID:    ownerUUID,
		AddTime:      int(time.Now().Unix()),
		LastSeen:     int(time.Now().Unix()),
		Activated:    false,
		Properties:   model.InstanceProperties{Items: make(map[string]model.PropertyItem)},
	}

	// 初始化所有支持的属性
	d.initializeAllProperties(instance)

	return instance, nil
}

func (d deviceFactory) GetSupportedTypes() []string {
	//TODO implement me
	panic("implement me")
}

// NewDeviceFactory 创建设备工厂
func NewDeviceFactory() DeviceFactory {
	return &deviceFactory{
		propertyMetaMap: model.PropertyMetaMap,
	}
}
