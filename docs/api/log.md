# 日志接口（需 JWT 认证）

## 查询设备日志

```
GET /api/v1/logs/device
Authorization: Bearer <token>
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `device_uuid` | string | ✅ | 设备 UUID |
| `start_time` | int64 | 否 | 起始时间（Unix 时间戳），默认 24 小时前 |
| `end_time` | int64 | 否 | 结束时间（Unix 时间戳），默认当前时间 |
| `limit` | int | 否 | 默认 100 |
| `offset` | int | 否 | 默认 0 |

**响应示例**:
```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "total": 150,
    "entries": [
      {
        "timestamp": 1717000000,
        "level": "info",
        "message": "设备启动完成",
        "event_type": "device.log.upload",
        "source": "device",
        "metadata": {}
      }
    ]
  }
}
```

**错误响应**:
- `400` Missing parameter — device_uuid 必填

## 上传设备日志

```
POST /api/v1/logs/device/upload
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `device_uuid` | string | ✅ | 设备 UUID |
| `message` | string | ✅ | 日志消息 |
| `level` | string | 否 | 日志级别，默认 `"info"`（可选: info/warning/panic） |
| `event_type` | string | 否 | 事件类型，默认 `"device.log.upload"` |
| `metadata` | object | 否 | 附加元数据 |
| `action_name` | string | 否 | 指令名称 |
| `action_data` | object | 否 | 指令数据 |
| `result` | string | 否 | 执行结果 |
| `error_code` | string | 否 | 错误码 |
| `error_detail` | string | 否 | 错误详情 |

**响应示例**:
```json
{"code": 200, "message": "Success", "data": "Log uploaded successfully"}
```

**错误响应**:
- `400` Invalid request body
- `400` Missing required fields — device_uuid 和 message 必填

## 查询用户操作日志

```
GET /api/v1/logs/user
Authorization: Bearer <token>
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `user_uuid` | string | 否 | 用户 UUID，不传则返回所有用户的日志 |
| `start_time` | int64 | 否 | 起始时间（Unix 时间戳），默认 24 小时前 |
| `end_time` | int64 | 否 | 结束时间（Unix 时间戳），默认当前时间 |
| `limit` | int | 否 | 默认 100 |
| `offset` | int | 否 | 默认 0 |

**响应示例**: 同设备日志格式。
