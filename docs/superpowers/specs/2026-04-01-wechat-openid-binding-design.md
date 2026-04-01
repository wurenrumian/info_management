# 微信 OpenID 绑定与 JWT 认证设计

**日期**：2026-04-01  
**阶段**：Phase 2 前置基础设施升级

## 1. 背景与目标

Phase 1 基础底座已完成，`users` 表中 `openid` 字段已预留但未启用。当前认证方式为请求头注入（`X-User-Id` 等），仅适用于开发调试。

本设计目标：
- 实现微信 OpenID 绑定与登录能力
- 将认证方式升级为 JWT，为 Phase 2 并行开发提供统一标准
- 新增学号+密码验证，支持未登录场景下的身份绑定

## 2. 设计约束

- 严格遵循 Phase 1 分层结构（model/repo/service/http/auth）
- 不破坏现有 handler 的 `auth.GetActor(c)` 调用模式
- 所有 Phase 2 模块统一使用 JWT 认证
- 微信配置通过环境变量注入

## 3. 整体架构

### 3.1 认证流程

```
┌─────────────────────────────────────────────────────────┐
│  微信登录                                                │
│  小程序 wx.login() → code                                │
│  POST /api/v1/wechat/login { code }                      │
│  后端 code → openid → 查用户 → 生成 JWT → 返回 token     │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  绑定 OpenID（已登录）                                    │
│  Header: Authorization: Bearer <token>                   │
│  POST /api/v1/wechat/bind { code }                       │
│  后端 code → openid → 绑定到当前用户                      │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  绑定 OpenID（未登录）                                    │
│  POST /api/v1/wechat/bind { student_id, password, code } │
│  后端 验证学号+密码 → code → openid → 绑定               │
└─────────────────────────────────────────────────────────┘
```

### 3.2 分层结构

```
internal/model/user.go          — 新增 PasswordHash 字段
internal/service/wechat/        — 微信 API 调用（code→openid）
internal/service/auth/          — JWT 生成与验证
internal/http/middleware/auth.go — JWT 认证中间件（替换 IdentityFromHeaders）
internal/http/handler/wechat_handler.go — 绑定和登录接口
internal/http/router/router.go  — 注册 wechat 路由
```

## 4. 数据模型变更

### 4.1 users 表新增字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `password_hash` | varchar(255) NULL | 密码哈希（bcrypt），首次绑定后设置 |

说明：
- `openid` 字段已存在，本设计启用其绑定逻辑
- `password_hash` 为 NULL 表示用户尚未设置密码，只能通过管理员导入或首次绑定时设置

## 5. JWT 设计

### 5.1 Token 结构

```json
{
  "sub": 1,
  "role": 1,
  "class_id": 10,
  "grade": "2020",
  "exp": 1712000000
}
```

- `sub`：用户 ID
- `role`：角色等级（1-4）
- `class_id` / `grade`：数据范围信息
- `exp`：过期时间（7 天）

### 5.2 环境变量

| 变量 | 说明 |
|------|------|
| `JWT_SECRET` | JWT 签名密钥 |
| `WECHAT_APP_ID` | 微信小程序 AppID |
| `WECHAT_APP_SECRET` | 微信小程序 AppSecret |

## 6. API 设计

### 6.1 `POST /api/v1/wechat/login` — 微信登录

**无需认证**

**请求体**：
```json
{ "code": "wx_auth_code" }
```

**逻辑**：
1. 调用微信 `code2Session` 接口换取 openid
2. 查询 `users` 表中 `openid` 匹配的用户
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
    "role": 1
  }
}
```

**错误响应**：
| 场景 | 状态码 | 提示 |
|------|--------|------|
| code 无效 | 400 | 微信授权码无效 |
| 未绑定 | 404 | 未绑定账号，请先绑定 |

---

### 6.2 `POST /api/v1/wechat/bind` — 绑定 OpenID

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
1. 判断是否有有效 JWT token
   - 有：从 token 获取用户 ID
   - 无：验证学号+密码，匹配用户
2. 调用微信 code2Session 换取 openid
3. 检查 openid 是否已被其他用户绑定
4. 绑定 openid 到当前用户
5. 若用户尚无密码，将本次密码哈希存储

**成功响应**（200）：
```json
{ "ok": true, "message": "绑定成功" }
```

**错误响应**：
| 场景 | 状态码 | 提示 |
|------|--------|------|
| code 无效 | 400 | 微信授权码无效 |
| 学号/密码错误 | 401 | 学号或密码不正确 |
| openid 已被绑定 | 409 | 该微信已绑定其他账号 |
| 已绑定其他微信 | 409 | 当前账号已绑定其他微信 |

## 7. 认证中间件

### 7.1 JWT 中间件（替换现有 IdentityFromHeaders）

```go
func JWTAuth() gin.HandlerFunc
```

逻辑：
1. 从 `Authorization` 头提取 Bearer token
2. 解析并验证 JWT
3. 构造 `auth.Actor` 并注入 context
4. 验证失败返回 401

### 7.2 路由调整

| 路由 | 认证 |
|------|------|
| `GET /healthz` | 无 |
| `POST /api/v1/wechat/login` | 无 |
| `POST /api/v1/wechat/bind` | 可选（有 token 走场景1，无 token 走场景2） |
| 其他 `/api/v1/*` | JWT 必需 |

## 8. 错误处理

统一错误码：

| 错误码 | 含义 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 未认证/认证失败 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 409 | 冲突（重复绑定） |
| 500 | 服务器内部错误 |

## 9. 测试策略

### 9.1 测试辅助

```go
func GenerateTestToken(userID uint, role int, classID uint, grade string) string
```

### 9.2 测试覆盖

| 测试项 | 文件 |
|--------|------|
| JWT 生成与验证 | `internal/service/auth/jwt_test.go` |
| 微信 code 换 openid | `internal/service/wechat/service_test.go` |
| 绑定接口（已登录/未登录） | `internal/http/handler/wechat_handler_test.go` |
| 登录接口 | `internal/http/handler/wechat_handler_test.go` |
| 中间件认证 | `internal/http/middleware/auth_test.go` |

### 9.3 测试辅助函数

新增 `internal/testutil/token.go`：

```go
package testutil

func GenerateTestToken(userID uint, role int, classID uint, grade string) string
```

所有测试文件通过此函数生成 JWT token，避免重复代码。

### 9.4 现有测试迁移清单

| 文件 | 改动 |
|------|------|
| `internal/http/middleware/identity_test.go` | 改为测试 JWT 中间件 |
| `internal/http/handler/admin_user_handler_test.go` | Header → Bearer token |
| `internal/http/handler/knowledge_handler_test.go` | Header → Bearer token |
| `internal/http/handler/admin_class_handler_test.go` | Header → Bearer token |
| `tests/api_contract_test.go` | Header → Bearer token |

迁移方式统一为：
```go
// 原
req.Header.Set("X-User-Id", "100")
req.Header.Set("X-User-Role", "1")
req.Header.Set("X-User-Class-Id", "1")
req.Header.Set("X-User-Grade", "2023")

// 新
token := testutil.GenerateTestToken(100, 1, 1, "2023")
req.Header.Set("Authorization", "Bearer "+token)
```

### 9.5 脚本更新

| 文件 | 改动 |
|------|------|
| `scripts/dev/knowledge_api_curl.sh` | 新增 JWT token 生成逻辑，所有 `-H "X-User-Id: ..."` 替换为 `-H "Authorization: Bearer $TOKEN"` |

脚本新增 token 生成方式（使用 `jq` 和 `openssl`）：
```bash
JWT_SECRET="${JWT_SECRET:-dev-secret}"
HEADER=$(echo -n '{"alg":"HS256","typ":"JWT"}' | base64 | tr -d '\n')
PAYLOAD=$(echo -n "{\"sub\":$ADMIN_USER_ID,\"role\":$ADMIN_ROLE,\"class_id\":$ADMIN_CLASS_ID,\"grade\":\"$ADMIN_GRADE\"}" | base64 | tr -d '\n')
SIGNATURE=$(echo -n "$HEADER.$PAYLOAD" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | base64 | tr -d '\n' | tr '+/' '-_' | tr -d '=')
TOKEN="$HEADER.$PAYLOAD.$SIGNATURE"
```

所有测试文件通过此函数生成 JWT token，避免重复代码。

### 9.4 现有测试迁移清单

| 文件 | 改动 |
|------|------|
| `internal/http/middleware/identity_test.go` | 改为测试 JWT 中间件 |
| `internal/http/handler/admin_user_handler_test.go` | Header → Bearer token |
| `internal/http/handler/knowledge_handler_test.go` | Header → Bearer token |
| `internal/http/handler/admin_class_handler_test.go` | Header → Bearer token |
| `tests/api_contract_test.go` | Header → Bearer token |

迁移方式统一为：
```go
// 原
req.Header.Set("X-User-Id", "100")
req.Header.Set("X-User-Role", "1")
req.Header.Set("X-User-Class-Id", "1")
req.Header.Set("X-User-Grade", "2023")

// 新
token := testutil.GenerateTestToken(100, 1, 1, "2023")
req.Header.Set("Authorization", "Bearer "+token)
```

### 9.5 脚本更新

| 文件 | 改动 |
|------|------|
| `scripts/dev/knowledge_api_curl.sh` | 新增 JWT token 生成逻辑，所有 `-H "X-User-Id: ..."` 替换为 `-H "Authorization: Bearer $TOKEN"` |

脚本新增 token 生成方式（使用 `jq` 和 `openssl`）：
```bash
JWT_SECRET="${JWT_SECRET:-dev-secret}"
HEADER=$(echo -n '{"alg":"HS256","typ":"JWT"}' | base64 | tr -d '\n')
PAYLOAD=$(echo -n "{\"sub\":$ADMIN_USER_ID,\"role\":$ADMIN_ROLE,\"class_id\":$ADMIN_CLASS_ID,\"grade\":\"$ADMIN_GRADE\"}" | base64 | tr -d '\n')
SIGNATURE=$(echo -n "$HEADER.$PAYLOAD" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | base64 | tr -d '\n' | tr '+/' '-_' | tr -d '=')
TOKEN="$HEADER.$PAYLOAD.$SIGNATURE"
```

## 10. 与 Phase 2 的衔接

- Phase 2 所有模块（党团流程、知识库、审批、信息发布）统一使用 JWT 认证
- 各模块 handler 通过 `auth.GetActor(c)` 获取身份，无需改动
- 权限判断仍走 `authz.Authorize(action)` + `authz.BuildScope(actor)`
- 本设计为 Phase 2 提供统一认证标准，避免各模块自行实现

## 11. 验收标准

1. `POST /api/v1/wechat/login` 能正确完成微信登录并返回 JWT
2. `POST /api/v1/wechat/bind` 支持已登录和未登录两种场景
3. JWT 中间件正确解析 token 并注入 Actor
4. 所有现有测试通过（已迁移为 JWT 认证）
5. `go test ./... -count=1` 全通过
6. 新增测试覆盖绑定/登录/认证关键路径
