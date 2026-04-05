package handler

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/config"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/audit"
	"manage/internal/service/authz"
	ksvc "manage/internal/service/knowledge"
	"manage/internal/service/upload"
)

// FileHandler handles generic file upload/download APIs.
type FileHandler struct {
	svc         *upload.Service
	repo        *repo.DocumentRepo
	auditLogger *audit.Logger
	uploadDir   string
}

type fileSearchItem struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	FilePath    string `json:"file_path"`
	FileSize    int64  `json:"file_size"`
	ContentType string `json:"content_type"`
	UploaderID  uint   `json:"uploader_id"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet"`
}

// NewFileHandler creates a file handler.
func NewFileHandler(db *gorm.DB) *FileHandler {
	uploadDir := config.DocumentUploadDir()
	return &FileHandler{
		svc:         upload.NewService(uploadDir),
		repo:        repo.NewDocumentRepo(db),
		auditLogger: audit.NewLogger(repo.NewAdminLogRepo(db)),
		uploadDir:   uploadDir,
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
	scene := c.PostForm("scene")

	result, err := h.svc.SaveFileWithScene(file, scene)
	if err != nil {
		response.Error(c, 400, "save file failed")
		return
	}

	doc := model.Document{
		Title:       file.Filename,
		FilePath:    result.FilePath,
		FileSize:    result.FileSize,
		ContentType: result.ContentType,
		ContentText: ksvc.ExtractTextFromFile(filepath.Join(h.uploadDir, result.FilePath)),
		UploaderID:  actor.UserID,
	}
	if err := h.repo.Create(&doc); err != nil {
		response.Error(c, 500, "save document failed")
		return
	}

	response.OK(c, doc)
}

// Search handles file search by title/content text.
func (h *FileHandler) Search(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionFilesList) {
		response.Error(c, 403, "forbidden")
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		response.Error(c, 400, "missing q")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	docs, total, err := h.repo.SearchWithTotal(q, limit, offset)
	if err != nil {
		response.Error(c, 500, "search files failed")
		return
	}
	items := make([]fileSearchItem, 0, len(docs))
	for _, doc := range docs {
		items = append(items, fileSearchItem{
			ID:          doc.ID,
			Title:       doc.Title,
			FilePath:    doc.FilePath,
			FileSize:    doc.FileSize,
			ContentType: doc.ContentType,
			UploaderID:  doc.UploaderID,
			URL:         "/uploads/" + doc.FilePath,
			Snippet:     buildSnippet(doc.ContentText, q),
		})
	}
	response.List(c, items, total)
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
	response.List(c, docs, total)
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

	h.auditLogger.Log(c, actor, "document.delete", "document", uint(id64))

	response.OK(c, gin.H{"deleted": true})
}

func buildSnippet(content, query string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	query = strings.TrimSpace(query)
	if query == "" {
		if len([]rune(content)) > 120 {
			return string([]rune(content)[:120])
		}
		return content
	}
	contentLower := strings.ToLower(content)
	queryLower := strings.ToLower(query)
	idx := strings.Index(contentLower, queryLower)
	if idx < 0 {
		if len([]rune(content)) > 120 {
			return string([]rune(content)[:120])
		}
		return content
	}
	start := idx - 30
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 60
	if end > len(content) {
		end = len(content)
	}
	return strings.TrimSpace(content[start:end])
}
