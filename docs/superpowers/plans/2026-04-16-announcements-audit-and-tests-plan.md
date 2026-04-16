# Announcement 审计日志与测试补齐 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补齐 Announcement 模块的审计日志写入与缺失测试覆盖，并通过全量测试验证。

**Architecture:** 保持现有 Announcement 数据结构和 service 行为不变，只在 handler 成功路径接入 `audit.Logger`，并用测试覆盖管理端操作、学生端可见性以及 repo/service 核心规则。

**Tech Stack:** Go 1.25, Gin, GORM, SQLite in-memory

---

### Task 1: 补齐失败测试

**Files:**
- Modify: `internal/http/handler/announcement_handler_test.go`
- Modify: `internal/service/announcements/service_test.go`
- Create: `internal/repo/announcement_repo_test.go`

- [ ] **Step 1: 写 handler 缺失用例**
- [ ] **Step 2: 运行 handler 测试并确认因实现缺口失败**
- [ ] **Step 3: 写 service 缺失用例**
- [ ] **Step 4: 运行 service 测试并确认因实现缺口失败**
- [ ] **Step 5: 写 repo 用例**
- [ ] **Step 6: 运行 repo 测试并确认新增测试有效**

### Task 2: 补齐最小实现

**Files:**
- Modify: `internal/http/handler/announcement_handler.go`

- [ ] **Step 1: 为 AnnouncementHandler 注入 `auditLogger`**
- [ ] **Step 2: 在 Create/Patch/Publish/Archive 成功后写审计日志**
- [ ] **Step 3: 运行对应测试直到通过**

### Task 3: 回归验证

**Files:**
- Modify: `internal/http/handler/announcement_handler.go`
- Modify: `internal/http/handler/announcement_handler_test.go`
- Modify: `internal/service/announcements/service_test.go`
- Create: `internal/repo/announcement_repo_test.go`

- [ ] **Step 1: `go test ./internal/http/handler ./internal/service/announcements ./internal/repo -count=1`**
- [ ] **Step 2: `go test ./... -count=1`**
