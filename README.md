# OMEGA3-IOT 开发文档

## 项目概述

OMEGA3-IOT 是一个基于 Go 的物联网设备管理平台，采用 HTTP REST API + MQTT 双协议架构。 [1](#1-0) 

## 核心架构

- **设备注册流程**: 匿名注册 → RegCode绑定 → 正式实例 [2](#1-1) 
- **认证系统**: JWT (用户) + VerifyCode (设备) [3](#1-2) 
- **数据模型**: Instance (设备实例) + DeviceRegistrationRecord (临时注册) [4](#1-3) 

## 快速启动

```bash
# 配置环境变量
export JWT_SECRET=your_secret_key
export OMEGA3_IOT=omega3_iot

# 启动服务
go run main.go
```

服务端口:
- HTTP API: `:27015`
- MQTT Broker: `tcp://yuyuko.food:1883`

## TODO Checklist

### 🔧 代码优化
- [ ] **设备类型加载封装** - `LoadDeviceTypeFromYAML` 需要重构为通用加载器 [5](#1-4) 
- [ ] **实例创建验证** - `NewInstanceFromConfig` 需要加上验证Hash [6](#1-5) 
- [ ] **设备注册防刷** - `RegisterDeviceAnonymously` 需要添加频率限制 [7](#1-6) 
- [ ] **手动添加设备重构** - `AddDevice` 方法需要重构 [8](#1-7) 
- [ ] **MQTT解耦** - `PublishActionToDevice` 需要解耦处理 [9](#1-8) 
- [ ] **MQTT重试机制** - 添加Retry Pool处理发送失败 [10](#1-9) 
- [ ] **设备工厂实现** - `GetSupportedTypes` 方法待实现 [11](#1-10) 
- [ ] **配置地址修复** - `Broker.Address()` 方法需要修复 [12](#1-11) 
- [ ] **VerifyCode加盐** - `GenerateVerifyCode` 需要添加salt [13](#1-12) 

### 📋 功能增强
- [ ] **更好的Log保存系统** - 实现结构化日志和持久化存储
- [ ] **权限账号管理机制** - 实现Group、Team多级权限管理
- [ ] **属性类型验证** - PropertyMeta需要Required Type字段 [14](#1-13) 

## 关键文件

| 文件 | 用途 |
|------|------|
| `internal/model/device.go` | 设备数据模型和类型管理器 |
| `internal/service/device_service.go` | 设备注册和管理业务逻辑 |
| `internal/service/mqtt_service.go` | MQTT通信处理 |
| `internal/service/user_service.go` | 用户认证和设备绑定 |
| `DesignStandard.md` | 项目设计规范 |

## 开发规范

- JSON字段使用下划线命名法 [15](#1-14) 
- 设备类型通过YAML配置驱动 [16](#1-15) 
- 所有数据库操作使用GORM，设置10秒超时 [17](#1-16) 

## Notes

项目目前处于开发阶段，核心功能已实现但需要优化和扩展。重点关注设备注册流程、MQTT通信和权限管理系统的完善。

Wiki pages you might want to explore:
- [Device Lifecycle & Registration (ju1c3rSH/OMEGA3-IOT)](/wiki/ju1c3rSH/OMEGA3-IOT#5.1)
- [MQTT Communication System (ju1c3rSH/OMEGA3-IOT)](/wiki/ju1c3rSH/OMEGA3-IOT#5.2)
- [Authentication & Security (ju1c3rSH/OMEGA3-IOT)](/wiki/ju1c3rSH/OMEGA3-IOT#5.5)

### Citations

**File:** DesignStandard.md (L4-8)
```markdown
## 1. JSON 字段命名规范
- 所有 JSON 字段名使用小写字母
- 不同单词间使用下划线 `_` 分割
- 保持一致性，避免混用驼峰命名

```

**File:** DesignStandard.md (L35-36)
```markdown
//TODO Required Type ?
```
```

**File:** DesignStandard.md (L38-47)
```markdown
## 4. 设备录入系统流程
```text
设备注册遵循：设备开机通过网络/Lora的方式向主服务器进行报备注册。
Lora需要通过网关。
用户通过后续设备上显示的RegCode向服务器发送请求，将设备绑定至用户。

流程：匿名注册——> 临时UUID和RegCode与VerifyCode，开辟当前UUID的Topic——>RegCode被使用，Topic下发指令，停止重置计时，并在服务端处转为正式UUID，存入数据库
在存入instance表之后，服务端在broker发布一条广播：
/data/device/{Device_UUID}/action 其含有GO_ON的信息，设备会在此前订阅这里，接收到之后开始工作。
```
```

**File:** internal/service/device_service.go (L19-43)
```go
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
```

**File:** internal/service/device_service.go (L47-48)
```go
	//TODO这个不能直接用！！！！
	deviceType, valid := model.GlobalDeviceTypeManager.GetById(deviceTypeID)
```

**File:** internal/service/mqtt_service.go (L76-77)
```go
	//TODO 解耦
	if err != nil {
```

**File:** internal/service/mqtt_service.go (L95-130)
```go
func (m *MQTTService) handlePropertiesData(c mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()
	log.Printf("Received property data from MQTT topic [%s] (QOS %d): %s", topic, msg.Qos(), string(payload))
	deviceUUID, _ := extractDeviceUUIDFromTopic(topic)
	var message DeviceMessage

	if err := json.Unmarshal(payload, &message); err != nil {
		fmt.Errorf("error unmarshalling device message: %v", err)
	}

	hashedVerifyCode := utils.HashVerifyCode(message.VerifyCode)
	rawPropsData := message.Data.Properties
	fmt.Printf("Properties Object: %+v\n", rawPropsData)

	var instance model.Instance
	dbSession := m.db.Session(&gorm.Session{})
	if err := dbSession.Where("instance_uuid = ? AND verify_hash = ?", deviceUUID, hashedVerifyCode).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Unauthorized access attempt: No device found with UUID %s and provided verify code (hash: %s)", deviceUUID, hashedVerifyCode)
		} else {
			log.Printf("Database error during authentication for device %s: %v", deviceUUID, err)
		}
		return
	}

	// 确保 instance.Properties.Items 已初始化
	if instance.Properties.Items == nil {
		instance.Properties.Items = make(map[string]*model.PropertyItem)
	}

	if err := m.updateDeviceProperties(instance, rawPropsData); err != nil {

	}

}
```

**File:** internal/model/device.go (L11-72)
```go
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
	SN     string `gorm:"type:varchar(100);null" json:"sn,omitempty"`
	Remark string `gorm:"type:text" json:"remark,omitempty"`
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
```

**File:** internal/model/device.go (L111-114)
```go
var GlobalDeviceTypeManager = &DeviceTypeManager{
	types: make(map[string]*DeviceType),
	ids:   make(map[int]*DeviceType),
}
```

**File:** internal/model/device.go (L117-118)
```go
	//TODO 也许这里可以封装起来，让其可以load any?

```

**File:** internal/model/device.go (L187-188)
```go
	//TODO这里还不能用，要加上验证Hash
	props := Properties{Items: make(map[string]*PropertyItem)}
```

**File:** internal/service/user_service.go (L138-139)
```go
		//TODO 这里可以加上一个 Retry Pool..
	}
```

**File:** internal/handler/factory/DeviceFactory.go (L38-39)
```go
	//TODO implement me
	panic("implement me")
```

**File:** internal/config/config.go (L30-31)
```go
	//TODO修好！
	return fmt.Sprintf("%s://%s:%d", b.Protocol, b.Host, b.Port)
```

**File:** internal/utils/GeneralUtils.go (L44-45)
```go
	//TODO 加salt
	return string(b), nil
```
