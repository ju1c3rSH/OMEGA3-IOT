package factory

import (
	"OMEGA3-IOT/internal/model"
)

type DeviceFactory interface {
	CreateInstance(name, deviceType, description, ownerUUID string) (model.Instance, error)
	GetSupportedTypes() []string
}

type deviceFactory struct {
	propertyMetaMap map[string]model.PropertyMeta
}

func (d deviceFactory) GetSupportedTypes() []string {
	//TODO implement me
	panic("implement me")
}
