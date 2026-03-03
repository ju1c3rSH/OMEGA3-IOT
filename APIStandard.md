# OMEGA3-IOT HTTP API 接口文档

## 公共规范

### 基础路径
所有 API 接口均位于 `/api/v1` 路径下

### 认证方式
- **公开接口**：无需认证
- **受保护接口**：需在请求头中携带 JWT Token  
  `Authorization: Bearer <access_token>`

### 响应格式
所有接口返回统一 JSON 格式：
```json
{
  "code": 200,
  "message": "success message",
  "data": { ... }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | HTTP 状态码（200/201/400/401/403/404/500） |
| `message` | string | 操作结果描述 |
| `data` | object | 业务数据（成功时）或错误详情（失败时） |

### 错误码规范
| HTTP 状态码 | 业务场景 | 说明 |
|------------|----------|------|
| 400 | 参数错误 | 请求参数缺失/格式错误/逻辑校验失败 |
| 401 | 未认证 | JWT Token 无效/过期/缺失 |
| 403 | 权限不足 | 无设备访问权限/角色权限不足 |
| 404 | 资源不存在 | 用户/设备/记录不存在 |
| 422 | 业务约束 | 时间范围超限/属性不存在等业务规则 |
| 429 | 请求过多 | 触发速率限制（15次/分钟） |

---

## 一、公开接口（无需认证）

### 1.1 系统健康检查
**路径**：`GET /api/v1/health`  
**描述**：检查服务健康状态

#### 请求示例
```bash
curl https://api.example.com/api/v1/health
```

#### 响应示例 (200)
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "status": "ok"
  }
}
```

---

### 1.2 连通性测试
**路径**：`GET /api/v1/test`  
**描述**：调试用连通性测试接口

#### 请求参数
| 参数 | 位置 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| `msg` | query | string | 否 | `"hello world"` | 测试消息内容 |

#### 请求示例
```bash
curl "https://api.example.com/api/v1/test?msg=ping"
```

#### 响应示例 (200)
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "ping"
  }
}
```

---

### 1.3 用户注册
**路径**：`POST /api/v1/users/register`  
**描述**：创建新用户账号

#### 请求体
```json
{
  "username": "iot_user",
  "password": "SecurePass123!"
}
```

| 字段 | 类型 | 必填 | 约束 | 说明 |
|------|------|------|------|------|
| `username` | string | 是 | 3-20字符，字母数字下划线 | 用户名 |
| `password` | string | 是 | ≥8字符，含大小写字母+数字 | 密码 |

#### 响应示例 (201)
```json
{
  "code": 201,
  "message": "User created successfully",
  "data": {
    "id": 101,
    "username": "iot_user",
    "role": "standard",
    "created_at": "2026-03-03T10:24:00Z",
    "last_seen": "2026-03-03T10:24:00Z",
    "last_ip": "192.168.1.100"
  }
}
```

#### 错误响应 (400)
```json
{
  "code": 400,
  "message": "Username already taken",
  "data": "duplicate key value violates unique constraint"
}
```

---

### 1.4 用户登录
**路径**：`POST /api/v1/users/login`  
**描述**：获取 JWT 访问令牌

#### 请求体
```json
{
  "username": "iot_user",
  "password": "SecurePass123!"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | 是 | 已注册用户名 |
| `password` | string | 是 | 用户密码 |

#### 响应示例 (200)
```json
{
  "code": 200,
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xxxxx",
    "user": {
      "id": 101,
      "username": "iot_user",
      "role": "standard"
    }
  }
}
```

#### 错误响应 (401)
```json
{
  "code": 401,
  "message": "Invalid username or password",
  "data": "authentication failed"
}
```

---

### 1.5 设备匿名注册
**路径**：`POST /api/v1/device/deviceRegisterAnon`  
**描述**：设备首次上电时获取注册凭证（由设备端调用）

#### 请求体
```json
{
  "device_type_id": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `device_type_id` | int | 是 | 设备类型ID（参考设备类型配置） |

#### 响应示例 (200)
```json
{
  "code": 200,
  "message": "Device Registered successfully",
  "data": {
    "device": {
      "id": 154,
      "uuid": "7b64cea8-ed24-4e73-b0a9-2af503bd4e69",
      "reg_code": "A0WU@HG6",
      "type": 1,
      "verify_code": "tOFX*mc8=V}?Cnh2",
      "expires_at": 1760858932
    }
  }
}
```

| 响应字段 | 说明 |
|----------|------|
| `uuid` | 设备唯一标识符（UUID v4） |
| `reg_code` | 8位注册码（用户App绑定时输入） |
| `verify_code` | 16位校验码（设备端保存，**仅首次返回明文**） |
| `expires_at` | Unix时间戳，24小时后过期 |

> ⚠️ **安全提示**：`verify_code` 仅在注册时返回一次明文，后续通信使用 SHA-256 哈希验证

---

## 二、受保护接口（需 JWT 认证）

### 2.1 获取用户信息
**路径**：`GET /api/v1/users/info`  
**权限**：已认证用户  
**描述**：获取当前用户详细信息

#### 请求头
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xxxxx
```

#### 响应示例 (200)
```json
{
  "code": 200,
  "message": "User info retrieved successfully",
  "data": {
    "id": 101,
    "username": "iot_user",
    "role": "standard",
    "created_at": "2026-03-03T10:24:00Z",
    "last_seen": "2026-03-03T14:30:00Z",
    "last_ip": "192.168.1.100"
  }
}
```

---

### 2.2 获取用户所有设备
**路径**：`GET /api/v1/users/getUserAllDevices`  
**权限**：已认证用户  
**描述**：获取当前用户拥有的所有设备列表

#### 请求头
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xxxxx
```

#### 响应示例 (200)
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "devices": [
      {
        "id": 154,
        "uuid": "7b64cea8-ed24-4e73-b0a9-2af503bd4e69",
        "name": "Living Room Tracker",
        "type": 1,
        "online": true,
        "description": "GPS定位器",
        "created_at": 1704873600,
        "last_seen": 1704873900,
        "properties": {
          "battery_level": {
            "value": "85",
            "meta": {
              "writable": true,
              "description": "电池电量",
              "unit": "%",
              "range": [0, 100],
              "format": "int"
            }
          },
          "gps_location": {
            "value": "39.9042,116.4074",
            "meta": {
              "writable": false,
              "description": "GPS位置",
              "format": "string"
            }
          }
        },
        "remark": "客厅窗台",
        "verify_hash": "a1b2c3d4e5f6..."  // SHA-256哈希值
      }
    ]
  }
}
```

---

### 2.3 添加设备（直接创建）
**路径**：`POST /api/v1/users/addDevice`  
**权限**：已认证用户  
**描述**：直接创建设备实例（无需 RegCode）

#### 请求体
```json
{
  "name": "Bedroom Sensor",
  "device_type": 2,
  "description": "温湿度传感器"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 设备名称（3-50字符） |
| `device_type` | int | 是 | 设备类型ID |
| `description` | string | 否 | 设备描述（≤200字符） |

#### 响应示例 (201)
```json
{
  "code": 201,
  "message": "Device created successfully",
  "data": {
    "id": 155,
    "uuid": "8c75dfb9-fe35-4f84-a1b0-3c5d6e7f8g9h",
    "name": "Bedroom Sensor",
    "type": 2,
    "online": false,
    "description": "温湿度传感器",
    "created_at": 1704874000,
    "last_seen": 1704874000,
    "properties": {},
    "remark": "",
    "verify_hash": "f7e6d5c4b3a2..." 
  }
}
```

---

### 2.4 通过注册码绑定设备
**路径**：`POST /api/v1/users/bindDeviceByRegCode`  
**权限**：已认证用户  
**描述**：将匿名注册的设备绑定到当前用户

#### 请求体
```json
{
  "reg_code": "A0WU@HG6",
  "device_nick": "Car Tracker",
  "device_remark": "安装在车辆OBD接口"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `reg_code` | string | 是 | 8位设备注册码 |
| `device_nick` | string | 是 | 设备昵称（3-50字符） |
| `device_remark` | string | 否 | 设备备注（≤200字符） |

#### 响应示例 (201)
```json
{
  "code": 201,
  "message": "Device created successfully",
  "data": {
    "device": {
      "id": 154,
      "uuid": "7b64cea8-ed24-4e73-b0a9-2af503bd4e69",
      "name": "Car Tracker",
      "type": 1,
      "online": false,
      "description": "GPS定位器",
      "created_at": 1704873600,
      "last_seen": 1704873600,
      "properties": {},
      "remark": "安装在车辆OBD接口",
      "verify_hash": "a1b2c3d4e5f6..." 
    }
  }
}
```

#### 错误响应 (400)
```json
{
  "code": 400,
  "message": "Invalid or expired reg_code",
  "data": "registration record not found or expired"
}
```

---

### 2.5 获取可访问设备列表
**路径**：`GET /api/v1/devices/accessible`  
**权限**：已认证用户  
**描述**：获取用户可访问的所有设备（自有设备 + 他人分享的设备）

#### 请求头
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xxxxx
```

#### 响应示例 (200)
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "devices": [
      {
        "instance_uuid": "7b64cea8-ed24-4e73-b0a9-2af503bd4e69",
        "name": "Car Tracker",
        "type": 1,
        "online": true,
        "permission": "read_write",  // 当前用户权限
        "shared_by": null            // null=自有设备，否则为分享者UUID
      },
      {
        "instance_uuid": "9d8e7f6a-5b4c-3d2e-1f0a-9b8c7d6e5f4a",
        "name": "Office Sensor",
        "type": 2,
        "online": false,
        "permission": "read",        // 只读权限
        "shared_by": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
      }
    ]
  }
}
```

---

### 2.6 发送设备控制指令
**路径**：`POST /api/v1/devices/{instance_uuid}/actions`  
**权限**：设备写权限（`write` 或 `read_write`）  
**描述**：向设备下发控制指令（通过 MQTT 透传）

#### 路径参数
| 参数 | 说明 |
|------|------|
| `instance_uuid` | 设备唯一标识符（UUID v4） |

#### 请求体
```json
{
  "command": "set_upload_interval",
  "params": {
    "interval_sec": 30
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `command` | string | 是 | 指令名称（由设备类型定义） |
| `params` | object | 否 | 指令参数（键值对） |

#### 响应示例 (200)
```json
{
  "code": 200,
  "message": "Action sent successfully",
  "data": {
    "instance_uuid": "7b64cea8-ed24-4e73-b0a9-2af503bd4e69",
    "command": "set_upload_interval"
  }
}
```

#### 错误响应 (403)
```json
{
  "code": 403,
  "message": "Access denied: insufficient permissions",
  "data": "required permission: write"
}
```

---

### 2.7 分享设备给其他用户
**路径**：`POST /api/v1/devices/{instance_uuid}/share`  
**权限**：设备写权限（设备所有者）  
**描述**：将设备访问权限分享给其他用户

#### 路径参数
| 参数 | 说明 |
|------|------|
| `instance_uuid` | 设备唯一标识符（UUID v4） |

#### 请求体
```json
{
  "shared_with_uuid": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
  "permission": "read",
  "expires_at": 1760945332
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `shared_with_uuid` | string | 是 | 被分享用户的 UUID |
| `permission` | string | 是 | 权限级别：`read`/`write`/`read_write` |
| `expires_at` | int64 | 否 | 过期时间（Unix时间戳），**空值表示永不过期** |

#### 响应示例 (201)
```json
{
  "code": 201,
  "message": "Device shared successfully",
  "data": {
    "shared_with": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
    "permission": "read",
    "expires_at": 1760945332
  }
}
```

---

### 2.8 获取设备历史数据
**路径**：`POST /api/v1/devices/{instance_uuid}/getHistoryData`  
**权限**：设备读权限（`read` 或 `read_write`）  
**描述**：查询设备历史属性数据（从 IoTDB 时序数据库）

#### 路径参数
| 参数 | 说明 |
|------|------|
| `instance_uuid` | 设备唯一标识符（UUID v4） |

#### 请求体
```json
{
  "start_timestamp": 1704067200,
  "end_timestamp": 1704153600,
  "properties": ["battery_level", "gps_location"],
  "limit": 1000,
  "offset": 0
}
```

| 字段 | 类型 | 必填 | 默认值 | 约束 | 说明 |
|------|------|------|--------|------|------|
| `start_timestamp` | int64 | 是 | - | - | 起始时间（Unix秒） |
| `end_timestamp` | int64 | 是 | - | > start_timestamp | 结束时间（Unix秒） |
| `properties` | []string | 否 | 全部属性 | 设备类型定义的属性名 | 需查询的属性列表 |
| `limit` | int | 否 | 1000 | 1~5000 | 单次返回最大记录数 |
| `offset` | int | 否 | 0 | ≥0 | 分页偏移量 |

#### 响应示例 (200)
```json
{
  "code": 200,
  "message": "Get device history data successfully",
  "data": {
    "instance_uuid": "7b64cea8-ed24-4e73-b0a9-2af503bd4e69",
    "total_count": 2580,
    "returned_count": 1000,
    "has_more": true,
    "records": [
      {
        "timestamp": 1704067200,
        "properties": {
          "battery_level": {
            "value": "85",
            "meta": {
              "writable": true,
              "description": "电池电量",
              "unit": "%",
              "range": [0, 100],
              "format": "int"
            }
          },
          "gps_location": {
            "value": "39.9042,116.4074",
            "meta": {
              "writable": false,
              "description": "GPS位置",
              "format": "string"
            }
          }
        }
      },
      {
        "timestamp": 1704067260,
        "properties": {
          "battery_level": {
            "value": "84",
            "meta": { ... }
          },
          "gps_location": {
            "value": "39.9045,116.4078",
            "meta": { ... }
          }
        }
      }
    ]
  }
}
```

#### 错误响应 (422)
```json
{
  "code": 422,
  "message": "Time range exceeds maximum allowed (30 days)",
  "data": {
    "max_range_seconds": 2592000
  }
}
```

---

## 三、安全与最佳实践

### 3.1 敏感数据处理
| 数据类型 | 处理方式 | 说明 |
|----------|----------|------|
| 密码 | bcrypt(cost=14) | 永不返回明文 |
| VerifyCode | 仅注册时返回明文 | 后续通信使用 SHA-256 哈希验证 |
| VerifyHash | 永不返回 | 仅服务端内部验证使用 |

### 3.2 客户端调用建议
1. **Token 刷新**：JWT 有效期 24 小时，建议在过期前 1 小时刷新
2. **分页查询**：历史数据查询使用 `offset` + `limit` 循环获取
3. **错误重试**：
    - 429 错误：等待 60 秒后重试
    - 5xx 错误：指数退避重试（1s → 2s → 4s → 8s）
4. **设备绑定流程**：
   ```mermaid
   sequenceDiagram
       participant App
       participant Server
       participant Device
       
       App->>Server: POST /deviceRegisterAnon
       Server-->>App: 返回 reg_code + verify_code
       App->>Device: 通过蓝牙/扫码传输 verify_code
       App->>Server: POST /bindDeviceByRegCode(reg_code)
       Server->>Device: MQTT下发 GO_ON 指令
       Device-->>Server: 开始正常上报数据
   ```

### 3.3 生产环境配置
```yaml
# config.yaml 关键配置
security:
  jwt_secret: "your_strong_secret_here"  # 32+ 字符
  cors_origins: 
    - "https://web.yourdomain.com"
    - "https://android.yourdomain.com"
  
rate_limit:
  requests: 15
  window: "1m"
  
mqtt:
  qos: 1  # 属性上报/指令下发使用 QoS 1
  tls: true  # 生产环境强制启用 TLS
```