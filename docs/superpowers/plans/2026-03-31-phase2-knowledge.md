# Knowledge Base Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 交付 Phase2 知识库问答模块，支持数据库分词搜索与 role>=2 管理维护。

**Architecture:** 在现有分层基础上新增 knowledge model/repo/handler，并扩展 authz actions。搜索在 PostgreSQL/Kingbase 走全文检索 SQL，SQLite 走 LIKE 降级，保证测试稳定。管理写操作统一写 `admin_logs`。

**Tech Stack:** Go, Gin, GORM, PostgreSQL/Kingbase, SQLite(in-memory tests)

---

### Task 1: 模型与迁移

**Files:**
- Create: `internal/model/knowledge_item.go`
- Modify: `internal/store/db.go`
- Modify: `internal/model/model_migrate_test.go`

- [ ] Step 1: 先补迁移测试（RED）
- [ ] Step 2: 增加 KnowledgeItem 模型
- [ ] Step 3: 更新 AutoMigrate
- [ ] Step 4: 跑模型测试到 GREEN

### Task 2: 权限矩阵扩展

**Files:**
- Modify: `internal/service/authz/actions.go`
- Modify: `internal/service/authz/authorize.go`
- Modify: `internal/service/authz/authorize_test.go`

- [ ] Step 1: 先加失败权限用例（学生可搜、role>=2 可管理）
- [ ] Step 2: 最小修改 action 与 Authorize
- [ ] Step 3: 跑 authz 测试到 GREEN

### Task 3: Repo 搜索与维护

**Files:**
- Create: `internal/repo/knowledge_repo.go`
- Create: `internal/repo/knowledge_repo_test.go`

- [ ] Step 1: 写 repo 失败测试（关键词命中/无命中/创建更新）
- [ ] Step 2: 实现 SQLite LIKE 与 PG 全文检索双路径
- [ ] Step 3: 跑 repo 测试到 GREEN

### Task 4: Handler 与路由

**Files:**
- Create: `internal/http/handler/knowledge_handler.go`
- Create: `internal/http/handler/admin_knowledge_handler.go`
- Create: `internal/http/handler/knowledge_handler_test.go`
- Modify: `internal/http/router/router.go`

- [ ] Step 1: 先写 handler 失败测试（403/400/200）
- [ ] Step 2: 实现学生搜索与管理员列表/新增/修改
- [ ] Step 3: 注册路由并补 admin_logs 写入
- [ ] Step 4: 跑 handler 测试到 GREEN

### Task 5: API 文档与全量验证

**Files:**
- Create: `docs/api/phase2-knowledge-api.md`

- [ ] Step 1: 补 API 文档（权限、请求响应、错误码）
- [ ] Step 2: 运行 `go test ./... -count=1`
- [ ] Step 3: 修复失败并复跑，直到全绿
