# 基础约定

## Base URL

```
/api/v1
```

## 认证方式

### 密码认证（DH Challenge-Response）

本系统使用基于离散对数的 Challenge-Response 机制进行密码认证，密码永远不会以明文形式传输。

**公开参数**:
- `p`: 2048-bit 素数（RFC 3526 Group 14）
- `g`: 生成元（固定为 2）

**注册流程**:
1. 客户端计算 `commitment = g^SHA256(password) mod p`
2. 客户端发送 `commitment`（十六进制编码）到服务端
3. 服务端存储 `commitment`

**登录流程**:
1. 客户端请求 `POST /api/v1/users/challenge` 获取 `nonce`
2. 客户端计算 `proof = commitment^nonce mod p`
3. 客户端发送 `proof`（十六进制编码）到 `POST /api/v1/users/login`
4. 服务端验证 `proof == stored_commitment^nonce mod p`

**安全特性**:
- 密码永不传输（网络上只出现 `commitment^nonce mod p`）
- 防重放：Nonce 一次性使用，60 秒过期
- 防彩虹表：每次登录 `proof` 不同（Nonce 不同）
- 数据库泄漏保护：`commitment` 受离散对数问题保护

### JWT Token

JWT Token 支持三种传递方式（按优先级）：
1. `Authorization` Cookie
2. `Authorization: Bearer <token>` Header
3. `?token=<token>` Query 参数

Token 使用 HS256 签名，默认有效期 24 小时。登出时 Token 会被加入 Redis 黑名单。

## 响应格式

**成功响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {},
  "error_details": "",
  "timestamp": 1717000000,
  "traceId": ""
}
```

**错误响应**:
```json
{
  "code": 400,
  "message": "错误描述",
  "error_details": "详细错误信息",
  "timestamp": 1717000000
}
```

## 全局限流

所有 `/api/v1` 路由默认应用限流中间件：每个 IP 每 60 秒最多 15 次请求。超限返回 HTTP 429。

---

## 错误码

| Code | 场景 | HTTP Status |
|------|------|-------------|
| 200 | 成功 | 200 |
| 201 | 创建成功 | 201 |
| 400 | 参数错误 | 400 |
| 401 | 未认证 | 401 |
| 403 | 权限不足 | 403 |
| 404 | 资源不存在 | 404 |
| 409 | 冲突（如已是成员） | 409 |
| 422 | 业务约束 | 422 |
| 429 | 限流 | 429 |
| 500 | 服务器错误 | 500 |

---

## 角色与权限

### 角色定义

| 值 | 名称 | 说明 |
|----|------|------|
| 1 | Normal | 普通注册用户（默认） |
| 2 | Moderator | 只读管理员 |
| 3 | Admin | 管理员 |
| 4 | SuperAdmin | 超级管理员 |

### 权限矩阵

| 权限标识 | 说明 | Moderator | Admin | SuperAdmin |
|---------|------|-----------|-------|------------|
| `user:view` | 查看用户列表/详情 | ✅ | ✅ | ✅ |
| `user:edit` | 编辑用户资料 | ❌ | ✅ | ✅ |
| `user:status` | 修改用户状态 | ❌ | ✅ | ✅ |
| `user:delete` | 删除用户 | ❌ | ❌ | ✅ |
| `user:reset` | 重置用户密码 | ❌ | ✅ | ✅ |
| `device:view` | 查看设备列表/详情 | ✅ | ✅ | ✅ |
| `device:edit` | 编辑设备信息 | ❌ | ✅ | ✅ |
| `device:delete` | 删除设备 | ❌ | ✅ | ✅ |
| `device:transfer` | 转移设备所有权 | ❌ | ✅ | ✅ |
| `group:view` | 查看用户组列表/详情/成员 | ✅ | ✅ | ✅ |
| `group:manage` | 管理用户组（解散/移除成员） | ❌ | ✅ | ✅ |
| `admin:view` | 查看管理员列表 | ❌ | ❌ | ✅ |
| `admin:manage` | 管理管理员（提升/降级） | ❌ | ❌ | ✅ |
| `system:stats` | 查看系统统计 | ✅ | ✅ | ✅ |
| `system:logs` | 查看管理操作日志 | ✅ | ✅ | ✅ |

---

## 中间件链

请求经过以下中间件处理（按顺序）：

1. **CORS** — 允许跨域请求（开发环境 AllowOrigins: `*`）
2. **RateLimiter** — 每 IP 每 60 秒最多 15 次请求
3. **JwtAuthMiddleWare** — 验证 JWT Token，提取 `user_uuid`、`username`、`role` 写入上下文
4. **AdminAuthMiddleware** — 验证用户角色 ≥ 2（Moderator 及以上）
5. **RequirePermission** — 验证用户角色拥有指定权限
6. **DeviceAccessMiddleware** — 验证用户对指定设备的访问权限（read/write/read_write）

### 设备访问权限

| 权限 | 说明 | 使用场景 |
|------|------|----------|
| `read` | 只读 | 查询历史数据 |
| `write` | 只写 | 发送指令、分享设备 |
| `read_write` | 读写 | 完全访问 |
