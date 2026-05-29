# 数据模型参考

## 用户 (User)

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_uuid` | string | 用户 UUID（唯一标识） |
| `user_name` | string | 用户名（唯一，最长50字符） |
| `nickname` | string | 昵称 |
| `avatar` | string | 头像路径 |
| `role` | int | 角色 (1=Normal, 2=Moderator, 3=Admin, 4=SuperAdmin) |
| `status` | int | 状态 |
| `description` | string | 个人描述 |
| `created_at` | int64 | 创建时间（Unix 时间戳） |
| `last_seen` | int64 | 最后在线时间 |
| `ip` | string | 最后登录 IP |

## 设备实例 (Instance)

| 字段 | 类型 | 说明 |
|------|------|------|
| `instance_uuid` | string | 设备实例 UUID（唯一标识） |
| `name` | string | 设备名称 |
| `type` | string | 设备类型名称 |
| `online` | bool | 是否在线 |
| `owner_uuid` | string | 所有者 UUID |
| `description` | string | 设备描述 |
| `status` | string | 状态（默认 `"active"`） |
| `properties` | object | 属性数据 |
| `created_at` | time.Time | 创建时间 |
| `last_seen` | int64 | 最后在线时间 |
| `verify_hash` | string | 验证哈希 |
| `sn` | string | 序列号 |
| `remark` | string | 备注 |

## 设备分享 (DeviceShare)

| 字段 | 类型 | 说明 |
|------|------|------|
| `instance_uuid` | string | 设备 UUID |
| `shared_with_uuid` | string | 被分享者 UUID |
| `shared_by_uuid` | string | 分享者 UUID |
| `permission` | string | 权限: `read` / `write` / `read_write` |
| `status` | string | 状态: `active` / `revoked` |
| `expires_at` | *int64 | 过期时间（nil 表示永不过期） |

## 用户组 (UserGroup)

| 字段 | 类型 | 说明 |
|------|------|------|
| `group_uuid` | string | 组 UUID（唯一标识） |
| `name` | string | 组名 |
| `description` | string | 组描述 |
| `owner_uuid` | string | 所有者 UUID |
| `max_members` | int | 最大成员数（0=无限制） |
| `status` | int | 状态: 0=活跃, 1=已解散 |
| `created_at` | int64 | 创建时间 |
| `updated_at` | int64 | 更新时间 |

## 组成员 (GroupMember)

| 字段 | 类型 | 说明 |
|------|------|------|
| `group_uuid` | string | 组 UUID |
| `user_uuid` | string | 用户 UUID |
| `role` | int | 角色: 0=成员, 1=管理员, 2=所有者 |
| `status` | int | 状态: 0=活跃, 1=退出, 2=踢出, 3=待审批 |
| `joined_at` | int64 | 加入时间 |
| `invited_by` | string | 邀请人 UUID |

## 组策略 (GroupPolicy)

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `device_visibility` | int | 1 | 0=全部可见, 1=仅已分享, 2=选择性 |
| `admin_device_access` | int | 0 | 0=只读, 1=完全访问 |
| `allow_search_invite` | bool | true | 是否允许定向邀请 |
| `allow_invite_link` | bool | true | 是否允许邀请链接 |
| `allow_member_invite` | bool | false | 是否允许普通成员邀请 |
| `require_approval` | bool | false | 是否需要审批 |
| `approval_mode` | int | 0 | 0=仅管理员, 1=任何成员 |

## 组邀请 (GroupInvite)

| 字段 | 类型 | 说明 |
|------|------|------|
| `group_uuid` | string | 组 UUID |
| `invite_code` | string | 16位邀请码（唯一） |
| `inviter_uuid` | string | 邀请人 UUID |
| `invitee_uuid` | string | 被邀请人 UUID（链接邀请为空） |
| `type` | int | 类型: 0=搜索邀请, 1=链接邀请 |
| `status` | int | 状态: 0=待处理, 1=已接受, 2=已拒绝, 3=已过期 |
| `expires_at` | int64 | 过期时间 |

## 管理操作日志 (AdminLog)

| 字段 | 类型 | 说明 |
|------|------|------|
| `admin_uuid` | string | 操作管理员 UUID |
| `action` | string | 操作类型（如 `"user.delete"`） |
| `target_type` | string | 目标类型: `user` / `device` / `group` |
| `target_uuid` | string | 目标 UUID |
| `detail` | string | 操作详情（JSON 字符串） |
| `ip` | string | 操作 IP |
| `created_at` | int64 | 操作时间 |
