# Phase 2-0 文件 API

## 通用文件上传/下载

### 健康检查与静态访问

- `GET /healthz`：健康检查
- `GET /uploads/<file_path>`：静态附件访问（例如 `/uploads/knowledge/2026/04/xxx.pdf`）

### 上传文件

```
POST /api/v1/files/upload
Content-Type: multipart/form-data

file: <binary>
scene: <optional, avatar|knowledge|announcement|document>
```

**权限**：所有登录用户（role >= 1）

**响应**：
```json
{
  "data": {
    "id": 1,
    "title": "report.pdf",
    "file_path": "knowledge/2026/04/1712000000000000000_report.pdf",
    "file_size": 1048576,
    "content_type": "application/pdf",
    "uploader_id": 5,
    "created_at": "2026-04-01T10:00:00Z"
  }
}
```

**错误**：
- 400: `missing file` / `file too large` / `unsupported file type`
- 401: 未认证
- 403: 无权限

说明：
- `scene` 为可选字段，后端可据此进行分目录存储（例如 `avatars/`、`knowledge/`）。
- 若不传 `scene`，后端会按文件类型自动分流（图片优先进入 `images/`，其余进入 `documents/`）。

### 文件列表

```
GET /api/v1/files?limit=20&offset=0
```

**权限**：所有登录用户

**响应**：
```json
{
  "data": [
    {
      "id": 2,
      "title": "report.pdf",
      "file_path": "knowledge/2026/04/123_report.pdf",
      "file_size": 1048576,
      "content_type": "application/pdf",
      "uploader_id": 5,
      "created_at": "2026-04-01T10:00:00Z"
    }
  ],
  "total": 42
}
```

### 文件检索（按标题 + 文档正文）

```
GET /api/v1/files/search?q=...&limit=20&offset=0
```

**权限**：所有登录用户

搜索范围：
- `title`
- `content_text`（上传时抽取的文档正文，支持分词检索）

**响应**：
```json
{
  "data": [
    {
      "id": 2,
      "title": "scholarship.docx",
      "file_path": "knowledge/2026/04/123_scholarship.docx",
      "file_size": 20480,
      "content_type": "application/msword",
      "uploader_id": 5,
      "url": "/uploads/knowledge/2026/04/123_scholarship.docx",
      "snippet": "奖学金申请需要提交综测排名证明和成绩单"
    }
  ],
  "total": 1
}
```

**错误**：
- 400: `missing q`
- 401: 未认证
- 403: 无权限

### 获取文件元数据

```
GET /api/v1/files/:id
```

**权限**：所有登录用户

**响应**：同上传响应中的 `data` 字段。

**错误**：
- 404: `file not found`

### 下载文件

```
GET /api/v1/files/:id/download
```

**权限**：所有登录用户

响应文件流 + `Content-Disposition: attachment; filename="原始文件名"`。

### 删除文件

```
DELETE /api/v1/files/:id
```

**权限**：超级管理员（role = 4）

**响应**：
```json
{
  "data": {
    "deleted": true
  }
}
```

删除后记录到 `admin_logs`（action: `document.delete`）。

## 允许的文件类型

`.pdf`, `.doc`, `.docx`, `.xls`, `.xlsx`, `.jpg`, `.jpeg`, `.png`, `.zip`

## 文件大小限制

30MB

## 存储路径

文件存储于 `data/uploads/<category>/<YYYY>/<MM>/<unique_filename>`，按业务分类与年月分目录。

## 其他模块使用方式

需要文件附件的模块（党团流程、审批流程、信息发布等）：

1. 前端调用 `POST /api/v1/files/upload` 上传文件，获得 `file_id`
2. 在自身表中以 `jsonb` 字段存储引用，如 `{"file_id": 1, "title": "xxx"}`
3. 前端通过 `GET /api/v1/files/:id/download` 下载
4. 禁止各模块自行实现文件保存逻辑
