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
	return s.SaveFileWithScene(file, "")
}

// SaveFileWithScene saves an uploaded file using an optional business scene for directory routing.
func (s *Service) SaveFileWithScene(file *multipart.FileHeader, scene string) (*SaveResult, error) {
	if file.Size > maxFileSize {
		return nil, fmt.Errorf("file too large")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExts[ext] {
		return nil, fmt.Errorf("unsupported file type")
	}

	contentType := detectContentType(ext)
	now := time.Now()
	category := resolveCategory(scene, ext)
	dir := filepath.Join(s.baseDir, category, fmt.Sprintf("%d", now.Year()), fmt.Sprintf("%02d", now.Month()))
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

func resolveCategory(scene, ext string) string {
	switch strings.TrimSpace(strings.ToLower(scene)) {
	case "avatar", "avatars":
		return "avatars"
	case "knowledge":
		return "knowledge"
	case "announcement", "announcements":
		return "announcements"
	case "document", "documents":
		return "documents"
	}

	switch ext {
	case ".jpg", ".jpeg", ".png":
		return "images"
	default:
		return "documents"
	}
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
