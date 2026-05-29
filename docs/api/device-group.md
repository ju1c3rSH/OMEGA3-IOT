# 设备组接口（需 JWT 认证）

设备组是设备维度的分组，用于将多个设备归类管理。

## 创建组

```
POST /api/v1/devices/groups/create_group
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✅ | 组名（最大128字符） |
| `description` | string | 否 | 组描述 |

**响应示例**:
```json
{
  "code": 200,
  "message": "Group created successfully",
  "data": {
    "group_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "name": "客厅设备",
    "owner_uuid": "...",
    "description": "客厅所有传感器",
    "created_at": "2026-05-29T00:00:00Z",
    "updated_at": "2026-05-29T00:00:00Z",
    "valid": 1
  }
}
```

## 设备加入组

```
POST /api/v1/devices/{instance_uuid}/join_group
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `group_uuid` | string | ✅ | 组 UUID |

**响应示例**:
```json
{
  "code": 200,
  "message": "Device joined group successfully",
  "data": {"group_uuid": "...", "device_uuid": "..."}
}
```

**错误响应**:
- `400` Invalid device_uuid / Invalid request parameters
- `403` Access denied — 设备不属于当前用户
- `404` Device not found

## 设备退出组

```
POST /api/v1/devices/{instance_uuid}/quit_group
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `group_uuid` | string | ✅ | 组 UUID |

**响应示例**:
```json
{
  "code": 200,
  "message": "Device quit group successfully",
  "data": {"group_uuid": "...", "device_uuid": "..."}
}
```

## 获取用户的所有设备组

```
GET /api/v1/users/me/device_groups
Authorization: Bearer <token>
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页数量，默认 10，最大 100 |

**响应示例**:
```json
{
  "code": 200,
  "message": "Groups retrieved successfully",
  "data": {
    "groups": [...],
    "total": 10,
    "page": 1,
    "page_size": 10
  }
}
```

## 获取组成员

```
GET /api/v1/devices/groups/{group_uuid}/members
Authorization: Bearer <token>
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页数量，默认 10，最大 100 |

**响应示例**:
```json
{
  "code": 200,
  "message": "Group members retrieved successfully",
  "data": {
    "members": [
      {
        "instance_uuid": "...",
        "name": "客厅温湿度计",
        "type": "BaseTracker",
        "online": true,
        "owner_uuid": "...",
        "description": "",
        "properties": {},
        "status": "active",
        "joined_at": "2026-05-29T00:00:00Z"
      }
    ],
    "total": 5,
    "page": 1,
    "page_size": 10
  }
}
```

## 解散组

```
POST /api/v1/devices/groups/{group_uuid}/dismiss_group
Authorization: Bearer <token>
```

**响应示例**:
```json
{"code": 200, "message": "Group dismissed successfully", "data": {"group_uuid": "..."}}
```

**错误响应**:
- `400` Invalid group_uuid
- `404` Could not find a match group
