# 管理后台接口

所有管理接口（除登录外）需要 JWT 认证 + AdminAuthMiddleware（角色 ≥ 2）。各接口还有独立的权限要求，参见 [角色与权限](./conventions.md#角色与权限)。

## 管理员登出

```
POST /api/v1/admin/logout
Authorization: Bearer <token>
```

**响应示例**:
```json
{"code": 200, "message": "Logged out", "data": null}
```

## 获取管理员列表

```
GET /api/v1/admin/admins
Authorization: Bearer <token>
```

**所需权限**: `admin:view`（仅 SuperAdmin）

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "admins": [
      {
        "user_uuid": "...",
        "username": "admin",
        "nickname": "超级管理员",
        "role": 4,
        "status": 0,
        "created_at": 1717000000,
        "last_seen": 1717000000
      }
    ]
  }
}
```

## 提升用户为管理员

```
POST /api/v1/admin/admins
Authorization: Bearer <token>
Content-Type: application/json
```

**所需权限**: `admin:manage`（仅 SuperAdmin）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `user_uuid` | string | ✅ | 目标用户 UUID |
| `role` | int | ✅ | 2= Moderator, 3=Admin（不能设为 4=SuperAdmin） |

**业务规则**: 不能修改自己的角色。

**响应示例**:
```json
{"code": 200, "message": "User promoted", "data": {"user_uuid": "...", "role": 3}}
```

**错误响应**:
- `400` cannot change your own role
- `400` invalid target role: must be moderator or admin

## 更新管理员角色

```
PUT /api/v1/admin/admins/{user_uuid}
Authorization: Bearer <token>
Content-Type: application/json
```

**所需权限**: `admin:manage`（仅 SuperAdmin）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `role` | int | ✅ | 2 或 3（不能设为 4） |

**业务规则**:
- 不能通过 API 设置 SuperAdmin
- 不能修改 SuperAdmin 的角色

**响应示例**:
```json
{"code": 200, "message": "Role updated", "data": {"user_uuid": "...", "role": 2}}
```

**错误响应**:
- `400` cannot assign super_admin via API
- `400` cannot modify super_admin role

## 撤销管理员

```
DELETE /api/v1/admin/admins/{user_uuid}
Authorization: Bearer <token>
```

**所需权限**: `admin:manage`（仅 SuperAdmin）

**业务规则**:
- 不能撤销自己
- 不能撤销 SuperAdmin

**响应示例**:
```json
{"code": 200, "message": "Admin demoted", "data": {"user_uuid": "..."}}
```

**错误响应**:
- `400` cannot demote yourself
- `400` cannot demote super_admin

## 用户列表

```
GET /api/v1/admin/users
Authorization: Bearer <token>
```

**所需权限**: `user:view`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页数量，默认 20 |
| `search` | string | 否 | 搜索用户名、昵称或 UUID |
| `role` | int | 否 | 按角色筛选 |
| `status` | int | 否 | 按状态筛选 |
| `sort_by` | string | 否 | 排序字段: `created_at` / `last_seen` / `username`，默认 `created_at` |
| `sort_order` | string | 否 | `asc` / `desc`，默认 `desc` |

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "users": [...],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

## 获取用户详情

```
GET /api/v1/admin/users/{user_uuid}
Authorization: Bearer <token>
```

**所需权限**: `user:view`

**响应示例**:
```json
{"code": 200, "message": "OK", "data": {...}}
```

**错误响应**:
- `404` User not found

## 编辑用户资料

```
PUT /api/v1/admin/users/{user_uuid}
Authorization: Bearer <token>
Content-Type: application/json
```

**所需权限**: `user:edit`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `nickname` | string | 否 | 昵称 |
| `description` | string | 否 | 描述 |

**响应示例**:
```json
{"code": 200, "message": "User updated", "data": {"user_uuid": "..."}}
```

## 更新用户状态

```
PUT /api/v1/admin/users/{user_uuid}/status
Authorization: Bearer <token>
Content-Type: application/json
```

**所需权限**: `user:status`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `status` | int | ✅ | 新状态值 |

**业务规则**: 不能通过此接口修改管理员状态。

**响应示例**:
```json
{"code": 200, "message": "Status updated", "data": {"user_uuid": "...", "status": 1}}
```

**错误响应**:
- `403` cannot change admin status via this endpoint

## 删除用户

```
DELETE /api/v1/admin/users/{user_uuid}
Authorization: Bearer <token>
```

**所需权限**: `user:delete`（仅 SuperAdmin）

**业务规则**: 不能删除管理员用户。

**响应示例**:
```json
{"code": 200, "message": "User deleted", "data": {"user_uuid": "..."}}
```

**错误响应**:
- `403` cannot delete admin users

## 重置用户密码

```
POST /api/v1/admin/users/{user_uuid}/reset-password
Authorization: Bearer <token>
Content-Type: application/json
```

**所需权限**: `user:reset`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `new_password` | string | ✅ | 新密码（最少 6 字符） |

**响应示例**:
```json
{"code": 200, "message": "Password reset successful", "data": {"user_uuid": "..."}}
```

## 设备列表

```
GET /api/v1/admin/devices
Authorization: Bearer <token>
```

**所需权限**: `device:view`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页数量，默认 20 |
| `search` | string | 否 | 搜索名称、UUID 或 SN |
| `type` | string | 否 | 按设备类型筛选 |
| `status` | string | 否 | 按状态筛选（如 `"active"`） |
| `owner` | string | 否 | 按所有者 UUID 筛选 |
| `online` | string | 否 | `"true"` / `"false"` |
| `sort_by` | string | 否 | `created_at` / `last_seen` / `name`，默认 `created_at` |
| `sort_order` | string | 否 | `asc` / `desc`，默认 `desc` |

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "devices": [...],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

## 获取设备详情

```
GET /api/v1/admin/devices/{instance_uuid}
Authorization: Bearer <token>
```

**所需权限**: `device:view`

**响应示例**:
```json
{"code": 200, "message": "OK", "data": {...}}
```

**错误响应**:
- `404` Device not found

## 编辑设备

```
PUT /api/v1/admin/devices/{instance_uuid}
Authorization: Bearer <token>
Content-Type: application/json
```

**所需权限**: `device:edit`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 否 | 设备名称 |
| `description` | string | 否 | 设备描述 |
| `remark` | string | 否 | 备注 |
| `sn` | string | 否 | 序列号 |

**响应示例**:
```json
{"code": 200, "message": "Device updated", "data": {"instance_uuid": "..."}}
```

## 删除设备

```
DELETE /api/v1/admin/devices/{instance_uuid}
Authorization: Bearer <token>
```

**所需权限**: `device:delete`

**响应示例**:
```json
{"code": 200, "message": "Device deleted", "data": {"instance_uuid": "..."}}
```

## 转移设备所有权

```
POST /api/v1/admin/devices/{instance_uuid}/transfer
Authorization: Bearer <token>
Content-Type: application/json
```

**所需权限**: `device:transfer`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `new_owner_uuid` | string | ✅ | 新所有者 UUID |
| `keep_original_access` | bool | 否 | 是否保留原所有者访问权限，默认 false |

**业务规则**: 不能转移给自己。

**响应示例**:
```json
{
  "code": 200,
  "message": "Device transferred",
  "data": {
    "instance_uuid": "...",
    "new_owner": "...",
    "keep_original": false
  }
}
```

**错误响应**:
- `400` device already belongs to this user

## 用户组列表

```
GET /api/v1/admin/groups
Authorization: Bearer <token>
```

**所需权限**: `group:view`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页数量，默认 20 |

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "groups": [...],
    "total": 30,
    "page": 1,
    "page_size": 20
  }
}
```

## 获取用户组详情

```
GET /api/v1/admin/groups/{group_uuid}
Authorization: Bearer <token>
```

**所需权限**: `group:view`

**响应示例**:
```json
{"code": 200, "message": "OK", "data": {...}}
```

**错误响应**:
- `404` Group not found

## 获取用户组成员

```
GET /api/v1/admin/groups/{group_uuid}/members
Authorization: Bearer <token>
```

**所需权限**: `group:view`

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {"members": [...]}
}
```

## 解散用户组

```
DELETE /api/v1/admin/groups/{group_uuid}
Authorization: Bearer <token>
```

**所需权限**: `group:manage`

**响应示例**:
```json
{"code": 200, "message": "Group dissolved", "data": {"group_uuid": "..."}}
```

## 移除用户组成员

```
DELETE /api/v1/admin/groups/{group_uuid}/members/{user_uuid}
Authorization: Bearer <token>
```

**所需权限**: `group:manage`

**响应示例**:
```json
{"code": 200, "message": "Member removed", "data": {"group_uuid": "...", "user_uuid": "..."}}
```

## 系统统计概览

```
GET /api/v1/admin/stats/overview
Authorization: Bearer <token>
```

**所需权限**: `system:stats`

**响应示例**:
```json
{"code": 200, "message": "OK", "data": {...}}
```

## 管理操作日志

```
GET /api/v1/admin/logs
Authorization: Bearer <token>
```

**所需权限**: `system:logs`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页数量，默认 20 |

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "logs": [
      {
        "id": 1,
        "admin_uuid": "...",
        "action": "user.delete",
        "target_type": "user",
        "target_uuid": "...",
        "detail": "{\"username\":\"test\"}",
        "ip": "127.0.0.1",
        "created_at": 1717000000
      }
    ],
    "total": 25,
    "page": 1,
    "page_size": 20
  }
}
```
