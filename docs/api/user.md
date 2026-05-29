# 用户接口（需 JWT 认证）

## 用户登出

```
POST /api/v1/users/logout
Authorization: Bearer <token>
```

**行为**: 将当前 Token 的 JTI 加入 Redis 黑名单，并清除 Authorization Cookie。

**响应示例**:
```json
{"code": 200, "message": "Logged out successfully", "data": null}
```

## 获取用户信息

```
GET /api/v1/users/info
Authorization: Bearer <token>
```

**响应示例**:
```json
{
  "code": 200,
  "message": "User info retrieved successfully",
  "data": {
    "id": 1,
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "username": "admin",
    "nickname": "管理员",
    "avatar_url": "/uploads/avatars/admin.png",
    "description": "系统管理员",
    "role": 1,
    "created_at": 1717000000,
    "last_seen": 1717000000,
    "last_ip": "127.0.0.1"
  }
}
```

**错误响应**:
- `401` Authentication required
- `404` User not found

## 获取用户所有设备

```
GET /api/v1/users/getUserAllDevices
Authorization: Bearer <token>
```

**响应示例**: 返回用户拥有的所有设备列表。

**错误响应**:
- `401` Authentication required
- `500` Failed to retrieve devices

## 更新用户资料

```
PUT /api/v1/users/profile
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `nickname` | string | 否 | 昵称 |
| `description` | string | 否 | 个人描述 |

**响应示例**:
```json
{"code": 200, "message": "Profile updated successfully", "data": null}
```

**错误响应**:
- `401` Authentication required
- `400` Invalid input
- `500` Failed to update profile

## 上传头像

```
POST /api/v1/users/avatar
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `avatar` | file | ✅ | 头像图片文件 |

**响应示例**:
```json
{
  "code": 200,
  "message": "Avatar uploaded successfully",
  "data": {"avatar_url": "/uploads/avatars/550e8400.png"}
}
```

**错误响应**:
- `401` Authentication required
- `400` Missing avatar file
- `500` Failed to upload avatar

## 重置头像

```
DELETE /api/v1/users/avatar
Authorization: Bearer <token>
```

**响应示例**:
```json
{
  "code": 200,
  "message": "Avatar reset to default",
  "data": {"avatar_url": "/uploads/avatars/default.png"}
}
```

**错误响应**:
- `401` Authentication required
- `500` Failed to reset avatar

## 创建设备

```
POST /api/v1/users/addDevice
Authorization: Bearer <token>
Content-Type: application/x-www-form-urlencoded
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✅ | 设备名称（唯一） |
| `device_type` | int | ✅ | 设备类型 ID |
| `description` | string | 否 | 设备描述 |

**响应示例**:
```json
{
  "code": 200,
  "message": "Device Created Successfully.",
  "data": {
    "id": 1,
    "instance_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "name": "客厅温湿度计",
    "type": "BaseTracker",
    "online": false,
    "owner_uuid": "...",
    "status": "active",
    "created_at": "2026-05-29T00:00:00Z",
    "updated_at": "2026-05-29T00:00:00Z"
  }
}
```

**错误响应**:
- `400` Invalid or missing query parameter
- `401` User not authenticated
- `400` Unsupported device type
- `400` Device name already exists

## 绑定设备

```
POST /api/v1/users/bindDeviceByRegCode
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `reg_code` | string | ✅ | 注册码（设备匿名注册时获得） |
| `device_nick` | string | 否 | 设备昵称 |
| `device_remark` | string | 否 | 设备备注 |

**响应示例**:
```json
{
  "code": 200,
  "message": "Device created successfully",
  "data": {
    "device": {
      "id": 1,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "name": "客厅温湿度计",
      "type": "BaseTracker",
      "online": false,
      "description": "",
      "created_at": 1717000000,
      "last_seen": 0,
      "remark": ""
    }
  }
}
```

**错误响应**:
- `400` Invalid input
- `401` User not authenticated
- `400` Unsupported device type
- `500` Failed to create device
