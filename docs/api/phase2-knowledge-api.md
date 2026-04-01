# Phase2 Knowledge API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`（JWT token，由微信登录接口返回）

## Student Endpoint

- `GET /api/v1/knowledge/search?q=...&limit=20&offset=0`

权限：`role >= 1`

搜索范围：
- `question`
- `answer`
- `keywords`
- `content_text`（由导入文件抽取出的正文文本）

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
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`

## Admin Endpoints

- `GET /api/v1/admin/knowledge?query=...&limit=20&offset=0`
- `GET /api/v1/admin/knowledge/:id`
- `POST /api/v1/admin/knowledge`
- `POST /api/v1/admin/knowledge/import`（multipart 上传文件并入库）
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

### POST /api/v1/admin/knowledge/import

`Content-Type: multipart/form-data`

表单字段：
- `question`：问题（必填）
- `answer`：答案（必填）
- `keywords`：逗号分隔关键词（可选，如 `奖学金,政策`）
- `files`：附件文件（必填，可多文件）

当前支持文件类型：
- `pdf`（已支持正文抽取）
- `doc`
- `docx`
- `xls`
- `xlsx`

成功后会自动生成附件链接，格式为：`/uploads/documents/<relative_path>`
并在服务端尝试抽取文档正文写入 `content_text` 用于检索：
- 已支持正文抽取：`docx`、`xlsx`、`pdf`
- 当前不保证正文抽取：`doc`、`xls`（仍可作为附件导入与展示）

PDF 正文抽取说明：
- 需要 PDF 包含可复制文本层（非扫描件）
- 逐页提取，合并空格；无文本则不影响附件保存

管理端错误响应：

- `400 {"error":"invalid body"}`
- `400 {"error":"invalid id"}`
- `400 {"error":"missing fields"}`
- `400 {"error":"missing files"}`
- `400 {"error":"unsupported file type"}`
- `400 {"error":"empty patch"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"knowledge not found"}`
- `500 {"error":"list knowledge failed"}`
- `500 {"error":"get knowledge failed"}`
- `500 {"error":"patch knowledge failed"}`
- `500 {"error":"delete knowledge failed"}`

## Audit Log

以下管理动作会写入 `admin_logs`：

- `knowledge.create`
- `knowledge.import`
- `knowledge.patch`
- `knowledge.delete`
