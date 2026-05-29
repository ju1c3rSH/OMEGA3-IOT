# 设备接口（需 JWT 认证）

## 可访问设备列表

```
GET /api/v1/devices/accessible
Authorization: Bearer <token>
```

返回用户拥有和被分享的所有设备。

**响应示例**:
```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "devices": [
      {
        "instance_uuid": "550e8400-e29b-41d4-a716-446655440000",
        "name": "客厅温湿度计",
        "type": "BaseTracker",
        "online": true,
        "permission": "read_write",
        "shared_by": null
      }
    ]
  }
}
```

**错误响应**:
- `401` User not authenticated
- `500` Failed to get devices

## 发送指令

```
POST /api/v1/devices/{instance_uuid}/actions
Authorization: Bearer <token>
Content-Type: application/json
```

**中间件**: DeviceAccessMiddleware（`write` 权限）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `command` | string | ✅ | 指令名称（须符合设备类型 spec 定义） |
| `params` | object | 否 | 指令参数 |

**响应示例**:
```json
{
  "code": 200,
  "message": "Action sent successfully",
  "data": {
    "instance_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "command": "reboot"
  }
}
```

**错误响应**:
- `400` Invalid or missing query parameter
- `403` Access denied — 无写权限
- `404` Device not found
- `400` Action validation failed — 指令不符合设备 spec

## 历史数据

```
POST /api/v1/devices/{instance_uuid}/getHistoryData
Authorization: Bearer <token>
Content-Type: application/json
```

**中间件**: DeviceAccessMiddleware（`read` 权限）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `start_timestamp` | int64 | ✅ | 起始时间（Unix 时间戳） |
| `end_timestamp` | int64 | ✅ | 结束时间（Unix 时间戳，须大于 start） |
| `properties` | []string | 否 | 指定查询的属性列表 |
| `limit` | int | 否 | 1-5000，默认 1000 |
| `offset` | int | 否 | ≥0，默认 0 |

**业务规则**:
- `start_timestamp` 必须小于 `end_timestamp`
- 时间范围不超过 30 天（15,552,000 秒）

**响应示例**:
```json
{
  "code": 200,
  "message": "Device shared successfully",
  "data": {
    "instance_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "total_count": 2580,
    "returned_count": 1000,
    "has_more": true,
    "records": [
      {
        "timestamp": 1704067200,
        "properties": {"battery_level": {"v": 85, "type": "int"}}
      }
    ]
  }
}
```

**错误响应**:
- `400` Missing instance_uuid
- `401` Unauthorized
- `400` start_timestamp must be less than end_timestamp
- `422` Time range exceeds maximum allowed (30 days)
- `403` Access denied
- `404` Device not found

## 分享设备

```
POST /api/v1/devices/{instance_uuid}/share
Authorization: Bearer <token>
Content-Type: application/json
```

**中间件**: DeviceAccessMiddleware（`write` 权限）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `shared_with_uuid` | string | ✅ | 被分享用户的 UUID |
| `permission` | string | ✅ | `read` / `write` / `read_write` |
| `expires_at` | int64 | 否 | 过期时间（Unix 时间戳），0 或不传表示永不过期 |

**响应示例**:
```json
{"code": 200, "message": "Device shared successfully", "data": "[]"}
```

**错误响应**:
- `401` User not authenticated
- `403` Access denied
- `400` Invalid or missing query parameter
