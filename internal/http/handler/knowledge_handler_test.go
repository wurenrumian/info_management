package handler_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/http/router"
	"manage/internal/model"
	"manage/internal/testutil"
)

func setupKnowledgeTestRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}, &model.AdminLog{}, &model.KnowledgeItem{}))

	require.NoError(t, db.Create(&model.KnowledgeItem{
		Question:    "休学申请怎么办理",
		Answer:      "先联系辅导员并提交休学申请表",
		Keywords:    datatypes.JSON(`["休学","申请"]`),
		Attachments: datatypes.JSON(`[{"title":"休学申请表","url":"https://example.com/leave"}]`),
		CreatedBy:   999,
		UpdatedBy:   999,
	}).Error)

	r := router.New(db)
	return db, r
}

func TestKnowledgeSearchByStudent(t *testing.T) {
	_, r := setupKnowledgeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/search?q=休学", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "休学申请怎么办理")
	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	var dataWrap map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload["data"], &dataWrap))
	var total int64
	require.NoError(t, json.Unmarshal(dataWrap["total"], &total))
	require.Equal(t, int64(1), total)
}

func TestAdminKnowledgeGetByIDNotFound(t *testing.T) {
	_, r := setupKnowledgeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/knowledge/99999", nil)
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "knowledge not found")
}

func TestAdminKnowledgeDeleteByID(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)
	var existing model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "休学申请怎么办理").First(&existing).Error)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/knowledge/"+strconv.Itoa(int(existing.ID)), nil)
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"deleted":true`)

	var count int64
	require.NoError(t, db.Model(&model.KnowledgeItem{}).Where("id = ?", existing.ID).Count(&count).Error)
	require.Equal(t, int64(0), count)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "knowledge.delete").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, existing.ID, logs[0].TargetID)
}

func TestAdminKnowledgeDeleteByIDNotFound(t *testing.T) {
	_, r := setupKnowledgeTestRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/knowledge/99999", nil)
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "knowledge not found")
}

func TestAdminKnowledgeImportWithFiles(t *testing.T) {
	uploadDir := t.TempDir()
	t.Setenv("DOCUMENT_UPLOAD_DIR", uploadDir)

	_, r := setupKnowledgeTestRouter(t)

	docxContent, err := buildDocx("奖学金申请需要提交综测排名证明")
	require.NoError(t, err)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("question", "奖学金申请材料"))
	require.NoError(t, writer.WriteField("answer", "请参考附件文档"))
	part, err := writer.CreateFormFile("files", "scholarship.docx")
	require.NoError(t, err)
	_, err = part.Write(docxContent)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	importReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/import", &body)
	importReq.Header.Set("Content-Type", writer.FormDataContentType())
	importToken := testutil.GenerateTestToken(200, 2, 0, "")
	importReq.Header.Set("Authorization", "Bearer "+importToken)
	importW := httptest.NewRecorder()
	r.ServeHTTP(importW, importReq)
	require.Equal(t, http.StatusOK, importW.Code)

	searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/search?q=综测排名证明", nil)
	searchToken := testutil.GenerateTestToken(100, 1, 1, "2023")
	searchReq.Header.Set("Authorization", "Bearer "+searchToken)
	searchW := httptest.NewRecorder()
	r.ServeHTTP(searchW, searchReq)

	require.Equal(t, http.StatusOK, searchW.Code)
	require.Contains(t, searchW.Body.String(), "奖学金申请材料")
}

func TestKnowledgeSearchHitsImportedPDFContent(t *testing.T) {
	uploadDir := t.TempDir()
	t.Setenv("DOCUMENT_UPLOAD_DIR", uploadDir)

	_, r := setupKnowledgeTestRouter(t)

	wd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Join(wd, "..", "..", "..")
	pdfPath := filepath.Join(root, "internal", "service", "knowledge", "testdata", "sample.pdf")
	pdfBytes, err := os.ReadFile(pdfPath)
	require.NoError(t, err)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("question", "C++学习路线"))
	require.NoError(t, writer.WriteField("answer", "请查看附件学习建议"))
	part, err := writer.CreateFormFile("files", "cpp_guide.pdf")
	require.NoError(t, err)
	_, err = part.Write(pdfBytes)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	importReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/import", &body)
	importReq.Header.Set("Content-Type", writer.FormDataContentType())
	importToken := testutil.GenerateTestToken(200, 2, 0, "")
	importReq.Header.Set("Authorization", "Bearer "+importToken)
	importW := httptest.NewRecorder()
	r.ServeHTTP(importW, importReq)
	require.Equal(t, http.StatusOK, importW.Code)

	searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/search?q=C++技术栈", nil)
	searchToken := testutil.GenerateTestToken(100, 1, 1, "2023")
	searchReq.Header.Set("Authorization", "Bearer "+searchToken)
	searchW := httptest.NewRecorder()
	r.ServeHTTP(searchW, searchReq)

	require.Equal(t, http.StatusOK, searchW.Code)
	require.Contains(t, searchW.Body.String(), "C++学习路线")
}

func buildDocx(text string) ([]byte, error) {
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
