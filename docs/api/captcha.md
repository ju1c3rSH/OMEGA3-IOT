# 验证码接口（无需认证）

验证码系统用于注册、登录等敏感操作的人机验证。采用动画 GIF + 问答形式。

## 获取验证码

```
GET /api/v1/captcha?theme=touhou&complexity=2
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `theme` | string | 否 | 主题，默认 "touhou" |
| `complexity` | int | 否 | 复杂度：0=Simple, 1=Easy, 2=Medium, 3=Hard，默认 2 |

**响应示例**:
```json
{
  "captcha_id": "550e8400-e29b-41d4-a716-446655440000",
  "image": "R0lGODlh...",
  "question_type": 1,
  "question_hint": "图中有几个红色圆形？",
  "options": [],
  "answer_mode": "input",
  "expires_in": 300,
  "sign": ""
}
```

**字段说明**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `captcha_id` | string | 验证码唯一标识 |
| `image` | string | base64 编码的动画 GIF |
| `question_type` | int | 问题类型：0=数数, 1=颜色, 2=角色, 3=位置 |
| `question_hint` | string | 问题提示文字 |
| `options` | string[] | 选项列表（select 模式时有值） |
| `answer_mode` | string | `"input"`（文本输入）或 `"select"`（选择按钮） |
| `expires_in` | int | 过期时间（秒），默认 300 |
| `sign` | string | HMAC 签名（可选） |

**问题类型详情**:

| question_type | 说明 | answer_mode | 示例 |
|---------------|------|-------------|------|
| 0 | 数数题 | input | "图中有几个红色圆形？" → 输入 "3" |
| 1 | 颜色题 | select | "选择图中出现的颜色" → 点击 "红色" |
| 2 | 角色题 | input | "图中角色的名字是？" → 输入 "Reimu" |
| 3 | 位置题 | select | "红色圆形在什么位置？" → 点击 "左上" |

## 验证验证码

```
POST /api/v1/captcha/verify
Content-Type: application/json
```

```json
{
  "captcha_id": "550e8400-e29b-41d4-a716-446655440000",
  "answer": "3",
  "sign": "",
  "timestamp": 0
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `captcha_id` | string | ✅ | 验证码 ID |
| `answer` | string | ✅ | 用户答案 |
| `sign` | string | 否 | HMAC 签名 |
| `timestamp` | int64 | 否 | 时间戳 |

**成功响应**:
```json
{
  "valid": true,
  "message": "验证成功"
}
```

**失败响应**:
```json
{
  "valid": false,
  "message": "验证码错误或已过期"
}
```

**注意**:
- 验证码是**一次性**的，验证后即失效
- 过期时间默认 5 分钟
- `sign` 和 `timestamp` 用于高级安全场景，通常可传空值

## JS SDK

```
GET /api/v1/captcha/sdk.js
```

提供前端 JS SDK，用于 Web 页面集成：

```javascript
EinkiCaptcha.init('#captcha-container', {
    baseUrl: 'https://your-server.com/api/v1',
    theme: 'touhou',
    complexity: 2,
    onVerify: function(success) {
        if (success) {
            // 验证通过，提交表单
        }
    }
});
```
