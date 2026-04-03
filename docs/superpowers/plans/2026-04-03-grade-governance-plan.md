# Grade Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `classes.grade` the source of truth while keeping `users.grade` as system-managed snapshot, and guarantee permission/JWT behavior stays correct.

**Architecture:** Add a small grade governance service that resolves effective grade and performs sync operations on user/class update paths. Keep existing RBAC scope contract (`class_id OR grade`) but ensure grade values are generated from class facts. Block manual user-grade edits through admin user patch.

**Tech Stack:** Go, Gin, GORM, SQLite in-memory tests, existing authz and audit modules

---

### Task 1: Add failing tests for grade governance behavior

**Files:**
- Modify: `internal/http/handler/admin_user_handler_test.go`
- Modify: `internal/http/handler/admin_class_handler_test.go`
- Create: `internal/service/auth/grade_service_test.go`

- [ ] **Step 1: Write failing test for blocking direct user grade patch**

```go
func TestPatchUserRejectsGradeField(t *testing.T) {}
```

- [ ] **Step 2: Write failing test for class patch triggering user grade sync**

```go
func TestPatchClassUpdatesUsersGradeWhenGradeChanged(t *testing.T) {}
```

- [ ] **Step 3: Write failing test for effective grade resolution**

```go
func TestResolveEffectiveGrade_PreferClassGradeThenFallbackUserGrade(t *testing.T) {}
```

- [ ] **Step 4: Run focused tests to verify RED**

Run: `go test ./internal/http/handler -run 'TestPatchUserRejectsGradeField|TestPatchClassUpdatesUsersGradeWhenGradeChanged' -count=1`  
Expected: FAIL (behavior not implemented)

Run: `go test ./internal/service/auth -run TestResolveEffectiveGrade_PreferClassGradeThenFallbackUserGrade -count=1`  
Expected: FAIL (service does not exist yet)

### Task 2: Introduce grade governance service

**Files:**
- Create: `internal/service/auth/grade_service.go`
- Modify: `internal/repo/user_repo.go`
- Modify: `internal/repo/class_repo.go`
- Test: `internal/service/auth/grade_service_test.go`

- [ ] **Step 1: Add minimal service API**

```go
type GradeService struct { ... }
func (s *GradeService) ResolveEffectiveGrade(user *model.User) (string, error) {}
func (s *GradeService) SyncUserGradeByClassID(userID uint, classID uint) error {}
func (s *GradeService) SyncUsersGradeByClassID(classID uint) (int64, error) {}
```

- [ ] **Step 2: Add minimal repo helpers needed by service**

```go
func (r *UserRepo) BulkUpdateGradeByClassID(classID uint, grade string) (int64, error) {}
func (r *ClassRepo) GetByID(id uint) (*model.Class, error) {}
```

- [ ] **Step 3: Run service tests to verify GREEN**

Run: `go test ./internal/service/auth -run TestResolveEffectiveGrade_PreferClassGradeThenFallbackUserGrade -count=1`  
Expected: PASS

### Task 3: Enforce system-managed grade in admin user patch

**Files:**
- Modify: `internal/http/handler/admin_user_handler.go`
- Modify: `internal/http/handler/admin_user_handler_test.go`
- Modify: `internal/service/audit/logger.go` (only if extra action name helper needed)

- [ ] **Step 1: Keep request struct backward-compatible but reject `grade` when present**

```go
if req.Grade != nil {
    response.Error(c, 400, "grade is system-managed")
    return
}
```

- [ ] **Step 2: If `class_id` updated, call sync service to refresh user grade**

```go
if req.ClassID != nil { _ = gradeSvc.SyncUserGradeByClassID(uint(id), *req.ClassID) }
```

- [ ] **Step 3: Add/adjust tests for class_id patch auto-sync**

```go
func TestPatchUserClassIDSyncsGrade(t *testing.T) {}
```

- [ ] **Step 4: Run focused handler tests**

Run: `go test ./internal/http/handler -run 'TestPatchUserRejectsGradeField|TestPatchUserClassIDSyncsGrade' -count=1`  
Expected: PASS

### Task 4: Sync users on class grade update

**Files:**
- Modify: `internal/http/handler/admin_class_handler.go`
- Modify: `internal/http/handler/admin_class_handler_test.go`
- Modify: `internal/service/audit/logger.go` (only if extra action name helper needed)

- [ ] **Step 1: Capture old class grade before patch**

```go
oldClass, err := h.classRepo.GetByIDInScope(authz.BuildScope(actor), uint(id))
```

- [ ] **Step 2: After class patch succeeds, detect grade change and sync users**

```go
if req.Grade != nil && *req.Grade != oldClass.Grade {
    _, err := h.gradeSvc.SyncUsersGradeByClassID(uint(id))
}
```

- [ ] **Step 3: Add/adjust tests to prove same-class users get new grade**

```go
func TestPatchClassUpdatesUsersGradeWhenGradeChanged(t *testing.T) {}
```

- [ ] **Step 4: Run focused class handler tests**

Run: `go test ./internal/http/handler -run TestPatchClassUpdatesUsersGradeWhenGradeChanged -count=1`  
Expected: PASS

### Task 5: Use effective grade in token issuance paths

**Files:**
- Modify: `internal/http/handler/wechat_handler.go`
- Modify: `internal/service/auth/dev_login.go`
- Modify: `internal/http/handler/wechat_handler_test.go`
- Modify: `internal/service/auth/dev_login_test.go`

- [ ] **Step 1: Resolve effective grade before calling `GenerateToken`**

```go
effectiveGrade, err := h.gradeSvc.ResolveEffectiveGrade(user)
token, err := jwtauth.GenerateToken(user.ID, user.Role, user.ClassID, effectiveGrade, h.jwtSecret)
```

- [ ] **Step 2: Keep response payload unchanged except consistent grade value**

```go
"grade": effectiveGrade,
```

- [ ] **Step 3: Add regression tests for class-grade precedence**

```go
func TestLoginUsesClassGradeAsTokenGrade(t *testing.T) {}
```

- [ ] **Step 4: Run focused auth/handler tests**

Run: `go test ./internal/service/auth ./internal/http/handler -run 'TestLoginUsesClassGradeAsTokenGrade|TestDevRegisterOrLogin' -count=1`  
Expected: PASS

### Task 6: Docs and verification

**Files:**
- Modify: `docs/api/phase1-foundation-api.md`
- Modify: `docs/development-standard.md`

- [ ] **Step 1: Document grade governance contract**

```md
- `classes.grade` is source-of-truth
- `users.grade` is system-managed snapshot
- `PATCH /admin/users/:id` does not accept grade
```

- [ ] **Step 2: Run formatting**

Run: `go fmt ./...`  
Expected: no errors

- [ ] **Step 3: Run focused tests**

Run: `go test ./internal/http/handler ./internal/service/auth ./internal/repo -count=1`  
Expected: PASS

- [ ] **Step 4: Run full test suite**

Run: `go test ./... -count=1`  
Expected: PASS

- [ ] **Step 5: Scope check before handoff**

Run: `git status --short`  
Expected: only grade governance related files changed
