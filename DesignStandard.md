# OMEGA3-IOT 设计规范

## 1. 核心原则

- **配置驱动**: 设备类型、系统行为通过 YAML 配置
- **依赖注入**: 服务间通过构造函数注入，避免全局状态
- **接口隔离**: Repository/Service 面向接口编程
- **错误透明**: 错误逐层包裹，保留完整上下文

## 2. 项目结构

```
OMEGA3-IOT/
├── cmd/                    # 程序入口
│   └── http-api/           # HTTP API 服务
├── internal/
│   ├── config/             # 配置加载
│   ├── db/                 # 数据库连接
│   ├── eventbus/           # 内部事件总线
│   ├── handler/            # HTTP 处理器
│   │   └── middlewares/    # Gin 中间件
│   ├── logger/             # 日志服务
│   ├── model/              # 数据模型
│   ├── repository/         # 数据访问层
│   ├── service/            # 业务逻辑层
│   ├── types/              # 共享类型定义
│   └── utils/              # 工具函数
├── internal/config/        # 配置文件目录
│   ├── device_type_list.yaml
│   └── GeneralConfig.yaml
└── main.go
```

## 3. 数据模型

### 3.1 Instance (设备实例)

| 字段 | 类型 | 说明 |
|------|------|------|
| InstanceUUID | string | 设备唯一标识 (UUID v4) |
| Name | string | 设备名称 |
| Type | string | 设备类型名称 |
| Online | bool | 在线状态 |
| OwnerUUID | string | 所有者 UUID |
| Properties | Properties | 当前属性值 (JSON) |
| VerifyHash | string | 设备认证哈希 |
| Status | string | active/inactive |

### 3.2 DeviceRegistrationRecord (注册记录)

| 字段 | 类型 | 说明 |
|------|------|------|
| DeviceUUID | string | 预分配设备 UUID |
| RegCode | string | 8位注册码 (用户绑定用) |
| VerifyHash | string | VerifyCode 的 SHA-256 哈希 |
| ExpiresAt | int64 | 过期时间 (24小时) |
| IsBound | bool | 是否已绑定 |

### 3.3 DeviceShare (设备分享)

| 字段 | 类型 | 说明 |
|------|------|------|
| InstanceUUID | string | 被分享设备 |
| SharedWithUUID | string | 被分享用户 |
| Permission | string | read/write/read_write |
| ExpiresAt | *int64 | 过期时间 (nil=永不过期) |

### 3.4 Properties (属性系统)

```go
type Properties struct {
    Items map[string]*Deprecated_PropertyItem `json:"items"`
}

type Deprecated_PropertyItem struct {
    Value string       `json:"value"`
    Meta  PropertyMeta `json:"meta"`
}

type PropertyMeta struct {
    Writable    bool     `json:"writable"`    // 是否可写
    Description string   `json:"description"` // 描述
    Unit        string   `json:"unit"`        // 单位
    Range       []int    `json:"range"`       // 取值范围
    Format      string   `json:"format"`      // string/int/float/bool
    Enum        []string `json:"enum"`        // 枚举值
}
```

## 4. 通信协议

### 4.1 MQTT 主题规范

```
data/device/{instance_uuid}/
├── properties    # 设备 → 服务器：属性上报 (QoS 1)
├── event         # 设备 → 服务器：事件通知 (QoS 1)
└── action        # 服务器 → 设备：指令下发 (QoS 1)
```

### 4.2 属性上报格式

```json
{
  "verify_code": "tOFX*mc8=V}?Cnh2",
  "timestamp": 1756882749,
  "data": {
    "properties": {
      "battery_level": {
        "value": "85",
        "meta": { ... }
      }
    }
  }
}
```

### 4.3 指令下发格式

```json
{
  "command": "set_upload_interval",
  "params": { "interval_sec": 30 },
  "timestamp": 1729450800
}
```

## 5. 设备注册流程

```
阶段1: 匿名注册 (设备端)
  POST /api/v1/device/deviceRegisterAnon
  ← 返回 {uuid, reg_code, verify_code}

阶段2: 用户绑定 (App端)
  POST /api/v1/users/bindDeviceByRegCode
  Body: {reg_code, device_nick}
  ← 设备激活，MQTT 下发 GO_ON 指令
```

## 6. 开发规范

### 6.1 命名规则

| 场景 | 规范 | 示例 |
|------|------|------|
| JSON 字段 | 小写 + 下划线 | `instance_uuid` |
| Go 结构体 | 大驼峰 | `InstanceUUID` |
| 数据库列 | 小写 + 下划线 | `instance_uuid` |
| 常量 | 大驼峰 | `DefaultTimeout` |
| 接口 | 动词 + er | `UserRepository` |

### 6.2 错误处理

```go
// 错误包装，保留上下文
if err := db.Create(&instance).Error; err != nil {
    return fmt.Errorf("create instance failed: %w", err)
}

// HTTP 层统一响应
c.JSON(http.StatusInternalServerError, response.Error(
    http.StatusInternalServerError,
    "Failed to create device",
))
```

### 6.3 服务初始化

```go
// 依赖注入模式
func NewUserService(
    mqtt *service.MQTTService,
    userRepo repository.UserRepository,
    // ...
) *UserService {
    return &UserService{
        mqtt: mqtt,
        userRepo: userRepo,
    }
}
```

## 7. 数据库设计

### 7.1 MySQL 表

```sql
-- 核心表：users, instances, device_registration_records, device_shares
-- 详见 migrations/ 目录 (TODO)
```

### 7.2 IoTDB 结构

```
root.device_data.{instance_uuid}.{property_name}
root.device_latest.{instance_uuid}.{property_name}
```

## 8. 配置文件

### 8.1 device_type_list.yaml

```yaml
device_types:
  - id: 1
    name: "BaseTracker"
    description: "定位器"
    properties:
      battery_level:
        writable: true
        description: "电量"
        unit: "%"
        range: [0, 100]
        format: "int"
      gps_location:
        writable: false
        description: "GPS位置"
        format: "string"
```

### 8.2 GeneralConfig.yaml

```yaml
database:
  mysqldsn: "user:pass@tcp(host:port)/dbname"

server:
  port: 27015

IoTDB:
  host: "localhost"
  port: 6667
  username: root
  password: root
```

## 9. 事件系统

```go
// 发布事件
eventBus.Publish(eventbus.EventDeviceOnline, eventbus.DeviceEvent{
    InstanceUUID: uuid,
    Timestamp:    time.Now().Unix(),
})

// 订阅事件
eventBus.Subscribe(eventbus.EventDeviceOnline, func(data interface{}) {
    evt := data.(eventbus.DeviceEvent)
    // 处理事件
})
```

## 10. 安全规范

| 数据 | 存储方式 | 传输 |
|------|----------|------|
| 用户密码 | bcrypt(cost=14) | HTTPS |
| VerifyCode | SHA-256 哈希 | MQTT TLS |
| JWT Token | HS256 签名 | HTTPS Header |
