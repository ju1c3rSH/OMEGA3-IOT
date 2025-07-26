package model

import (
	"OMEGA3-IOT/internal/utils"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"time"
)

type Instance struct {
	ID           int        `gorm:"primaryKey;autoIncrement" json:"id" `
	InstanceUUID string     `json:"instance_uuid"`
	Name         string     `json:"name" `
	Type         string     `json:"type" example:"door_sensor"`
	Online       bool       `json:"online" example:"true"`
	OwnerUUID    string     `json:"owner_uuid" example:""`
	Description  string     `json:"description,omitempty" example:"前门防盗设备"`
	AddTime      int64      `json:"add_time" example:"1751193600"`
	LastSeen     int64      `json:"last_seen" example:"1751193600"`
	Activated    bool       `json:"activated" gorm:"default:false"`
	ActivateTime int64      `json:"activate_time,omitempty"`
	Properties   Properties `json:"properties" gorm:"type:json"`
	//privatekey?
}

type DeviceTemplate struct {
	Type        string                  `json:"type" gorm:"primaryKey"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Properties  map[string]PropertyMeta `json:"properties" gorm:"type:json"`
	Actions     []ActionMeta            `json:"actions" gorm:"type:json"`
}
type DeviceType struct {
	ID   int    `yaml:"id" json:"id"`
	Name string `yaml:"name" json:"name"`
	//DisplayName string                 `yaml:"display_name" json:"display_name"`
	Description string                  `yaml:"description" json:"description"`
	Properties  map[string]PropertyMeta `yaml:"properties" json:"properties"`
}
type DeviceAddTemplate struct {
	Name        string `json:"name" gorm:"primaryKey"`
	Description string `json:"description"`
	Type        int    `json:"type" gorm:"primaryKey"`
	SN          string `json:"sn" gorm:"primaryKey"`
}

type ActionMeta struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}
type DeviceTypeManager struct {
	types map[string]*DeviceType
	ids   map[int]*DeviceType
}

var GlobalDeviceTypeManager = &DeviceTypeManager{
	types: make(map[string]*DeviceType),
	ids:   make(map[int]*DeviceType),
}

func (dtm *DeviceTypeManager) LoadFromYAML(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	var config struct {
		DeviceTypes []*DeviceType `yaml:"device_types"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	dtm.types = make(map[string]*DeviceType)
	dtm.ids = make(map[int]*DeviceType)

	for _, dt := range config.DeviceTypes {
		dtm.types[dt.Name] = dt
		dtm.ids[dt.ID] = dt
	}

	return nil
}

// 获取设备的类型
func (dtm *DeviceTypeManager) GetByName(name string) (*DeviceType, bool) {
	dt, exists := dtm.types[name]
	return dt, exists
}

// 验证设备类型
func (dtm *DeviceTypeManager) IsValidType(name string) bool {
	_, exists := dtm.types[name]
	return exists
}

func NewInstanceFromConfig(name string, ownerUuid string, deviceType *DeviceType) (*Instance, error) {
	props := Properties{Items: make(map[string]*PropertyItem)}
	for propName, meta := range deviceType.Properties {
		props.Items[propName] = &PropertyItem{
			//Value: getDefaultValue(meta.Type),
			Meta: meta,
		}

	}
	return &Instance{
		InstanceUUID: utils.GenerateUUID().String(),
		Name:         name,
		Type:         deviceType.Name,
		Online:       false,
		OwnerUUID:    ownerUuid,
		Description:  deviceType.Description,
		AddTime:      time.Now().Unix(),
		LastSeen:     time.Now().Unix(),
		Properties:   props,
	}, nil

}
