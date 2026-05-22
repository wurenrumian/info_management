# Phase2 Announcements API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`

## Overview

- `GET /api/v1/announcements` 只返回当前用户可见的 `published` 公告。
- `GET /api/v1/announcements/all` 和 `GET /api/v1/announcements/all/:id` 不做受众范围过滤，只给有权限的教职工/管理员侧使用。
- 管理端 `create / patch / publish / archive` 都走 `/api/v1/admin/announcements/*`。

公告模型对应 `model.Announcement`，响应里会直接返回该模型的 JSON 字段。

## Student Endpoints

- `GET /api/v1/announcements?limit=20&offset=0`
- `GET /api/v1/announcements/all?limit=20&offset=0`
- `GET /api/v1/announcements/all/:id`
- `GET /api/v1/announcements/:id`

### GET /api/v1/announcements

说明：

- 只返回 `status=published` 的公告。
- `target_scope` 按当前用户的 `class.grade`、`major`、`class_id`、`role`、`student_id` 做匹配。
- 结果使用 `{"data":[...],"total":N}` 包装。

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "title": "五一假期安全提醒",
      "content": "请同学们离校前做好登记。",
      "status": "published",
      "audience_type": "targeted",
      "target_scope": {
        "grades": ["2023"]
      },
      "tags": ["假期", "安全"],
      "attachment_file_ids": [21],
      "external_links": [
        {
          "title": "学院官网原文",
          "url": "https://example.edu.cn/news/holiday",
          "source": "school_site"
        }
      ],
      "author_id": 999,
      "published_at": "2026-04-08T10:00:00Z",
      "created_at": "2026-04-08T09:00:00Z",
      "updated_at": "2026-04-08T10:00:00Z"
    }
  ],
  "total": 1
}
```

### GET /api/v1/announcements/:id

说明：

- 只返回当前用户可见且已发布的公告。
- 不可见或未发布时返回 `404 announcement not found`。

成功响应：

```json
{
  "data": {
    "id": 1,
    "title": "五一假期安全提醒",
    "content": "请同学们离校前做好登记。",
    "status": "published",
    "audience_type": "targeted",
    "target_scope": {
      "grades": ["2023"]
    },
    "tags": ["假期", "安全"],
    "attachment_file_ids": [21],
    "external_links": [
      {
        "title": "学院官网原文",
        "url": "https://example.edu.cn/news/holiday",
        "source": "school_site"
      }
    ],
    "author_id": 999,
    "published_at": "2026-04-08T10:00:00Z",
    "created_at": "2026-04-08T09:00:00Z",
    "updated_at": "2026-04-08T10:00:00Z"
  }
}
```

### GET /api/v1/announcements/all

说明：

- 只对有 `announcements:list:all` 权限的用户开放。
- 返回所有 `published` 公告，不做受众范围过滤。
- 响应结构同普通列表。

### GET /api/v1/announcements/all/:id

说明：

- 只对有 `announcements:get:all` 权限的用户开放。
- 返回指定 `published` 公告的完整内容，不做受众范围过滤。

## Admin Endpoints

- `GET /api/v1/admin/announcements?status=draft&limit=20&offset=0`
- `GET /api/v1/admin/announcements/:id`
- `POST /api/v1/admin/announcements`
- `PATCH /api/v1/admin/announcements/:id`
- `POST /api/v1/admin/announcements/:id/publish`
- `POST /api/v1/admin/announcements/:id/archive`

权限：

- 走 `authz` 权限控制，不是简单的硬编码角色判断。
- 管理端列表支持 `status=draft|published|archived` 过滤。

### GET /api/v1/admin/announcements

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "title": "五一假期安全提醒",
      "content": "请同学们离校前做好登记。",
      "status": "draft",
      "audience_type": "targeted",
      "target_scope": {
        "grades": ["2023"]
      },
      "tags": ["假期", "安全"],
      "attachment_file_ids": [21],
      "external_links": [],
      "author_id": 999,
      "published_at": null,
      "created_at": "2026-04-08T09:00:00Z",
      "updated_at": "2026-04-08T09:00:00Z"
    }
  ],
  "total": 1
}
```

### GET /api/v1/admin/announcements/:id

说明：

- 返回 `model.Announcement` 的完整数据。
- 未找到时返回 `404 announcement not found`。

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

说明：

- `audience_type` 为空时默认 `all`。
- 当 `audience_type=all` 时，`target_scope` 会被清空为 `{}`。
- 该接口只做创建，新增公告初始状态固定为 `draft`。

成功响应：

```json
{
  "data": {
    "id": 1,
    "title": "五一假期安全提醒",
    "content": "请同学们离校前做好登记。",
    "status": "draft",
    "audience_type": "targeted",
    "target_scope": {
      "grades": ["2023"]
    },
    "tags": ["假期", "安全"],
    "attachment_file_ids": [21],
    "external_links": [],
    "author_id": 999,
    "published_at": null,
    "created_at": "2026-04-08T09:00:00Z",
    "updated_at": "2026-04-08T09:00:00Z"
  }
}
```

### PATCH /api/v1/admin/announcements/:id

请求体可以只传部分字段：

```json
{
  "title": "五一假期安全提醒（更新版）",
  "content": "请同学们离校前做好登记并保持手机畅通。",
  "audience_type": "all",
  "tags": ["假期", "安全", "通知"]
}
```

说明：

- 支持更新 `title`、`content`、`audience_type`、`target_scope`、`tags`、`attachment_file_ids`、`external_links`。
- `audience_type` 设为 `all` 时会同时清空 `target_scope`。
- 如果请求体没有任何可更新字段，返回 `400 invalid request`。

### POST /api/v1/admin/announcements/:id/publish

请求体可为空；如果传 body，也只会读取下面两个字段：

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
    "published_at": "2026-04-08T10:00:00Z",
    "notification_summary": {
      "attempted": 10,
      "sent": 9,
      "failed": 1,
      "first_error": "wechat api error",
      "failures": [
        {
          "user_id": 1001,
          "error": "wechat api error"
        }
      ]
    }
  }
}
```

说明：

- `send_notification=false` 或未传时，`notification_summary` 可能为 `null`。
- 默认模板码是 `announcement`。

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

## Error Responses

- `400 {"error":"invalid audience_type"}`
- `400 {"error":"invalid status"}`
- `400 {"error":"invalid id"}`
- `400 {"error":"invalid request"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"announcement not found"}`
- `500 {"error":"query failed"}`
- `500 {"error":"create failed"}`
- `500 {"error":"patch failed"}`
- `500 {"error":"publish failed"}`
- `500 {"error":"archive failed"}`

## Notes

- `target_scope` 为空时等同于 `{}`。
- `tags`、`attachment_file_ids`、`external_links` 都存为 JSON 字段，接口不做更深层的结构校验。
