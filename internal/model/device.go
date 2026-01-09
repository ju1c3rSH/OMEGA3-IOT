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
	VerifyHash   string     `gorm:"type:varchar(255)" json:"verify_hash"`
	//IsActivated  bool       `gorm:"default:false" json:"is_activated"`不需要，因为有DeviceRegistrationRecord的机制，出现在这个库里的肯定是激活绑定了的
	SN          string `gorm:"type:varchar(100);null" json:"sn,omitempty"`
	Status      string `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	IsShared    bool   `gorm:"default:false" json:"is_shared"`
	SharedCount int    `gorm:"default:0" json:"shared_count"`
	Remark      string `gorm:"type:text" json:"remark,omitempty"`
}

type DeviceTemplate struct {
	Type        string                  `json:"type" gorm:"primaryKey"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Properties  map[string]PropertyMeta `json:"properties" gorm:"type:json"`
	Actions     []ActionMeta            `json:"actions" gorm:"type:json"`
}

type DeviceRegistrationRecord struct {
	// ID 是记录的主键
	ID uint `gorm:"primaryKey" json:"id"`

	// DeviceUUID 是分配给设备的唯一标识符，用于后续通信和绑定。
	// 数据库层面有唯一索引 idx_device_uuid 保障其唯一性。
	DeviceUUID string `gorm:"type:varchar(36);uniqueIndex:idx_device_uuid" json:"device_uuid"`

	// RegCode 是提供给用户用于绑定设备的 8 位随机码。
	// 数据库层面有唯一索引 idx_reg_code 保障其唯一性。
	RegCode string `gorm:"type:varchar(8);uniqueIndex:idx_reg_code" json:"reg_code"`

	// DeviceTypeID 关联到 GlobalDeviceTypeManager 中的设备类型 ID。
	DeviceTypeID int `gorm:"type:int" json:"device_type_id"`

	// SN (Serial Number) 是设备的序列号（如果有的话）。
	// 可以为 NULL。如果需要唯一性，应在数据库层面通过允许 NULL 的唯一索引来实现。
	SN string `gorm:"type:varchar(100);null" json:"sn,omitempty"`

	// VerifyHash 是用于设备数据上传鉴权的哈希值（基于 VerifyCode 生成）。
	VerifyHash string `gorm:"type:varchar(255)" json:"verify_hash"`

	// CreatedAt 记录创建时的 Unix 时间戳。
	CreatedAt int64 `gorm:"not null" json:"created_at"` // 或使用 time.Time 配合 gorm:"autoCreateTime"

	// ExpiresAt 记录此注册码过期时的 Unix 时间戳。
	// 添加索引 idx_expires_at 以优化过期记录清理或查询。
	ExpiresAt int64 `gorm:"index:idx_expires_at" json:"expires_at"`

	// IsBound 标记此注册记录是否已被成功用于绑定设备。
	// 添加索引 idx_is_bound 以优化查询未绑定的记录。
	// default:false 确保新记录默认为未绑定。
	IsBound bool `gorm:"default:false;index:idx_is_bound" json:"is_bound"`

	//以上信息由QWEN3--CODER生成 （这玩意还挺好用）
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

/*
↓无用
*/
type DeviceAddTemplate struct {
	Name        string `json:"name" gorm:"primaryKey"`
	Description string `json:"description"`
	Type        int    `json:"type" gorm:"primaryKey"`
	SN          string `json:"sn" gorm:"primaryKey"`
}

type DeviceShare struct {
	ID             uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	InstanceUUID   string `gorm:"type:varchar(36);not null;index" json:"instance_uuid"`
	SharedWithUUID string `gorm:"type:varchar(36);not null;index" json:"shared_with_uuid"`
	SharedByUUID   string `gorm:"type:varchar(36);not null;index" json:"shared_by_uuid"`
	Permission     string `gorm:"type:varchar(20);not null;default:'read'" json:"permission" validate:"oneof=read write read_write"`
	// 移除 IsValid 字段
	Status    string `gorm:"type:varchar(20);not null;default:'active'" json:"status"` // active, revoked
	CreatedAt int64  `json:"created_at"`                                               // 移除 autoCreateTime 标签
	UpdatedAt int64  `json:"updated_at"`                                               // 移除 autoUpdateTime 标签
	ExpiresAt *int64 `gorm:"index" json:"expires_at,omitempty"`                        // 使用 *int64，nil 表示永不过期
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
func NewRegistrationRecord(deviceTypeID int, hashedVerifyCode string) (*DeviceRegistrationRecord, error) {
	return &DeviceRegistrationRecord{
		DeviceUUID:   utils.GenerateUUID().String(),
		DeviceTypeID: deviceTypeID,
		RegCode:      utils.GenerateRegCode(),
		ExpiresAt:    time.Now().Add(time.Hour * 24).Unix(),
		CreatedAt:    time.Now().Unix(),
		VerifyHash:   hashedVerifyCode,
	}, nil
}
func NewInstanceFromConfig(name string, ownerUuid string, deviceType *DeviceType, verifyHash string, remark string, deviceUUID string) (*Instance, error) {
	//TODO这里还不能用，要加上验证Hash
	props := Properties{Items: make(map[string]*PropertyItem)}
	for propName, meta := range deviceType.Properties {
		props.Items[propName] = &PropertyItem{
			//Value: getDefaultValue(meta.Type),
			Meta: meta,
		}

	}
	return &Instance{
		InstanceUUID: deviceUUID,
		Name:         name,
		Remark:       remark,
		Type:         deviceType.Name,
		Online:       false,
		OwnerUUID:    ownerUuid,
		Description:  deviceType.Description,
		AddTime:      time.Now().Unix(),
		LastSeen:     time.Now().Unix(),
		Properties:   props,
		VerifyHash:   verifyHash,
	}, nil

}

//Let me think , what should i do in next step?shall i put this factory_function to web api directly?maybe not.Because i havent deal the problems in database to save props.After it, i should make a history system to record the changes of the devices,but i dont know how to design a LINEAR Save System....
//So,deal with the Properties to save in mysql(in Json)First.
