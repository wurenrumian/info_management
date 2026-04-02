package handler

import (
	"errors"
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
	svc       *upload.Service
	repo      *repo.DocumentRepo
	logRepo   *repo.AdminLogRepo
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
		response.Error(c, 400, "save file failed")
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
	response.OK(c, gin.H{
		"data":  docs,
		"total": total,
	})
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

	_ = h.logRepo.Create(&model.AdminLog{
		AdminID:    actor.UserID,
		Action:     "document.delete",
		TargetType: "document",
		TargetID:   uint(id64),
		IPAddress:  c.ClientIP(),
	})

	response.OK(c, gin.H{"deleted": true})
}
