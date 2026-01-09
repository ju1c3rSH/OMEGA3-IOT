# OMEGA3-IOT

## 项目概述

OMEGA3-IOT 是一个基于 Go 的物联网设备管理平台，采用 HTTP REST API + MQTT 双协议架构。 [1](#1-0) 

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

###  代码优化
- [ ] **设备类型加载封装** - `LoadDeviceTypeFromYAML` 需要重构为通用加载器 [5](#1-4) 
- [ ] **实例创建验证** - `NewInstanceFromConfig` 需要加上验证Hash [6](#1-5) 
- [ ] **设备注册防刷** - `RegisterDeviceAnonymously` 需要添加频率限制 [7](#1-6)
- [ ] **MQTT解耦** - `PublishActionToDevice` 需要解耦处理 [9](#1-8) 
- [ ] **MQTT重试机制** - 添加Retry Pool处理发送失败 [10](#1-9) 
- [ ] **设备工厂实现** - `GetSupportedTypes` 方法待实现 [11](#1-10)
- [ ] **VerifyCode加盐** - `GenerateVerifyCode` 需要添加salt [13](#1-12) 

###  功能增强
- [ ] **更好的Log保存系统** - 实现结构化日志和持久化存储
- [ ] **权限账号管理机制** - 实现Group、Team多级权限管理
- [ ] **属性类型验证** - PropertyMeta需要Required Type字段 [14](#1-13) 


## 开发规范 
- 详情请查看Designstandard.md
## Notes

项目目前处于开发阶段，核心功能已实现但需要优化和扩展。重点关注设备注册流程、MQTT通信和权限管理系统的完善。

Wiki pages you might want to explore:
- [Device Lifecycle & Registration (ju1c3rSH/OMEGA3-IOT)](/wiki/ju1c3rSH/OMEGA3-IOT#5.1)
- [MQTT Communication System (ju1c3rSH/OMEGA3-IOT)](/wiki/ju1c3rSH/OMEGA3-IOT#5.2)
- [Authentication & Security (ju1c3rSH/OMEGA3-IOT)](/wiki/ju1c3rSH/OMEGA3-IOT#5.5)
