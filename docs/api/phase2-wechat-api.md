# Phase2 WeChat API

## Base URL

`/api/v1`

## Authentication

- `/auth/public-register` — 无需认证
- `/wechat/login` — 无需认证
- `/wechat/bind` — 可选认证（已登录走 token，未登录走学号+密码）
- `/dev/register-or-login` — 仅 `APP_ENV=dev` 时启用
- `/dev/login-and-send-subscribe-check` — 仅 `APP_ENV=dev` 时启用

## Endpoints

### POST /api/v1/auth/public-register — 公开注册/激活

无需认证。

**请求体**：
```json
{
  "student_id": "2020001",
  "name": "张三",
  "code": "wx_auth_code_optional"
}
```

**逻辑**：
1. 按 `student_id` 查用户
2. 用户存在：校验 `name` 一致，否则拒绝
3. 用户不存在：自动创建学生账号（role=student），并默认挂到 `未绑定班级`
4. 若传 `code`：换取 openid 并绑定（冲突则拒绝）
5. 生成 JWT token 并返回

**成功响应**（200）：
```json
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "student_id": "2020001",
      "name": "张三",
      "role": 1,
      "class_id": 9999,
      "grade": "",
      "major": ""
    }
  }
}
```

**默认班级说明**：
- 新注册且库中不存在该学号时，后端会自动创建或复用默认班级 `未绑定班级`
- 该默认班级当前固定使用 `class_id = 9999`

**错误响应**：
| 状态码 | 响应体 | 说明 |
|--------|--------|------|
| 400 | `{"error":"missing student_id or name"}` | 缺少学号或姓名 |
| 400 | `{"error":"invalid authorization code"}` | 微信 code 无效 |
| 401 | `{"error":"student id and name do not match"}` | 同学号姓名不匹配 |
| 409 | `{"error":"this wechat account is already bound to another user"}` | openid 已被他人绑定 |
| 500 | `{"error":"public register failed"}` | 创建或查询用户失败 |

---

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
  "data": {
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
}
```

**错误响应**：
| 状态码 | 响应体 | 说明 |
|--------|--------|------|
| 400 | `{"error":"missing code"}` | 缺少 code |
| 400 | `{"error":"invalid authorization code"}` | code 无效或过期 |
| 404 | `{"error":"account not bound, please bind first"}` | openid 未绑定用户 |

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

> 注意：学号/密码分支要求目标用户已设置密码，否则会在 `PasswordHash == nil` 时直接返回 `401 incorrect student id or password`。

**成功响应**（200）：
```json
{
  "data": {
    "ok": true,
    "message": "bind success"
  }
}
```

**错误响应**：
| 状态码 | 响应体 | 说明 |
|--------|--------|------|
| 400 | `{"error":"missing code"}` | 缺少 code |
| 400 | `{"error":"invalid authorization code"}` | code 无效或过期 |
| 401 | `{"error":"incorrect student id or password"}` | 学号/密码错误 |
| 401 | `{"error":"please login first or provide student id and password"}` | 既无 token 也无学号密码 |
| 409 | `{"error":"this wechat account is already bound to another user"}` | openid 已被其他用户绑定 |
| 500 | `{"error":"bind failed"}` | 数据库错误 |

---

### POST /api/v1/dev/register-or-login — 开发环境快捷注册或登录

仅在 `APP_ENV=dev` 时可用。

**请求体**：
```json
{ "student_id": "2020001", "role": 1 }
```

**成功响应**（200）：
```json
{
  "data": {
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
}
```

**错误响应**：
| 状态码 | 响应体 | 说明 |
|--------|--------|------|
| 400 | `{"error":"missing student_id"}` | 缺少学号 |
| 400 | `{"error":"invalid role"}` | role 非法 |
| 403 | `{"error":"dev register-or-login is disabled"}` | 非开发环境 |
| 500 | `{"error":"dev register-or-login failed"}` | 创建或签发 token 失败 |

**行为说明**：
1. `student_id` 已存在时，直接签发 token
2. `student_id` 不存在时，自动创建测试用户并签发 token
3. `role` 可选，默认学生角色

---

### POST /api/v1/dev/login-and-send-subscribe-check — 开发环境登录并验证订阅发送

仅在 `APP_ENV=dev` 时可用。

**用途**：一条请求串联验证“注册/登录 + 订阅状态记录 + 订阅消息发送”。

**请求体**：
```json
{
  "student_id": "2020001",
  "role": 1,
  "template_code": "dev_login_check",
  "wechat_template_id": "tmpl_dev_check",
  "status": "accept",
  "open_id": "dev-openid-2020001",
  "page": "/pages/index/index",
  "template_data": {
    "thing1": { "value": "Dev登录订阅验证" }
  }
}
```

**字段说明**：
- `student_id`：必填
- `role`：可选，默认学生角色
- `template_code`：可选，默认 `dev_login_check`
- `wechat_template_id`：可选，默认 `tmpl_dev_login_check`
- `status`：可选，`accept` / `reject`，默认 `accept`
- `open_id`：可选；若用户无 openid 且未提供，将自动生成 `dev-openid-<student_id>`
- `page` / `template_data`：可选

**成功响应**（200）：
```json
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "student_id": "2020001"
    },
    "template_code": "dev_login_check",
    "subscription_status": "subscribed",
    "granted_count": 1,
    "consumed_count": 0,
    "remaining_count": 1,
    "send_ok": true,
    "send_error": ""
  }
}
```

**错误响应**：
| 状态码 | 响应体 | 说明 |
|--------|--------|------|
| 400 | `{"error":"missing student_id"}` | 缺少学号 |
| 400 | `{"error":"invalid role"}` | role 非法 |
| 400 | `{"error":"status must be accept or reject"}` | status 非法 |
| 403 | `{"error":"dev login-and-send-subscribe-check is disabled"}` | 非开发环境 |
| 500 | `{"error":"...failed"}` | 模板创建/订阅记录/发送失败 |
| 500 | `{"error":"notification service unavailable"}` | 通知服务未初始化 |
| 500 | `{"error":"create dev notification template failed"}` | 模板自动创建失败 |

若 `status` 为 `reject`，接口只更新订阅状态为 `unsubscribed`，并返回 `send_ok: false` 以及 `send_error: "subscription status is reject"`，不会尝试发送模板消息。
若 `status` 为 `accept`，接口会将对应模板的订阅可用次数累加 1；只有实际发送订阅消息成功后，后端才会消耗 1 次。
