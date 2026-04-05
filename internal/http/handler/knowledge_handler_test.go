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
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}, &model.AdminLog{}, &model.KnowledgeItem{}, &model.Document{}, &model.KnowledgeAttachment{}))

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

func TestKnowledgeGetByIDByStudent(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)
	var existing model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "休学申请怎么办理").First(&existing).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/"+strconv.Itoa(int(existing.ID)), nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "休学申请怎么办理")
	require.Contains(t, w.Body.String(), "\"id\":"+strconv.Itoa(int(existing.ID)))
}

func TestKnowledgeGetByIDInvalidID(t *testing.T) {
	_, r := setupKnowledgeTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/abc", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid id")
}

func TestKnowledgeGetByIDNotFound(t *testing.T) {
	_, r := setupKnowledgeTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/99999", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "knowledge not found")
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

func TestAdminKnowledgeList(t *testing.T) {
	_, r := setupKnowledgeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/knowledge?query=休学&limit=20&offset=0", nil)
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"total":`)
	require.Contains(t, w.Body.String(), "休学申请怎么办理")
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
	require.Contains(t, w.Body.String(), `"id":`+strconv.Itoa(int(existing.ID)))
	require.Contains(t, w.Body.String(), "休学申请怎么办理")
}

func TestAdminKnowledgeCreate(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge", bytes.NewBufferString(`{
		"question":"复学流程是什么",
		"answer":"提交复学申请并等待审批",
		"keywords":["复学","审批"],
		"attachments":[{"title":"复学指引","url":"https://example.com/back"}]
	}`))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "复学流程是什么")

	var created model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "复学流程是什么").First(&created).Error)
	require.Equal(t, "提交复学申请并等待审批", created.Answer)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "knowledge.create").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, created.ID, logs[0].TargetID)
}

func TestAdminKnowledgePatchByID(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)
	var existing model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "休学申请怎么办理").First(&existing).Error)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/knowledge/"+strconv.Itoa(int(existing.ID)), bytes.NewBufferString(`{"answer":"先提交休学申请表，再联系辅导员"}`))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"updated":true`)

	var updated model.KnowledgeItem
	require.NoError(t, db.First(&updated, existing.ID).Error)
	require.Equal(t, "先提交休学申请表，再联系辅导员", updated.Answer)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "knowledge.patch").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, existing.ID, logs[0].TargetID)
}

func TestAdminKnowledgePatchByIDInvalidID(t *testing.T) {
	_, r := setupKnowledgeTestRouter(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/knowledge/abc", bytes.NewBufferString(`{"answer":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid id")
}

func TestAdminKnowledgePatchByIDEmptyPatch(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)
	var existing model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "休学申请怎么办理").First(&existing).Error)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/knowledge/"+strconv.Itoa(int(existing.ID)), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "empty patch")
}

func TestAdminKnowledgeBindAttachments(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)
	var existing model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "休学申请怎么办理").First(&existing).Error)
	doc := model.Document{
		Title:       "leave-guide.pdf",
		FilePath:    "2026/04/leave-guide.pdf",
		FileSize:    128,
		ContentType: "application/pdf",
		UploaderID:  200,
	}
	require.NoError(t, db.Create(&doc).Error)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/"+strconv.Itoa(int(existing.ID))+"/attachments", bytes.NewBufferString(`{"file_ids":[`+strconv.Itoa(int(doc.ID))+`]}`))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"added_count":1`)
	require.Contains(t, w.Body.String(), `"file_id":`+strconv.Itoa(int(doc.ID)))

	var relCount int64
	require.NoError(t, db.Model(&model.KnowledgeAttachment{}).Where("knowledge_id = ? AND file_id = ?", existing.ID, doc.ID).Count(&relCount).Error)
	require.Equal(t, int64(1), relCount)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "knowledge.attach").Find(&logs).Error)
	require.Len(t, logs, 1)
}

func TestAdminKnowledgeListAttachments(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)
	var existing model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "休学申请怎么办理").First(&existing).Error)
	doc := model.Document{
		Title:       "leave-flow.docx",
		FilePath:    "2026/04/leave-flow.docx",
		FileSize:    256,
		ContentType: "application/msword",
		UploaderID:  200,
	}
	require.NoError(t, db.Create(&doc).Error)
	require.NoError(t, db.Create(&model.KnowledgeAttachment{KnowledgeID: existing.ID, FileID: doc.ID, CreatedBy: 200}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/knowledge/"+strconv.Itoa(int(existing.ID))+"/attachments", nil)
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"file_id":`+strconv.Itoa(int(doc.ID)))
	require.Contains(t, w.Body.String(), "leave-flow.docx")
}

func TestAdminKnowledgeDeleteAttachment(t *testing.T) {
	db, r := setupKnowledgeTestRouter(t)
	var existing model.KnowledgeItem
	require.NoError(t, db.Where("question = ?", "休学申请怎么办理").First(&existing).Error)
	doc := model.Document{
		Title:       "remove-me.pdf",
		FilePath:    "2026/04/remove-me.pdf",
		FileSize:    200,
		ContentType: "application/pdf",
		UploaderID:  200,
	}
	require.NoError(t, db.Create(&doc).Error)
	require.NoError(t, db.Create(&model.KnowledgeAttachment{KnowledgeID: existing.ID, FileID: doc.ID, CreatedBy: 200}).Error)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/knowledge/"+strconv.Itoa(int(existing.ID))+"/attachments/"+strconv.Itoa(int(doc.ID)), nil)
	token := testutil.GenerateTestToken(200, 2, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"deleted":true`)

	var relCount int64
	require.NoError(t, db.Model(&model.KnowledgeAttachment{}).Where("knowledge_id = ? AND file_id = ?", existing.ID, doc.ID).Count(&relCount).Error)
	require.Equal(t, int64(0), relCount)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "knowledge.detach").Find(&logs).Error)
	require.Len(t, logs, 1)
}

func TestAdminKnowledgeImportEndpointRemoved(t *testing.T) {
	uploadDir := t.TempDir()
	t.Setenv("DOCUMENT_UPLOAD_DIR", uploadDir)

	_, r := setupKnowledgeTestRouter(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", "only-file.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("pdf-text-layer"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	importReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/import", &body)
	importReq.Header.Set("Content-Type", writer.FormDataContentType())
	importToken := testutil.GenerateTestToken(200, 2, 0, "")
	importReq.Header.Set("Authorization", "Bearer "+importToken)
	importW := httptest.NewRecorder()
	r.ServeHTTP(importW, importReq)

	require.Equal(t, http.StatusNotFound, importW.Code)
}

func TestKnowledgeSearchHitsBoundDocxContent(t *testing.T) {
	uploadDir := t.TempDir()
	t.Setenv("DOCUMENT_UPLOAD_DIR", uploadDir)

	db, r := setupKnowledgeTestRouter(t)

	docxContent, err := buildDocx("奖学金申请需要提交综测排名证明")
	require.NoError(t, err)

	fileID := uploadFileViaAPI(t, r, "scholarship.docx", docxContent)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge", bytes.NewBufferString(`{"question":"奖学金申请材料","answer":"请参考附件文档","keywords":["奖学金","材料"]}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(200, 2, 0, ""))
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusOK, createW.Code)
	knowledgeID := extractDataID(t, createW.Body.Bytes())

	bindReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/"+strconv.Itoa(int(knowledgeID))+"/attachments", bytes.NewBufferString(`{"file_ids":[`+strconv.Itoa(int(fileID))+`]}`))
	bindReq.Header.Set("Content-Type", "application/json")
	bindReq.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(200, 2, 0, ""))
	bindW := httptest.NewRecorder()
	r.ServeHTTP(bindW, bindReq)
	require.Equal(t, http.StatusOK, bindW.Code)

	searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/search?q=综测排名证明", nil)
	searchToken := testutil.GenerateTestToken(100, 1, 1, "2023")
	searchReq.Header.Set("Authorization", "Bearer "+searchToken)
	searchW := httptest.NewRecorder()
	r.ServeHTTP(searchW, searchReq)

	require.Equal(t, http.StatusOK, searchW.Code)
	require.Contains(t, searchW.Body.String(), "奖学金申请材料")
	require.Contains(t, searchW.Body.String(), `"file_id":`+strconv.Itoa(int(fileID)))

	var relCount int64
	require.NoError(t, db.Model(&model.KnowledgeAttachment{}).Where("knowledge_id = ? AND file_id = ?", knowledgeID, fileID).Count(&relCount).Error)
	require.Equal(t, int64(1), relCount)
}

func TestKnowledgeSearchHitsBoundPDFContent(t *testing.T) {
	uploadDir := t.TempDir()
	t.Setenv("DOCUMENT_UPLOAD_DIR", uploadDir)

	_, r := setupKnowledgeTestRouter(t)

	wd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Join(wd, "..", "..", "..")
	pdfPath := filepath.Join(root, "internal", "service", "knowledge", "testdata", "sample.pdf")
	pdfBytes, err := os.ReadFile(pdfPath)
	require.NoError(t, err)

	fileID := uploadFileViaAPI(t, r, "cpp_guide.pdf", pdfBytes)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge", bytes.NewBufferString(`{"question":"C++学习路线","answer":"请查看附件学习建议","keywords":["C++","路线"]}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(200, 2, 0, ""))
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusOK, createW.Code)
	knowledgeID := extractDataID(t, createW.Body.Bytes())

	bindReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/"+strconv.Itoa(int(knowledgeID))+"/attachments", bytes.NewBufferString(`{"file_ids":[`+strconv.Itoa(int(fileID))+`]}`))
	bindReq.Header.Set("Content-Type", "application/json")
	bindReq.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(200, 2, 0, ""))
	bindW := httptest.NewRecorder()
	r.ServeHTTP(bindW, bindReq)
	require.Equal(t, http.StatusOK, bindW.Code)

	searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/search?q=C++技术栈", nil)
	searchToken := testutil.GenerateTestToken(100, 1, 1, "2023")
	searchReq.Header.Set("Authorization", "Bearer "+searchToken)
	searchW := httptest.NewRecorder()
	r.ServeHTTP(searchW, searchReq)

	require.Equal(t, http.StatusOK, searchW.Code)
	require.Contains(t, searchW.Body.String(), "C++学习路线")
}

func uploadFileViaAPI(t *testing.T, r http.Handler, filename string, content []byte) uint {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(200, 2, 0, ""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	return extractDataID(t, w.Body.Bytes())
}

func extractDataID(t *testing.T, b []byte) uint {
	t.Helper()
	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &payload))
	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload["data"], &data))
	var id uint
	require.NoError(t, json.Unmarshal(data["id"], &id))
	return id
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
