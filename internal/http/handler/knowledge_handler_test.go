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
	var total int64
	require.NoError(t, json.Unmarshal(payload["total"], &total))
	require.Equal(t, int64(1), total)
}

func TestKnowledgeSearchRejectsEmptyQuery(t *testing.T) {
	_, r := setupKnowledgeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/search", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "missing q")
}

func TestAdminKnowledgeListForbiddenForStudent(t *testing.T) {
	_, r := setupKnowledgeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/knowledge", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminKnowledgeCreateAndPatchWriteAdminLog(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)

	createBody := []byte(`{"question":"复学流程是什么","answer":"提交复学申请并等待审批","keywords":["复学","审批"],"attachments":[{"title":"复学指引","url":"https://example.com/back"}]}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(200, 2, 0, "")
	createReq.Header.Set("Authorization", "Bearer "+token)
	wCreate := httptest.NewRecorder()
	r.ServeHTTP(wCreate, createReq)

	require.Equal(t, http.StatusOK, wCreate.Code)
	require.Contains(t, wCreate.Body.String(), "复学流程是什么")

	var created model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "复学流程是什么").First(&created).Error)

	patchBody := []byte(`{"answer":"先提交复学材料，再由学院审批"}`)
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/knowledge/"+strconv.Itoa(int(created.ID)), bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+token)
	wPatch := httptest.NewRecorder()
	r.ServeHTTP(wPatch, patchReq)

	require.Equal(t, http.StatusOK, wPatch.Code)

	var logs []model.AdminLog
	require.NoError(t, db.Order("id asc").Find(&logs).Error)
	require.Len(t, logs, 2)
	require.Equal(t, "knowledge.create", logs[0].Action)
	require.Equal(t, "knowledge.patch", logs[1].Action)
	require.Equal(t, uint(200), logs[0].AdminID)
	require.Equal(t, created.ID, logs[0].TargetID)
	require.Equal(t, created.ID, logs[1].TargetID)
}

func TestAdminKnowledgePatchReturns404WhenNotFound(t *testing.T) {
	_, r := setupKnowledgeTestRouter(t)

	patchBody := []byte(`{"answer":"不存在记录"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/knowledge/99999", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "knowledge not found")
}

func TestAdminKnowledgeListIncludesTotal(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)
	require.NoError(t, db.Create(&model.KnowledgeItem{
		Question:  "奖学金申请条件",
		Answer:    "绩点和综测达标",
		Keywords:  datatypes.JSON(`["奖学金"]`),
		CreatedBy: 999,
		UpdatedBy: 999,
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/knowledge?limit=1&offset=0", nil)
	token := testutil.GenerateTestToken(200, 2, 0, "")
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

func TestAdminKnowledgeGetByID(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)
	var existing model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "休学申请怎么办理").First(&existing).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/knowledge/"+strconv.Itoa(int(existing.ID)), nil)
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "休学申请怎么办理")
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
	t.Setenv("KNOWLEDGE_UPLOAD_DIR", uploadDir)

	db, r := setupKnowledgeTestRouter(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("question", "奖学金政策在哪里看"))
	require.NoError(t, writer.WriteField("answer", "请查看附件政策文件"))
	require.NoError(t, writer.WriteField("keywords", "奖学金,政策"))

	part, err := writer.CreateFormFile("files", "policy.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-pdf-content"))
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "奖学金政策在哪里看")
	require.Contains(t, w.Body.String(), "/uploads/knowledge/")

	var item model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "奖学金政策在哪里看").First(&item).Error)

	var attachments []map[string]string
	require.NoError(t, json.Unmarshal(item.Attachments, &attachments))
	require.Len(t, attachments, 1)
	require.Equal(t, "policy.pdf", attachments[0]["title"])

	filename := filepath.Base(attachments[0]["url"])
	_, err = os.Stat(filepath.Join(uploadDir, filename))
	require.NoError(t, err)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "knowledge.import").Find(&logs).Error)
	require.Len(t, logs, 1)
}

func TestKnowledgeSearchHitsImportedDocContent(t *testing.T) {
	uploadDir := t.TempDir()
	t.Setenv("KNOWLEDGE_UPLOAD_DIR", uploadDir)

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
	t.Setenv("KNOWLEDGE_UPLOAD_DIR", uploadDir)

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
