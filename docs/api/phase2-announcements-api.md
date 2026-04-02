# Phase2 Announcements API (v0 Placeholder)

## Base URL

`/api/v1`

## Scope

该文档作为并行开发占位稿，后续由模块 owner 补充完整接口细节。

## Planned Endpoints (Draft)

- `GET /api/v1/announcements`
- `GET /api/v1/announcements/:id`
- `POST /api/v1/admin/announcements`
- `PATCH /api/v1/admin/announcements/:id`

## Permissions (Draft)

- 学生：查看命中的通知
- 管理员（role >= 2）：按 scope 发布与更新通知

## Response Convention

- 成功：`{"data": ...}`
- 失败：`{"error": "..."}`
- 列表：`{"data": [...], "total": N}`

## TODO

- 补充筛选条件定义（班级/年级/角色）
- 补充附件引用格式（`file_id`）
- 补充测试命令与覆盖清单
