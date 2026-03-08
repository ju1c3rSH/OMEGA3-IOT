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
Body: {"username": "admin", "password": "123456"}
```

### 用户登录
```
POST /api/v1/users/login
Body: {"username": "admin", "password": "123456"}
Response: {"access_token": "...", "user": {...}}
```

### 设备匿名注册
```
POST /api/v1/device/deviceRegisterAnon
Body: {"device_type_id": 1}
Response: {
  "uuid": "...",
  "reg_code": "A0WU@HG6",
  "verify_code": "...",  // 仅返回一次
  "expires_at": 1760858932
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
Body: {
  "name": "测试设备",
  "device_type": 1,
  "description": ""
}
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
      "type": 1,
      "online": true,
      "permission": "read_write",
      "shared_by": null  // null=自有, uuid=被分享
    }
  ]
}
```

### 发送指令
```
POST /api/v1/devices/{instance_uuid}/actions
Permission: write
Body: {
  "command": "set_upload_interval",
  "params": {"interval_sec": 30}
}
```

### 分享设备
```
POST /api/v1/devices/{instance_uuid}/share
Permission: write (仅所有者)
Body: {
  "shared_with_uuid": "...",
  "permission": "read",  // read/write/read_write
  "expires_at": 1760945332  // null=永不过期
}
```

### 历史数据
```
POST /api/v1/devices/{instance_uuid}/getHistoryData
Permission: read
Body: {
  "start_timestamp": 1704067200,  // Unix秒
  "end_timestamp": 1704153600,    // 最大30天范围
  "properties": ["battery_level"], // null=全部
  "limit": 1000,                  // 1-5000
  "offset": 0
}
Response: {
  "instance_uuid": "...",
  "total_count": 2580,
  "returned_count": 1000,
  "has_more": true,
  "records": [
    {
      "timestamp": 1704067200,
      "properties": {
        "battery_level": {
          "value": "85",
          "meta": {...}
        }
      }
    }
  ]
}
```

---

## 快速参考

### 设备绑定完整流程

```bash
# 1. 设备端：获取注册凭证
curl -X POST http://localhost:27015/api/v1/device/deviceRegisterAnon \
  -d '{"device_type_id":1}'

# 2. 用户端：登录获取 token
curl -X POST http://localhost:27015/api/v1/users/login \
  -d '{"username":"admin","password":"123456"}'

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

# POST
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' $BASE/devices/{uuid}/actions
```
