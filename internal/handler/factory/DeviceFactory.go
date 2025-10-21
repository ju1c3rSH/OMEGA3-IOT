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
		AddTime:      int64(time.Now().Unix()),
		LastSeen:     int64(time.Now().Unix()),
		Properties:   model.Properties{Items: make(map[string]model.PropertyItem)},
	}

	//d.initializeAllProperties(instance)

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
