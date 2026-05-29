# 公开接口（无需认证）

## 健康检查

```
GET /api/v1/health
```

**响应示例**:
```json
{"code": 200, "message": "OK", "data": {"status": "ok"}}
```

## 测试端点

```
GET /api/v1/test
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `msg` | string | 否 | 测试消息，默认 `"hello world"` |

**响应示例**:
```json
{"code": 200, "message": "OK", "data": {"message": "hello world"}}
```

## 用户注册

```
POST /api/v1/users/register
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | ✅ | 用户名（唯一，最长50字符） |
| `commitment` | string | ✅ | DH 承诺值（十六进制编码） |

**客户端计算流程**:
```javascript
// 1. 计算 SHA256(password) 作为指数
const passwordHash = sha256(password);  // 返回 ArrayBuffer
const exponent = bufferToBigInt(passwordHash);

// 2. 计算承诺值 A = g^exponent mod p
const commitment = modPow(g, exponent, p);

// 3. 发送承诺值的十六进制编码
const commitmentHex = commitment.toString(16);
```

**响应示例**:
```json
{
  "code": 200,
  "message": "User created successfully",
  "data": {
    "id": 1,
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "username": "admin",
    "role": 1,
    "created_at": 1717000000,
    "last_seen": 1717000000,
    "last_ip": "127.0.0.1"
  }
}
```

**错误响应**:
- `400` Invalid input — 参数校验失败
- `400` Username already taken — 用户名已存在
- `400` Invalid commitment format — 承诺值格式错误

## 用户登录挑战

```
POST /api/v1/users/challenge
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | ✅ | 用户名 |

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "nonce": "a1b2c3d4e5f6...",
    "p": "ffffffffffffffffc90fdaa2...",
    "g": "2"
  }
}
```

**说明**:
- `nonce`: 一次性随机数，有效期 60 秒
- `p`: DH 素数（2048-bit，RFC 3526 Group 14）
- `g`: DH 生成元（固定为 2）

**错误响应**:
- `400` Invalid input — 参数校验失败
- `400` User not found — 用户不存在

## 用户登录

```
POST /api/v1/users/login
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | ✅ | 用户名 |
| `proof` | string | ✅ | DH 证明值（十六进制编码） |

**客户端计算流程**:
```javascript
// 1. 从 challenge 响应获取 nonce, p, g
const nonce = hexToBigInt(response.data.nonce);
const p = hexToBigInt(response.data.p);
const g = hexToBigInt(response.data.g);

// 2. 计算承诺值（与注册时相同）
const passwordHash = sha256(password);
const exponent = bufferToBigInt(passwordHash);
const commitment = modPow(g, exponent, p);

// 3. 计算证明值 B = commitment^nonce mod p
const proof = modPow(commitment, nonce, p);

// 4. 发送证明值的十六进制编码
const proofHex = proof.toString(16);
```

**响应示例**:
```json
{
  "code": 200,
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "username": "admin",
      "role": 1
    }
  }
}
```

**错误响应**:
- `400` Invalid input — 参数校验失败
- `401` Invalid credentials — 证明值验证失败
- `401` Challenge expired, please request a new one — Nonce 已过期或已使用

## 设备匿名注册

```
POST /api/v1/device/deviceRegisterAnon
Content-Type: application/x-www-form-urlencoded
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `device_type_id` | int | ✅ | 设备类型 ID |

**响应示例**:
```json
{
  "code": 200,
  "message": "Device Registered successfully",
  "data": {
    "device": {
      "id": 1,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "reg_code": "A0WU@HG6",
      "type": 1,
      "expires_at": 1760858932,
      "verify_code": "a1b2c3d4"
    }
  }
}
```

**错误响应**:
- `400` Invalid or missing query parameter
- `400` Unsupported device type
- `500` Failed to generate verification code

## 管理员登录挑战

```
POST /api/v1/admin/challenge
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | ✅ | 用户名 |

**业务规则**: 用户角色必须 ≥ 2（Moderator 及以上）

**响应示例**:
```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "nonce": "a1b2c3d4e5f6...",
    "p": "ffffffffffffffffc90fdaa2...",
    "g": "2"
  }
}
```

**错误响应**:
- `400` Invalid input — 参数校验失败
- `400` Challenge failed — 挑战失败
- `403` Account is not an admin — 账号不是管理员
- `403` Account is disabled — 账号已禁用

## 管理员登录

```
POST /api/v1/admin/login
Content-Type: application/json
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | ✅ | 用户名 |
| `proof` | string | ✅ | DH 证明值（十六进制编码） |

**业务规则**: 用户角色必须 ≥ 2（Moderator 及以上）

**响应示例**:
```json
{
  "code": 200,
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "user_uuid": "550e8400-e29b-41d4-a716-446655440000",
      "username": "admin",
      "nickname": "超级管理员",
      "role": 4
    }
  }
}
```

**错误响应**:
- `400` Invalid input — 参数校验失败
- `401` Invalid credentials — 证明值验证失败
- `401` Challenge expired, please request a new one — Nonce 已过期或已使用
- `403` Account is not an admin — 账号不是管理员
- `403` Account is disabled — 账号已禁用
