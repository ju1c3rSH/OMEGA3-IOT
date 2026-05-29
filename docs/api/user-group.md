# 用户组接口（需 JWT 认证）

用户组是用户维度的协作分组，支持成员管理、权限控制、邀请机制和设备共享。

## 创建用户组

```
POST /api/v1/groups
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✅ | 组名（1-128字符） |
| `description` | string | 否 | 组描述 |
| `max_members` | int | 否 | 最大成员数，0 表示无限制 |

**响应示例** (HTTP 201):
```json
{
  "code": 201,
  "message": "Group created",
  "data": {
    "id": 1,
    "group_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "name": "研发团队",
    "description": "研发部门设备管理组",
    "owner_uuid": "...",
    "max_members": 50,
    "status": 0,
    "created_at": 1717000000,
    "updated_at": 1717000000
  }
}
```

## 获取我的用户组列表

```
GET /api/v1/groups
Authorization: Bearer <token>
```

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "groups": [
      {
        "id": 1,
        "group_uuid": "...",
        "name": "研发团队",
        "description": "",
        "owner_uuid": "...",
        "max_members": 50,
        "status": 0,
        "created_at": 1717000000,
        "updated_at": 1717000000
      }
    ]
  }
}
```

## 获取用户组详情

```
GET /api/v1/groups/{group_uuid}
Authorization: Bearer <token>
```

**业务规则**: 必须是组成员才能查看详情。

**错误响应**:
- `404` Group not found — 组不存在或已解散
- `403` Access denied — 不是组成员

## 更新用户组

```
PUT /api/v1/groups/{group_uuid}
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 否 | 新组名 |
| `description` | string | 否 | 新描述 |
| `max_members` | int | 否 | 新的最大成员数 |

**业务规则**: 需要组管理员或所有者权限。

**响应示例**:
```json
{"code": 200, "message": "Group updated", "data": {"group_uuid": "..."}}
```

**错误响应**:
- `403` Access denied — 权限不足

## 解散用户组

```
DELETE /api/v1/groups/{group_uuid}
Authorization: Bearer <token>
```

**业务规则**: 仅组所有者可解散。

**响应示例**:
```json
{"code": 200, "message": "Group dissolved", "data": {"group_uuid": "..."}}
```

**错误响应**:
- `403` Access denied — 非所有者

## 获取用户组成员列表

```
GET /api/v1/groups/{group_uuid}/members
Authorization: Bearer <token>
```

**业务规则**: 必须是组成员。

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "members": [
      {
        "id": 1,
        "group_uuid": "...",
        "user_uuid": "...",
        "role": 2,
        "status": 0,
        "joined_at": 1717000000,
        "invited_by": ""
      }
    ]
  }
}
```

**成员角色 (`role`)**:
| 值 | 说明 |
|----|------|
| 0 | 普通成员 |
| 1 | 组管理员 |
| 2 | 组所有者 |

**成员状态 (`status`)**:
| 值 | 说明 |
|----|------|
| 0 | 活跃 |
| 1 | 已退出 |
| 2 | 已踢出 |
| 3 | 待审批 |

## 定向邀请

```
POST /api/v1/groups/{group_uuid}/invite/search
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `invitee_uuid` | string | ✅ | 被邀请用户的 UUID |
| `expires_at` | int64 | 否 | 过期时间（Unix 时间戳），默认 7 天后 |

**业务规则**:
- 检查组策略 `allow_search_invite` 和 `allow_member_invite`
- 被邀请用户必须存在
- 被邀请用户不能已是组成员
- 不能有重复的待处理邀请

**响应示例** (HTTP 201):
```json
{
  "code": 201,
  "message": "Invite created",
  "data": {
    "id": 1,
    "group_uuid": "...",
    "invite_code": "ABCD1234EFGH5678",
    "inviter_uuid": "...",
    "invitee_uuid": "...",
    "type": 0,
    "status": 0,
    "expires_at": 1717604800,
    "created_at": 1717000000
  }
}
```

**错误响应**:
- `403` Access denied — 不是组成员或成员无权邀请
- `404` User not found

## 创建邀请链接

```
POST /api/v1/groups/{group_uuid}/invite/link
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `expires_at` | int64 | 否 | 过期时间（Unix 时间戳），默认 7 天后 |

**业务规则**:
- 检查组策略 `allow_invite_link` 和 `allow_member_invite`
- 链接邀请的 `invitee_uuid` 为空，任何有链接的人都可以加入

**响应示例** (HTTP 201):
```json
{
  "code": 201,
  "message": "Invite link created",
  "data": {
    "id": 2,
    "group_uuid": "...",
    "invite_code": "XYZW9876ABCD5432",
    "inviter_uuid": "...",
    "invitee_uuid": "",
    "type": 1,
    "status": 0,
    "expires_at": 1717604800,
    "created_at": 1717000000
  }
}
```

## 接受邀请

```
POST /api/v1/groups/invite/{invite_code}/accept
Authorization: Bearer <token>
```

**业务规则**:
- 邀请必须是 `pending` 状态且未过期
- 搜索邀请 (`type=0`): `invitee_uuid` 必须匹配当前用户
- 链接邀请 (`type=1`): 任何人都可以接受
- 如果组策略 `require_approval` 为 true，成员状态为 `pending`（待审批）
- 已是成员则返回 409

**响应示例**:
```json
{
  "code": 200,
  "message": "Invite accepted",
  "data": {
    "id": 1,
    "group_uuid": "...",
    "user_uuid": "...",
    "role": 0,
    "status": 0,
    "joined_at": 1717000000,
    "invited_by": "..."
  }
}
```

**错误响应**:
- `404` Invite not found
- `400` Invalid invite — 已过期、已处理或不匹配
- `409` Already a member

## 审批成员

```
POST /api/v1/groups/{group_uuid}/members/{user_uuid}/approve
Authorization: Bearer <token>
```

**业务规则**: 需要组管理员或所有者权限（取决于 `approval_mode` 策略）。

**响应示例**:
```json
{"code": 200, "message": "Member approved", "data": {"group_uuid": "...", "user_uuid": "..."}}
```

## 拒绝成员

```
POST /api/v1/groups/{group_uuid}/members/{user_uuid}/reject
Authorization: Bearer <token>
```

**业务规则**: 需要组管理员或所有者权限。

**响应示例**:
```json
{"code": 200, "message": "Member rejected", "data": {"group_uuid": "...", "user_uuid": "..."}}
```

## 移除成员

```
DELETE /api/v1/groups/{group_uuid}/members/{user_uuid}
Authorization: Bearer <token>
```

**业务规则**:
- 需要组管理员或所有者权限
- 不能移除组所有者

**响应示例**:
```json
{"code": 200, "message": "Member removed", "data": {"group_uuid": "...", "user_uuid": "..."}}
```

**错误响应**:
- `400` Cannot remove member — 不能移除所有者

## 退出用户组

```
POST /api/v1/groups/{group_uuid}/leave
Authorization: Bearer <token>
```

**业务规则**: 组所有者不能退出（需要先转让所有权或解散组）。

**响应示例**:
```json
{"code": 200, "message": "Left group", "data": {"group_uuid": "..."}}
```

**错误响应**:
- `400` Cannot leave group — 所有者不能退出

## 更新成员角色

```
PUT /api/v1/groups/{group_uuid}/members/{user_uuid}/role
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `role` | int | ✅ | 0=成员, 1=管理员（不能设置为 2=所有者） |

**业务规则**:
- 需要组所有者权限
- 不能修改自己的角色
- `role` 只能是 0 或 1

**响应示例**:
```json
{"code": 200, "message": "Role updated", "data": {"group_uuid": "...", "user_uuid": "...", "role": 1}}
```

## 获取用户组设备列表

```
GET /api/v1/groups/{group_uuid}/devices
Authorization: Bearer <token>
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页数量，默认 10，最大 100 |

**业务规则**: 必须是组成员。

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "devices": [...],
    "total": 3,
    "page": 1,
    "page_size": 10
  }
}
```

## 分享设备到用户组

```
POST /api/v1/groups/{group_uuid}/devices/share
Authorization: Bearer <token>
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `instance_uuid` | string | ✅ | 设备实例 UUID |
| `permission` | string | ✅ | `read` / `write` / `read_write` |

**业务规则**:
- 必须是组成员
- 必须是设备的所有者

**响应示例** (HTTP 201):
```json
{"code": 201, "message": "Device shared", "data": {"group_uuid": "...", "instance_uuid": "..."}}
```

## 撤销用户组设备分享

```
DELETE /api/v1/groups/{group_uuid}/devices/{instance_uuid}
Authorization: Bearer <token>
```

**响应示例**:
```json
{"code": 200, "message": "Share revoked", "data": {"group_uuid": "...", "instance_uuid": "..."}}
```

## 获取用户组策略

```
GET /api/v1/groups/{group_uuid}/policy
Authorization: Bearer <token>
```

**业务规则**: 必须是组成员。

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "id": 1,
    "group_uuid": "...",
    "device_visibility": 1,
    "admin_device_access": 0,
    "allow_search_invite": true,
    "allow_invite_link": true,
    "allow_member_invite": false,
    "require_approval": false,
    "approval_mode": 0,
    "created_at": 1717000000,
    "updated_at": 1717000000
  }
}
```

**策略字段说明**:

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `device_visibility` | int | 1 | 0=全部可见, 1=仅已分享, 2=选择性 |
| `admin_device_access` | int | 0 | 0=只读, 1=完全访问 |
| `allow_search_invite` | bool | true | 是否允许定向邀请 |
| `allow_invite_link` | bool | true | 是否允许邀请链接 |
| `allow_member_invite` | bool | false | 是否允许普通成员邀请 |
| `require_approval` | bool | false | 新成员是否需要审批 |
| `approval_mode` | int | 0 | 0=仅管理员可审批, 1=任何成员可审批 |

## 更新用户组策略

```
PUT /api/v1/groups/{group_uuid}/policy
Authorization: Bearer <token>
Content-Type: application/json
```

所有字段均为可选（指针类型，传 null 或不传表示不修改）。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `device_visibility` | int | 否 | 0/1/2 |
| `admin_device_access` | int | 否 | 0/1 |
| `allow_search_invite` | bool | 否 | |
| `allow_invite_link` | bool | 否 | |
| `allow_member_invite` | bool | 否 | |
| `require_approval` | bool | 否 | |
| `approval_mode` | int | 否 | 0/1 |

**业务规则**: 需要组管理员或所有者权限。

**响应示例**:
```json
{"code": 200, "message": "Policy updated", "data": {"group_uuid": "..."}}
```

## 获取待处理邀请列表

```
GET /api/v1/groups/{group_uuid}/invites
Authorization: Bearer <token>
```

**业务规则**: 需要组管理员或所有者权限。

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "invites": [
      {
        "id": 1,
        "group_uuid": "...",
        "invite_code": "ABCD1234EFGH5678",
        "inviter_uuid": "...",
        "invitee_uuid": "...",
        "type": 0,
        "status": 0,
        "expires_at": 1717604800,
        "created_at": 1717000000
      }
    ]
  }
}
```

**邀请类型 (`type`)**:
| 值 | 说明 |
|----|------|
| 0 | 搜索邀请（定向，有指定 `invitee_uuid`） |
| 1 | 链接邀请（公开，`invitee_uuid` 为空） |

**邀请状态 (`status`)**:
| 值 | 说明 |
|----|------|
| 0 | 待处理 |
| 1 | 已接受 |
| 2 | 已拒绝 |
| 3 | 已过期 |
