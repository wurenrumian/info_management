# PartyFlow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现党团流程的学生查询、管理员维护、批量导入与固定规则提醒最小闭环。

**Architecture:** 使用 `party_progresses` 保存当前状态，`party_progress_events` 保存历史事件；HTTP handler 只做参数与权限处理，业务规则放在 `service/partyflow`，提醒发送复用现有 `notification.Service`。

**Tech Stack:** Go 1.25, Gin, GORM, SQLite (testing), PostgreSQL/Kingbase (production)

---

## File Structure

| 操作 | 文件路径 | 说明 |
|------|----------|------|
| 创建 | `internal/model/party_progress.go` | 当前党团流程模型 |
| 创建 | `internal/model/party_progress_event.go` | 党团流程事件模型 |
| 创建 | `internal/repo/party_progress_repo.go` | 当前状态查询与写入 |
| 创建 | `internal/repo/party_progress_event_repo.go` | 历史事件写入与查询 |
| 创建 | `internal/repo/party_progress_repo_test.go` | repo 测试 |
| 创建 | `internal/service/partyflow/service.go` | 创建、更新、导入、学生视图 |
| 创建 | `internal/service/partyflow/reminder.go` | 规则扫描与提醒 |
| 创建 | `internal/service/partyflow/service_test.go` | service 测试 |
| 创建 | `internal/http/handler/partyflow_handler.go` | PartyFlow API |
| 创建 | `internal/http/handler/partyflow_handler_test.go` | handler 测试 |
| 修改 | `internal/service/authz/actions.go` | 新增 partyflow 动作 |
| 修改 | `internal/service/authz/authorize.go` | 新增权限规则 |
| 修改 | `internal/store/db.go` | AutoMigrate 新模型 |
| 修改 | `internal/http/router/router.go` | 注册 PartyFlow 路由 |
| 修改 | `docs/api/phase2-partyflow-api.md` | 将 placeholder 升级为正式文档 |

### Task 1: 模型、authz 与迁移

**Files:**
- Create: `internal/model/party_progress.go`
- Create: `internal/model/party_progress_event.go`
- Modify: `internal/service/authz/actions.go`
- Modify: `internal/service/authz/authorize.go`
- Modify: `internal/store/db.go`

- [ ] **Step 1: 写模型定义**

```go
// internal/model/party_progress.go
package model

import (
	"time"

	"gorm.io/datatypes"
)

type PartyProgress struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"index:idx_party_user_flow,unique;not null" json:"user_id"`
	FlowType         string         `gorm:"type:varchar(20);index:idx_party_user_flow,unique;not null" json:"flow_type"`
	CurrentStage     string         `gorm:"type:varchar(40);not null" json:"current_stage"`
	StageStartedAt   time.Time      `json:"stage_started_at"`
	NextActionHint   string         `gorm:"type:varchar(200)" json:"next_action_hint"`
	ReminderRuleCode string         `gorm:"type:varchar(40)" json:"reminder_rule_code"`
	Metadata         datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedBy        uint           `json:"created_by"`
	UpdatedBy        uint           `json:"updated_by"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}
```

```go
// internal/model/party_progress_event.go
package model

import "time"

type PartyProgressEvent struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ProgressID uint      `gorm:"index;not null" json:"progress_id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	FlowType   string    `gorm:"type:varchar(20);index;not null" json:"flow_type"`
	FromStage  string    `gorm:"type:varchar(40)" json:"from_stage"`
	ToStage    string    `gorm:"type:varchar(40);not null" json:"to_stage"`
	EventType  string    `gorm:"type:varchar(20);not null" json:"event_type"`
	Note       string    `gorm:"type:varchar(500)" json:"note"`
	OperatorID uint      `json:"operator_id"`
	HappenedAt time.Time `json:"happened_at"`
	CreatedAt  time.Time `json:"created_at"`
}
```

- [ ] **Step 2: 新增 authz 动作**

在 `internal/service/authz/actions.go` 末尾追加：

```go
	ActionPartyflowMeGet     = "partyflow:me:get"
	ActionPartyflowList      = "partyflow:list"
	ActionPartyflowGet       = "partyflow:get"
	ActionPartyflowCreate    = "partyflow:create"
	ActionPartyflowPatch     = "partyflow:patch"
	ActionPartyflowImport    = "partyflow:import"
	ActionPartyflowRemindRun = "partyflow:remind:run"
```

- [ ] **Step 3: 新增权限规则**

在 `internal/service/authz/authorize.go` 中增加：

```go
case model.RoleStudent:
	return action == ActionGetMe ||
		action == ActionMePatch ||
		action == ActionProfileHomeGet ||
		action == ActionKnowledgeSearch ||
		action == ActionNotifUnreadGet ||
		action == ActionFilesUpload ||
		action == ActionFilesGet ||
		action == ActionFilesList ||
		action == ActionPartyflowMeGet
```

```go
case model.RoleCadre:
	return action == ActionUsersList ||
		action == ActionUsersGet ||
		action == ActionGetMe ||
		action == ActionMePatch ||
		action == ActionProfileHomeGet ||
		action == ActionKnowledgeSearch ||
		action == ActionNotifUnreadGet ||
		action == ActionKnowledgeList ||
		action == ActionKnowledgeGet ||
		action == ActionKnowledgeCreate ||
		action == ActionKnowledgePatch ||
		action == ActionClassesList ||
		action == ActionClassesGet ||
		action == ActionFilesUpload ||
		action == ActionFilesGet ||
		action == ActionFilesList ||
		action == ActionPartyflowMeGet ||
		action == ActionPartyflowList ||
		action == ActionPartyflowGet ||
		action == ActionPartyflowCreate ||
		action == ActionPartyflowPatch ||
		action == ActionPartyflowImport
```

- [ ] **Step 4: 加入迁移**

修改 `internal/store/db.go`：

```go
if err := db.AutoMigrate(
	&model.User{},
	&model.Class{},
	&model.AdminLog{},
	&model.KnowledgeItem{},
	&model.KnowledgeAttachment{},
	&model.Document{},
	&model.NotificationTemplate{},
	&model.NotificationLog{},
	&model.UserSubscribe{},
	&model.PartyProgress{},
	&model.PartyProgressEvent{},
); err != nil {
	return nil, err
}
```

- [ ] **Step 5: 运行测试**

Run: `go test ./internal/service/authz ./internal/model ./internal/store -count=1`  
Expected: PASS

### Task 2: Repo 与 Service

**Files:**
- Create: `internal/repo/party_progress_repo.go`
- Create: `internal/repo/party_progress_event_repo.go`
- Create: `internal/repo/party_progress_repo_test.go`
- Create: `internal/service/partyflow/service.go`
- Create: `internal/service/partyflow/reminder.go`
- Create: `internal/service/partyflow/service_test.go`

- [ ] **Step 1: 写 repo**

```go
// internal/repo/party_progress_repo.go
package repo

import (
	"manage/internal/model"
	"manage/internal/service/authz"

	"gorm.io/gorm"
)

type PartyProgressRepo struct{ db *gorm.DB }

func NewPartyProgressRepo(db *gorm.DB) *PartyProgressRepo { return &PartyProgressRepo{db: db} }

func (r *PartyProgressRepo) ListByScope(scope authz.Scope, flowType string, limit, offset int) ([]model.PartyProgress, int64, error) {
	q := ApplyUserScope(r.db.Model(&model.PartyProgress{}), scope, "user_id")
	if flowType != "" {
		q = q.Where("flow_type = ?", flowType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []model.PartyProgress
	err := q.Order("id desc").Limit(limit).Offset(offset).Find(&out).Error
	return out, total, err
}
```

```go
// internal/service/partyflow/service.go
package partyflow

type Service struct {
	progressRepo *repo.PartyProgressRepo
	eventRepo    *repo.PartyProgressEventRepo
}
```

- [ ] **Step 2: 写核心行为测试**

```go
func TestServicePatchWritesEvent(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Class{}, &model.PartyProgress{}, &model.PartyProgressEvent{}))

	svc := partyflow.NewService(
		repo.NewPartyProgressRepo(db),
		repo.NewPartyProgressEventRepo(db),
	)

	progress, err := svc.CreateOrUpdate(200, partyflow.UpsertInput{
		UserID:         100,
		FlowType:       "party",
		CurrentStage:   "activist",
		StageStartedAt: time.Now(),
		Note:           "seed",
	})
	require.NoError(t, err)

	_, err = svc.CreateOrUpdate(200, partyflow.UpsertInput{
		ID:             progress.ID,
		UserID:         100,
		FlowType:       "party",
		CurrentStage:   "development_target",
		StageStartedAt: time.Now(),
		Note:           "upgrade",
	})
	require.NoError(t, err)
}
```

- [ ] **Step 3: 运行测试**

Run: `go test ./internal/repo ./internal/service/partyflow -count=1`  
Expected: PASS

### Task 3: Handler、路由与 API 文档

**Files:**
- Create: `internal/http/handler/partyflow_handler.go`
- Create: `internal/http/handler/partyflow_handler_test.go`
- Modify: `internal/http/router/router.go`
- Modify: `docs/api/phase2-partyflow-api.md`

- [ ] **Step 1: 写 handler**

```go
// internal/http/handler/partyflow_handler.go
package handler

type PartyflowHandler struct {
	svc *partyflow.Service
}

func NewPartyflowHandler(db *gorm.DB) *PartyflowHandler {
	return &PartyflowHandler{
		svc: partyflow.NewService(
			repo.NewPartyProgressRepo(db),
			repo.NewPartyProgressEventRepo(db),
		),
	}
}
```

```go
func (h *PartyflowHandler) GetMine(c *gin.Context) {}
func (h *PartyflowHandler) ListAdmin(c *gin.Context) {}
func (h *PartyflowHandler) GetAdmin(c *gin.Context) {}
func (h *PartyflowHandler) Create(c *gin.Context) {}
func (h *PartyflowHandler) Patch(c *gin.Context) {}
func (h *PartyflowHandler) Import(c *gin.Context) {}
```

- [ ] **Step 2: 注册路由**

修改 `internal/http/router/router.go`：

```go
partyflowHandler := handler.NewPartyflowHandler(db)

api.GET("/partyflow/me", partyflowHandler.GetMine)

admin.GET("/partyflow/progress", partyflowHandler.ListAdmin)
admin.GET("/partyflow/progress/:id", partyflowHandler.GetAdmin)
admin.POST("/partyflow/progress", partyflowHandler.Create)
admin.PATCH("/partyflow/progress/:id", partyflowHandler.Patch)
admin.POST("/partyflow/progress/import", partyflowHandler.Import)
```

- [ ] **Step 3: 补正式 API 文档**

把 `docs/api/phase2-partyflow-api.md` 替换为：

```md
# Phase2 PartyFlow API

## Base URL

`/api/v1`

## Student Endpoint

- `GET /api/v1/partyflow/me`

## Admin Endpoints

- `GET /api/v1/admin/partyflow/progress`
- `GET /api/v1/admin/partyflow/progress/:id`
- `POST /api/v1/admin/partyflow/progress`
- `PATCH /api/v1/admin/partyflow/progress/:id`
- `POST /api/v1/admin/partyflow/progress/import`
```

- [ ] **Step 4: 运行测试**

Run: `go test ./internal/http/handler ./internal/http/router -count=1`  
Expected: PASS

### Task 4: 固定规则提醒

**Files:**
- Modify: `internal/service/partyflow/reminder.go`
- Modify: `internal/service/partyflow/service_test.go`

- [ ] **Step 1: 写提醒扫描入口**

```go
func (s *Service) ScanAndSendReminders(ctx context.Context, now time.Time) error {
	items, err := s.progressRepo.ListReminderCandidates(now)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := s.sendReminder(ctx, item); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 2: 写测试**

```go
func TestScanAndSendReminders(t *testing.T) {
	// seed one activist progress older than 90d
	// expect one reminder event written
}
```

- [ ] **Step 3: 全量验证**

Run: `go test ./... -count=1`  
Expected: PASS

