package model

import (
	"encoding/json"
	"fmt"
)

type Instance struct {
	ID           int        `gorm:"primaryKey;autoIncrement" json:"id" `
	InstanceUUID string     `json:"instance_uuid"`
	Name         string     `json:"name" `
	Type         string     `json:"type" example:"door_sensor"`
	Online       bool       `json:"online" example:"true"`
	OwnerUUID    string     `json:"owner_uuid" example:""`
	Description  string     `json:"description,omitempty" example:"前门防盗设备"`
	AddTime      int        `json:"add_time" example:"1751193600"`
	LastSeen     int        `json:"last_seen" example:"1751193600"`
	Activated    bool       `json:"activated" gorm:"default:false"`
	ActivateTime int64      `json:"activate_time,omitempty"`
	Properties   Properties `json:"properties" gorm:"type:json"`
	//privatekey?
}
type Properties struct {
	Items map[string]*Property `json:"items"`
}

type PropertyItem struct {
	Value string       `json:"value"`
	Meta  PropertyMeta `json:"meta"`
}

type PropertyMeta struct {
	Writable    bool
	Description string
	Unit        string
	Range       []int
	Format      string
	Enum        []string
	//TODO Required Type ?
}
type DeviceTemplate struct {
	Type        string                  `json:"type" gorm:"primaryKey"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Properties  map[string]PropertyMeta `json:"properties" gorm:"type:json"`
	Actions     []ActionMeta            `json:"actions" gorm:"type:json"`
}

type ActionMeta struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}
type Property interface {
	GetValue(name string) (interface{}, error)
	SetValue(name string, value interface{}) error
	GetAll() map[string]interface{}
	GetMeta() map[string]PropertyMeta
}

// TODO: 动态 Meta 管理模块
// - [ ] 支持从数据库加载 Meta 数据 [[4]]
// - [ ] 支持通过 API 更新 Meta 数据
// - [ ] 支持热更新（不重启服务）[[1]]
// - [ ] 支持多设备类型统一管理 [[7]]

var PropertyMetaMap = map[string]map[string]PropertyMeta{
	"smart_lamp": {
		"switch": {
			Writable:    true,
			Description: "开关状态",
		},
		"brightness": {
			Writable:    true,
			Description: "亮度百分比",
			Unit:        "%",
			Range:       []int{0, 100},
		},
		"color": {
			Writable:    true,
			Description: "颜色值",
			Format:      "hex",
		},
	},
}

type TestDevice struct {
	Type         string `json:"type" meta:"writeable:true,description: A type of test"`
	BatteryLevel int    `json:"battery_level" meta:"writeable:true,description: Battery level of test"`
}

func NewProperty(data []byte) (Property, error) {
	typeHint := struct {
		Type string `json:"type"`
	}{}

	if err := json.Unmarshal(data, &typeHint); err != nil {
		return nil, err
	}

	switch typeHint.Type {
	case "test_device":
		var p TestDevice
		return &p, json.Unmarshal(data, &p)

	default:
		return nil, fmt.Errorf("unknown property type: %s", typeHint.Type)
	}
}
