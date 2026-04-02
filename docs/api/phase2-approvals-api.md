# Phase2 Approvals API (v0 Placeholder)

## Base URL

`/api/v1`

## Scope

该文档作为并行开发占位稿，后续由模块 owner 补充完整接口细节。

## Planned Endpoints (Draft)

- `POST /api/v1/approvals`
- `GET /api/v1/approvals/me`
- `GET /api/v1/admin/approvals`
- `PATCH /api/v1/admin/approvals/:id`

## Permissions (Draft)

- 学生：发起/查看本人申请
- 管理员（role >= 2）：按 scope 审批处理

## Response Convention

- 成功：`{"data": ...}`
- 失败：`{"error": "..."}`
- 列表：`{"data": [...], "total": N}`

## TODO

- 补充状态流转定义（发起/通过/驳回/撤回）
- 补充请求/响应字段定义
- 补充测试命令与覆盖清单
