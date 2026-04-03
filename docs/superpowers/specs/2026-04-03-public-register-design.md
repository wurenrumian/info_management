# 公开注册接口设计（方案 A）

## 1. 目标

为非开发环境提供公开注册入口，支持学生通过 `student_id + name` 完成账号创建/激活，并直接拿到 JWT，用于后续访问受保护接口。

本方案选择：
- 纯学号 + 姓名注册
- 若学号不存在则自动建号
- 与微信绑定流程兼容（可选传 `code` 同步绑定 openid）

## 2. 范围

### In Scope

- 新增公开注册接口：`POST /api/v1/auth/public-register`
- 支持“已存在学号激活”与“不存在学号自动创建”
- 可选绑定微信 openid（传 `code` 时）
- 注册成功后直接返回 JWT + 用户信息
- 记录最小审计日志（注册行为）

### Out of Scope

- 短信验证码/注册码
- 复杂风控（图形验证码、设备指纹等）
- 完整账号找回体系

## 3. 接口设计

### 3.1 POST `/api/v1/auth/public-register`

鉴权：无需 JWT  
请求体：

```json
{
  "student_id": "2020001",
  "name": "张三",
  "code": "wx_auth_code_optional"
}
```

字段规则：
- `student_id`：必填，非空
- `name`：必填，非空
- `code`：可选；有值时尝试 `code -> openid` 并绑定

成功响应（200）：

```json
{
  "data": {
    "token": "jwt...",
    "user": {
      "id": 1,
      "student_id": "2020001",
      "name": "张三",
      "role": 1,
      "class_id": 10,
      "grade": "2020",
      "major": "信息管理"
    }
  }
}
```

失败响应：
- 400：缺少 `student_id/name` 或参数非法
- 401：同学号存在但姓名不匹配
- 409：微信 openid 已绑定他人（仅传 `code` 时）
- 500：数据库或签发 token 失败

## 4. 业务流程

1. 参数校验（`student_id`、`name`）。
2. 以 `student_id` 查用户：
   - 存在：校验 `name` 必须一致，否则拒绝。
   - 不存在：自动创建学生账号（`role=student`，其余字段使用系统默认值）。
3. 若传 `code`：
   - 调用微信 `code2Session` 获取 openid；
   - 检查 openid 是否已绑定其他用户；
   - 将 openid 绑定当前用户。
4. 生成 JWT 并返回 `token + user`。
5. 写入注册行为日志（最小审计）。

## 5. 数据与默认值策略

自动建号时默认值：
- `role = student`
- `class_id = 10`
- `grade = "2020"`
- `major = "信息管理"`

说明：该默认策略与现有 dev 用户创建策略对齐，后续可改为管理员维护或批量导入覆盖。

## 6. 安全与风险

风险：纯学号+姓名存在冒名注册风险。  
当前最小缓解：
- 同学号姓名必须完全匹配
- 注册日志保留，支持审计追溯
- 同 openid 不可跨账号绑定

后续增强建议（非本次范围）：
- 增加短信/注册码
- 增加 IP 限流与黑名单

## 7. 实施改动点

- `internal/http/handler/wechat_handler.go`
  - 新增 `PublicRegister` handler
- `internal/http/router/router.go`
  - 注册 `POST /api/v1/auth/public-register`
- `docs/api/phase2-wechat-api.md`
  - 增补公开注册接口文档
- 测试：
  - `internal/http/handler/wechat_handler_test.go`
  - 覆盖：创建新用户、激活已有用户、姓名不匹配、可选 code 绑定

## 8. 测试策略

- Handler 单测：
  - 新用户注册成功，返回 token
  - 旧用户激活成功，返回 token
  - 姓名不匹配返回 401
  - code 无效返回 400
  - openid 冲突返回 409
- 回归：
  - 不影响现有 `/wechat/login` `/wechat/bind` `/dev/*`

## 9. 兼容性

- 不修改现有登录与绑定接口语义，仅新增入口。
- 前端可逐步切换至新接口，不影响已有调用。
