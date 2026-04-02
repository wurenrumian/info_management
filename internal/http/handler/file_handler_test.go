package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
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

	doc := model.Document{
		Title:       "test.pdf",
		FilePath:    "2026/04/test.pdf",
		FileSize:    100,
		ContentType: "application/pdf",
		UploaderID:  100,
	}
	require.NoError(t, db.Create(&doc).Error)

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
