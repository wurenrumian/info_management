# Phase2 PartyFlow API (v0 Placeholder)

## Base URL

`/api/v1`

## Scope

该文档作为并行开发占位稿，后续由模块 owner 补充完整接口细节。

## Planned Endpoints (Draft)

- `GET /api/v1/partyflow/me`
- `GET /api/v1/admin/partyflow/progress`
- `PATCH /api/v1/admin/partyflow/progress/:id`

## Permissions (Draft)

- 学生：仅可查看本人党团进度
- 管理员（role >= 2）：可按 scope 查询与更新

## Response Convention

- 成功：`{"data": ...}`
- 失败：`{"error": "..."}`
- 列表：`{"data": [...], "total": N}`

## TODO

- 补充请求/响应字段定义
- 补充错误码细则
- 补充测试命令与覆盖清单
