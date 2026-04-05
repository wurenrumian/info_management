# Phase2 Knowledge API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`（JWT token，由微信登录接口返回）

## Student Endpoint

- `GET /api/v1/knowledge/search?q=...&limit=20&offset=0`
- `GET /api/v1/knowledge/:id`

权限：`role >= 1`

搜索范围：
- `question`
- `answer`
- `keywords`
- `content_text`（由已绑定附件抽取出的正文文本）

检索策略（当前实现）：
- 先进行全文检索（FTS）
- 若无命中，回退到分词后的 `LIKE` 检索（中文场景更稳定）

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "question": "休学申请怎么办理",
      "answer": "先联系辅导员并提交休学申请表",
      "keywords": ["休学", "申请"],
      "attachments": [{"title": "休学申请表", "url": "https://example.com/leave"}],
      "created_by": 999,
      "updated_by": 999,
      "created_at": "2026-03-31T00:00:00Z",
      "updated_at": "2026-03-31T00:00:00Z"
    }
  ],
  "total": 42
}
```

错误响应：

- `400 {"error":"missing q"}`
- `400 {"error":"invalid id"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"knowledge not found"}`

### GET /api/v1/knowledge/:id

成功响应：

```json
{
  "data": {
    "id": 1,
    "question": "休学申请怎么办理",
    "answer": "先联系辅导员并提交休学申请表",
    "keywords": ["休学", "申请"],
    "attachments": [
      {
        "file_id": 10,
        "title": "休学申请表.docx",
        "url": "/uploads/knowledge/2026/04/1712000000000000000_leave.docx",
        "content_type": "application/msword",
        "file_size": 10240
      }
    ],
    "created_by": 999,
    "updated_by": 999,
    "created_at": "2026-03-31T00:00:00Z",
    "updated_at": "2026-03-31T00:00:00Z"
  }
}
```

## Admin Endpoints

- `GET /api/v1/admin/knowledge?query=...&limit=20&offset=0`
- `GET /api/v1/admin/knowledge/:id`
- `POST /api/v1/admin/knowledge`
- `POST /api/v1/admin/knowledge/:id/attachments`（批量绑定附件）
- `GET /api/v1/admin/knowledge/:id/attachments`（查询已绑定附件）
- `DELETE /api/v1/admin/knowledge/:id/attachments/:file_id`（解绑单个附件）
- `PATCH /api/v1/admin/knowledge/:id`
- `DELETE /api/v1/admin/knowledge/:id`

权限：`role >= 2`

### GET /api/v1/admin/knowledge

成功响应：

```json
{
  "data": [
    {
      "id": 2,
      "question": "复学流程是什么",
      "answer": "提交复学申请并等待审批",
      "keywords": ["复学", "审批"],
      "attachments": [{"title": "复学指引", "url": "https://example.com/back"}],
      "created_by": 200,
      "updated_by": 200,
      "created_at": "2026-03-31T00:00:00Z",
      "updated_at": "2026-03-31T00:00:00Z"
    }
  ],
  "total": 12
}
```

### GET /api/v1/admin/knowledge/:id

成功响应：

```json
{
  "data": {
    "id": 2,
    "question": "复学流程是什么",
    "answer": "提交复学申请并等待审批",
    "keywords": ["复学", "审批"],
    "attachments": [{"title": "复学指引", "url": "https://example.com/back"}],
    "created_by": 200,
    "updated_by": 200,
    "created_at": "2026-03-31T00:00:00Z",
    "updated_at": "2026-03-31T00:00:00Z"
  }
}
```

### POST /api/v1/admin/knowledge

请求体：

```json
{
  "question": "复学流程是什么",
  "answer": "提交复学申请并等待审批",
  "keywords": ["复学", "审批"],
  "attachments": [{"title": "复学指引", "url": "https://example.com/back"}]
}
```

成功响应：

```json
{
  "data": {
    "id": 2,
    "question": "复学流程是什么",
    "answer": "提交复学申请并等待审批",
    "keywords": ["复学", "审批"],
    "attachments": [{"title": "复学指引", "url": "https://example.com/back"}],
    "created_by": 200,
    "updated_by": 200,
    "created_at": "2026-03-31T00:00:00Z",
    "updated_at": "2026-03-31T00:00:00Z"
  }
}
```

### PATCH /api/v1/admin/knowledge/:id

请求体（部分字段）：

```json
{
  "answer": "先提交复学材料，再由学院审批"
}
```

成功响应：

```json
{
  "data": {
    "updated": true
  }
}
```

### DELETE /api/v1/admin/knowledge/:id

成功响应：

```json
{
  "data": {
    "deleted": true
  }
}
```

管理端错误响应：

- `400 {"error":"invalid body"}`
- `400 {"error":"invalid id"}`
- `400 {"error":"missing file_ids"}`
- `400 {"error":"file not found"}`
- `400 {"error":"invalid file_id"}`
- `400 {"error":"empty patch"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"knowledge not found"}`
- `500 {"error":"list knowledge failed"}`
- `500 {"error":"get knowledge failed"}`
- `500 {"error":"patch knowledge failed"}`
- `500 {"error":"delete knowledge failed"}`
- `500 {"error":"bind attachment failed"}`
- `500 {"error":"list attachment failed"}`
- `500 {"error":"delete attachment failed"}`

### POST /api/v1/admin/knowledge/:id/attachments

请求体：

```json
{
  "file_ids": [1, 2, 3]
}
```

成功响应：

```json
{
  "data": {
    "added_count": 2,
    "already_count": 1,
    "attachments": [
      {
        "file_id": 1,
        "title": "report.pdf",
        "url": "/uploads/documents/2026/04/123_report.pdf",
        "content_type": "application/pdf",
        "file_size": 1048576
      }
    ]
  }
}
```

### GET /api/v1/admin/knowledge/:id/attachments

成功响应：

```json
{
  "data": [
    {
      "file_id": 1,
      "title": "report.pdf",
      "url": "/uploads/documents/2026/04/123_report.pdf",
      "content_type": "application/pdf",
      "file_size": 1048576
    }
  ]
}
```

### DELETE /api/v1/admin/knowledge/:id/attachments/:file_id

成功响应：

```json
{
  "data": {
    "deleted": true
  }
}
```

## Audit Log

以下管理动作会写入 `admin_logs`：

- `knowledge.create`
- `knowledge.import`
- `knowledge.patch`
- `knowledge.delete`
- `knowledge.attach`
- `knowledge.detach`
