# Phase 2-0 文件 API

## 通用文件上传/下载

### 上传文件

```
POST /api/v1/files/upload
Content-Type: multipart/form-data

file: <binary>
```

**权限**：所有登录用户（role >= 1）

**响应**：
```json
{
  "data": {
    "id": 1,
    "title": "report.pdf",
    "file_path": "2026/04/1712000000000000000_report.pdf",
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
      "file_path": "2026/04/123_report.pdf",
      "file_size": 1048576,
      "content_type": "application/pdf",
      "uploader_id": 5,
      "created_at": "2026-04-01T10:00:00Z"
    }
  ],
  "total": 42
}
```

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

**权限**：管理员（role >= 2）

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

文件存储于 `data/uploads/documents/<YYYY>/<MM>/<unique_filename>`，按年月分目录。

## 其他模块使用方式

需要文件附件的模块（党团流程、审批流程、信息发布等）：

1. 前端调用 `POST /api/v1/files/upload` 上传文件，获得 `file_id`
2. 在自身表中以 `jsonb` 字段存储引用，如 `{"file_id": 1, "title": "xxx"}`
3. 前端通过 `GET /api/v1/files/:id/download` 下载
4. 禁止各模块自行实现文件保存逻辑
