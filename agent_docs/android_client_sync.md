# OMEGA3-IOT Android 客户端同步指南

> 本文档用于指导 Android 端 Claude Code Agent 实现与服务端的 API 和 Model 同步。
> 服务端已完成 Spec 模型重构、设备在线状态管理、用户头像和昵称功能。

---

## 一、用户模型变更

### 1.1 User 数据模型

服务端 `User` 模型新增了两个字段。Android 端的 User data class 需要同步：

```kotlin
data class User(
    val id: Long,
    @SerializedName("user_uuid") val userUuid: String,
    @SerializedName("user_name") val userName: String,
    val nickname: String?,          // 新增：用户自定义昵称，可为空
    val avatar: String?,            // 新增：头像 URL 路径，如 "/uploads/avatars/{uuid}.png"
    val type: Int,
    val online: Boolean,
    val description: String?,
    @SerializedName("last_seen") val lastSeen: Long,
    val ip: String?,
    val role: Int,
    val status: Int,
    @SerializedName("created_at") val createdAt: Long,
    @SerializedName("updated_at") val updatedAt: Long
)
```

**关键规则**：
- `nickname` 为 `null` 或空字符串时，UI 应回退显示 `userName`
- `avatar` 为相对路径（如 `/uploads/avatars/abc123.png`），需要拼接 Base URL 才能请求
- `avatar` 为 `null` 或空时，UI 应显示默认占位图

---

## 二、API 变更

### 2.1 修改的端点

#### GET /api/v1/users/info

**变更**：响应新增 `nickname`、`avatar_url`、`description` 字段。

```json
// 响应
{
  "code": 200,
  "message": "User info retrieved successfully",
  "data": {
    "id": 1,
    "uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "username": "admin",
    "nickname": "管理员",            // 新增
    "avatar_url": "/uploads/avatars/xxx.png",  // 新增
    "description": "",               // 新增
    "role": 1,
    "created_at": 1704067200,
    "last_seen": 1704153600,
    "last_ip": "192.168.1.1"
  }
}
```

#### POST /api/v1/users/register

**变更**：响应新增 `nickname`、`avatar_url` 字段（注册时 nickname 为空，avatar 为默认 identicon）。

#### POST /api/v1/users/login

**变更**：响应中的 `user` 对象新增 `nickname`、`avatar_url` 字段。

### 2.2 新增端点

#### PUT /api/v1/users/profile — 更新个人资料

```
PUT /api/v1/users/profile
Authorization: Bearer <token>
Content-Type: application/json

Body:
{
  "nickname": "新昵称",        // 可选，传 null 或不传则不更新
  "description": "个人简介"    // 可选，传 null 或不传则不更新
}

Response:
{
  "code": 200,
  "message": "Profile updated successfully",
  "data": null
}
```

**约束**：
- `nickname` 最大 50 字符
- `description` 最大长度不限（text 类型）
- 两个字段都是可选的，至少传一个

#### POST /api/v1/users/avatar — 上传头像

```
POST /api/v1/users/avatar
Authorization: Bearer <token>
Content-Type: multipart/form-data

Form Field:
  avatar: <image file>    // 支持 JPEG/PNG/GIF/BMP，服务端自动缩放到 256x256 PNG

Response:
{
  "code": 200,
  "message": "Avatar uploaded successfully",
  "data": {
    "avatar_url": "/uploads/avatars/xxx.png"
  }
}
```

**约束**：
- 图片会被裁剪为正方形（居中裁剪）后缩放到 256x256
- 存储为 PNG 格式
- 文件大小建议限制在 5MB 以内

#### DELETE /api/v1/users/avatar — 重置头像

```
DELETE /api/v1/users/avatar
Authorization: Bearer <token>

Response:
{
  "code": 200,
  "message": "Avatar reset to default",
  "data": {
    "avatar_url": "/uploads/avatars/xxx.png"
  }
}
```

将头像重置为基于 UUID 生成的 GitHub 风格 identicon。

### 2.3 新增静态文件端点

```
GET /uploads/avatars/{filename}

示例：
GET http://your-server:27015/uploads/avatars/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.png
```

直接返回图片文件，无需认证。

---

## 三、设备模型变更

### 3.1 设备在线状态

服务端已实现完整的在线/离线管理：

| 条件 | 行为 |
|------|------|
| 设备通过 MQTT 上报数据 | 自动标记 `online = true` |
| 设备发送 `shutdown` 或 `offline` 事件 | 立即标记 `online = false` |
| 设备超过 5 分钟无消息 | 自动标记 `online = false` |

**Android 端需要做的**：
- 设备列表中的 `online` 字段现在是可信的，可以直接用于 UI 状态显示
- 建议在设备卡片上用绿色/灰色圆点表示在线/离线状态
- `last_seen` 字段可用于显示 "最后在线: X 分钟前"

### 3.2 Action 校验（行为变更）

服务端现在会校验 Action 的合法性：
- 不存在的 `command` 会返回 400 错误
- 缺少必填参数会返回 400 错误
- 参数值超出范围/不在枚举内会返回 400 错误

**Android 端需要做的**：
- 发送 Action 时需要处理 400 错误响应，向用户展示具体错误信息
- 建议在 UI 上根据设备类型只展示合法的 Action 按钮

### 3.3 属性校验（行为变更）

服务端现在会校验设备上报的属性值：
- 类型不匹配的属性会被跳过（不中断其他属性更新）
- 超出范围的值会被拒绝
- 不在枚举内的值会被拒绝

**Android 端需要做的**：
- 历史数据查询返回的数据现在更加可靠
- 可以信任 `online` 和 `last_seen` 字段的准确性

---

## 四、UI 重构指南

### 4.1 用户个人资料页

**布局建议**（Material 3 风格）：

```
┌─────────────────────────────────┐
│         [返回]  个人资料         │
├─────────────────────────────────┤
│                                 │
│        ┌──────────┐             │
│        │  头像 256 │             │
│        │  可点击   │             │
│        └──────────┘             │
│        点击更换头像              │
│                                 │
│  ┌───────────────────────────┐  │
│  │ 昵称          [管理员  >] │  │
│  ├───────────────────────────┤  │
│  │ 用户名        [admin]     │  │  ← 不可编辑
│  ├───────────────────────────┤  │
│  │ 个人简介      [...    >] │  │
│  └───────────────────────────┘  │
│                                 │
│  ┌───────────────────────────┐  │
│  │ 修改密码              >  │  │
│  ├───────────────────────────┤  │
│  │ 退出登录                  │  │
│  └───────────────────────────┘  │
└─────────────────────────────────┘
```

**交互逻辑**：
1. 点击头像 → 弹出底部弹窗：「拍照」「从相册选择」「重置为默认头像」
2. 选择图片后 → 裁剪（可选，服务端会自动裁剪居中）→ 上传 → 显示新头像
3. 点击昵称 → 跳转编辑页或弹出 Dialog → 保存调用 `PUT /users/profile`
4. 点击个人简介 → 跳转编辑页 → 保存调用 `PUT /users/profile`
5. 显示逻辑：`nickname` 为空时回退显示 `userName`

### 4.2 头像显示规则

```kotlin
// 在所有显示用户头像的地方统一使用此逻辑
fun loadAvatar(imageView: ImageView, avatarUrl: String?, userName: String) {
    if (avatarUrl.isNullOrBlank()) {
        // 显示默认占位图（首字母圆形头像或内置默认图）
        imageView.setImageResource(R.drawable.default_avatar)
    } else {
        val fullUrl = "${BASE_URL}$avatarUrl"  // 拼接完整 URL
        Glide.with(imageView.context)
            .load(fullUrl)
            .placeholder(R.drawable.default_avatar)
            .error(R.drawable.default_avatar)
            .circleCrop()
            .into(imageView)
    }
}
```

### 4.3 设备列表项

**布局建议**：

```
┌─────────────────────────────────┐
│ [设备图标]  客厅温度传感器        │
│             SmartSensor          │
│  ● 在线              最后在线: 刚刚 │
│  温度: 25.3°C  湿度: 62%        │
└─────────────────────────────────┘
```

**关键点**：
- 在线状态圆点：`online == true` 用绿色，`false` 用灰色
- `last_seen` 显示为相对时间：刚刚 / X 分钟前 / X 小时前 / 昨天
- 属性值只显示 required 和当前有值的属性

### 4.4 Action 发送页

**约束**：
- 根据设备类型的 `actions` 定义动态渲染表单
- 每个 `input_param` 渲染为对应的输入控件：
  - `type: "string"` → EditText
  - `type: "int"` / `"float"` → 数字输入 EditText，带范围提示
  - `type: "bool"` → Switch / Checkbox
  - `enum: [...]` → Spinner / Dropdown
- `required: true` 的参数标记星号
- 发送失败（400）时展示服务端返回的 `error` 信息

---

## 五、网络层更新清单

### 5.1 ApiService 接口新增

```kotlin
interface ApiService {
    // 已有端点（响应结构变更）
    @GET("/api/v1/users/info")
    suspend fun getUserInfo(): Response<StandardResponse<UserInfoResponse>>

    @POST("/api/v1/users/register")
    suspend fun register(@Body request: RegisterRequest): Response<StandardResponse<RegisterResponse>>

    @POST("/api/v1/users/login")
    suspend fun login(@Body request: LoginRequest): Response<StandardResponse<LoginResponse>>

    // 新增端点
    @PUT("/api/v1/users/profile")
    suspend fun updateProfile(@Body request: UpdateProfileRequest): Response<StandardResponse<Unit>>

    @Multipart
    @POST("/api/v1/users/avatar")
    suspend fun uploadAvatar(@Part avatar: MultipartBody.Part): Response<StandardResponse<AvatarResponse>>

    @DELETE("/api/v1/users/avatar")
    suspend fun resetAvatar(): Response<StandardResponse<AvatarResponse>>
}
```

### 5.2 Request/Response 数据类

```kotlin
// Request
data class UpdateProfileRequest(
    val nickname: String? = null,
    val description: String? = null
)

// Response
data class AvatarResponse(
    @SerializedName("avatar_url") val avatarUrl: String
)

data class UserInfoResponse(
    val id: Long,
    val uuid: String,
    val username: String,
    val nickname: String?,
    @SerializedName("avatar_url") val avatarUrl: String?,
    val description: String?,
    val role: Int,
    @SerializedName("created_at") val createdAt: Long,
    @SerializedName("last_seen") val lastSeen: Long,
    @SerializedName("last_ip") val lastIp: String?
)

data class LoginResponse(
    @SerializedName("access_token") val accessToken: String,
    val user: UserInfoResponse
)

data class RegisterResponse(
    val id: Long,
    val uuid: String,
    val username: String,
    val nickname: String?,
    @SerializedName("avatar_url") val avatarUrl: String?,
    val role: Int,
    @SerializedName("created_at") val createdAt: Long,
    @SerializedName("last_seen") val lastSeen: Long,
    @SerializedName("last_ip") val lastIp: String?
)
```

---

## 六、实现优先级

| 优先级 | 任务 | 说明 |
|:---:|------|------|
| P0 | User 模型同步 | 新增 nickname、avatar 字段 |
| P0 | GET /users/info 响应适配 | 解析新字段，头像 URL 拼接 |
| P1 | PUT /users/profile | 昵称/描述编辑功能 |
| P1 | POST /users/avatar | 头像上传（multipart） |
| P1 | DELETE /users/avatar | 重置默认头像 |
| P1 | 用户资料页 UI | 头像显示、昵称编辑 |
| P2 | 设备列表 online 状态 | 绿色/灰色圆点 |
| P2 | Action 400 错误处理 | 展示校验失败原因 |
| P3 | Action 动态表单 | 根据设备类型渲染参数表单 |

---

## 七、注意事项

1. **Base URL 拼接**：`avatar` 字段是相对路径，请求图片时需要拼接 `http://host:port`
2. **图片压缩**：客户端上传前建议压缩到 1MB 以内，减少上传时间
3. **缓存策略**：头像 URL 不变时可使用 Glide/Coil 磁盘缓存；上传新头像后 URL 会变化，自动刷新缓存
4. **Token 刷新**：当前服务端不支持 refresh token，token 过期后需重新登录
5. **错误码统一**：所有响应的 `code` 字段遵循 HTTP 状态码，`error` 字段包含具体错误信息
6. **昵称回退**：所有显示用户名的地方，优先使用 `nickname`，为空时回退到 `userName`

---

## 八、WebSocket 实时推送通道

### 8.1 连接方式

```
GET /api/v1/ws
Header: Authorization: Bearer <token>
Upgrade: websocket
```

使用 OkHttp + WebSocket 客户端库。连接时通过 Header 传递 JWT token。

### 8.2 消息格式 (JSON)

所有消息共享统一信封格式：

```json
{
  "type": "消息类型",
  "seq": 1001,
  "ts": 1716800000,
  "payload": { ... }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | String | 消息类型，决定路由 |
| `seq` | Long | 序列号，用于 ACK 匹配 |
| `ts` | Long | Unix 时间戳（秒） |
| `payload` | Object | 载荷，结构由 type 决定 |

### 8.3 服务端 → App 消息类型

#### device.status — 设备在线状态变更
```json
{
  "type": "device.status",
  "seq": 1,
  "ts": 1716800000,
  "payload": {
    "device_uuid": "xxx",
    "online": true,
    "last_seen": 1716800000
  }
}
```

#### event.push — 设备事件推送 (warning/critical 需要 ACK)
```json
{
  "type": "event.push",
  "seq": 2,
  "ts": 1716800000,
  "payload": {
    "event_key": "low_battery_alarm",
    "device_uuid": "xxx",
    "severity": "warning",
    "data": {"current_level": 15}
  }
}
```

#### property.update — 设备属性更新
```json
{
  "type": "property.update",
  "ts": 1716800000,
  "payload": {
    "device_uuid": "xxx",
    "properties": {
      "temperature": 25.3,
      "humidity": 62
    }
  }
}
```

#### action.result — 指令执行结果
```json
{
  "type": "action.result",
  "ts": 1716800000,
  "payload": {
    "device_uuid": "xxx",
    "command": "reboot",
    "success": true
  }
}
```

### 8.4 App → 服务端消息类型

#### ack — 确认收妥 (收到带 seq 的消息后发送)
```json
{"type": "ack", "payload": {"ack_seq": 2}}
```

#### action.send — 发送控制指令
```json
{
  "type": "action.send",
  "payload": {
    "device_uuid": "xxx",
    "command": "reboot",
    "params": {}
  }
}
```

#### ping — 心跳
```json
{"type": "ping"}
```
服务端回复: `{"type": "pong", "ts": ...}`

### 8.5 ACK 规则

- 收到带 `seq` 的消息后，10 秒内回传 `ack`
- 仅 `event.push` (severity = warning/critical) 需要 ACK
- `device.status` 和 `property.update` 不需要 ACK

### 8.6 Android 端实现建议

```kotlin
// WebSocket 管理器核心逻辑
class PushWebSocketManager(
    private val okHttpClient: OkHttpClient,
    private val baseUrl: String,
    private val tokenProvider: () -> String
) {
    private var webSocket: WebSocket? = null
    private val messageDispatcher = MessageDispatcher()
    private val reconnectHandler = ReconnectHandler()

    fun connect() {
        val request = Request.Builder()
            .url("${baseUrl.replace("http", "ws")}/api/v1/ws")
            .addHeader("Authorization", "Bearer ${tokenProvider()}")
            .build()
        webSocket = okHttpClient.newWebSocket(request, createListener())
    }

    fun send(message: String) {
        webSocket?.send(message)
    }

    // 消息分发器 — 按 type 路由到不同的 Handler
    class MessageDispatcher {
        private val handlers = mutableMapOf<String, (JSONObject) -> Unit>()

        fun register(type: String, handler: (JSONObject) -> Unit) {
            handlers[type] = handler
        }

        fun dispatch(message: String) {
            val json = JSONObject(message)
            val type = json.getString("type")
            handlers[type]?.invoke(json)
            // 自动 ACK (如果消息带 seq)
            if (json.has("seq") && type == "event.push") {
                send("""{"type":"ack","payload":{"ack_seq":${json.getLong("seq")}}}""")
            }
        }
    }
}
```

### 8.7 重连策略

- 断线后指数退避重连：1s → 2s → 4s → 8s → 16s → 30s (上限)
- 重连成功后重置退避
- App 进入后台时保持连接（Android 需要 Foreground Service）
- App 被杀死后，下次启动时自动重连
