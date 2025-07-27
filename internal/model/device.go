package model

import (
	"OMEGA3-IOT/internal/utils"
	"fmt"
	"github.com/spf13/viper"
	"sync"
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

/*
	type DeviceType struct {
		ID   int    `mapstructure:"id" yaml:"id" json:"id"`
		Name string `mapsyructure:"name" yaml:"name" json:"name"`
		//DisplayName string                 `yaml:"display_name" json:"display_name"`
		Description string                  `mapstructure:"description" yaml:"description" json:"description"`
		Properties  map[string]PropertyMeta `mapstructure:"description" yaml:"properties" json:"properties"`
	}
*/
type DeviceType struct {
	ID          int                     `mapstructure:"id" yaml:"id"`
	Name        string                  `mapstructure:"name" yaml:"name"`
	Description string                  `mapstructure:"description" yaml:"description"`
	Properties  map[string]PropertyMeta `mapstructure:"properties" yaml:"properties"`
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
	mu    sync.RWMutex
}

var GlobalDeviceTypeManager = &DeviceTypeManager{
	types: make(map[string]*DeviceType),
	ids:   make(map[int]*DeviceType),
}

func (dtm *DeviceTypeManager) LoadDeviceTypeFromYAML(filePath string) error {
	//TODO 也许这里可以封装起来，让其可以load any?

	v := viper.New()
	v.SetConfigFile(filePath)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("could not load config: %v", err)
	}

	//var deviceTypes []*DeviceType不适用指针数组
	var deviceTypesConfig struct {
		DeviceTypes []DeviceType `mapstructure:"device_types" yaml:"device_types"`
	}
	if err := v.Unmarshal(&deviceTypesConfig); err != nil {
		return fmt.Errorf("could not unmarshal config: %v", err)
	}

	dtm.mu.Lock()
	defer dtm.mu.Unlock()

	dtm.types = make(map[string]*DeviceType)
	dtm.ids = make(map[int]*DeviceType)

	for i, dt := range deviceTypesConfig.DeviceTypes {
		deviceType := &dt
		fmt.Printf("Processing: %+v\n", deviceType)

		if deviceType.Name == "" {
			fmt.Printf("Warning: Device type %d has empty name\n", i)
			continue
			//debug msg...
		}
		if deviceType.ID <= 0 {
			fmt.Printf("Warning: Device type %s has invalid ID\n", deviceType.Name)
			continue
		}

		dtm.ids[deviceType.ID] = deviceType
		dtm.types[deviceType.Name] = deviceType
	}
	return nil
}

func (dtm *DeviceTypeManager) GetByName(name string) (*DeviceType, bool) {
	dt, exists := dtm.types[name]
	//fmt.Println(dtm.ids[0])
	return dt, exists
}

func (dtm *DeviceTypeManager) GetById(id int) (*DeviceType, bool) {
	dt, exists := dtm.ids[id]
	return dt, exists
}

// IsValidType 是验证设备类型
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
