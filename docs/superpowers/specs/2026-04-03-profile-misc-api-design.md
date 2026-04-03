# 个人主页杂项 API 设计（Issue #24）

## 1. 目标

补齐个人主页资料编辑与展示闭环，满足前端统一走 `PATCH /api/v1/me` 的联调方式，并确保 `GET /api/v1/me` 与 `GET /api/v1/profile/home` 字段语义一致。

本次优先级：P0。

## 2. 范围

### In Scope

- 完成 `GET /api/v1/me` 资料字段输出对齐
- 完成 `PATCH /api/v1/me` 统一部分更新
- 完成 `GET /api/v1/profile/home` 的 `basic` 字段语义对齐
- 明确可编辑与只读字段
- 提供稳定的错误码（通过响应 `code` 字段承载）
- 覆盖成功与失败测试用例，并更新接口文档示例

### Out of Scope

- 管理端批量改资料
- 资料修改审批流
- 资料完善度等衍生画像

## 3. 字段模型与存储策略

### 3.1 字段语义

- `real_name`：实名，学生只读（映射 `users.name`）
- `nickname`：昵称，可编辑
- `major`：专业，可编辑
- `college`：学院，可编辑
- `enrollment_year`：入学学年，可编辑
- `bio`：个性签名，可编辑
- `avatar_url`：头像地址，可编辑

### 3.2 存储映射

- `real_name` -> `users.name`（已有列）
- `major` -> `users.major`（已有列）
- `college` -> `users.college`（新增列）
- `enrollment_year` -> `users.enrollment_year`（新增列）
- `nickname` -> `users.profile_attrs.nickname`
- `bio` -> `users.profile_attrs.bio`
- `avatar_url` -> `users.profile_attrs.avatar_url`

选择理由：
- `major/college/enrollment_year` 为稳定结构化字段，未来更可能参与筛选与统计，提前入列可避免二次迁移。
- `nickname/bio/avatar_url` 暂按展示属性保存在 `jsonb`，兼顾灵活性与改动规模。

## 4. 接口契约

### 4.1 GET `/api/v1/me`

鉴权：需要 JWT。  
返回结构继续采用统一 `{"data": ...}` 包裹，`data` 内字段为：

```json
{
  "id": 10001,
  "student_id": "2023123456",
  "real_name": "张三",
  "nickname": "阿三",
  "role": 1,
  "major": "人工智能",
  "college": "信息学院",
  "enrollment_year": 2023,
  "bio": "今天也在认真生活",
  "avatar_url": "https://xxx/avatar/10001.png",
  "updated_at": "2026-04-03T10:20:30Z"
}
```

### 4.2 PATCH `/api/v1/me`

鉴权：需要 JWT。  
语义：部分更新，只更新请求体里传入的可编辑字段。

可编辑字段：
- `nickname`
- `major`
- `college`
- `enrollment_year`
- `bio`
- `avatar_url`

只读字段（出现即报错）：
- `real_name`
- `student_id`
- `id`
- `role`

成功响应：
- 状态码 `200`
- 返回 `{"data": <与 GET /me 相同语义对象>}`

### 4.3 GET `/api/v1/profile/home`

保持现有聚合结构不变：
- `data.basic`
- `data.quick_entry`
- `data.account`

其中 `data.basic` 统一为与 `/api/v1/me` 同语义字段集，至少包含：
- `real_name`
- `nickname`
- `major`
- `college`
- `enrollment_year`
- `bio`
- `avatar_url`

## 5. 校验规则

- `nickname`：去首尾空格后长度 `1~20`
- `bio`：去首尾空格后长度 `0~100`
- `major`：去首尾空格后长度 `1~50`
- `college`：去首尾空格后长度 `1~50`
- `enrollment_year`：`2000 ~ 当前年份+1`
- `avatar_url`：必须是合法 `http/https` URL

说明：`PATCH` 请求体必须至少包含一个可编辑字段，否则返回参数错误。

## 6. 错误码设计

当前仓库错误响应格式为 `{"error":"..."}`。为保持兼容并满足联调要求，扩展为：

```json
{
  "error": "real_name is read-only",
  "code": 40002
}
```

错误码约定：
- `40001` 参数非法（长度/格式错误）
- `40002` 修改只读字段
- `40003` 业务规则冲突（如入学年份超范围）
- `40100` 未登录或 token 无效
- `40300` 无权限
- `50000` 服务内部错误

其中 HTTP 状态仍遵循现有规范：
- `400` 参数/业务冲突
- `401` 未认证
- `403` 无权限
- `500` 内部错误

## 7. 实施改动点

- `internal/model/user.go`
  - 新增 `College`、`EnrollmentYear` 字段
- `internal/service/profile/service.go`
  - 引入 Me/Profile DTO，统一 `GET /me` 和 `profile/home.basic` 映射
  - 扩展 Patch 输入结构与合并策略
- `internal/http/handler/me_handler.go`
  - 扩展 `PATCH /me` 请求体解析、只读字段检测、参数校验和错误码映射
- `internal/http/response/response.go`
  - 增加带业务错误码的失败响应方法
- 测试：
  - `internal/http/handler/me_handler_test.go`
  - `internal/service/profile/service_test.go`
- 文档：
  - `docs/api/phase1-foundation-api.md` 增加成功/失败示例

## 8. 测试策略

- Handler 测试：
  - `GET /me` 返回新字段语义
  - `PATCH /me` 成功更新混合字段（列 + jsonb）
  - 只读字段更新返回 `400 + code=40002`
  - 非法参数返回 `400 + code=40001`
  - 学年越界返回 `400 + code=40003`
- Service 测试：
  - `profile_attrs` 合并不丢失无关 key
  - `GetHome.basic` 与 `GetMe` 字段语义一致
- 回归：
  - 现有权限与聚合计数逻辑保持不变

## 9. 兼容性与风险

- 兼容点：
  - API 成功响应仍为 `{"data": ...}`
  - 失败响应保留 `error`，仅新增 `code`
- 风险点：
  - 增加表字段后，历史 SQLite/PG 环境需依赖 AutoMigrate 生效
  - `profile_attrs` 中历史脏值（例如非字符串）可能影响类型断言

缓解：
- 读取 `profile_attrs` 时做宽松解析与安全类型转换
- 关键路径用 handler + service 测试覆盖
