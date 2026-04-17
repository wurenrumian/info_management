# Phase2 Announcements API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`

## Student Endpoints

- `GET /api/v1/announcements?limit=20&offset=0`
- `GET /api/v1/announcements/all?limit=20&offset=0`（仅 `role > 2`）
- `GET /api/v1/announcements/all/:id`（仅 `role > 2`）
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

筛选规则说明：
- 当 `audience_type=targeted` 且 `target_scope.grades` 非空时，学生端按用户所属班级的 `classes.grade` 进行匹配。
- `users.grade` 是系统维护快照字段，不作为公告年级筛选的一手事实来源。

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

### GET /api/v1/announcements/all

说明：
- 仅教师/超级管理员可访问（`role > 2`）。
- 返回所有 `published` 公告，不应用 `target_scope` 的年级/专业/班级筛选。

成功响应：

```json
{
  "data": [
    {
      "id": 2,
      "title": "仅2024级可见",
      "status": "published",
      "audience_type": "targeted"
    },
    {
      "id": 1,
      "title": "全员公告",
      "status": "published",
      "audience_type": "all"
    }
  ],
  "total": 2
}
```

### GET /api/v1/announcements/all/:id

说明：
- 仅教师/超级管理员可访问（`role > 2`）。
- 返回单条 `published` 公告完整内容，不应用 `target_scope` 筛选。

成功响应：

```json
{
  "data": {
    "id": 2,
    "title": "仅2024级可见",
    "content": "这是定向公告完整正文",
    "status": "published",
    "audience_type": "targeted",
    "target_scope": {
      "grades": ["2024"]
    }
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
    "majors": ["信息管理"],
    "class_ids": [1],
    "roles": [2],
    "student_ids": ["S1001", "S1002"]
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
  "template_code": "announcement"
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
