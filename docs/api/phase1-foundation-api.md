# Phase1 Foundation API

## Base URL

`/api/v1`

## Common Public Routes

- `GET /healthz`
- `GET /uploads/<file_path>`（静态文件访问，例如 `/uploads/knowledge/...`、`/uploads/avatars/...`）

## Authentication

All endpoints (except `/auth/public-register`, `/wechat/login`, `/wechat/bind`, `/wechat/callback`, `/dev/register-or-login`, and `/dev/login-and-send-subscribe-check`) require a JWT token in the `Authorization` header:

```
Authorization: Bearer <token>
```

Token payload contains: `sub` (user ID), `role`, `class_id`, `grade`.

`grade` governance:
- `classes.grade` is source-of-truth.
- `users.grade` is system-managed snapshot.
- `PATCH /api/v1/admin/users/:id` does not accept `grade`.

## Student Endpoint

- `GET /api/v1/me`
- `PATCH /api/v1/me`
- `GET /api/v1/profile/home`

## Admin User Endpoints

- `GET /api/v1/admin/users`
- `GET /api/v1/admin/users/:id`
- `PATCH /api/v1/admin/users/:id`
- `POST /api/v1/admin/users/import`（仅超级管理员）

## Admin Class Endpoints

- `GET /api/v1/admin/classes`
- `GET /api/v1/admin/classes/:id`
- `POST /api/v1/admin/classes`
- `PATCH /api/v1/admin/classes/:id`

## Admin Logs Endpoint

- `GET /api/v1/admin/logs`

## WeChat Endpoints

- `POST /api/v1/auth/public-register` — 无需认证
- `POST /api/v1/wechat/login` — 无需认证
- `POST /api/v1/wechat/bind` — 可选认证
- `POST /api/v1/wechat/callback` — 无需认证（微信服务器回调）
- `POST /api/v1/dev/register-or-login` — 仅开发环境启用
- `POST /api/v1/dev/login-and-send-subscribe-check` — 仅开发环境启用

## File Endpoints

- `POST /api/v1/files/upload`
- `GET /api/v1/files/search`
- `GET /api/v1/files`
- `GET /api/v1/files/:id`
- `GET /api/v1/files/:id/download`
- `DELETE /api/v1/files/:id`

## Knowledge Endpoints

- `GET /api/v1/knowledge/search`
- `GET /api/v1/knowledge/:id`
- `GET /api/v1/admin/knowledge`
- `GET /api/v1/admin/knowledge/:id`
- `POST /api/v1/admin/knowledge`
- `POST /api/v1/admin/knowledge/:id/attachments`
- `GET /api/v1/admin/knowledge/:id/attachments`
- `DELETE /api/v1/admin/knowledge/:id/attachments/:file_id`
- `PATCH /api/v1/admin/knowledge/:id`
- `DELETE /api/v1/admin/knowledge/:id`

## Notification Endpoints

- `POST /api/v1/admin/notification/templates`
- `GET /api/v1/admin/notification/templates/:code`
- `GET /api/v1/admin/notification/logs`
- `GET /api/v1/notifications/unread/count`
- `POST /api/v1/user/subscribe/report`

## Example Request

```http
GET /api/v1/admin/users HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

## Profile Examples

### PATCH /api/v1/me (Success)

Request:

```json
{
  "nickname": "阿三同学",
  "major": "人工智能",
  "college": "信息学院",
  "enrollment_year": 2023,
  "bio": "今天也在认真生活",
  "avatar_url": "/uploads/avatars/2026/04/1775373038139423368_avatar.jpg"
}
```

Response:

```json
{
  "data": {
    "id": 10001,
    "student_id": "2023123456",
    "real_name": "张三",
    "nickname": "阿三同学",
    "role": 1,
    "major": "人工智能",
    "college": "信息学院",
    "enrollment_year": 2023,
    "bio": "今天也在认真生活",
    "avatar_url": "/uploads/avatars/2026/04/1775373038139423368_avatar.jpg",
    "updated_at": "2026-04-03T10:20:30Z"
  }
}
```

### PATCH /api/v1/me (Failure: read-only field)

Request:

```json
{
  "real_name": "李四"
}
```

Response:

```json
{
  "error": "real_name is read-only",
  "code": 40002
}
```

## User Import API

### POST /api/v1/admin/users/import

仅超级管理员可用。

支持三种输入格式：
- `application/json`
- `multipart/form-data` 上传 `.csv`
- `multipart/form-data` 上传 `.xlsx`

导入规则：
- 必填字段：`student_id`, `name`, `class_id`
- 可选字段：`role`, `major`, `college`, `enrollment_year`
- `grade` 不允许导入写入（系统按 `class_id` 同步）
- `student_id` 已存在时，跳过并记失败（不更新）
- `class_id` 不存在时，跳过并记失败
- 单行失败不影响其他行

JSON 请求示例：
```json
{
  "users": [
    {
      "student_id": "20260001",
      "name": "张三",
      "class_id": 1,
      "role": 1,
      "major": "计算机科学与技术",
      "college": "信息学院",
      "enrollment_year": 2026
    }
  ]
}
```

成功响应（200）：
```json
{
  "data": {
    "imported": 1,
    "failed": 2,
    "errors": [
      {
        "row": 3,
        "student_id": "20260002",
        "error": "duplicate student_id"
      },
      {
        "row": 4,
        "student_id": "20260003",
        "error": "class not found"
      }
    ]
  }
}
```

错误响应：
| 状态码 | 响应体 | 说明 |
|--------|--------|------|
| 400 | `{"error":"invalid import payload"}` | JSON 结构非法 |
| 400 | `{"error":"file is required"}` | 非 JSON 请求且未上传文件 |
| 400 | `{"error":"invalid import file"}` | CSV/XLSX 内容非法 |
| 400 | `{"error":"unsupported file type"}` | 文件扩展名不支持 |
| 403 | `{"error":"forbidden"}` | 非超级管理员 |
| 500 | `{"error":"import users failed"}` | 服务端导入失败 |
