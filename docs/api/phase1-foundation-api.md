# Phase1 Foundation API

## Base URL

`/api/v1`

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
- `GET /api/v1/files`
- `GET /api/v1/files/:id`
- `GET /api/v1/files/:id/download`
- `DELETE /api/v1/files/:id`

## Knowledge Endpoints

- `GET /api/v1/knowledge/search`
- `GET /api/v1/admin/knowledge`
- `GET /api/v1/admin/knowledge/:id`
- `POST /api/v1/admin/knowledge`
- `POST /api/v1/admin/knowledge/import`
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
  "avatar_url": "https://example.com/avatar/10001.png"
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
    "avatar_url": "https://example.com/avatar/10001.png",
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
