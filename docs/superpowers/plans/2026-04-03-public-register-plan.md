# Public Register API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a production-safe public register endpoint that supports `student_id + name` registration/activation and optional WeChat code binding, then returns JWT.

**Architecture:** Extend the existing WeChat handler with a focused public register flow that reuses current repo/JWT/wechat service layers. Keep current login/bind APIs unchanged and add one new route under `/api/v1/auth/public-register`.

**Tech Stack:** Go, Gin, GORM, existing `repo.UserRepo`, existing JWT service, existing WeChat code2Session service.

---

### Task 1: Add failing handler tests for public register

**Files:**
- Modify: `internal/http/handler/wechat_handler_test.go`

- [ ] **Step 1: Write failing tests**
- [ ] **Step 2: Run targeted tests to verify failure**
- [ ] **Step 3: Confirm failures map to missing route/handler behavior**

### Task 2: Implement public register handler logic

**Files:**
- Modify: `internal/http/handler/wechat_handler.go`
- Modify: `internal/http/router/router.go`

- [ ] **Step 1: Add request model and handler method**
- [ ] **Step 2: Implement create-or-activate logic**
- [ ] **Step 3: Implement optional `code -> openid` binding checks**
- [ ] **Step 4: Register route in router**

### Task 3: Verify and document

**Files:**
- Modify: `docs/api/phase2-wechat-api.md`

- [ ] **Step 1: Run targeted tests (new cases)**
- [ ] **Step 2: Run handler+wechat+notification regression tests**
- [ ] **Step 3: Update API docs for the new endpoint**
