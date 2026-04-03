# Profile Misc API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Issue #24 P0 profile editing and display contract with unified PATCH `/api/v1/me`, read-only protection, validation, and semantic consistency between `/me` and `/profile/home`.

**Architecture:** Extend the existing `profile` service with a dedicated response DTO shared by `GetMe` and `GetHome.basic`. Persist structured profile fields (`major`, `college`, `enrollment_year`) in table columns and flexible display fields (`nickname`, `bio`, `avatar_url`) in `profile_attrs` JSONB. Keep existing HTTP status conventions while adding stable business error codes in error payloads.

**Tech Stack:** Go, Gin, GORM, SQLite in-memory tests, existing auth/authz middleware

---

### Task 1: Add failing tests for new profile contract and error codes

**Files:**
- Modify: `internal/http/handler/me_handler_test.go`
- Modify: `internal/service/profile/service_test.go`

- [ ] **Step 1: Write failing handler tests for profile fields and error codes**

```go
func TestGetMeReturnsProfileFields(t *testing.T) {}
func TestPatchMeUpdatesColumnsAndProfileAttrs(t *testing.T) {}
func TestPatchMeRejectsReadOnlyFields(t *testing.T) {}
func TestPatchMeRejectsInvalidEnrollmentYear(t *testing.T) {}
```

- [ ] **Step 2: Write failing service test for semantic consistency**

```go
func TestGetHomeBasicMatchesGetMeSemantics(t *testing.T) {}
```

- [ ] **Step 3: Run focused tests to verify RED**

Run: `go test ./internal/http/handler -run 'TestGetMeReturnsProfileFields|TestPatchMeUpdatesColumnsAndProfileAttrs|TestPatchMeRejectsReadOnlyFields|TestPatchMeRejectsInvalidEnrollmentYear' -count=1`  
Expected: FAIL (missing fields/validation/error code behavior)

Run: `go test ./internal/service/profile -run TestGetHomeBasicMatchesGetMeSemantics -count=1`  
Expected: FAIL (current service returns `model.User` in `basic`)

### Task 2: Add model and response-layer support

**Files:**
- Modify: `internal/model/user.go`
- Modify: `internal/http/response/response.go`
- Test: `internal/http/handler/me_handler_test.go`

- [ ] **Step 1: Add new table-backed fields to user model**

```go
College        string `gorm:"type:varchar(100)" json:"college"`
EnrollmentYear int    `gorm:"index" json:"enrollment_year"`
```

- [ ] **Step 2: Add error response helper with business code**

```go
func ErrorWithCode(c *gin.Context, status int, code int, msg string) {
    c.JSON(status, gin.H{"error": msg, "code": code})
}
```

- [ ] **Step 3: Run focused tests to confirm still RED but compiles where needed**

Run: `go test ./internal/http/handler -run TestPatchMeRejectsReadOnlyFields -count=1`  
Expected: FAIL on behavior assertions, not compile errors

### Task 3: Implement profile DTO mapping in service

**Files:**
- Modify: `internal/service/profile/service.go`
- Modify: `internal/service/profile/service_test.go`

- [ ] **Step 1: Implement shared profile DTO and mapping helper**

```go
type MeData struct {
    ID             uint      `json:"id"`
    StudentID      string    `json:"student_id"`
    RealName       string    `json:"real_name"`
    Nickname       string    `json:"nickname"`
    Role           int       `json:"role"`
    Major          string    `json:"major"`
    College        string    `json:"college"`
    EnrollmentYear int       `json:"enrollment_year"`
    Bio            string    `json:"bio"`
    AvatarURL      string    `json:"avatar_url"`
    UpdatedAt      time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Update `GetMe` to return DTO and `GetHome.basic` to use same DTO**

```go
func (s *Service) GetMe(actor auth.Actor) (*MeData, error) {}
func (s *Service) GetHome(actor auth.Actor) (*HomeData, error) {}
```

- [ ] **Step 3: Run focused service tests to verify GREEN**

Run: `go test ./internal/service/profile -run 'TestGetHomeBasicMatchesGetMeSemantics|TestGetHomeReturnsKnowledgeCountAndWechatBoundFlag' -count=1`  
Expected: PASS

### Task 4: Implement PATCH behavior, validation, and read-only checks

**Files:**
- Modify: `internal/http/handler/me_handler.go`
- Modify: `internal/service/profile/service.go`
- Modify: `internal/http/handler/me_handler_test.go`
- Modify: `internal/service/profile/service_test.go`

- [ ] **Step 1: Add failing test expectation for merged update result payload**

```go
require.Equal(t, "AI", resp.Data.Major)
require.Equal(t, "信息学院", resp.Data.College)
require.Equal(t, 2023, resp.Data.EnrollmentYear)
```

- [ ] **Step 2: Implement request parsing with raw key inspection**

```go
raw := map[string]json.RawMessage{}
if err := c.ShouldBindJSON(&raw); err != nil { ... }
if _, ok := raw["real_name"]; ok { response.ErrorWithCode(c, 400, 40002, "real_name is read-only"); return }
```

- [ ] **Step 3: Implement validation and business rule mapping**

```go
if !validURL(*req.AvatarURL) { return 40001 }
if year < 2000 || year > time.Now().Year()+1 { return 40003 }
```

- [ ] **Step 4: Extend service patch input to include column + JSON fields**

```go
type PatchMeInput struct {
    Nickname       *string
    Major          *string
    College        *string
    EnrollmentYear *int
    Bio            *string
    AvatarURL      *string
}
```

- [ ] **Step 5: Run focused handler tests to verify GREEN**

Run: `go test ./internal/http/handler -run 'TestGetMeReturnsProfileFields|TestPatchMeUpdatesColumnsAndProfileAttrs|TestPatchMeRejectsReadOnlyFields|TestPatchMeRejectsInvalidEnrollmentYear' -count=1`  
Expected: PASS

### Task 5: Update API docs with success and failure examples

**Files:**
- Modify: `docs/api/phase1-foundation-api.md`

- [ ] **Step 1: Add profile API example section**

```md
### PATCH /api/v1/me (Example Success)
### PATCH /api/v1/me (Example Failure: read-only field)
```

- [ ] **Step 2: Verify docs mention error `code` field**

Run: `rg -n '"code"|/api/v1/me|/api/v1/profile/home' docs/api/phase1-foundation-api.md`  
Expected: matching lines exist for both success and failure examples

### Task 6: Full verification and cleanup

**Files:**
- Verify: `internal/http/handler/me_handler.go`
- Verify: `internal/service/profile/service.go`
- Verify: `internal/model/user.go`
- Verify: `docs/api/phase1-foundation-api.md`

- [ ] **Step 1: Run formatter**

Run: `go fmt ./...`  
Expected: no errors

- [ ] **Step 2: Run focused package tests**

Run: `go test ./internal/service/profile ./internal/http/handler -count=1`  
Expected: PASS

- [ ] **Step 3: Run full test suite**

Run: `go test ./... -count=1`  
Expected: PASS

- [ ] **Step 4: Review git diff for scope**

Run: `git status --short`  
Expected: only intended files are modified
