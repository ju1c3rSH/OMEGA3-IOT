# OMEGA3-IOT HTTP API 规范

## 文档目录

| 文件 | 内容 |
|------|------|
| [认证迁移指南](./AuthMigration.md) | **Android/iOS 客户端必读** - DH Challenge-Response 认证实现 |
| [基础约定](./conventions.md) | Base URL、认证方式、响应格式、错误码、角色权限、中间件链 |
| [公开接口](./public.md) | 无需认证的接口：注册、登录、设备匿名注册、健康检查 |
| [用户接口](./user.md) | 用户资料管理、头像、设备创建与绑定 |
| [设备接口](./device.md) | 设备操作：指令发送、历史数据、设备分享 |
| [设备组接口](./device-group.md) | 设备维度分组管理 |
| [用户组接口](./user-group.md) | 用户协作分组：成员管理、邀请、策略、设备共享 |
| [WebSocket 推送](./websocket.md) | 实时推送通道 |
| [日志接口](./log.md) | 设备日志与用户操作日志 |
| [管理后台接口](./admin.md) | 管理员管理、用户/设备/组管理、系统统计 |
| [数据模型](./models.md) | 所有数据模型的字段定义 |
| [接口总览与快速参考](./reference.md) | 全部接口一览表、cURL 示例 |

## 快速导航

- **Android/iOS 开发？** 必看 [认证迁移指南](./AuthMigration.md) - 包含完整的 Kotlin 实现示例
- **新接入？** 先看 [基础约定](./conventions.md)，然后看 [公开接口](./public.md) 完成注册登录
- **设备开发？** 看 [设备接口](./device.md) 和 [WebSocket 推送](./websocket.md)
- **管理后台？** 看 [管理后台接口](./admin.md)
- **查字段？** 看 [数据模型](./models.md)
