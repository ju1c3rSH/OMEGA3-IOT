# OMEGA3-IOT API 规范

## 基础约定

- **Base URL**: `/api/v1`
- **认证**: `Authorization: Bearer <token>`
- **响应格式**:
```json
{
  "code": 200,
  "message": "success",
  "data": { }
}
```

## 错误码

| Code | 场景 | HTTP Status |
|------|------|-------------|
| 200 | 成功 | 200 |
| 201 | 创建成功 | 201 |
| 400 | 参数错误 | 400 |
| 401 | 未认证 | 401 |
| 403 | 权限不足 | 403 |
| 404 | 资源不存在 | 404 |
| 422 | 业务约束 | 422 |
| 429 | 限流 | 429 |
| 500 | 服务器错误 | 500 |

---

## 公开接口

### 健康检查
```
GET /api/v1/health
```

### 用户注册
```
POST /api/v1/users/register
Content-Type: application/x-www-form-urlencoded

Body: username=admin&password=123456
```

### 用户登录
```
POST /api/v1/users/login
Content-Type: application/x-www-form-urlencoded

Body: username=admin&password=123456
Response: {"access_token": "...", "user": {...}}
```

### 设备匿名注册
```
POST /api/v1/device/deviceRegisterAnon
Content-Type: application/x-www-form-urlencoded

Body: device_type_id=1
Response: {
  "device": {
    "id": 1,
    "uuid": "...",
    "reg_code": "A0WU@HG6",
    "type": 1,
    "expires_at": 1760858932,
    "verify_code": "..."
  }
}
```

---

## 认证接口 (需 JWT)

### 用户信息
```
GET /api/v1/users/info
```

### 我的设备
```
GET /api/v1/users/getUserAllDevices
```

### 绑定设备
```
POST /api/v1/users/bindDeviceByRegCode
Body: {
  "reg_code": "A0WU@HG6",
  "device_nick": "客厅传感器",
  "device_remark": ""
}
```

### 创建设备
```
POST /api/v1/users/addDevice
Content-Type: application/x-www-form-urlencoded

Body: name=测试设备&device_type=1&description=
```

---

## 设备接口

### 可访问设备列表
```
GET /api/v1/devices/accessible
Response: {
  "devices": [
    {
      "instance_uuid": "...",
      "name": "...",
      "type": "BaseTracker",
      "online": true,
      "permission": "read_write",
      "shared_by": null
    }
  ]
}
```

### 发送指令
```
POST /api/v1/devices/{instance_uuid}/actions
Permission: write
Body: {
  "command": "reboot",
  "params": {"interval_sec": 30}
}
```

### 分享设备
```
POST /api/v1/devices/{instance_uuid}/share
Permission: write (仅所有者)
Body: {
  "shared_with_uuid": "...",
  "permission": "read",
  "expires_at": 1760945332
}
```

### 历史数据
```
POST /api/v1/devices/{instance_uuid}/getHistoryData
Permission: read
Body: {
  "start_timestamp": 1704067200,
  "end_timestamp": 1704153600,
  "properties": ["battery_level"],
  "limit": 1000,
  "offset": 0
}
Response: {
  "instance_uuid": "...",
  "total_count": 2580,
  "returned_count": 1000,
  "has_more": true,
  "records": [...]
}
```

---

## 设备组接口

### 创建组
```
POST /api/v1/devices/groups/create_group
Body: {
  "name": "客厅设备",
  "description": "客厅所有传感器"
}
Response: {
  "id": 1,
  "name": "客厅设备",
  "owner_id": 1,
  "description": "客厅所有传感器",
  "created_at": "...",
  "updated_at": "...",
  "valid": 1
}
```

### 设备加入组
```
POST /api/v1/devices/{instance_uuid}/join_group
Body: {
  "group_id": 1
}
```

### 设备退出组
```
POST /api/v1/devices/{instance_uuid}/quit_group
Body: {
  "group_id": 1
}
```

### 获取用户的所有组
```
GET /api/v1/users/me/device_groups?page=1&page_size=10
Response: {
  "groups": [...],
  "total": 10,
  "page": 1,
  "page_size": 10
}
```

### 获取组成员
```
GET /api/v1/devices/groups/{group_id}/members?page=1&page_size=10
Response: {
  "members": [...],
  "total": 5,
  "page": 1,
  "page_size": 10
}
```

### 解散组
```
POST /api/v1/devices/groups/{group_id}/dismiss_group
Response: {
  "group_id": 1
}
```

---

## 日志接口

### 查询设备日志
```
GET /api/v1/logs/device?device_uuid={uuid}&start_time={unix_ts}&end_time={unix_ts}&limit=100&offset=0
```

### 上传设备日志
```
POST /api/v1/logs/device/upload
Body: {
  "device_uuid": "...",
  "level": "info",
  "message": "设备启动完成",
  "event_type": "device_log_upload",
  "metadata": {...},
  "action_name": "reboot",
  "result": "success"
}
```

### 查询用户日志
```
GET /api/v1/logs/user?user_uuid={uuid}&start_time={unix_ts}&end_time={unix_ts}&limit=100&offset=0
```

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
  -d '{"reg_code":"A0WU@HG6","device_nick":"我的设备"}'

# 4. 设备端：开始上报数据 (MQTT)
# Topic: data/device/{uuid}/properties
# 设备收到 GO_ON 指令后激活
```

### cURL 模板

```bash
BASE="http://localhost:27015/api/v1"
TOKEN="your_jwt_token"

# GET
curl -H "Authorization: Bearer $TOKEN" $BASE/users/info

# POST (JSON)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' $BASE/devices/{uuid}/actions

# POST (Form)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -d 'name=设备名&device_type=1' $BASE/users/addDevice
```
