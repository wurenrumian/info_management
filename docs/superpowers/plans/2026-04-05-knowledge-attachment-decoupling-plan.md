# Knowledge Attachment Decoupling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decouple knowledge file import from Q&A management by introducing explicit `knowledge_attachments` relations and new attachment-binding APIs.

**Architecture:** Keep `knowledge_items` CRUD and document library intact, add `knowledge_attachments` as the source of truth for Q&A-file relationships, and provide explicit bind/unbind/list endpoints. Preserve compatibility by keeping `attachments` in response shape during transition and support deprecated `import` in file-only mode.

**Tech Stack:** Go, Gin, GORM, SQLite (tests), PostgreSQL-compatible schema via AutoMigrate.

---

### Task 1: Add KnowledgeAttachment Model and Migration Wiring

**Files:**
- Create: `internal/model/knowledge_attachment.go`
- Modify: `internal/store/db.go`
- Modify: `internal/model/model_migrate_test.go`

- [ ] **Step 1: Write failing migration test**

Add a test assertion that `AutoMigrate` includes `KnowledgeAttachment`:

```go
err = db.AutoMigrate(
  &model.User{},
  &model.Class{},
  &model.AdminLog{},
  &model.KnowledgeItem{},
  &model.Document{},
  &model.KnowledgeAttachment{},
)
require.NoError(t, err)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/model -run TestAutoMigrateCoreTables`
Expected: FAIL because `KnowledgeAttachment` is undefined.

- [ ] **Step 3: Add model and migrate registration**

Create model with unique index on `(knowledge_id, file_id)`:

```go
type KnowledgeAttachment struct {
  ID          uint      `gorm:"primaryKey" json:"id"`
  KnowledgeID uint      `gorm:"index:idx_knowledge_file,unique;not null" json:"knowledge_id"`
  FileID      uint      `gorm:"index:idx_knowledge_file,unique;not null" json:"file_id"`
  CreatedBy   uint      `gorm:"index;not null" json:"created_by"`
  CreatedAt   time.Time `json:"created_at"`
}
```

Register it in `OpenAndMigrate` and migrate test.

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/model -run TestAutoMigrateCoreTables`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/knowledge_attachment.go internal/store/db.go internal/model/model_migrate_test.go
git commit -m "feat(model): add knowledge attachment relation model"
```

### Task 2: Add Repository Layer for Knowledge Attachments

**Files:**
- Create: `internal/repo/knowledge_attachment_repo.go`
- Create: `internal/repo/knowledge_attachment_repo_test.go`

- [ ] **Step 1: Write failing repo tests**

Cover:
- batch create ignores duplicates by unique constraint behavior
- list by knowledge id
- delete by knowledge+file id

Example test target:

```go
func TestKnowledgeAttachmentRepo_ListByKnowledgeID(t *testing.T) { ... }
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/repo -run KnowledgeAttachmentRepo`
Expected: FAIL due missing repo implementation.

- [ ] **Step 3: Implement minimal repo**

Implement:
- `CreateBatch(rows []model.KnowledgeAttachment) error`
- `ListByKnowledgeID(knowledgeID uint) ([]model.KnowledgeAttachment, error)`
- `DeleteByKnowledgeAndFileID(knowledgeID, fileID uint) error`

Use GORM clauses for conflict ignore (idempotent insert).

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/repo -run KnowledgeAttachmentRepo`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/repo/knowledge_attachment_repo.go internal/repo/knowledge_attachment_repo_test.go
git commit -m "feat(repo): add knowledge attachment repository"
```

### Task 3: Extend Admin Knowledge Handler with Attachment APIs

**Files:**
- Modify: `internal/http/handler/admin_knowledge_handler.go`
- Modify: `internal/http/router/router.go`

- [ ] **Step 1: Write failing handler tests for new endpoints**

Add tests first in `internal/http/handler/knowledge_handler_test.go` for:
- `POST /api/v1/admin/knowledge/:id/attachments`
- `GET /api/v1/admin/knowledge/:id/attachments`
- `DELETE /api/v1/admin/knowledge/:id/attachments/:file_id`

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/http/handler -run KnowledgeAttachment`
Expected: FAIL with 404 route/missing handler behavior.

- [ ] **Step 3: Implement handlers and register routes**

Add methods:
- `BindAttachments`
- `ListAttachments`
- `DeleteAttachment`

Register routes under admin group:

```go
admin.POST("/knowledge/:id/attachments", adminKnowledgeHandler.BindAttachments)
admin.GET("/knowledge/:id/attachments", adminKnowledgeHandler.ListAttachments)
admin.DELETE("/knowledge/:id/attachments/:file_id", adminKnowledgeHandler.DeleteAttachment)
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/http/handler -run KnowledgeAttachment`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/http/handler/admin_knowledge_handler.go internal/http/router/router.go internal/http/handler/knowledge_handler_test.go
git commit -m "feat(api): add explicit knowledge attachment binding endpoints"
```

### Task 4: Implement Attachment DTO Assembly from Document Records

**Files:**
- Modify: `internal/http/handler/admin_knowledge_handler.go`
- Modify: `internal/repo/document_repo.go` (if query helpers are needed)

- [ ] **Step 1: Write failing tests for attachment response fields**

Ensure attachment list response includes deterministic fields:
- `file_id`
- `title`
- `url` (`/uploads/documents/<file_path>`)
- `content_type`
- `file_size`

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/http/handler -run KnowledgeAttachmentList`
Expected: FAIL on missing fields or shape mismatch.

- [ ] **Step 3: Implement DTO mapper**

Join `knowledge_attachments` with `documents` (repo or handler-level orchestration), map to response DTO.

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/http/handler -run KnowledgeAttachmentList`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/http/handler/admin_knowledge_handler.go internal/repo/document_repo.go internal/http/handler/knowledge_handler_test.go
git commit -m "feat(api): return document-backed knowledge attachments"
```

### Task 5: Refactor Deprecated Import Endpoint to Decoupled Behavior

**Files:**
- Modify: `internal/http/handler/admin_knowledge_handler.go`
- Modify: `internal/http/handler/knowledge_handler_test.go`

- [ ] **Step 1: Write failing tests for import file-only mode**

Add tests:
- posting `files` only to `/admin/knowledge/import` succeeds and returns file IDs
- posting `question/answer/files` still works (compat mode), but relation is written to `knowledge_attachments`

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/http/handler -run KnowledgeImport`
Expected: FAIL for file-only mode and relation assertions.

- [ ] **Step 3: Implement import dual-mode**

Behavior:
- If `question/answer` empty: upload file(s), create `documents`, return imported files.
- If `question/answer` provided: create knowledge then create relation rows by returned `document.id`.

Mark comments/docs as deprecated for this endpoint.

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/http/handler -run KnowledgeImport`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/http/handler/admin_knowledge_handler.go internal/http/handler/knowledge_handler_test.go
git commit -m "refactor(api): decouple deprecated knowledge import from implicit binding"
```

### Task 6: Make Knowledge Read APIs Return Explicitly Bound Attachments

**Files:**
- Modify: `internal/http/handler/admin_knowledge_handler.go`
- Modify: `internal/http/handler/knowledge_handler.go`
- Modify: `internal/http/handler/knowledge_handler_test.go`

- [ ] **Step 1: Write failing tests for read-path attachment source**

Add assertions that:
- `GET /admin/knowledge/:id` returns only explicitly bound docs
- `GET /knowledge/search` returns only explicitly bound docs in attachments

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/http/handler -run 'Knowledge(Search|GetByID).*Attachment'`
Expected: FAIL due legacy `attachments jsonb` path.

- [ ] **Step 3: Implement read-path projection**

On read APIs, hydrate attachments from relation table + document metadata and set response payload accordingly.

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/http/handler -run 'Knowledge(Search|GetByID).*Attachment'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/http/handler/admin_knowledge_handler.go internal/http/handler/knowledge_handler.go internal/http/handler/knowledge_handler_test.go
git commit -m "feat(api): serve knowledge attachments from explicit relations"
```

### Task 7: Add Audit Logs for Bind/Unbind Actions

**Files:**
- Modify: `internal/http/handler/admin_knowledge_handler.go`
- Modify: `internal/http/handler/knowledge_handler_test.go`

- [ ] **Step 1: Write failing tests for audit actions**

Assert `admin_logs` contains:
- `knowledge.attach`
- `knowledge.detach`

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/http/handler -run 'KnowledgeAttachment.*(Audit|Delete)'`
Expected: FAIL due missing audit actions.

- [ ] **Step 3: Add audit logging**

After successful bind/unbind, write admin logs with target type `knowledge`.

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/http/handler -run 'KnowledgeAttachment.*(Audit|Delete)'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/http/handler/admin_knowledge_handler.go internal/http/handler/knowledge_handler_test.go
git commit -m "feat(audit): log knowledge attachment bind and unbind"
```

### Task 8: Update API Documentation and Script Coverage

**Files:**
- Modify: `docs/api/phase2-knowledge-api.md`
- Modify: `scripts/dev/knowledge_api_curl.sh`

- [ ] **Step 1: Write failing expectations in script/comments**

Add calls in curl script for:
- bind attachments
- list attachments
- detach attachment
- import file-only mode

- [ ] **Step 2: Implement docs and script updates**

Update doc sections with:
- new endpoints and request/response examples
- `import` deprecation and dual-mode behavior
- compatibility note for `attachments` response field

- [ ] **Step 3: Run script lint/smoke checks**

Run:
- `bash -n scripts/dev/knowledge_api_curl.sh`
- optional live check against local service if available

Expected: no syntax errors.

- [ ] **Step 4: Commit**

```bash
git add docs/api/phase2-knowledge-api.md scripts/dev/knowledge_api_curl.sh
git commit -m "docs(test): cover decoupled knowledge attachment APIs"
```

### Task 9: Final Verification

**Files:**
- Modify: none (verification only)

- [ ] **Step 1: Run focused test suites**

Run:
- `go test ./internal/http/handler -run Knowledge`
- `go test ./internal/repo -run KnowledgeAttachmentRepo`
- `go test ./internal/model -run TestAutoMigrateCoreTables`

Expected: all PASS.

- [ ] **Step 2: Run broader regression**

Run: `go test ./...`
Expected: PASS (or record existing unrelated failures explicitly).

- [ ] **Step 3: Validate requirements checklist**

Check each issue acceptance criterion against test evidence and API docs.

- [ ] **Step 4: Commit verification note**

```bash
git status
```

Expected: clean working tree before merge/PR handoff.

