# 设备文件夹接口（需 JWT 认证）

设备文件夹（Device Folder）是设备维度的**组织工具**，用于将用户自己的多个设备归类管理（类似文件夹/标签）。

> **语义说明**：此接口原名「设备组（Device Group）」，已重命名为「设备文件夹（Device Folder）」以避免与「用户组（User Group）」混淆。设备文件夹是单人使用的设备组织工具，用户组是多人协作的团队。

## 创建文件夹

```
POST /api/v1/devices/folders
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✅ | 文件夹名称（最大128字符） |
| `description` | string | 否 | 文件夹描述 |

**响应示例**:
```json
{
  "code": 200,
  "message": "Folder created successfully",
  "data": {
    "folder_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "name": "客厅设备",
    "owner_uuid": "...",
    "description": "客厅所有传感器",
    "created_at": "2026-05-29T00:00:00Z",
    "updated_at": "2026-05-29T00:00:00Z",
    "valid": 1
  }
}
```

## 设备加入文件夹

```
POST /api/v1/devices/{instance_uuid}/folders
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `folder_uuid` | string | ✅ | 文件夹 UUID |

**响应示例**:
```json
{
  "code": 200,
  "message": "Device added to folder successfully",
  "data": {"folder_uuid": "...", "device_uuid": "..."}
}
```

**错误响应**:
- `400` Invalid device_uuid / Invalid request parameters
- `403` Access denied — 设备不属于当前用户
- `404` Device not found

## 设备移出文件夹

```
DELETE /api/v1/devices/{instance_uuid}/folders/{folder_uuid}
Authorization: Bearer <token>
```

**响应示例**:
```json
{
  "code": 200,
  "message": "Device removed from folder successfully",
  "data": {"folder_uuid": "...", "device_uuid": "..."}
}
```

## 获取用户的所有文件夹

```
GET /api/v1/users/me/device_folders
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
  "message": "Folders retrieved successfully",
  "data": {
    "folders": [
      {
        "folder_uuid": "...",
        "name": "客厅设备",
        "owner_uuid": "...",
        "description": "",
        "created_at": "2026-05-29T00:00:00Z",
        "updated_at": "2026-05-29T00:00:00Z",
        "valid": 1
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 10
  }
}
```

## 获取文件夹中的设备

```
GET /api/v1/devices/folders/{folder_uuid}/devices
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
  "message": "Folder devices retrieved successfully",
  "data": {
    "devices": [
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

## 删除文件夹

```
DELETE /api/v1/devices/folders/{folder_uuid}
Authorization: Bearer <token>
```

**响应示例**:
```json
{"code": 200, "message": "Folder deleted successfully", "data": {"folder_uuid": "..."}}
```

**错误响应**:
- `400` Invalid folder_uuid
- `403` Permission denied
- `404` Folder not found
