# 文档库 — 最小文件基础设施设计（v0）

**日期**：2026-04-01
**阶段**：Phase 2 基础设施（先行模块）

## 1. 目标

构建通用文件上传/下载基础设施，为后续并行模块（党团流程、审批流程、信息发布）提供统一的文件管理能力。当前阶段只实现核心 CRUD API，不包含分类浏览、搜索等前端功能。

## 2. 范围

### 包含

- `documents` 表与 GORM 模型
- 通用文件上传 API（30MB 限制，本地存储）
- 通用文件下载 API（按 ID 获取）
- 文件列表 API（分页）
- 管理员删除文件 API（记录 admin_logs）
- 共享 `upload` service（从知识库抽离）
- 文件文本提取器复用（PDF/DOCX/XLSX，供知识库使用）
- 知识库 handler 重构为使用共享 service
- 统一静态文件路由 `/uploads/documents`

### 不包含

- 分类管理（`category` 字段延后）
- 全文检索/内容搜索
- 分片上传
- 前端浏览/搜索界面
- OSS/云存储适配

## 3. 权限

- 所有登录用户：可上传文件、查看文件元数据、下载文件
- 管理员（role >= 2）：可删除文件
- 删除操作记录到 `admin_logs`（action: `document.delete`）

## 4. 数据模型

新增 `documents` 表：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| title | varchar(200) | 原始文件名 |
| file_path | varchar(500) | 相对存储路径 |
| file_size | bigint | 文件大小（bytes） |
| content_type | varchar(100) | MIME 类型 |
| uploader_id | bigint FK → users | 上传人 |
| created_at | timestamp | 上传时间 |

对比原始技术方案（`original_request/技术方案.md` Section 3.9）：
- 增加 `content_type`：下载时设置正确的 Content-Type header
- 去掉 `category`：最小实现不需要分类，后续可扩展
- 去掉 `updated_at`：文件上传后内容不变

## 5. 存储路径

```
data/uploads/documents/<YYYY>/<MM>/<unique_filename>
```

按年月分目录，避免单目录文件过多。文件名格式：`<unix_nano>_<sanitized_original>`。

## 6. API 端点

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| `POST` | `/api/v1/files/upload` | 登录用户 | 上传文件（multipart，字段名 `file`），返回文件元数据 |
| `GET` | `/api/v1/files` | 登录用户 | 文件列表（分页，`limit`/`offset`），返回 `{"data": [...], "total": N}` |
| `GET` | `/api/v1/files/:id` | 登录用户 | 获取文件元数据 |
| `GET` | `/api/v1/files/:id/download` | 登录用户 | 下载文件（返回文件流 + Content-Disposition） |
| `DELETE` | `/api/v1/files/:id` | 管理员 | 删除文件 + 记录 admin_logs |

### 上传请求格式

```
POST /api/v1/files/upload
Content-Type: multipart/form-data

file: <binary>
```

### 上传响应

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

### 下载响应

```
GET /api/v1/files/:id/download
Content-Type: application/pdf
Content-Disposition: attachment; filename="report.pdf"

<binary data>
```

## 7. 代码结构

```
internal/
├── model/
│   └── document.go              # Document 模型
├── repo/
│   └── document_repo.go         # CRUD + 分页
├── service/
│   ├── upload/
│   │   ├── service.go           # 文件保存、校验、命名（共享）
│   │   └── extractor.go         # PDF/DOCX/XLSX 文本提取（从 knowledge 移动）
│   └── knowledge/
│       └── service.go           # 保留，调用 upload.ExtractTextFromFile
└── http/
    └── handler/
        ├── file_handler.go      # 通用文件 API
        └── admin_knowledge_handler.go  # 重构：使用 upload.Service
```

### `upload.Service` 核心接口

```go
package upload

type SaveResult struct {
    FilePath     string
    FileName     string
    OriginalName string
    FileSize     int64
    ContentType  string
}

type Service struct {
    baseDir string
}

func NewService(baseDir string) *Service
func (s *Service) SaveFile(file *multipart.FileHeader) (*SaveResult, error)
func ExtractTextFromFile(path string) string  // 从 knowledge 包移动
```

### 允许的文件类型

`.pdf`, `.doc`, `.docx`, `.xls`, `.xlsx`, `.jpg`, `.jpeg`, `.png`, `.zip`

### 文件大小限制

30MB（`30 * 1024 * 1024` bytes）

## 8. 知识库重构

`admin_knowledge_handler.go` 的 `saveUploadedFiles()` 方法改为：

1. 调用 `upload.Service.SaveFile()` 保存文件
2. 调用 `upload.ExtractTextFromFile()` 提取文本
3. 删除重复的 `allowedAttachment()`、`uniqueFileName()`、`os.MkdirAll()` 逻辑

handler 构造函数注入 `upload.Service` 实例。

## 9. 路由变更

`router.go` 中：

- 移除硬编码的 `r.Static("/uploads/knowledge", uploadDir)`
- 新增 `r.Static("/uploads/documents", documentUploadDir)`
- 新增文件 API 路由组

## 10. 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DOCUMENT_UPLOAD_DIR` | 文件存储目录 | `./data/uploads/documents` |

`KNOWLEDGE_UPLOAD_DIR` 保留兼容（知识库重构后不再需要，但避免破坏现有部署）。

## 11. authz 动作

新增以下动作常量（`internal/service/authz/actions.go`）：

```go
ActionFilesUpload = "files:upload"
ActionFilesGet    = "files:get"
ActionFilesList   = "files:list"
ActionFilesDelete = "files:delete"
```

权限矩阵：
- `files:upload`, `files:get`, `files:list` → role >= 1（所有登录用户）
- `files:delete` → role >= 2（管理员）

## 12. 测试要求

- handler 测试：至少覆盖参数错误、403 权限拒绝、上传成功、下载成功、删除成功
- repo 测试：覆盖分页查询、按 ID 查找、删除不存在记录
- 至少 1 条 403 用例（学生尝试删除文件）
- 文件大小超限测试（>30MB 拒绝）
- 非法文件类型测试
- 全量通过：`go test ./... -count=1`

## 13. 后续模块使用方式

其他模块需要文件附件时：

1. 前端调用 `POST /api/v1/files/upload` 上传文件，获得 `file_id`
2. 业务模块在自身表中存储 `file_id` 引用（如 `attachments jsonb` 存 `{"file_id": 1, "title": "xxx"}`）
3. 前端通过 `GET /api/v1/files/:id/download` 下载

业务模块不需要自己实现文件上传逻辑。
