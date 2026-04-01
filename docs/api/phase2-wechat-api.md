# Phase2 WeChat API

## Base URL

`/api/v1`

## Authentication

- `/wechat/login` — 无需认证
- `/wechat/bind` — 可选认证（已登录走 token，未登录走学号+密码）

## Endpoints

### POST /api/v1/wechat/login — 微信登录

无需认证。

**请求体**：
```json
{ "code": "wx_auth_code" }
```

**逻辑**：
1. 调用微信 `code2Session` 换取 openid
2. 查询已绑定该 openid 的用户
3. 生成 JWT token
4. 返回 token + 用户信息

**成功响应**（200）：
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "student_id": "2020001",
    "name": "张三",
    "role": 1,
    "class_id": 10,
    "grade": "2020",
    "major": "计算机科学与技术"
  }
}
```

**错误响应**：
| 状态码 | 响应体 | 说明 |
|--------|--------|------|
| 400 | `{"error":"missing code"}` | 缺少 code |
| 400 | `{"error":"微信授权码无效"}` | code 无效或过期 |
| 404 | `{"error":"未绑定账号，请先绑定"}` | openid 未绑定用户 |

---

### POST /api/v1/wechat/bind — 绑定 OpenID

**场景1：已登录（带 JWT token）**

请求头：`Authorization: Bearer <token>`

请求体：
```json
{ "code": "wx_auth_code" }
```

**场景2：未登录（学号+密码验证）**

请求体：
```json
{
  "student_id": "2020001",
  "password": "mypassword",
  "code": "wx_auth_code"
}
```

**逻辑**：
1. 调用微信 `code2Session` 换取 openid
2. 判断认证方式：
   - 有 JWT token：从 token 获取用户 ID
   - 无 token：验证学号+密码匹配用户
3. 检查 openid 是否已被其他用户绑定
4. 绑定 openid 到当前用户
5. 若用户首次设置密码，存储密码哈希

**成功响应**（200）：
```json
{
  "data": {
    "ok": true,
    "message": "绑定成功"
  }
}
```

**错误响应**：
| 状态码 | 响应体 | 说明 |
|--------|--------|------|
| 400 | `{"error":"missing code"}` | 缺少 code |
| 400 | `{"error":"微信授权码无效"}` | code 无效或过期 |
| 401 | `{"error":"学号或密码不正确"}` | 学号/密码错误 |
| 401 | `{"error":"请登录后绑定或提供学号和密码"}` | 既无 token 也无学号密码 |
| 409 | `{"error":"该微信已绑定其他账号"}` | openid 已被其他用户绑定 |
| 500 | `{"error":"绑定失败"}` | 数据库错误 |
