package handler_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
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

func TestFileUploadWithScene(t *testing.T) {
	db, r := setupFileTestRouter(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("scene", "avatar"))
	part, err := writer.CreateFormFile("file", "avatar.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-png-content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "avatar.png")

	var doc model.Document
	require.NoError(t, db.Where("title = ?", "avatar.png").First(&doc).Error)
	require.Contains(t, doc.FilePath, "avatars/")
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

func TestFileDeleteForbiddenForTeacher(t *testing.T) {
	_, r := setupFileTestRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/files/1", nil)
	token := testutil.GenerateTestToken(100, 3, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestFileDeleteBySuperAdmin(t *testing.T) {
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
	token := testutil.GenerateTestToken(200, 4, 0, "")
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

func TestFileSearchByContentText(t *testing.T) {
	db, r := setupFileTestRouter(t)

	require.NoError(t, db.Create(&model.Document{
		Title:       "奖学金说明.docx",
		FilePath:    "documents/2026/04/a.docx",
		FileSize:    123,
		ContentType: "application/msword",
		ContentText: "奖学金申请需要提交综测排名证明和成绩单",
		UploaderID:  100,
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/search?q=综测排名证明", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "奖学金说明.docx")
	require.Contains(t, w.Body.String(), "\"url\":\"/uploads/documents/")
	require.Contains(t, w.Body.String(), "综测排名证明")
}

func TestFileUploadExtractsContentTextForDocx(t *testing.T) {
	db, r := setupFileTestRouter(t)
	docxContent, err := buildDocxForFileUpload("奖学金申请需要提交综测排名证明")
	require.NoError(t, err)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "scholarship.docx")
	require.NoError(t, err)
	_, err = part.Write(docxContent)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var doc model.Document
	require.NoError(t, db.Where("title = ?", "scholarship.docx").First(&doc).Error)
	require.Contains(t, doc.ContentText, "综测排名证明")
}

func buildDocxForFileUpload(text string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body>
</w:document>`,
	}

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(w, content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
