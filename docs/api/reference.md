# 接口总览与快速参考

## 接口总览

| 方法 | 路径 | 认证 | 权限 | 说明 |
|------|------|------|------|------|
| `GET` | `/api/v1/health` | ❌ | — | 健康检查 |
| `GET` | `/api/v1/test` | ❌ | — | 测试端点 |
| `POST` | `/api/v1/users/register` | ❌ | — | 用户注册 |
| `POST` | `/api/v1/users/login` | ❌ | — | 用户登录 |
| `POST` | `/api/v1/device/deviceRegisterAnon` | ❌ | — | 设备匿名注册 |
| `POST` | `/api/v1/admin/login` | ❌ | — | 管理员登录 |
| `POST` | `/api/v1/users/logout` | ✅ | — | 用户登出 |
| `GET` | `/api/v1/users/info` | ✅ | — | 获取用户信息 |
| `GET` | `/api/v1/users/getUserAllDevices` | ✅ | — | 获取用户所有设备 |
| `PUT` | `/api/v1/users/profile` | ✅ | — | 更新用户资料 |
| `POST` | `/api/v1/users/avatar` | ✅ | — | 上传头像 |
| `DELETE` | `/api/v1/users/avatar` | ✅ | — | 重置头像 |
| `POST` | `/api/v1/users/addDevice` | ✅ | — | 创建设备 |
| `POST` | `/api/v1/users/bindDeviceByRegCode` | ✅ | — | 绑定设备 |
| `GET` | `/api/v1/devices/accessible` | ✅ | — | 可访问设备列表 |
| `POST` | `/api/v1/devices/{uuid}/getHistoryData` | ✅ | read | 历史数据 |
| `POST` | `/api/v1/devices/{uuid}/actions` | ✅ | write | 发送指令 |
| `POST` | `/api/v1/devices/{uuid}/share` | ✅ | write | 分享设备 |
| `POST` | `/api/v1/devices/groups/create_group` | ✅ | — | 创建设备组 |
| `POST` | `/api/v1/devices/{uuid}/join_group` | ✅ | — | 设备加入组 |
| `POST` | `/api/v1/devices/{uuid}/quit_group` | ✅ | — | 设备退出组 |
| `GET` | `/api/v1/devices/groups/{uuid}/members` | ✅ | — | 设备组成员 |
| `POST` | `/api/v1/devices/groups/{uuid}/dismiss_group` | ✅ | — | 解散设备组 |
| `GET` | `/api/v1/users/me/device_groups` | ✅ | — | 我的设备组 |
| `POST` | `/api/v1/groups` | ✅ | — | 创建用户组 |
| `GET` | `/api/v1/groups` | ✅ | — | 我的用户组列表 |
| `GET` | `/api/v1/groups/{uuid}` | ✅ | — | 用户组详情 |
| `PUT` | `/api/v1/groups/{uuid}` | ✅ | — | 更新用户组 |
| `DELETE` | `/api/v1/groups/{uuid}` | ✅ | — | 解散用户组 |
| `GET` | `/api/v1/groups/{uuid}/members` | ✅ | — | 用户组成员列表 |
| `POST` | `/api/v1/groups/{uuid}/invite/search` | ✅ | — | 定向邀请 |
| `POST` | `/api/v1/groups/{uuid}/invite/link` | ✅ | — | 创建邀请链接 |
| `POST` | `/api/v1/groups/invite/{code}/accept` | ✅ | — | 接受邀请 |
| `POST` | `/api/v1/groups/{uuid}/members/{uid}/approve` | ✅ | — | 审批成员 |
| `POST` | `/api/v1/groups/{uuid}/members/{uid}/reject` | ✅ | — | 拒绝成员 |
| `DELETE` | `/api/v1/groups/{uuid}/members/{uid}` | ✅ | — | 移除成员 |
| `POST` | `/api/v1/groups/{uuid}/leave` | ✅ | — | 退出用户组 |
| `PUT` | `/api/v1/groups/{uuid}/members/{uid}/role` | ✅ | — | 更新成员角色 |
| `GET` | `/api/v1/groups/{uuid}/devices` | ✅ | — | 用户组设备列表 |
| `POST` | `/api/v1/groups/{uuid}/devices/share` | ✅ | — | 分享设备到组 |
| `DELETE` | `/api/v1/groups/{uuid}/devices/{uuid}` | ✅ | — | 撤销组设备分享 |
| `GET` | `/api/v1/groups/{uuid}/policy` | ✅ | — | 获取用户组策略 |
| `PUT` | `/api/v1/groups/{uuid}/policy` | ✅ | — | 更新用户组策略 |
| `GET` | `/api/v1/groups/{uuid}/invites` | ✅ | — | 待处理邀请列表 |
| `GET` | `/api/v1/ws` | ✅ | — | WebSocket 推送通道 |
| `GET` | `/api/v1/logs/device` | ✅ | — | 查询设备日志 |
| `POST` | `/api/v1/logs/device/upload` | ✅ | — | 上传设备日志 |
| `GET` | `/api/v1/logs/user` | ✅ | — | 查询用户操作日志 |
| `POST` | `/api/v1/admin/logout` | ✅ | admin | 管理员登出 |
| `GET` | `/api/v1/admin/admins` | ✅ | admin:view | 管理员列表 |
| `POST` | `/api/v1/admin/admins` | ✅ | admin:manage | 提升管理员 |
| `PUT` | `/api/v1/admin/admins/{uuid}` | ✅ | admin:manage | 更新管理员角色 |
| `DELETE` | `/api/v1/admin/admins/{uuid}` | ✅ | admin:manage | 撤销管理员 |
| `GET` | `/api/v1/admin/users` | ✅ | user:view | 用户列表 |
| `GET` | `/api/v1/admin/users/{uuid}` | ✅ | user:view | 用户详情 |
| `PUT` | `/api/v1/admin/users/{uuid}` | ✅ | user:edit | 编辑用户 |
| `PUT` | `/api/v1/admin/users/{uuid}/status` | ✅ | user:status | 更新用户状态 |
| `DELETE` | `/api/v1/admin/users/{uuid}` | ✅ | user:delete | 删除用户 |
| `POST` | `/api/v1/admin/users/{uuid}/reset-password` | ✅ | user:reset | 重置密码 |
| `GET` | `/api/v1/admin/devices` | ✅ | device:view | 设备列表 |
| `GET` | `/api/v1/admin/devices/{uuid}` | ✅ | device:view | 设备详情 |
| `PUT` | `/api/v1/admin/devices/{uuid}` | ✅ | device:edit | 编辑设备 |
| `DELETE` | `/api/v1/admin/devices/{uuid}` | ✅ | device:delete | 删除设备 |
| `POST` | `/api/v1/admin/devices/{uuid}/transfer` | ✅ | device:transfer | 转移设备 |
| `GET` | `/api/v1/admin/groups` | ✅ | group:view | 用户组列表 |
| `GET` | `/api/v1/admin/groups/{uuid}` | ✅ | group:view | 用户组详情 |
| `GET` | `/api/v1/admin/groups/{uuid}/members` | ✅ | group:view | 用户组成员 |
| `DELETE` | `/api/v1/admin/groups/{uuid}` | ✅ | group:manage | 解散用户组 |
| `DELETE` | `/api/v1/admin/groups/{uuid}/members/{uid}` | ✅ | group:manage | 移除组成员 |
| `GET` | `/api/v1/admin/stats/overview` | ✅ | system:stats | 系统统计 |
| `GET` | `/api/v1/admin/logs` | ✅ | system:logs | 管理操作日志 |

---

## 快速参考

### 设备绑定完整流程

```bash
# 1. 设备端：获取注册凭证
curl -X POST http://localhost:27015/api/v1/device/deviceRegisterAnon \
  -d 'device_type_id=1'

# 2. 用户端：登录获取 token
curl -X POST http://localhost:27015/api/v1/users/login \
  -d 'username=admin&password=123456'

# 3. 用户端：绑定设备
curl -X POST http://localhost:27015/api/v1/users/bindDeviceByRegCode \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reg_code":"A0WU@HG6","device_nick":"我的设备"}'

# 4. 设备端：开始上报数据 (MQTT)
# Topic: data/device/{uuid}/properties
# 设备收到 GO_ON 指令后激活
```

### 用户组协作流程

```bash
# 1. 创建用户组
curl -X POST http://localhost:27015/api/v1/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"研发团队","description":"研发部门设备管理组"}'

# 2. 生成邀请链接
curl -X POST http://localhost:27015/api/v1/groups/$GROUP_UUID/invite/link \
  -H "Authorization: Bearer $TOKEN"

# 3. 其他用户通过邀请码加入
curl -X POST http://localhost:27015/api/v1/groups/invite/$INVITE_CODE/accept \
  -H "Authorization: Bearer $OTHER_TOKEN"

# 4. 分享设备到组
curl -X POST http://localhost:27015/api/v1/groups/$GROUP_UUID/devices/share \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"instance_uuid":"$DEVICE_UUID","permission":"read_write"}'
```

### cURL 模板

```bash
BASE="http://localhost:27015/api/v1"
TOKEN="your_jwt_token"

# GET 请求
curl -H "Authorization: Bearer $TOKEN" $BASE/users/info

# POST (JSON)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"command":"reboot","params":{}}' \
  $BASE/devices/{uuid}/actions

# POST (Form)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -d 'name=设备名&device_type=1' \
  $BASE/users/addDevice

# PUT (JSON)
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"nickname":"新昵称"}' \
  $BASE/users/profile

# DELETE
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  $BASE/users/avatar
```
