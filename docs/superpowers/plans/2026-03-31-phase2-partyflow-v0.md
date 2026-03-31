# Phase 2 PartyFlow v0 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a minimal, parallel-friendly party/league progress module with student self-query, admin list/update/import endpoints, event audit trail, and reminder placeholder hooks.

**Architecture:** Reuse Phase 1 authz/scope and layered structure. Add new models (`party_progress`, `party_progress_events`), scope-aware repositories, service-level stage transition recording, and HTTP handlers. Keep APIs stable while leaving stage rules configurable later.

**Tech Stack:** Go 1.22+, Gin, GORM, PostgreSQL-compatible schema, SQLite tests, Testify

---

### Task 1: Add PartyFlow Models and Migration Coverage

**Files:**
- Create: `internal/model/party_progress.go`
- Create: `internal/model/party_progress_event.go`
- Modify: `internal/store/db.go`
- Test: `internal/model/party_progress_migrate_test.go`

- [ ] **Step 1: Write migration test for new tables**

```go
package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/model"
)

func TestAutoMigratePartyFlowTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&model.PartyProgress{}, &model.PartyProgressEvent{})
	require.NoError(t, err)
}
```

- [ ] **Step 2: Add `PartyProgress` model**

```go
type PartyProgress struct {
	ID             uint           `gorm:"primaryKey"`
	UserID         uint           `gorm:"index;not null"`
	Type           string         `gorm:"size:10;index;not null"` // party/league
	Stage          string         `gorm:"size:32;index;not null"`
	StageUpdatedAt time.Time      `gorm:"not null"`
	ExtraInfo      datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
```

- [ ] **Step 3: Add `PartyProgressEvent` model**

```go
type PartyProgressEvent struct {
	ID         uint           `gorm:"primaryKey"`
	ProgressID uint           `gorm:"index;not null"`
	FromStage  *string        `gorm:"size:32"`
	ToStage    string         `gorm:"size:32;not null"`
	EventTime  time.Time      `gorm:"not null"`
	OperatorID uint           `gorm:"index;not null"`
	Remark     *string
	Attachments datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt  time.Time
}
```

- [ ] **Step 4: Extend DB migration bootstrap**

```go
if err := db.AutoMigrate(
	&model.User{}, &model.Class{}, &model.AdminLog{},
	&model.PartyProgress{}, &model.PartyProgressEvent{},
); err != nil { return nil, err }
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/model -run 'TestAutoMigrateCoreTables|TestAutoMigratePartyFlowTables' -count=1`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/model/party_progress.go internal/model/party_progress_event.go internal/model/party_progress_migrate_test.go internal/store/db.go
git commit -m "feat: add partyflow models and migration coverage"
```

### Task 2: Add PartyFlow Repositories With Scope Filtering

**Files:**
- Create: `internal/repo/party_progress_repo.go`
- Create: `internal/repo/party_event_repo.go`
- Test: `internal/repo/party_progress_repo_test.go`

- [ ] **Step 1: Add repository tests for role scope visibility**

```go
func TestPartyProgressRepo_ListByScope(t *testing.T) {
	// seed users and progress rows across class/grade
	// assert self/class/class+grade/all scope behavior
}
```

- [ ] **Step 2: Implement list/query/update methods**

```go
func (r *PartyProgressRepo) ListByScope(scope authz.Scope, tpe string, limit, offset int) ([]model.PartyProgress, error)
func (r *PartyProgressRepo) GetByIDInScope(scope authz.Scope, id uint) (*model.PartyProgress, error)
func (r *PartyProgressRepo) UpdateStage(id uint, toStage string, when time.Time) error
```

- [ ] **Step 3: Implement event append/query methods**

```go
func (r *PartyEventRepo) Create(evt *model.PartyProgressEvent) error
func (r *PartyEventRepo) ListByProgressID(progressID uint) ([]model.PartyProgressEvent, error)
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/repo -run TestPartyProgressRepo_ListByScope -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/repo/party_progress_repo.go internal/repo/party_event_repo.go internal/repo/party_progress_repo_test.go
git commit -m "feat: add scope-aware repositories for partyflow"
```

### Task 3: Add PartyFlow Service for Stage Transition + Event Audit

**Files:**
- Create: `internal/service/partyflow/service.go`
- Test: `internal/service/partyflow/service_test.go`

- [ ] **Step 1: Write service test for stage transition audit**

```go
func TestService_UpdateStage_WritesEvent(t *testing.T) {
	// update from stage A to B; assert progress updated and event recorded
}
```

- [ ] **Step 2: Implement service**

```go
type Service struct {
	progressRepo *repo.PartyProgressRepo
	eventRepo    *repo.PartyEventRepo
}

func (s *Service) UpdateStage(progress *model.PartyProgress, operatorID uint, toStage string, remark *string) error
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/service/partyflow -count=1`  
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/service/partyflow/service.go internal/service/partyflow/service_test.go
git commit -m "feat: add partyflow transition service with event audit"
```

### Task 4: Implement Student and Admin PartyFlow Handlers

**Files:**
- Create: `internal/http/handler/party_progress_handler.go`
- Modify: `internal/http/router/router.go`
- Test: `internal/http/handler/party_progress_handler_test.go`

- [ ] **Step 1: Add handler tests for required endpoints**

```go
func TestGetMyPartyProgress(t *testing.T) {}
func TestAdminListPartyProgressRespectsScope(t *testing.T) {}
func TestAdminPatchStageWritesEvent(t *testing.T) {}
```

- [ ] **Step 2: Add endpoint handlers**

```go
GET  /api/v1/party-progress/me
GET  /api/v1/admin/party-progress
PATCH /api/v1/admin/party-progress/:id/stage
```

- [ ] **Step 3: Wire authorization actions (temporary string constants)**

```go
// add in authz/actions.go
const (
	ActionPartyProgressMeGet    = "party_progress:me_get"
	ActionPartyProgressList     = "party_progress:list"
	ActionPartyProgressStageSet = "party_progress:stage_set"
)
```

- [ ] **Step 4: Update `Authorize` matrix for partyflow actions**

```go
// student -> me_get
// cadre/teacher -> list + stage_set (within scope)
// superadmin -> all
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/http/handler -run 'TestGetMyPartyProgress|TestAdminListPartyProgressRespectsScope|TestAdminPatchStageWritesEvent' -count=1`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/http/handler/party_progress_handler.go internal/http/handler/party_progress_handler_test.go internal/http/router/router.go internal/service/authz/actions.go internal/service/authz/authorize.go
git commit -m "feat: add partyflow query and stage-update endpoints"
```

### Task 5: Add Import Endpoint Skeleton and Contract Validation

**Files:**
- Create: `internal/http/handler/party_progress_import_handler.go`
- Modify: `internal/http/router/router.go`
- Test: `internal/http/handler/party_progress_import_handler_test.go`

- [ ] **Step 1: Add import endpoint behavior test**

```go
func TestAdminImportPartyProgress_ValidatesFileAndRole(t *testing.T) {}
```

- [ ] **Step 2: Implement v0 import skeleton**

```go
POST /api/v1/admin/party-progress/import
// validate role, multipart file existence, parse header row, return accepted count
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/http/handler -run TestAdminImportPartyProgress_ValidatesFileAndRole -count=1`  
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/http/handler/party_progress_import_handler.go internal/http/handler/party_progress_import_handler_test.go internal/http/router/router.go
git commit -m "feat: add partyflow import endpoint skeleton"
```

### Task 6: Add Reminder Placeholder Job and API Docs

**Files:**
- Create: `internal/service/partyflow/reminder.go`
- Test: `internal/service/partyflow/reminder_test.go`
- Create: `docs/api/phase2-partyflow-api.md`
- Modify: `README.md`

- [ ] **Step 1: Add reminder placeholder test**

```go
func TestReminderPlanner_SelectsDueProgressRows(t *testing.T) {}
```

- [ ] **Step 2: Implement reminder planner skeleton**

```go
func BuildDueReminderCandidates(now time.Time, items []model.PartyProgress) []model.PartyProgress
```

- [ ] **Step 3: Add phase2 API doc**

```md
GET /api/v1/party-progress/me
GET /api/v1/admin/party-progress
PATCH /api/v1/admin/party-progress/:id/stage
POST /api/v1/admin/party-progress/import
```

- [ ] **Step 4: Update README with Phase 2 module status**

```md
Phase 2 modules:
- PartyFlow (v0): in progress
```

- [ ] **Step 5: Run full verification**

Run: `go test ./... -count=1`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/partyflow/reminder.go internal/service/partyflow/reminder_test.go docs/api/phase2-partyflow-api.md README.md
git commit -m "test/docs: add partyflow reminder placeholder and phase2 api docs"
```
