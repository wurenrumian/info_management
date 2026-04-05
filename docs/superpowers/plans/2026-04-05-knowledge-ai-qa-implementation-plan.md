# Knowledge AI QA Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add super-admin-only AI preview generation from uploaded document text plus all-or-nothing batch knowledge creation.

**Architecture:** Extend `AdminKnowledgeHandler` with two endpoints: `qa-generate-preview` (read-only draft generation) and `batch` (transactional write). AI generation is isolated behind a small `knowledge` service interface so handler logic remains testable and fallback-safe. Input source is existing `documents.content_text`; output is strict JSON mapped to a draft DTO that frontend can edit then submit.

**Tech Stack:** Go, Gin, GORM, existing sqlite test stack, existing JWT role middleware.

---

## File Structure

- Modify: `internal/http/router/router.go`
  - Register two new admin routes.
- Modify: `internal/http/handler/admin_knowledge_handler.go`
  - Add request/response DTOs and two handlers.
  - Add role=4 guard helper.
  - Add transactional batch create helper.
- Create: `internal/service/knowledge/qa_generator.go`
  - Define generator interface and production implementation.
  - Define strict draft schema and validation.
- Modify: `internal/config/config.go`
  - Add AI config getters (`AIBaseURL`, `AIAPIKey`, `AIModel`).
- Modify: `internal/http/handler/knowledge_handler_test.go`
  - Add handler tests for new endpoints and role behavior.
- Create: `internal/service/knowledge/qa_generator_test.go`
  - Add parser/validator tests for strict JSON conversion.
- Modify: `docs/api/phase2-knowledge-api.md`
  - Document two endpoints + error table updates.

---

### Task 1: Add Failing Handler Tests First (TDD Red)

**Files:**
- Modify: `internal/http/handler/knowledge_handler_test.go`

- [ ] **Step 1: Write failing test for super-admin-only preview generation**

```go
func TestAdminKnowledgeQAGeneratePreviewForbiddenForTeacher(t *testing.T) {
    _, r := setupKnowledgeTestRouter(t)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/qa-generate-preview", bytes.NewBufferString(`{"file_ids":[1],"qa_count_range":{"min":1,"max":2}}`))
    req.Header.Set("Content-Type", "application/json")
    token := testutil.GenerateTestToken(300, 3, 0, "")
    req.Header.Set("Authorization", "Bearer "+token)

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    require.Equal(t, http.StatusForbidden, w.Code)
    require.Contains(t, w.Body.String(), "forbidden")
}
```

- [ ] **Step 2: Write failing test for preview success payload shape**

```go
func TestAdminKnowledgeQAGeneratePreviewBySuperAdmin(t *testing.T) {
    db, r := setupKnowledgeTestRouter(t)
    doc := model.Document{Title: "k.docx", FilePath: "knowledge/2026/04/k.docx", ContentText: "奖学金申请需要成绩单", ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", FileSize: 100, UploaderID: 100}
    require.NoError(t, db.Create(&doc).Error)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/qa-generate-preview", bytes.NewBufferString(`{"file_ids":[`+strconv.Itoa(int(doc.ID))+`],"qa_count_range":{"min":1,"max":2}}`))
    req.Header.Set("Content-Type", "application/json")
    token := testutil.GenerateTestToken(400, 4, 0, "")
    req.Header.Set("Authorization", "Bearer "+token)

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    require.Equal(t, http.StatusOK, w.Code)
    require.Contains(t, w.Body.String(), `"question"`)
    require.Contains(t, w.Body.String(), `"attachment_file_ids"`)
    require.Contains(t, w.Body.String(), `"total":`)
}
```

- [ ] **Step 3: Write failing tests for batch all-or-nothing transaction**

```go
func TestAdminKnowledgeBatchAllOrNothing(t *testing.T) {
    db, r := setupKnowledgeTestRouter(t)

    body := `{"items":[{"question":"Q1","answer":"A1","keywords":["k1"],"attachment_file_ids":[]},{"question":"","answer":"A2","keywords":["k2"],"attachment_file_ids":[]}]}`
    req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/batch", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    token := testutil.GenerateTestToken(400, 4, 0, "")
    req.Header.Set("Authorization", "Bearer "+token)

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    require.Equal(t, http.StatusBadRequest, w.Code)

    var count int64
    require.NoError(t, db.Model(&model.KnowledgeItem{}).Where("question = ?", "Q1").Count(&count).Error)
    require.Equal(t, int64(0), count)
}
```

- [ ] **Step 4: Run tests to verify RED**

Run:
```bash
go test ./internal/http/handler -run 'TestAdminKnowledgeQAGeneratePreviewForbiddenForTeacher|TestAdminKnowledgeQAGeneratePreviewBySuperAdmin|TestAdminKnowledgeBatchAllOrNothing' -count=1
```

Expected: FAIL with route not found or missing handler behavior.

- [ ] **Step 5: Commit failing tests**

```bash
git add internal/http/handler/knowledge_handler_test.go
git commit -m "test: add failing tests for knowledge ai preview and batch api"
```

---

### Task 2: Build Draft Schema + Strict Validation in Service Layer

**Files:**
- Create: `internal/service/knowledge/qa_generator.go`
- Create: `internal/service/knowledge/qa_generator_test.go`

- [ ] **Step 1: Write failing parser/validator tests**

```go
func TestParseDraftsRejectsInvalidJSON(t *testing.T) {}
func TestParseDraftsRejectsOutOfRangeCount(t *testing.T) {}
func TestParseDraftsAcceptsValidPayload(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify RED**

Run:
```bash
go test ./internal/service/knowledge -run 'TestParseDraftsRejectsInvalidJSON|TestParseDraftsRejectsOutOfRangeCount|TestParseDraftsAcceptsValidPayload' -count=1
```

Expected: FAIL because parser/validator does not exist.

- [ ] **Step 3: Add minimal schema + validator implementation**

```go
type QACountRange struct { Min int `json:"min"`; Max int `json:"max"` }
type QADraft struct {
    Question string   `json:"question"`
    Answer   string   `json:"answer"`
    Keywords []string `json:"keywords"`
    AttachmentFileIDs []uint `json:"attachment_file_ids"`
}

type QAPreviewResult struct { Items []QADraft `json:"items"` }
```

```go
func ValidateCountRange(r QACountRange) error {
    if r.Min < 1 || r.Max < r.Min || r.Max > 30 { return errors.New("invalid qa_count_range") }
    return nil
}
```

- [ ] **Step 4: Re-run tests to verify GREEN**

Run:
```bash
go test ./internal/service/knowledge -run 'TestParseDraftsRejectsInvalidJSON|TestParseDraftsRejectsOutOfRangeCount|TestParseDraftsAcceptsValidPayload' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/knowledge/qa_generator.go internal/service/knowledge/qa_generator_test.go
git commit -m "feat: add knowledge qa draft schema and strict validation"
```

---

### Task 3: Wire AI Generation Service (Minimal Production Path)

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/service/knowledge/qa_generator.go`

- [ ] **Step 1: Add failing test for missing AI config path**

```go
func TestGeneratePreviewFailsWhenAIConfigMissing(t *testing.T) {}
```

- [ ] **Step 2: Run RED test**

Run:
```bash
go test ./internal/service/knowledge -run 'TestGeneratePreviewFailsWhenAIConfigMissing' -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement minimal AI client and config getters**

```go
func AIBaseURL() string { return getEnv("AI_BASE_URL", "") }
func AIAPIKey() string  { return strings.TrimSpace(os.Getenv("AI_API_KEY")) }
func AIModel() string   { return getEnv("AI_MODEL", "gpt-4.1-mini") }
```

```go
type QAGenerator interface {
    Generate(ctx context.Context, docs []DocumentInput, countRange QACountRange) ([]QADraft, error)
}
```

```go
if config.AIAPIKey() == "" || config.AIBaseURL() == "" {
    return nil, errors.New("generate preview failed")
}
```

- [ ] **Step 4: Re-run targeted tests**

Run:
```bash
go test ./internal/service/knowledge -run 'TestGeneratePreviewFailsWhenAIConfigMissing|TestParseDraftsRejectsInvalidJSON|TestParseDraftsAcceptsValidPayload' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/service/knowledge/qa_generator.go internal/service/knowledge/qa_generator_test.go
git commit -m "feat: add configurable AI generator for knowledge preview"
```

---

### Task 4: Implement Preview Endpoint in Admin Handler

**Files:**
- Modify: `internal/http/handler/admin_knowledge_handler.go`
- Modify: `internal/http/router/router.go`

- [ ] **Step 1: Add failing route-level test expectation**

```go
req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/qa-generate-preview", bytes.NewBufferString(`{"file_ids":[1],"qa_count_range":{"min":1,"max":1}}`))
// expect non-404 after route wiring
```

- [ ] **Step 2: Implement route + handler skeleton**

```go
admin.POST("/knowledge/qa-generate-preview", adminKnowledgeHandler.GenerateQAPreview)
```

```go
func (h *AdminKnowledgeHandler) GenerateQAPreview(c *gin.Context) {
    actor, ok := auth.GetActor(c)
    if !ok { response.Error(c, 401, "unauthorized"); return }
    if actor.Role != model.RoleSuperAdmin { response.Error(c, 403, "forbidden"); return }
    // bind req -> validate -> load docs -> generator.Generate -> response.List
}
```

- [ ] **Step 3: Add request DTO and validation path**

```go
type qaGeneratePreviewReq struct {
    FileIDs []uint `json:"file_ids"`
    QACountRange knowledge.QACountRange `json:"qa_count_range"`
}
```

- [ ] **Step 4: Run targeted handler tests**

Run:
```bash
go test ./internal/http/handler -run 'TestAdminKnowledgeQAGeneratePreviewForbiddenForTeacher|TestAdminKnowledgeQAGeneratePreviewBySuperAdmin' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/http/router/router.go internal/http/handler/admin_knowledge_handler.go internal/http/handler/knowledge_handler_test.go
git commit -m "feat: add super-admin knowledge qa preview endpoint"
```

---

### Task 5: Implement Batch Create Endpoint with Single Transaction

**Files:**
- Modify: `internal/http/handler/admin_knowledge_handler.go`
- Modify: `internal/http/router/router.go`

- [ ] **Step 1: Add/extend failing tests for all-or-nothing behavior**

```go
require.Equal(t, http.StatusBadRequest, w.Code)
require.Equal(t, int64(0), count) // no partial writes
```

- [ ] **Step 2: Implement batch request DTO + transaction helper**

```go
type batchKnowledgeReq struct { Items []knowledge.QADraft `json:"items"` }
```

```go
err := h.svc.WithTx(func(tx *gorm.DB) error {
    for _, item := range req.Items {
        // validate item -> create knowledge -> bind attachments
    }
    return nil
})
```

(If `WithTx` does not exist, add local `h.db.Transaction(...)` by storing `db *gorm.DB` in handler.)

- [ ] **Step 3: Return list response aligned with current style**

```go
response.List(c, createdItems, int64(len(createdItems)))
```

- [ ] **Step 4: Run targeted tests**

Run:
```bash
go test ./internal/http/handler -run 'TestAdminKnowledgeBatchAllOrNothing' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/http/handler/admin_knowledge_handler.go internal/http/handler/knowledge_handler_test.go internal/http/router/router.go
git commit -m "feat: add transactional knowledge batch create endpoint"
```

---

### Task 6: Docs + Full Verification

**Files:**
- Modify: `docs/api/phase2-knowledge-api.md`

- [ ] **Step 1: Update API docs for new endpoints**

```md
- POST /api/v1/admin/knowledge/qa-generate-preview
- POST /api/v1/admin/knowledge/batch
权限：role = 4
```

- [ ] **Step 2: Add error response entries**

```md
- 400 {"error":"invalid qa_count_range"}
- 500 {"error":"generate preview failed"}
- 500 {"error":"batch create knowledge failed"}
```

- [ ] **Step 3: Run full related test suites**

Run:
```bash
go test ./internal/service/knowledge ./internal/http/handler ./internal/http/router -count=1
```

Expected: PASS.

- [ ] **Step 4: Run quick regression for existing knowledge APIs**

Run:
```bash
go test ./internal/http/handler -run 'TestKnowledgeSearchByStudent|TestAdminKnowledgeCreate|TestAdminKnowledgeBindAttachments|TestAdminKnowledgeDeleteByID' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit docs + final code**

```bash
git add docs/api/phase2-knowledge-api.md
git commit -m "docs: document knowledge ai preview and batch endpoints"
```

---

## Self-Review Checklist

- Spec coverage:
  - Super-admin-only access: covered in Task 4 tests + handler guard.
  - AI preview from `documents.content_text`: covered in Task 3/4.
  - Frontend-reviewed drafts: preview response DTO in Task 4.
  - Batch all-success transaction: Task 5.
  - API docs update: Task 6.
- Placeholder scan: no TODO/TBD placeholders remain.
- Type consistency: `QADraft` used consistently between preview output and batch input.
