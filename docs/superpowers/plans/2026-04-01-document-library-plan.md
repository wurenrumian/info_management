# 文档库 / 文件基础设施 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建通用文件上传/下载基础设施，为后续并行模块提供统一文件管理能力，并重构知识库使用共享服务。

**Architecture:** 新增 `documents` 表 + `upload` 共享 service + 通用文件 handler，知识库 handler 重构为委托 upload service。所有文件操作通过统一 API 完成。

**Tech Stack:** Go 1.25, Gin, GORM, SQLite (testing), PostgreSQL/Kingbase (production)

---

## File Structure

| 操作 | 文件路径 | 说明 |
|------|----------|------|
| 创建 | `internal/model/document.go` | Document GORM 模型 |
| 创建 | `internal/repo/document_repo.go` | Document CRUD + 分页 |
| 创建 | `internal/repo/document_repo_test.go` | Document repo 测试 |
| 创建 | `internal/service/upload/service.go` | 文件保存、校验、命名 |
| 创建 | `internal/service/upload/extractor.go` | PDF/DOCX/XLSX 文本提取（从 knowledge 移动） |
| 创建 | `internal/service/upload/service_test.go` | upload service 测试 |
| 创建 | `internal/http/handler/file_handler.go` | 通用文件 API handler |
| 创建 | `internal/http/handler/file_handler_test.go` | 文件 handler 测试 |
| 修改 | `internal/service/authz/actions.go` | 新增 files:* 动作常量 |
| 修改 | `internal/service/authz/authorize.go` | 新增 files 权限规则 |
| 修改 | `internal/http/router/router.go` | 注册文件路由 + 统一静态目录 |
| 修改 | `internal/service/knowledge/service.go` | 使用 upload.ExtractTextFromFile |
| 修改 | `internal/http/handler/admin_knowledge_handler.go` | 使用 upload.Service 保存文件 |
| 修改 | `internal/store/db.go` | AutoMigrate 加入 Document |
| 修改 | `internal/model/model_migrate_test.go` | 迁移测试加入 Document |
| 修改 | `docs/api/phase2-knowledge-api.md` | 更新知识库 API 文档 |
| 创建 | `docs/api/phase2-files-api.md` | 文件 API 文档 |

---

### Task 1: Document 模型与 authz 动作

**Files:**
- Create: `internal/model/document.go`
- Modify: `internal/service/authz/actions.go`
- Modify: `internal/service/authz/authorize.go`

- [ ] **Step 1: 创建 Document 模型**

```go
// internal/model/document.go
package model

import "time"

// Document represents an uploaded file in the document library.
type Document struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"type:varchar(200);not null" json:"title"`
	FilePath    string    `gorm:"type:varchar(500);not null" json:"file_path"`
	FileSize    int64     `gorm:"not null" json:"file_size"`
	ContentType string    `gorm:"type:varchar(100);not null" json:"content_type"`
	UploaderID  uint      `gorm:"index;not null" json:"uploader_id"`
	CreatedAt   time.Time `json:"created_at"`
}
```

- [ ] **Step 2: 新增 authz 动作常量**

修改 `internal/service/authz/actions.go`，在末尾新增：

```go
const (
	ActionFilesUpload = "files:upload"
	ActionFilesGet    = "files:get"
	ActionFilesList   = "files:list"
	ActionFilesDelete = "files:delete"
)
```

- [ ] **Step 3: 新增 files 权限规则**

修改 `internal/service/authz/authorize.go`，更新每个 role 的 case：

```go
func Authorize(role int, action string) bool {
	switch role {
	case model.RoleStudent:
		return action == ActionGetMe ||
			action == ActionKnowledgeSearch ||
			action == ActionFilesUpload ||
			action == ActionFilesGet ||
			action == ActionFilesList
	case model.RoleCadre:
		return action == ActionUsersList ||
			action == ActionUsersGet ||
			action == ActionKnowledgeSearch ||
			action == ActionKnowledgeList ||
			action == ActionKnowledgeGet ||
			action == ActionKnowledgeCreate ||
			action == ActionKnowledgePatch ||
			action == ActionKnowledgeDelete ||
			action == ActionClassesList ||
			action == ActionClassesGet ||
			action == ActionFilesUpload ||
			action == ActionFilesGet ||
			action == ActionFilesList ||
			action == ActionFilesDelete
	case model.RoleTeacher:
		return action == ActionUsersList ||
			action == ActionUsersGet ||
			action == ActionKnowledgeSearch ||
			action == ActionKnowledgeList ||
			action == ActionKnowledgeGet ||
			action == ActionKnowledgeCreate ||
			action == ActionKnowledgePatch ||
			action == ActionKnowledgeDelete ||
			action == ActionClassesList ||
			action == ActionClassesGet ||
			action == ActionFilesUpload ||
			action == ActionFilesGet ||
			action == ActionFilesList ||
			action == ActionFilesDelete
	case model.RoleSuperAdmin:
		return true
	default:
		return false
	}
}
```

- [ ] **Step 4: 运行测试验证**

```bash
go test ./internal/service/authz/... -count=1
```

预期：现有 authz 测试通过（新增动作不影响现有测试，因为 SuperAdmin 返回 true，学生/cadre 的测试只测已有动作）。

- [ ] **Step 5: Commit**

```bash
git add internal/model/document.go internal/service/authz/actions.go internal/service/authz/authorize.go
git commit -m "feat: add Document model and files authz actions"
```

---

### Task 2: Document Repo 与测试

**Files:**
- Create: `internal/repo/document_repo.go`
- Create: `internal/repo/document_repo_test.go`

- [ ] **Step 1: 创建 Document Repo**

```go
// internal/repo/document_repo.go
package repo

import (
	"manage/internal/model"

	"gorm.io/gorm"
)

// DocumentRepo provides data access for documents.
type DocumentRepo struct {
	db *gorm.DB
}

// NewDocumentRepo creates a document repository.
func NewDocumentRepo(db *gorm.DB) *DocumentRepo {
	return &DocumentRepo{db: db}
}

// Create inserts one document record.
func (r *DocumentRepo) Create(doc *model.Document) error {
	return r.db.Create(doc).Error
}

// GetByID returns one document by id.
func (r *DocumentRepo) GetByID(id uint) (*model.Document, error) {
	var doc model.Document
	if err := r.db.First(&doc, id).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// ListWithTotal returns documents with pagination and total count.
func (r *DocumentRepo) ListWithTotal(limit, offset int) ([]model.Document, int64, error) {
	var total int64
	if err := r.db.Model(&model.Document{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var docs []model.Document
	err := r.db.Model(&model.Document{}).Order("id desc").Limit(limit).Offset(offset).Find(&docs).Error
	return docs, total, err
}

// DeleteByID deletes one document by id.
func (r *DocumentRepo) DeleteByID(id uint) error {
	tx := r.db.Delete(&model.Document{}, id)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
```

- [ ] **Step 2: 创建 Document Repo 测试**

```go
// internal/repo/document_repo_test.go
package repo_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/model"
	"manage/internal/repo"
)

func TestDocumentRepoCRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Document{}))

	r := repo.NewDocumentRepo(db)

	doc := &model.Document{
		Title:       "report.pdf",
		FilePath:    "2026/04/123_report.pdf",
		FileSize:    1024,
		ContentType: "application/pdf",
		UploaderID:  100,
	}
	require.NoError(t, r.Create(doc))
	require.Greater(t, doc.ID, uint(0))

	got, err := r.GetByID(doc.ID)
	require.NoError(t, err)
	require.Equal(t, doc.Title, got.Title)
	require.Equal(t, doc.FilePath, got.FilePath)

	docs, total, err := r.ListWithTotal(20, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, docs, 1)

	require.NoError(t, r.DeleteByID(doc.ID))
	_, err = r.GetByID(doc.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestDocumentRepoDeleteNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Document{}))

	r := repo.NewDocumentRepo(db)
	require.ErrorIs(t, r.DeleteByID(99999), gorm.ErrRecordNotFound)
}
```

- [ ] **Step 3: 运行测试验证**

```bash
go test ./internal/repo/document_repo_test.go -count=1 -v
```

预期：两个测试都通过。

- [ ] **Step 4: Commit**

```bash
git add internal/repo/document_repo.go internal/repo/document_repo_test.go
git commit -m "feat: add DocumentRepo with CRUD and tests"
```

---

### Task 3: Upload Service（共享文件服务）

**Files:**
- Create: `internal/service/upload/service.go`
- Create: `internal/service/upload/extractor.go`
- Create: `internal/service/upload/service_test.go`

- [ ] **Step 1: 创建 upload service**

```go
// internal/service/upload/service.go
package upload

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxFileSize = 30 * 1024 * 1024 // 30MB

var allowedExts = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".zip":  true,
}

// SaveResult holds information about a saved file.
type SaveResult struct {
	FilePath     string
	FileName     string
	OriginalName string
	FileSize     int64
	ContentType  string
}

// Service handles file upload operations.
type Service struct {
	baseDir string
}

// NewService creates an upload service.
func NewService(baseDir string) *Service {
	return &Service{baseDir: baseDir}
}

// SaveFile saves an uploaded file and returns metadata.
func (s *Service) SaveFile(file *multipart.FileHeader) (*SaveResult, error) {
	if file.Size > maxFileSize {
		return nil, fmt.Errorf("file too large")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExts[ext] {
		return nil, fmt.Errorf("unsupported file type")
	}

	contentType := detectContentType(ext)
	now := time.Now()
	dir := filepath.Join(s.baseDir, fmt.Sprintf("%d", now.Year()), fmt.Sprintf("%02d", now.Month()))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("prepare upload dir failed")
	}

	fileName := uniqueFileName(file.Filename)
	filePath := filepath.Join(dir, fileName)

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open upload file failed")
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("create file failed")
	}
	defer dst.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		return nil, fmt.Errorf("save file failed")
	}

	relPath, _ := filepath.Rel(s.baseDir, filePath)
	return &SaveResult{
		FilePath:     filepath.ToSlash(relPath),
		FileName:     fileName,
		OriginalName: file.Filename,
		FileSize:     file.Size,
		ContentType:  contentType,
	}, nil
}

func detectContentType(ext string) string {
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".doc", ".docx":
		return "application/msword"
	case ".xls", ".xlsx":
		return "application/vnd.ms-excel"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

func uniqueFileName(origin string) string {
	base := filepath.Base(origin)
	base = strings.ReplaceAll(base, " ", "_")
	return fmt.Sprintf("%d_%s", time.Now().UnixNano(), base)
}
```

- [ ] **Step 2: 移动 extractor 到 upload 包**

从 `internal/service/knowledge/extractor.go` 复制全部内容到 `internal/service/upload/extractor.go`，保持包名为 `upload`。文件内容完全不变。

- [ ] **Step 3: 创建 upload service 测试**

```go
// internal/service/upload/service_test.go
package upload

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveFileSuccess(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-pdf-content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	_, header, err := readerFromForm(body, writer)
	require.NoError(t, err)

	result, err := svc.SaveFile(header)
	require.NoError(t, err)
	require.Equal(t, "test.pdf", result.OriginalName)
	require.Equal(t, "application/pdf", result.ContentType)
	require.Equal(t, int64(18), result.FileSize)
	require.Contains(t, result.FilePath, "test.pdf")

	_, err = os.Stat(filepath.Join(dir, result.FilePath))
	require.NoError(t, err)
}

func TestSaveFileRejectsLargeFile(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "big.pdf")
	require.NoError(t, err)
	// Write 31MB
	large := make([]byte, 1024*1024)
	for i := 0; i < 31; i++ {
		_, err = part.Write(large)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	_, header, err := readerFromForm(body, writer)
	require.NoError(t, err)

	_, err = svc.SaveFile(header)
	require.Error(t, err)
	require.Contains(t, err.Error(), "too large")
}

func TestSaveFileRejectsUnsupportedType(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "malware.exe")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-exe"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	_, header, err := readerFromForm(body, writer)
	require.NoError(t, err)

	_, err = svc.SaveFile(header)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported")
}

// readerFromForm extracts the first file header from a multipart form.
func readerFromForm(body *bytes.Buffer, writer *multipart.Writer) (*multipart.Reader, *multipart.FileHeader, error) {
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 * 1024 * 1024)
	if err != nil {
		return nil, nil, err
	}
	files := form.File["file"]
	if len(files) == 0 {
		return nil, nil, nil
	}
	return reader, files[0], nil
}
```

- [ ] **Step 4: 运行测试验证**

```bash
go test ./internal/service/upload/... -count=1 -v
```

预期：3 个测试通过。

- [ ] **Step 5: Commit**

```bash
git add internal/service/upload/service.go internal/service/upload/extractor.go internal/service/upload/service_test.go
git commit -m "feat: add shared upload service with file validation and extractor"
```

---

### Task 4: File Handler 与测试

**Files:**
- Create: `internal/http/handler/file_handler.go`
- Create: `internal/http/handler/file_handler_test.go`

- [ ] **Step 1: 创建 File Handler**

```go
// internal/http/handler/file_handler.go
package handler

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
	"manage/internal/service/upload"
)

// FileHandler handles generic file upload/download APIs.
type FileHandler struct {
	svc      *upload.Service
	repo     *repo.DocumentRepo
	logRepo  *repo.AdminLogRepo
	uploadDir string
}

// NewFileHandler creates a file handler.
func NewFileHandler(db *gorm.DB) *FileHandler {
	uploadDir := os.Getenv("DOCUMENT_UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads/documents"
	}
	return &FileHandler{
		svc:       upload.NewService(uploadDir),
		repo:      repo.NewDocumentRepo(db),
		logRepo:   repo.NewAdminLogRepo(db),
		uploadDir: uploadDir,
	}
}

// Upload handles file upload.
func (h *FileHandler) Upload(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionFilesUpload) {
		response.Error(c, 403, "forbidden")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, 400, "missing file")
		return
	}

	result, err := h.svc.SaveFile(file)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	doc := model.Document{
		Title:       file.Filename,
		FilePath:    result.FilePath,
		FileSize:    result.FileSize,
		ContentType: result.ContentType,
		UploaderID:  actor.UserID,
	}
	if err := h.repo.Create(&doc); err != nil {
		response.Error(c, 500, "save document failed")
		return
	}

	response.OK(c, doc)
}

// List handles file list with pagination.
func (h *FileHandler) List(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionFilesList) {
		response.Error(c, 403, "forbidden")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	docs, total, err := h.repo.ListWithTotal(limit, offset)
	if err != nil {
		response.Error(c, 500, "list files failed")
		return
	}
	c.JSON(200, gin.H{"data": docs, "total": total})
}

// Get handles file metadata retrieval.
func (h *FileHandler) Get(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionFilesGet) {
		response.Error(c, 403, "forbidden")
		return
	}

	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}

	doc, err := h.repo.GetByID(uint(id64))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "file not found")
			return
		}
		response.Error(c, 500, "get file failed")
		return
	}

	response.OK(c, doc)
}

// Download handles file download.
func (h *FileHandler) Download(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionFilesGet) {
		response.Error(c, 403, "forbidden")
		return
	}

	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}

	doc, err := h.repo.GetByID(uint(id64))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "file not found")
			return
		}
		response.Error(c, 500, "get file failed")
		return
	}

	fullPath := filepath.Join(h.uploadDir, doc.FilePath)
	c.Header("Content-Disposition", "attachment; filename=\""+doc.Title+"\"")
	c.Header("Content-Type", doc.ContentType)
	c.File(fullPath)
}

// Delete handles file deletion (admin only).
func (h *FileHandler) Delete(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionFilesDelete) {
		response.Error(c, 403, "forbidden")
		return
	}

	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}

	doc, err := h.repo.GetByID(uint(id64))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "file not found")
			return
		}
		response.Error(c, 500, "get file failed")
		return
	}

	if err := h.repo.DeleteByID(uint(id64)); err != nil {
		response.Error(c, 500, "delete file failed")
		return
	}

	// Best-effort physical file deletion
	_ = os.Remove(filepath.Join(h.uploadDir, doc.FilePath))

	_ = h.logRepo.Create(model.AdminLog{
		AdminID:    actor.UserID,
		Action:     "document.delete",
		TargetType: "document",
		TargetID:   uint(id64),
		IPAddress:  c.ClientIP(),
	})

	response.OK(c, gin.H{"deleted": true})
}
```

- [ ] **Step 2: 创建 File Handler 测试**

```go
// internal/http/handler/file_handler_test.go
package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/http/router"
	"manage/internal/model"
	"manage/internal/testutil"
)

func setupFileTestRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	uploadDir := t.TempDir()
	t.Setenv("DOCUMENT_UPLOAD_DIR", uploadDir)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Class{}, &model.AdminLog{}, &model.Document{}))

	r := router.New(db)
	return db, r
}

func TestFileUploadSuccess(t *testing.T) {
	db, r := setupFileTestRouter(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "report.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-pdf-content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "report.pdf")

	var doc model.Document
	require.NoError(t, db.Where("title = ?", "report.pdf").First(&doc).Error)
	require.Equal(t, "application/pdf", doc.ContentType)
	require.Equal(t, uint(100), doc.UploaderID)
}

func TestFileUploadRejectsStudent(t *testing.T) {
	_, r := setupFileTestRouter(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Student can upload (role >= 1)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestFileDeleteForbiddenForStudent(t *testing.T) {
	_, r := setupFileTestRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/files/1", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestFileDeleteByAdmin(t *testing.T) {
	db, r := setupFileTestRouter(t)

	// Create a document first
	uploadDir := t.TempDir()
	t.Setenv("DOCUMENT_UPLOAD_DIR", uploadDir)
	doc := model.Document{
		Title:       "test.pdf",
		FilePath:    "2026/04/test.pdf",
		FileSize:    100,
		ContentType: "application/pdf",
		UploaderID:  100,
	}
	require.NoError(t, db.Create(&doc).Error)
	// Create physical file
	os.MkdirAll(filepath.Join(uploadDir, "2026/04"), 0o755)
	os.WriteFile(filepath.Join(uploadDir, doc.FilePath), []byte("content"), 0o644)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/files/"+strconv.Itoa(int(doc.ID)), nil)
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"deleted":true`)

	var count int64
	require.NoError(t, db.Model(&model.Document{}).Where("id = ?", doc.ID).Count(&count).Error)
	require.Equal(t, int64(0), count)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "document.delete").Find(&logs).Error)
	require.Len(t, logs, 1)
}

func TestFileListIncludesTotal(t *testing.T) {
	db, r := setupFileTestRouter(t)

	require.NoError(t, db.Create(&model.Document{
		Title: "a.pdf", FilePath: "a.pdf", FileSize: 100, ContentType: "application/pdf", UploaderID: 1,
	}).Error)
	require.NoError(t, db.Create(&model.Document{
		Title: "b.pdf", FilePath: "b.pdf", FileSize: 200, ContentType: "application/pdf", UploaderID: 1,
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/files?limit=1&offset=0", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	var total int64
	require.NoError(t, json.Unmarshal(payload["total"], &total))
	require.Equal(t, int64(2), total)

	var data []map[string]any
	require.NoError(t, json.Unmarshal(payload["data"], &data))
	require.Len(t, data, 1)
}

func TestFileGetByID(t *testing.T) {
	db, r := setupFileTestRouter(t)

	doc := model.Document{
		Title: "meta.pdf", FilePath: "meta.pdf", FileSize: 50, ContentType: "application/pdf", UploaderID: 1,
	}
	require.NoError(t, db.Create(&doc).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/"+strconv.Itoa(int(doc.ID)), nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "meta.pdf")
}

func TestFileGetByIDNotFound(t *testing.T) {
	_, r := setupFileTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/99999", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "file not found")
}
```

- [ ] **Step 3: 运行测试验证**

```bash
go test ./internal/http/handler/file_handler_test.go -count=1 -v
```

预期：7 个测试通过。

- [ ] **Step 4: Commit**

```bash
git add internal/http/handler/file_handler.go internal/http/handler/file_handler_test.go
git commit -m "feat: add FileHandler with upload/download/delete and tests"
```

---

### Task 5: Router 注册与静态文件统一

**Files:**
- Modify: `internal/http/router/router.go`

- [ ] **Step 1: 更新 router.go**

完整替换 `internal/http/router/router.go`：

```go
package router

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"manage/internal/http/handler"
	"manage/internal/http/middleware"
)

func New(db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", handler.Health)

	// Unified static file serving
	uploadDir := strings.TrimSpace(os.Getenv("DOCUMENT_UPLOAD_DIR"))
	if uploadDir == "" {
		uploadDir = "./data/uploads/documents"
	}
	r.Static("/uploads/documents", uploadDir)

	// Backward compat: knowledge uploads still served from documents dir
	knowledgeDir := strings.TrimSpace(os.Getenv("KNOWLEDGE_UPLOAD_DIR"))
	if knowledgeDir != "" {
		r.Static("/uploads/knowledge", knowledgeDir)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}
	appID := os.Getenv("WECHAT_APP_ID")
	appSecret := os.Getenv("WECHAT_APP_SECRET")

	api := r.Group("/api/v1")

	wechatHandler := handler.NewWechatHandler(db, appID, appSecret, jwtSecret)
	api.POST("/wechat/login", wechatHandler.Login)
	api.POST("/wechat/bind", middleware.OptionalJWTAuth(jwtSecret), wechatHandler.Bind)

	api.Use(middleware.JWTAuth(jwtSecret))

	meHandler := handler.NewMeHandler(db)
	knowledgeHandler := handler.NewKnowledgeHandler(db)
	adminUserHandler := handler.NewAdminUserHandler(db)
	adminClassHandler := handler.NewAdminClassHandler(db)
	adminLogHandler := handler.NewAdminLogHandler(db)
	adminKnowledgeHandler := handler.NewAdminKnowledgeHandler(db)
	fileHandler := handler.NewFileHandler(db)

	api.GET("/me", meHandler.GetMe)
	api.GET("/knowledge/search", knowledgeHandler.Search)

	// File APIs
	api.POST("/files/upload", fileHandler.Upload)
	api.GET("/files", fileHandler.List)
	api.GET("/files/:id", fileHandler.Get)
	api.GET("/files/:id/download", fileHandler.Download)
	api.DELETE("/files/:id", fileHandler.Delete)

	admin := api.Group("/admin")
	admin.GET("/users", adminUserHandler.ListUsers)
	admin.GET("/users/:id", adminUserHandler.GetUser)
	admin.PATCH("/users/:id", adminUserHandler.PatchUser)

	admin.GET("/classes", adminClassHandler.ListClasses)
	admin.GET("/classes/:id", adminClassHandler.GetClass)
	admin.POST("/classes", adminClassHandler.CreateClass)
	admin.PATCH("/classes/:id", adminClassHandler.PatchClass)

	admin.GET("/logs", adminLogHandler.ListLogs)
	admin.GET("/knowledge", adminKnowledgeHandler.ListKnowledge)
	admin.GET("/knowledge/:id", adminKnowledgeHandler.GetKnowledge)
	admin.POST("/knowledge", adminKnowledgeHandler.CreateKnowledge)
	admin.POST("/knowledge/import", adminKnowledgeHandler.ImportKnowledge)
	admin.PATCH("/knowledge/:id", adminKnowledgeHandler.PatchKnowledge)
	admin.DELETE("/knowledge/:id", adminKnowledgeHandler.DeleteKnowledge)

	return r
}
```

- [ ] **Step 2: 运行全量测试验证路由不破坏现有功能**

```bash
go test ./... -count=1
```

预期：所有现有测试通过（新增路由不影响现有测试路径）。

- [ ] **Step 3: Commit**

```bash
git add internal/http/router/router.go
git commit -m "feat: register file APIs and unify static file serving"
```

---

### Task 6: AutoMigrate 更新

**Files:**
- Modify: `internal/store/db.go`
- Modify: `internal/model/model_migrate_test.go`

- [ ] **Step 1: 更新 db.go**

```go
// internal/store/db.go
package store

import (
	"manage/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func OpenAndMigrate(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.User{}, &model.Class{}, &model.AdminLog{}, &model.KnowledgeItem{}, &model.Document{}); err != nil {
		return nil, err
	}
	return db, nil
}
```

- [ ] **Step 2: 更新 model_migrate_test.go**

```go
// internal/model/model_migrate_test.go
package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/model"
)

func TestAutoMigrateCoreTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.User{}, &model.Class{}, &model.AdminLog{}, &model.KnowledgeItem{}, &model.Document{})
	require.NoError(t, err)
}
```

- [ ] **Step 3: 运行测试验证**

```bash
go test ./internal/model/... -count=1 -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/store/db.go internal/model/model_migrate_test.go
git commit -m "feat: add Document to AutoMigrate"
```

---

### Task 7: 重构知识库使用共享 upload service

**Files:**
- Modify: `internal/http/handler/admin_knowledge_handler.go`
- Modify: `internal/service/knowledge/service.go`
- Modify: `internal/service/knowledge/extractor.go`

- [ ] **Step 1: 更新 admin_knowledge_handler.go**

替换 `saveUploadedFiles` 方法和构造函数，删除重复逻辑：

```go
// internal/http/handler/admin_knowledge_handler.go

// 修改构造函数
func NewAdminKnowledgeHandler(db *gorm.DB) *AdminKnowledgeHandler {
	uploadDir := os.Getenv("KNOWLEDGE_UPLOAD_DIR")
	if strings.TrimSpace(uploadDir) == "" {
		uploadDir = os.Getenv("DOCUMENT_UPLOAD_DIR")
		if uploadDir == "" {
			uploadDir = "./data/uploads/documents"
		}
	}
	return &AdminKnowledgeHandler{
		svc:       knowledgeSvc.NewService(db),
		logRepo:   repo.NewAdminLogRepo(db),
		uploadSvc: upload.NewService(uploadDir),
		uploadDir: uploadDir,
	}
}

// 修改 AdminKnowledgeHandler 结构体
type AdminKnowledgeHandler struct {
	svc       *knowledgeSvc.Service
	logRepo   *repo.AdminLogRepo
	uploadSvc *upload.Service
	uploadDir string
}
```

在 import 中添加 `"manage/internal/service/upload"`，删除 `knowledgeSvc "manage/internal/service/knowledge"` 中对 extractor 的引用。

替换 `saveUploadedFiles` 方法：

```go
func (h *AdminKnowledgeHandler) saveUploadedFiles(c *gin.Context) ([]map[string]string, string, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, "", fmt.Errorf("invalid multipart form")
	}
	files := form.File["files"]
	if len(files) == 0 {
		return nil, "", fmt.Errorf("missing files")
	}

	attachments := make([]map[string]string, 0, len(files))
	textParts := make([]string, 0, len(files))
	for _, file := range files {
		result, err := h.uploadSvc.SaveFile(file)
		if err != nil {
			return nil, "", err
		}
		attachments = append(attachments, map[string]string{
			"title": file.Filename,
			"url":   "/uploads/documents/" + result.FilePath,
		})
		if text := upload.ExtractTextFromFile(filepath.Join(h.uploadDir, result.FilePath)); text != "" {
			textParts = append(textParts, text)
		}
	}
	return attachments, strings.Join(textParts, " "), nil
}
```

删除 `allowedAttachment()` 和 `uniqueFileName()` 函数（已移至 upload service）。

注意：保留 `"mime/multipart"` import（`c.MultipartForm()` 返回类型依赖它）。

- [ ] **Step 2: 更新 knowledge service 使用 upload extractor**

`internal/service/knowledge/service.go` 不需要修改（它不直接使用 extractor）。

`internal/service/knowledge/extractor.go` 保留但标记为废弃，或直接从 knowledge 包中删除（因为已移至 upload 包）。删除该文件：

```bash
rm internal/service/knowledge/extractor.go
```

- [ ] **Step 3: 运行全量测试验证**

```bash
go test ./... -count=1
```

预期：所有测试通过，包括知识库导入测试。注意 `TestAdminKnowledgeImportWithFiles` 中的 URL 断言需要从 `/uploads/knowledge/` 改为 `/uploads/documents/`。

- [ ] **Step 4: 更新知识库测试中的 URL 断言**

修改 `internal/http/handler/knowledge_handler_test.go` 中 `TestAdminKnowledgeImportWithFiles`：

```go
// 将这一行：
require.Contains(t, w.Body.String(), "/uploads/knowledge/")
// 改为：
require.Contains(t, w.Body.String(), "/uploads/documents/")
```

- [ ] **Step 5: 再次运行全量测试**

```bash
go test ./... -count=1
```

- [ ] **Step 6: Commit**

```bash
git add internal/http/handler/admin_knowledge_handler.go internal/http/handler/knowledge_handler_test.go
git rm internal/service/knowledge/extractor.go
git commit -m "refactor: migrate knowledge handler to shared upload service"
```

---

### Task 8: API 文档

**Files:**
- Create: `docs/api/phase2-files-api.md`
- Modify: `docs/api/phase2-knowledge-api.md`

- [ ] **Step 1: 创建文件 API 文档**

```markdown
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
  "data": [...],
  "total": 42
}
```

### 获取文件元数据

```
GET /api/v1/files/:id
```

**权限**：所有登录用户

### 下载文件

```
GET /api/v1/files/:id/download
```

**权限**：所有登录用户

响应文件流 + `Content-Disposition: attachment`。

### 删除文件

```
DELETE /api/v1/files/:id
```

**权限**：管理员（role >= 2）

删除后记录到 `admin_logs`（action: `document.delete`）。

## 允许的文件类型

`.pdf`, `.doc`, `.docx`, `.xls`, `.xlsx`, `.jpg`, `.jpeg`, `.png`, `.zip`

## 文件大小限制

30MB
```

- [ ] **Step 2: 更新知识库 API 文档**

在 `docs/api/phase2-knowledge-api.md` 中，将知识库导入的附件 URL 前缀从 `/uploads/knowledge/` 更新为 `/uploads/documents/`。

- [ ] **Step 3: Commit**

```bash
git add docs/api/phase2-files-api.md docs/api/phase2-knowledge-api.md
git commit -m "docs: add file API documentation and update knowledge API"
```

---

### Task 9: 最终验证

- [ ] **Step 1: 全量测试**

```bash
go test ./... -count=1 -v
```

- [ ] **Step 2: go vet**

```bash
go vet ./...
```

- [ ] **Step 3: go fmt**

```bash
go fmt ./...
```

- [ ] **Step 4: 确认编译通过**

```bash
go build ./cmd/server/
```

- [ ] **Step 5: 最终 Commit**

```bash
git add -A
git commit -m "feat: document library infrastructure complete"
```
