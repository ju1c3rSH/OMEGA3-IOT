# OMEGA3-IOT

> **⚠️ 学习型项目 | Learning Project**
>
> 本项目处于快速迭代阶段，架构和 API 可能随时变更。**不建议用于生产环境**。

## 项目定位

OMEGA3-IOT 是一个**高度可定制、易于部署**的物联网设备管理平台。

- **协议**: HTTP REST API + MQTT 双协议
- **数据库**: MySQL (元数据) + Apache IoTDB (时序数据)
- **部署**: 单二进制文件 + YAML 配置，开箱即用

## 快速启动

```bash
# 1. 准备数据库 (MySQL + IoTDB)
# 2. 复制并修改配置
cp internal/config/GeneralConfig_example.yaml internal/config/GeneralConfig.yaml

# 3. 设置环境变量
export JWT_SECRET=your_strong_secret

# 4. 运行
go run main.go
```

## 核心特性

| 特性 | 状态 | 说明 |
|------|------|------|
| 设备类型系统 | ✅ | YAML 配置驱动，动态扩展 |
| 两阶段注册 | ✅ | 匿名注册 → 用户绑定 |
| MQTT 通信 | ✅ | 属性上报 / 指令下发 |
| 设备分享 | ✅ | 支持 read/write/read_write 权限 |
| 历史数据 | ✅ | 基于 IoTDB 的时序查询 |
| 日志系统 | ✅ | 结构化事件日志 |

## TODO Checklist

### 功能开发
- [ ] **自动化测试** - 单元测试 + 集成测试覆盖
- [ ] **配置热重载** - 无需重启更新配置
- [ ] **设备固件 OTA** - 远程固件升级支持
- [ ] **告警规则引擎** - 基于属性的触发器
- [ ] **数据可视化** - 内置简易 Dashboard
- [ ] **多租户支持** - 组织/团队级别的资源隔离
- [ ] **Webhook 通知** - 设备事件外部推送

### 架构优化
- [ ] **插件系统** - 设备协议插件化
- [ ] **缓存层** - Redis 缓存热点数据
- [ ] **消息队列** - 异步处理设备数据
- [ ] **限流熔断** - API 速率限制
- [ ] **服务发现** - 分布式部署支持

### 体验改进
- [ ] **Docker 一键部署** - compose 配置
- [ ] **CLI 工具** - 设备管理命令行
- [ ] **文档站点** - 自动生成 API 文档
- [ ] **SDK 发布** - Go/Python/JS 客户端

## 项目文档

- [DesignStandard.md](DesignStandard.md) - 架构设计与开发规范
- [APIStandard.md](APIStandard.md) - HTTP API 接口文档

## 技术栈

```
┌─────────────────────────────────────────┐
│  HTTP API (Gin)  │  MQTT (paho.mqtt)   │
├─────────────────────────────────────────┤
│  Service Layer (Business Logic)         │
├─────────────────────────────────────────┤
│  Repository Layer (GORM)                │
├─────────────────────────────────────────┤
│  MySQL      │    IoTDB    │   EventBus │
└─────────────────────────────────────────┘
```

## 贡献

这是一个个人学习项目，欢迎提 Issue 和 PR。

## License

MIT
