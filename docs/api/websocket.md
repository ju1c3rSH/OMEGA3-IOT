# WebSocket 推送通道

## 建立 WebSocket 连接

```
GET /api/v1/ws
Authorization: Bearer <token>
Upgrade: websocket
```

**行为**: 将 HTTP 连接升级为 WebSocket，用于实时接收设备属性更新、事件推送等。

**连接参数**:
- 读缓冲: 1024 字节
- 写缓冲: 1024 字节
- 允许所有来源（开发环境）

**认证失败响应** (HTTP 401):
```json
{"error": "authentication required"}
```
