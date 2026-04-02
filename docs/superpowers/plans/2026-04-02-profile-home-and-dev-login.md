# Profile Home And Dev Login Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the missing profile/homepage support APIs that this backend can truthfully serve now.

**Architecture:** Extend the existing `me` and `notification` HTTP handlers with small repository helpers instead of introducing a new service layer. Keep `PATCH /api/v1/me` data in `users.profile_attrs`, compute homepage counters from current persisted tables, and gate dev login behind an explicit environment variable check.

**Tech Stack:** Go, Gin, GORM, SQLite test DB, existing JWT auth service

---

### Task 1: Add failing tests for profile home and profile editing

**Files:**
- Modify: `internal/http/handler/me_handler_test.go`
- Verify: `internal/http/router/router.go`

- [ ] **Step 1: Write failing tests for `GET /api/v1/profile/home` and `PATCH /api/v1/me`**

```go
func TestGetProfileHomeReturnsAggregatedData(t *testing.T) {}
func TestPatchMeUpdatesProfileAttrs(t *testing.T) {}
```

- [ ] **Step 2: Run the focused tests and verify they fail**

Run: `go test ./internal/http/handler -run 'TestGetProfileHomeReturnsAggregatedData|TestPatchMeUpdatesProfileAttrs' -count=1`
Expected: FAIL because routes/handlers do not exist yet.

- [ ] **Step 3: Implement the minimal handler and router changes**

```go
api.GET("/profile/home", meHandler.GetProfileHome)
api.PATCH("/me", meHandler.PatchMe)
```

- [ ] **Step 4: Re-run the focused tests and verify they pass**

Run: `go test ./internal/http/handler -run 'TestGetProfileHomeReturnsAggregatedData|TestPatchMeUpdatesProfileAttrs' -count=1`
Expected: PASS

### Task 2: Add failing tests for unread notification count

**Files:**
- Modify: `internal/http/handler/notification_handler_test.go`
- Modify: `internal/http/handler/notification_handler.go`
- Modify: `internal/service/notification/repo.go`

- [ ] **Step 1: Write the failing unread count test**

```go
func TestUnreadCountReturnsPendingCountForCurrentUser(t *testing.T) {}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./internal/http/handler -run TestUnreadCountReturnsPendingCountForCurrentUser -count=1`
Expected: FAIL because route/handler/repo method do not exist yet.

- [ ] **Step 3: Implement the minimal repo and handler support**

```go
func (r *Repo) CountUnreadByUser(userID uint) (int64, error) {}
func (h *NotificationHandler) UnreadCount(c *gin.Context) {}
```

- [ ] **Step 4: Re-run the focused test and verify it passes**

Run: `go test ./internal/http/handler -run TestUnreadCountReturnsPendingCountForCurrentUser -count=1`
Expected: PASS

### Task 3: Add failing tests for dev login

**Files:**
- Modify: `internal/http/handler/wechat_handler_test.go`
- Modify: `internal/http/handler/wechat_handler.go`
- Modify: `internal/http/router/router.go`

- [ ] **Step 1: Write failing tests for dev login enabled and disabled cases**

```go
func TestDevLoginReturnsTokenWhenEnabled(t *testing.T) {}
func TestDevLoginForbiddenWhenDisabled(t *testing.T) {}
```

- [ ] **Step 2: Run the focused tests and verify they fail**

Run: `go test ./internal/http/handler -run 'TestDevLoginReturnsTokenWhenEnabled|TestDevLoginForbiddenWhenDisabled' -count=1`
Expected: FAIL because route/handler do not exist yet.

- [ ] **Step 3: Implement the minimal dev-only login endpoint**

```go
api.POST("/dev/login", wechatHandler.DevLogin)
```

- [ ] **Step 4: Re-run the focused tests and verify they pass**

Run: `go test ./internal/http/handler -run 'TestDevLoginReturnsTokenWhenEnabled|TestDevLoginForbiddenWhenDisabled' -count=1`
Expected: PASS

### Task 4: Full verification

**Files:**
- Verify: `internal/http/handler/*.go`
- Verify: `internal/http/router/router.go`

- [ ] **Step 1: Run handler package tests**

Run: `go test ./internal/http/handler -count=1`
Expected: PASS

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS
