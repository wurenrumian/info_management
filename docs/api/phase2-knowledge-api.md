# Phase2 Knowledge API

## Base URL

`/api/v1`

## Required Headers

- `X-User-Id`
- `X-User-Role`
- `X-User-Class-Id` (按角色 scope 使用)
- `X-User-Grade` (按角色 scope 使用)

## Student Endpoint

- `GET /api/v1/knowledge/search?q=...&limit=20&offset=0`

权限：`role >= 1`

搜索范围：
- `question`
- `answer`
- `keywords`
- `content_text`（由导入文件抽取出的正文文本）

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
  ]
}
```

错误响应：

- `400 {"error":"missing q"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`

## Admin Endpoints

- `GET /api/v1/admin/knowledge?query=...&limit=20&offset=0`
- `POST /api/v1/admin/knowledge`
- `POST /api/v1/admin/knowledge/import`（multipart 上传文件并入库）
- `PATCH /api/v1/admin/knowledge/:id`

权限：`role >= 2`

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

### POST /api/v1/admin/knowledge/import

`Content-Type: multipart/form-data`

表单字段：
- `question`：问题（必填）
- `answer`：答案（必填）
- `keywords`：逗号分隔关键词（可选，如 `奖学金,政策`）
- `files`：附件文件（必填，可多文件）

当前支持文件类型：
- `pdf`
- `doc`
- `docx`
- `xls`
- `xlsx`

成功后会自动生成附件链接，格式为：`/uploads/knowledge/<filename>`
并在服务端尝试抽取文档正文写入 `content_text` 用于检索：
- 已支持正文抽取：`docx`、`xlsx`
- 当前不保证正文抽取：`pdf`、`doc`、`xls`（仍可作为附件导入与展示）

管理端错误响应：

- `400 {"error":"invalid body"}`
- `400 {"error":"invalid id"}`
- `400 {"error":"missing fields"}`
- `400 {"error":"missing files"}`
- `400 {"error":"unsupported file type"}`
- `400 {"error":"empty patch"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `500 {"error":"list knowledge failed"}`
- `500 {"error":"patch knowledge failed"}`

## Audit Log

以下管理动作会写入 `admin_logs`：

- `knowledge.create`
- `knowledge.patch`
