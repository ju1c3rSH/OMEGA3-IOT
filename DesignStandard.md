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
│   ├── db/                 # 数据库连接 (MySQL + IoTDB)
│   ├── eventbus/           # 内部事件总线
│   ├── handler/            # HTTP 处理器
│   │   └── middlewares/    # Gin 中间件
│   ├── logger/             # 日志服务 (IoTDB)
│   ├── model/              # 数据模型
│   ├── repository/         # 数据访问层
│   ├── service/            # 业务逻辑层
│   ├── types/              # 共享类型定义
│   └── utils/              # 工具函数
├── internal/config/        # 配置文件目录
│   ├── device_type_list.yaml
│   ├── GeneralConfig.yaml
│   └── mqtt_config.yaml
└── main.go
```

## 3. 数据模型

### 3.1 Instance (设备实例)

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 主键自增 |
| InstanceUUID | string | 设备唯一标识 (UUID v4) |
| Name | string | 设备名称 |
| Type | string | 设备类型名称 |
| Online | bool | 在线状态 |
| OwnerUUID | string | 所有者 UUID |
| Properties | Properties | 当前属性值 (JSON) |
| VerifyHash | string | 设备认证哈希 |
| Status | string | active/inactive |
| SN | string | 序列号 |
| IsShared | bool | 是否被分享 |
| SharedCount | int | 分享次数 |
| Remark | string | 备注 |

### 3.2 DeviceRegistrationRecord (注册记录)

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 主键 |
| DeviceUUID | string | 预分配设备 UUID |
| RegCode | string | 8位注册码 |
| DeviceTypeID | int | 设备类型ID |
| SN | string | 序列号 |
| VerifyHash | string | VerifyCode SHA-256 哈希 |
| CreatedAt | int64 | 创建时间戳 |
| ExpiresAt | int64 | 过期时间 (24小时) |
| IsBound | bool | 是否已绑定 |

### 3.3 DeviceShare (设备分享)

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 主键 |
| InstanceUUID | string | 被分享设备 |
| SharedWithUUID | string | 被分享用户 |
| SharedByUUID | string | 分享者 |
| Permission | string | read/write/read_write |
| Status | string | active/revoked |
| CreatedAt | int64 | 创建时间 |
| UpdatedAt | int64 | 更新时间 |
| ExpiresAt | *int64 | 过期时间 (nil=永不过期) |

### 3.4 DeviceGroup (设备组)

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | int64 | 主键 |
| Name | string | 组名称 |
| OwnerID | int64 | 所有者ID |
| Description | string | 描述 |
| CreatedAt | time.Time | 创建时间 |
| UpdatedAt | time.Time | 更新时间 |
| Valid | int8 | 是否有效 (1=有效) |

### 3.5 DeviceGroupRelation (组关系)

| 字段 | 类型 | 说明 |
|------|------|------|
| GroupID | int64 | 组ID |
| DeviceID | int64 | 设备ID |
| JoinedAt | time.Time | 加入时间 |
| Valid | int8 | 是否有效 |

### 3.6 Properties (属性系统)

```go
type Properties struct {
    Items map[string]*TypedInstancePropertyItem `json:"items"`
}

type TypedInstancePropertyItem struct {
    Value TypedValue   `json:"value"`
    Meta  PropertyMeta `json:"meta"`
}

type TypedValue struct {
    V         interface{} `json:"v"`
    Type      string      `json:"type"`      // string/int/float/bool/time/json/binary
    Timestamp int64       `json:"timestamp"`
}

type PropertyMeta struct {
    Writable           bool     `json:"writable"`
    Description        string   `json:"description"`
    Unit               string   `json:"unit"`
    Range              []int    `json:"range"`
    Format             string   `json:"format"`
    Enum               []string `json:"enum"`
    Required           bool     `json:"required"`
    Pattern            string   `json:"pattern"`            // 正则校验
    RequiredCapabilities []string `json:"required_capabilities"`
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
        "value": {
          "v": 85,
          "type": "int",
          "timestamp": 1756882749000
        },
        "meta": { ... }
      }
    }
  }
}
```

### 4.3 指令下发格式

```json
{
  "command": "reboot",
  "params": { "delay_sec": 5 },
  "timestamp": 1729450800
}
```

### 4.4 事件上报格式

```json
{
  "verify_code": "xxx",
  "timestamp": 1756882749,
  "event": {
    "type": "low_battery_alarm",
    "severity": "warning",
    "params": {
      "current_level": 15,
      "threshold": 20
    }
  }
}
```

## 5. 设备注册流程

```
阶段1: 匿名注册 (设备端)
  POST /api/v1/device/deviceRegisterAnon
  Body: device_type_id=1
  ← 返回 {uuid, reg_code, verify_code, expires_at}

阶段2: 用户绑定 (App端)
  POST /api/v1/users/bindDeviceByRegCode
  Body: {reg_code, device_nick, device_remark}
  ← 设备激活，MQTT 下发 GO_ON 指令

阶段3: 设备激活 (设备端)
  收到 GO_ON 指令后开始正常上报数据
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
c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
    http.StatusInternalServerError,
    "Failed to create device",
    err.Error(),
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

- **users**: 用户信息
- **instances**: 设备实例
- **device_registration_records**: 设备注册记录
- **device_shares**: 设备分享关系
- **device_group**: 设备组
- **device_group_relation**: 设备组关系

### 7.2 IoTDB 结构

```
root.device_data.{instance_uuid}.{property_name}
├── timestamp
├── value
└── quality

root.logs.{device|user|system}.{event_type}
├── timestamp
├── level
├── message
└── metadata
```

## 8. 配置文件

### 8.1 device_type_list.yaml

```yaml
device_types:
  - id: 1
    name: "BaseTracker"
    description: "定位器"

    capabilities:
      gps_module:
        description: "支持 GPS/北斗定位"
        required: true

    properties:
      battery_level:
        writable: true
        description: "当前电量百分比"
        unit: "%"
        range: [0, 100]
        format: "int"
        required: true

    events:
      low_battery_alarm:
        description: "电量低于阈值时主动上报"
        severity: "warning"
        output_params:
          - name: "current_level"
            type: "int"
        retention_days: 30

    actions:
      reboot:
        description: "远程重启设备"
        input_params: []
        timeout_sec: 30
        idempotent: true
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
  queryTimeoutMs: 4000
  pool:
    maxConnections: 5
    timeout: 60000
    fetchMetadataAuto: false
```

## 9. 事件系统

```go
// 发布事件
eventBus.Publish(ctx, event)

// 订阅事件 (类型安全)
eventbus.SubscribeTyped(eventBus, eventbus.LogEventDeviceLogUpload, func(evt DeviceLogEvent) {
    // 处理事件
})
```

### 9.1 事件类型

- `LogEventDeviceLogUpload`: 设备日志上传
- `LogEventUserAction`: 用户操作日志
- `LogEventSystemError`: 系统错误日志

## 10. 安全规范

| 数据 | 存储方式 | 传输 |
|------|----------|------|
| 用户密码 | bcrypt(cost=14) | HTTPS |
| VerifyCode | SHA-256 哈希 | MQTT TLS |
| JWT Token | HS256 签名 | HTTPS Header |
| 设备通信 | VerifyHash 验证 | MQTT QoS 1 |

## 11. 设备组功能

设备组允许用户将多个设备组织在一起进行管理：

- **创建组**: 用户可创建多个设备组
- **加入组**: 设备可加入多个组
- **权限**: 只有设备所有者才能将设备加入/退出组
- **组解散**: 只有组创建者可以解散组

## 12. 日志系统

日志数据存储在 IoTDB 中，支持以下类型：

- **设备日志**: 设备上报的运行日志、事件
- **用户日志**: 用户操作记录
- **系统日志**: 系统错误和异常

查询支持时间范围过滤和分页。
