# Approvals Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现请假与盖章两类审批的学生提交、管理员处理、撤回与历史留痕最小闭环。

**Architecture:** 使用 `approvals` 保存当前审批状态，`approval_actions` 保存动作历史；审批人按班级班主任规则计算；学生与管理员接口共用 `service/approvals`。

**Tech Stack:** Go 1.25, Gin, GORM, SQLite (testing), PostgreSQL/Kingbase (production)

---

## File Structure

| 操作 | 文件路径 | 说明 |
|------|----------|------|
| 创建 | `internal/model/approval.go` | 审批主表 |
| 创建 | `internal/model/approval_action.go` | 审批动作表 |
| 创建 | `internal/repo/approval_repo.go` | 审批查询与状态写入 |
| 创建 | `internal/repo/approval_action_repo.go` | 历史动作写入 |
| 创建 | `internal/repo/approval_repo_test.go` | repo 测试 |
| 创建 | `internal/service/approvals/service.go` | 审批业务规则 |
| 创建 | `internal/service/approvals/service_test.go` | service 测试 |
| 创建 | `internal/http/handler/approval_handler.go` | 审批 API |
| 创建 | `internal/http/handler/approval_handler_test.go` | handler 测试 |
| 修改 | `internal/service/authz/actions.go` | 新增 approvals 动作 |
| 修改 | `internal/service/authz/authorize.go` | 新增 approvals 权限 |
| 修改 | `internal/store/db.go` | 加入审批模型 |
| 修改 | `internal/http/router/router.go` | 注册审批路由 |
| 修改 | `docs/api/phase2-approvals-api.md` | 升级为正式 API 文档 |

### Task 1: 模型与权限

**Files:**
- Create: `internal/model/approval.go`
- Create: `internal/model/approval_action.go`
- Modify: `internal/service/authz/actions.go`
- Modify: `internal/service/authz/authorize.go`
- Modify: `internal/store/db.go`

- [ ] **Step 1: 创建审批模型**

```go
// internal/model/approval.go
package model

import (
	"time"

	"gorm.io/datatypes"
)

type Approval struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	ApplicantID       uint           `gorm:"index;not null" json:"applicant_id"`
	ApprovalType      string         `gorm:"type:varchar(20);index;not null" json:"approval_type"`
	Status            string         `gorm:"type:varchar(20);index;not null" json:"status"`
	Title             string         `gorm:"type:varchar(200);not null" json:"title"`
	FormData          datatypes.JSON `gorm:"type:jsonb" json:"form_data"`
	AttachmentFileIDs datatypes.JSON `gorm:"type:jsonb" json:"attachment_file_ids"`
	CurrentApproverID uint           `gorm:"index" json:"current_approver_id"`
	Semester          string         `gorm:"type:varchar(20);index;not null" json:"semester"`
	SubmittedAt       time.Time      `json:"submitted_at"`
	DecidedAt         *time.Time     `json:"decided_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}
```

```go
// internal/model/approval_action.go
package model

import (
	"time"

	"gorm.io/datatypes"
)

type ApprovalAction struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	ApprovalID uint           `gorm:"index;not null" json:"approval_id"`
	ActionType string         `gorm:"type:varchar(20);not null" json:"action_type"`
	OperatorID uint           `gorm:"index;not null" json:"operator_id"`
	Comment    string         `gorm:"type:varchar(500)" json:"comment"`
	Snapshot   datatypes.JSON `gorm:"type:jsonb" json:"snapshot"`
	CreatedAt  time.Time      `json:"created_at"`
}
```

- [ ] **Step 2: 新增 approvals authz 动作**

```go
	ActionApprovalsCreate   = "approvals:create"
	ActionApprovalsMyList   = "approvals:my:list"
	ActionApprovalsGet      = "approvals:get"
	ActionApprovalsWithdraw = "approvals:withdraw"
	ActionApprovalsList     = "approvals:list"
	ActionApprovalsReview   = "approvals:review"
```

- [ ] **Step 3: 写权限映射**

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
		action == ActionApprovalsCreate ||
		action == ActionApprovalsMyList ||
		action == ActionApprovalsGet ||
		action == ActionApprovalsWithdraw
```

- [ ] **Step 4: 迁移新表**

在 `internal/store/db.go` 的 `AutoMigrate` 中加入：

```go
&model.Approval{},
&model.ApprovalAction{},
```

- [ ] **Step 5: 运行测试**

Run: `go test ./internal/service/authz ./internal/model ./internal/store -count=1`  
Expected: PASS

### Task 2: Repo 与 Service

**Files:**
- Create: `internal/repo/approval_repo.go`
- Create: `internal/repo/approval_action_repo.go`
- Create: `internal/repo/approval_repo_test.go`
- Create: `internal/service/approvals/service.go`
- Create: `internal/service/approvals/service_test.go`

- [ ] **Step 1: 写 repo**

```go
type ApprovalRepo struct{ db *gorm.DB }

func (r *ApprovalRepo) Create(item *model.Approval) error { return r.db.Create(item).Error }

func (r *ApprovalRepo) GetByID(id uint) (*model.Approval, error) {
	var item model.Approval
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
```

- [ ] **Step 2: 写 service 状态流转**

```go
func (s *Service) Review(actor auth.Actor, id uint, action, comment string) (*model.Approval, error) {
	item, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if item.Status != "pending" {
		return nil, ErrInvalidStatus
	}
	switch action {
	case "approve":
		item.Status = "approved"
	case "reject":
		item.Status = "rejected"
	default:
		return nil, ErrInvalidAction
	}
	now := time.Now()
	item.DecidedAt = &now
	return item, s.repo.UpdateDecision(item)
}
```

- [ ] **Step 3: 写 service 测试**

```go
func TestWithdrawPendingApproval(t *testing.T) {
	// create pending approval
	// withdraw by applicant
	// expect status withdrawn and one action row
}
```

- [ ] **Step 4: 运行测试**

Run: `go test ./internal/repo ./internal/service/approvals -count=1`  
Expected: PASS

### Task 3: Handler、路由与 API 文档

**Files:**
- Create: `internal/http/handler/approval_handler.go`
- Create: `internal/http/handler/approval_handler_test.go`
- Modify: `internal/http/router/router.go`
- Modify: `docs/api/phase2-approvals-api.md`

- [ ] **Step 1: 写 handler**

```go
type ApprovalHandler struct {
	svc *approvals.Service
}

func NewApprovalHandler(db *gorm.DB) *ApprovalHandler {
	return &ApprovalHandler{
		svc: approvals.NewService(
			repo.NewApprovalRepo(db),
			repo.NewApprovalActionRepo(db),
			repo.NewUserRepo(db),
			repo.NewClassRepo(db),
		),
	}
}
```

```go
func (h *ApprovalHandler) Create(c *gin.Context) {}
func (h *ApprovalHandler) ListMine(c *gin.Context) {}
func (h *ApprovalHandler) Get(c *gin.Context) {}
func (h *ApprovalHandler) Withdraw(c *gin.Context) {}
func (h *ApprovalHandler) ListAdmin(c *gin.Context) {}
func (h *ApprovalHandler) Review(c *gin.Context) {}
```

- [ ] **Step 2: 注册路由**

```go
approvalHandler := handler.NewApprovalHandler(db)

api.POST("/approvals", approvalHandler.Create)
api.GET("/approvals/me", approvalHandler.ListMine)
api.GET("/approvals/:id", approvalHandler.Get)
api.POST("/approvals/:id/withdraw", approvalHandler.Withdraw)

admin.GET("/approvals", approvalHandler.ListAdmin)
admin.GET("/approvals/:id", approvalHandler.Get)
admin.PATCH("/approvals/:id", approvalHandler.Review)
```

- [ ] **Step 3: 替换 API 占位稿**

```md
# Phase2 Approvals API

## Student Endpoints

- `POST /api/v1/approvals`
- `GET /api/v1/approvals/me`
- `GET /api/v1/approvals/:id`
- `POST /api/v1/approvals/:id/withdraw`

## Admin Endpoints

- `GET /api/v1/admin/approvals`
- `GET /api/v1/admin/approvals/:id`
- `PATCH /api/v1/admin/approvals/:id`
```

- [ ] **Step 4: 运行测试**

Run: `go test ./internal/http/handler ./internal/http/router -count=1`  
Expected: PASS

### Task 4: 全量验证

**Files:**
- Modify: `internal/http/handler/approval_handler_test.go`
- Modify: `docs/api/phase2-approvals-api.md`

- [ ] **Step 1: 补 403 与越权测试**

```go
func TestAdminCannotReviewOutOfScopeApproval(t *testing.T) {
	// seed applicant outside scope
	// expect 403
}
```

- [ ] **Step 2: 跑全量测试**

Run: `go test ./... -count=1`  
Expected: PASS

