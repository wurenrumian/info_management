# Phase2 Announcements API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`

## Student Endpoints

- `GET /api/v1/announcements?limit=20&offset=0`
- `GET /api/v1/announcements/:id`

### GET /api/v1/announcements

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "title": "五一假期安全提醒",
      "status": "published",
      "tags": ["假期", "安全"],
      "published_at": "2026-04-08T10:00:00Z"
    }
  ],
  "total": 1
}
```

### GET /api/v1/announcements/:id

成功响应：

```json
{
  "data": {
    "id": 1,
    "title": "五一假期安全提醒",
    "content": "请同学们离校前做好登记。",
    "status": "published",
    "audience_type": "targeted",
    "tags": ["假期", "安全"],
    "attachment_file_ids": [21],
    "external_links": [
      {
        "title": "学院官网原文",
        "url": "https://example.edu.cn/news/holiday",
        "source": "school_site"
      }
    ],
    "published_at": "2026-04-08T10:00:00Z",
    "created_at": "2026-04-08T09:00:00Z",
    "updated_at": "2026-04-08T10:00:00Z"
  }
}
```

## Admin Endpoints

- `GET /api/v1/admin/announcements?status=draft&limit=20&offset=0`
- `GET /api/v1/admin/announcements/:id`
- `POST /api/v1/admin/announcements`
- `PATCH /api/v1/admin/announcements/:id`
- `POST /api/v1/admin/announcements/:id/publish`
- `POST /api/v1/admin/announcements/:id/archive`

权限：
- `role >= 2`

### POST /api/v1/admin/announcements

请求体：

```json
{
  "title": "五一假期安全提醒",
  "content": "请同学们离校前做好登记。",
  "audience_type": "targeted",
  "target_scope": {
    "grades": ["2023"],
    "majors": ["信息管理"]
  },
  "tags": ["假期", "安全"],
  "attachment_file_ids": [21],
  "external_links": [
    {
      "title": "学院官网原文",
      "url": "https://example.edu.cn/news/holiday",
      "source": "school_site"
    }
  ]
}
```

### PATCH /api/v1/admin/announcements/:id

请求体：

```json
{
  "title": "五一假期安全提醒（更新版）",
  "content": "请同学们离校前做好登记并保持手机畅通。"
}
```

### POST /api/v1/admin/announcements/:id/publish

请求体：

```json
{
  "send_notification": true,
  "template_code": "announcement_publish"
}
```

成功响应：

```json
{
  "data": {
    "id": 1,
    "status": "published",
    "published_at": "2026-04-08T10:00:00Z"
  }
}
```

### POST /api/v1/admin/announcements/:id/archive

成功响应：

```json
{
  "data": {
    "id": 1,
    "status": "archived",
    "updated_at": "2026-04-09T09:00:00Z"
  }
}
```

错误响应：

- `400 {"error":"invalid audience_type"}`
- `400 {"error":"invalid id"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"announcement not found"}`

## Audience Type

- `all`
- `targeted`

## Status Enum

- `draft`
- `published`
- `archived`

## Audit Log

写操作记录到 `admin_logs`：

- `announcements.create`
- `announcements.patch`
- `announcements.publish`
- `announcements.archive`

