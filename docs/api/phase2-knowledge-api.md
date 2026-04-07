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
- `POST /api/v1/admin/knowledge/qa-generate-preview`（AI 生成问答草稿，仅预览不入库）
- `POST /api/v1/admin/knowledge/qa-generate-preview/stream`（SSE 版本，便于前端实时消费）
- `POST /api/v1/admin/knowledge/batch`（批量提交问答，事务全成功）
- `POST /api/v1/admin/knowledge/:id/attachments`（批量绑定附件）
- `GET /api/v1/admin/knowledge/:id/attachments`（查询已绑定附件）
- `DELETE /api/v1/admin/knowledge/:id/attachments/:file_id`（解绑单个附件）
- `PATCH /api/v1/admin/knowledge/:id`
- `DELETE /api/v1/admin/knowledge/:id`

权限：
- 常规管理接口：`role >= 2`
- 删除相关接口：`role >= 3`
- AI 生成与批量提交接口：`role >= 3`

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
      "attachments": [
        {
          "file_id": 21,
          "title": "复学指引.pdf",
          "url": "/uploads/knowledge/2026/04/1712000000000000001_back.pdf",
          "content_type": "application/pdf",
          "file_size": 20480
        }
      ],
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
    "attachments": [
      {
        "file_id": 21,
        "title": "复学指引.pdf",
        "url": "/uploads/knowledge/2026/04/1712000000000000001_back.pdf",
        "content_type": "application/pdf",
        "file_size": 20480
      }
    ],
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
  "keywords": ["复学", "审批"]
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
    "attachments": [],
    "created_by": 200,
    "updated_by": 200,
    "created_at": "2026-03-31T00:00:00Z",
    "updated_at": "2026-03-31T00:00:00Z"
  }
}
```

### POST /api/v1/admin/knowledge/qa-generate-preview

权限：`role >= 3`

请求体：

```json
{
  "file_ids": [12, 18],
  "qa_count_range": {
    "min": 5,
    "max": 12
  }
}
```

约束：

- `file_ids` 不能为空
- `qa_count_range` 必须满足 `1 <= min <= max <= 30`

成功响应（草稿）：

```json
{
  "data": [
    {
      "question": "奖学金申请需要什么材料？",
      "answer": "需要成绩单、综测证明、申请表。",
      "attachment_file_ids": [12]
    }
  ],
  "total": 9
}
```

说明：

- 预览阶段 `keywords` 为可选字段，AI 可不返回。
- 该接口对前端仍返回普通 JSON（非流式响应）。
- 服务端调用 AI Provider 时使用 `stream=true`（SSE）流式接收并在服务端聚合后解析，兼容以下返回形态：
  - SSE 分片（`data: ...` + `data: [DONE]`）
  - 非流式一次性 JSON（兼容兜底）

### POST /api/v1/admin/knowledge/qa-generate-preview/stream

权限：`role >= 3`

请求体：与 `POST /api/v1/admin/knowledge/qa-generate-preview` 相同。

响应：`text/event-stream`

事件格式：

- `event: drafts`
  - `data`: `{"items":[...],"total":N}`
- `event: done`
  - `data`: `[DONE]`

说明：

- 该端点用于前端流式消费；服务端会在生成完成后推送 `drafts` 事件并以 `done` 收尾。

### POST /api/v1/admin/knowledge/batch

权限：`role >= 3`

请求体：

```json
{
  "items": [
    {
      "question": "奖学金申请需要什么材料？",
      "answer": "需要成绩单、综测证明、申请表。",
      "attachment_file_ids": [12]
    }
  ]
}
```

说明：

- `keywords` 可选；若不传则按空数组 `[]` 入库。

事务语义：

- 任一 item 校验或写入失败，整体回滚
- 全部成功才提交

成功响应：

```json
{
  "data": [
    {
      "id": 101,
      "question": "奖学金申请需要什么材料？",
      "answer": "需要成绩单、综测证明、申请表。",
      "keywords": ["奖学金", "申请材料"],
      "attachments": [
        {
          "file_id": 12,
          "title": "scholarship.docx",
          "url": "/uploads/knowledge/2026/04/123_scholarship.docx",
          "content_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
          "file_size": 20480
        }
      ],
      "created_by": 1,
      "updated_by": 1,
      "created_at": "2026-04-05T00:00:00Z",
      "updated_at": "2026-04-05T00:00:00Z"
    }
  ],
  "total": 1
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

权限：`role >= 3`

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
- `400 {"error":"empty items"}`
- `400 {"error":"invalid qa_count_range"}`
- `400 {"error":"invalid item"}`
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
- `500 {"error":"generate preview failed"}`
- `500 {"error":"batch create knowledge failed"}`

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
        "url": "/uploads/knowledge/2026/04/123_report.pdf",
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
      "url": "/uploads/knowledge/2026/04/123_report.pdf",
      "content_type": "application/pdf",
      "file_size": 1048576
    }
  ]
}
```

### DELETE /api/v1/admin/knowledge/:id/attachments/:file_id

权限：`role >= 3`

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
- `knowledge.patch`
- `knowledge.delete`
- `knowledge.attach`
- `knowledge.detach`
- `knowledge.ai_generate_preview`
- `knowledge.batch_create`
