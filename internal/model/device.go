package model

import (
	"OMEGA3-IOT/internal/utils"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"time"
)

type Instance struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	InstanceUUID string     `gorm:"uniqueIndex;type:varchar(36)" json:"instance_uuid"`
	Name         string     `gorm:"type:varchar(100);not null" json:"name"`
	Type         string     `gorm:"type:varchar(50);not null;index" json:"type"`
	Online       bool       `gorm:"default:false" json:"online"`
	OwnerUUID    string     `gorm:"type:varchar(36);not null;index" json:"owner_uuid"`
	Description  string     `gorm:"type:text" json:"description,omitempty"`
	AddTime      int64      `gorm:"not null" json:"add_time"`
	LastSeen     int64      `gorm:"not null" json:"last_seen"`
	Properties   Properties `gorm:"type:json" json:"properties"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
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
	fmt.Println(dtm.ids[0])
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

//Let me think , what should i do in next step?shall i put this factory_function to web api directly?maybe not.Because i havent deal the problems in database to save props.After it, i should make a history system to record the changes of the devices,but i dont know how to design a LINEAR Save System....
//So,deal with the Properties to save in mysql(in Json)First.
