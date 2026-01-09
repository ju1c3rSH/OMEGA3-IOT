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
