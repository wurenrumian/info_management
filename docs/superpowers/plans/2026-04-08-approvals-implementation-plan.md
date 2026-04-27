# Approvals Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现请假/离校与活动预算两类学院内部轻量审批，支持学生提交/查看/撤回，教师或超管审批/驳回/转交/提醒，团干部仅协助查看和提醒。

**Architecture:** 使用 `approvals` 保存当前审批状态和结构化表单，`approval_actions` 保存动作历史；附件与模板复用文档库文件 ID；超时提醒使用 Go 标准库定时扫描并复用 notification service。审批详情统一预留 `certificate_records` 字段，供 `certificates` 模块接入后返回关联 PDF 列表。

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
| 创建 | `internal/service/approvals/reminder.go` | 超时扫描与提醒 |
| 创建 | `internal/service/approvals/scheduler.go` | Go 标准库定时扫描 |
| 创建 | `internal/service/approvals/service_test.go` | service 测试 |
| 创建 | `internal/http/handler/approval_handler.go` | 审批 API |
| 创建 | `internal/http/handler/approval_handler_test.go` | handler 测试 |
| 修改 | `internal/service/authz/actions.go` | 新增 approvals 动作 |
| 修改 | `internal/service/authz/authorize.go` | 新增 approvals 权限 |
| 修改 | `internal/store/db.go` | 加入审批模型 |
| 修改 | `internal/app` 相关启动文件 | 按环境变量启动 scheduler |
| 修改 | `internal/http/router/router.go` | 注册审批路由 |
| 修改 | `docs/api/phase2-approvals-api.md` | 同步正式 API 文档 |

## Task 1: 模型与权限

**Files:**
- Create: `internal/model/approval.go`
- Create: `internal/model/approval_action.go`
- Modify: `internal/service/authz/actions.go`
- Modify: `internal/service/authz/authorize.go`
- Modify: `internal/store/db.go`

- [ ] **Step 1: 创建审批模型**

`Approval` 字段：

```go
type Approval struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	ApplicantID       uint           `gorm:"index;not null" json:"applicant_id"`
	ApprovalType      string         `gorm:"type:varchar(20);index;not null" json:"approval_type"`
	Status            string         `gorm:"type:varchar(20);index;not null" json:"status"`
	CurrentStep       string         `gorm:"type:varchar(40);index" json:"current_step"`
	Title             string         `gorm:"type:varchar(200);not null" json:"title"`
	FormData          datatypes.JSON `gorm:"type:jsonb" json:"form_data"`
	AttachmentFileIDs datatypes.JSON `gorm:"type:jsonb" json:"attachment_file_ids"`
	TemplateFileID    *uint          `gorm:"index" json:"template_file_id"`
	CurrentApproverID *uint          `gorm:"index" json:"current_approver_id"`
	Semester          string         `gorm:"type:varchar(20);index;not null" json:"semester"`
	DueAt             *time.Time     `gorm:"index" json:"due_at"`
	SubmittedAt       time.Time      `json:"submitted_at"`
	DecidedAt         *time.Time     `json:"decided_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}
```

`ApprovalAction` 字段：

```go
type ApprovalAction struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	ApprovalID uint           `gorm:"index;not null" json:"approval_id"`
	ActionType string         `gorm:"type:varchar(20);not null" json:"action_type"`
	OperatorID uint           `gorm:"index;not null" json:"operator_id"`
	FromStatus string         `gorm:"type:varchar(20)" json:"from_status"`
	ToStatus   string         `gorm:"type:varchar(20)" json:"to_status"`
	Comment    string         `gorm:"type:varchar(500)" json:"comment"`
	Snapshot   datatypes.JSON `gorm:"type:jsonb" json:"snapshot"`
	CreatedAt  time.Time      `json:"created_at"`
}
```

- [ ] **Step 2: 新增 approvals authz 动作**

```go
ActionApprovalsCreate      = "approvals:create"
ActionApprovalsMyList      = "approvals:my:list"
ActionApprovalsGet         = "approvals:get"
ActionApprovalsWithdraw    = "approvals:withdraw"
ActionApprovalsList        = "approvals:list"
ActionApprovalsReview      = "approvals:review"
ActionApprovalsAssign      = "approvals:assign"
ActionApprovalsRemind      = "approvals:remind"
ActionApprovalsOverdueScan = "approvals:overdue:scan"
```

- [ ] **Step 3: 写权限映射**

- 学生：创建、本人列表、本人详情、本人撤回
- 团干部：scope 内列表、详情、提醒；不可审批
- 教师：scope 内列表、详情、审批、转交、提醒、超时扫描
- 超管：全部 approvals 动作
- 修改 `authorize.go` 时只追加 action，不替换已有权限分支

- [ ] **Step 4: 迁移新表**

在 `internal/store/db.go` 的 `AutoMigrate` 中加入：

```go
&model.Approval{},
&model.ApprovalAction{},
```

- [ ] **Step 5: 运行测试**

Run: `go test ./internal/service/authz ./internal/model ./internal/store -count=1`
Expected: PASS

## Task 2: Repo 与 Service

**Files:**
- Create: `internal/repo/approval_repo.go`
- Create: `internal/repo/approval_action_repo.go`
- Create: `internal/repo/approval_repo_test.go`
- Create: `internal/service/approvals/service.go`
- Create: `internal/service/approvals/service_test.go`

- [ ] **Step 1: 写 repo**

`ApprovalRepo` 必须支持：
- `Create`
- `GetByID`
- `GetByIDInScope`
- `ListMine`
- `ListByScope`
- `UpdateStatus`
- `UpdateApprover`
- `ListOverduePending`

scope 查询必须通过 `ApplyUserScope(..., "applicant_id")` 限制申请人范围。

`ApprovalActionRepo` 必须支持：
- `Create`
- `ListByApprovalID`

- [ ] **Step 2: 写表单校验**

`leave` 校验：
- `reason`、`start_at`、`end_at`、`contact_phone` 必填
- `end_at > start_at`
- `leave_type` 允许 `short_leave` / `leave_city` / `miss_classes`

`budget` 校验：
- `activity_name`、`budget_amount`、`purpose` 必填
- `budget_amount > 0`
- 如提供 `items`，金额不得为负

- [ ] **Step 3: 写 service 状态流转**

必须支持：
- `Create`
- `ListMine`
- `Get`
- `Withdraw`
- `ListAdmin`
- `Review`
- `Assign`
- `Remind`

规则：
- 创建后状态为 `pending`
- `leave` 默认 `current_step=review`，默认 `due_at=submitted_at+24h`
- `budget` 默认 `current_step=budget_review`，默认 `due_at=submitted_at+72h`
- `pending` 才可审批、转交、撤回
- 审结后不可再次处理
- 每次动作写 `approval_actions`
- 管理员写操作记录 `admin_logs`
- `Get` 详情输出需稳定预留 `certificate_records` 字段；未接入或暂无可用记录时返回 `[]`

- [ ] **Step 4: 写 service 测试**

覆盖：
- 创建 `leave` 成功并写 submit action
- 创建 `budget` 成功并写 submit action
- 非法表单失败
- 撤回 pending 成功
- 已审结不可撤回
- 团干部审批被拒绝
- 教师审批成功
- 转交更新当前审批人并写 assign action

Run: `go test ./internal/repo ./internal/service/approvals -count=1`
Expected: PASS

## Task 3: 超时提醒与 Scheduler

**Files:**
- Create: `internal/service/approvals/reminder.go`
- Create: `internal/service/approvals/scheduler.go`
- Modify: `internal/service/approvals/service_test.go`
- Modify: `internal/app` 相关启动文件

- [ ] **Step 1: 写超时扫描入口**

```go
func (s *Service) ScanAndRemindOverdue(ctx context.Context, now time.Time) (OverdueScanResult, error)
```

扫描逻辑：
- 查找 `pending` 且 `due_at <= now` 的申请
- 调用 notification service 提醒 `current_approver_id`
- 写 `approval_actions(action_type=remind)`
- 不改变审批状态

- [ ] **Step 2: 写 scheduler**

要求：
- 使用 Go 标准库 `time.Timer` / `time.Ticker`
- `APPROVALS_OVERDUE_SCHEDULER_ENABLED=false` 时不启动
- `APPROVALS_OVERDUE_SCAN_HOUR=9` 控制每日扫描小时
- ctx 取消时正常退出
- 不引入 cron、队列或工作流引擎

- [ ] **Step 3: 写提醒测试**

覆盖：
- 未超时不提醒
- 超时 pending 写 remind action
- 已审结不提醒
- 重复扫描可再次提醒，但 snapshot 中记录扫描时间和发送结果

Run: `go test ./internal/service/approvals -count=1`
Expected: PASS

## Task 4: Handler、路由与 API 文档

**Files:**
- Create: `internal/http/handler/approval_handler.go`
- Create: `internal/http/handler/approval_handler_test.go`
- Modify: `internal/http/router/router.go`
- Modify: `docs/api/phase2-approvals-api.md`

- [ ] **Step 1: 写 handler**

接口：
- `Create`
- `ListMine`
- `Get`
- `Withdraw`
- `ListAdmin`
- `Review`
- `Assign`
- `Remind`
- `ScanOverdue`

handler 只做：
- 解析参数
- `auth.GetActor(c)`
- `authz.Authorize`
- 调 service
- 用 `response.OK` / `response.Error` 返回
- 学生端与管理端 `Get` 使用同一详情结构，并包含 `certificate_records`

- [ ] **Step 2: 注册路由**

```go
approvalHandler := handler.NewApprovalHandler(db)

api.POST("/approvals", approvalHandler.Create)
api.GET("/approvals/me", approvalHandler.ListMine)
api.GET("/approvals/:id", approvalHandler.Get)
api.POST("/approvals/:id/withdraw", approvalHandler.Withdraw)

admin.GET("/approvals", approvalHandler.ListAdmin)
admin.GET("/approvals/:id", approvalHandler.Get)
admin.PATCH("/approvals/:id/review", approvalHandler.Review)
admin.PATCH("/approvals/:id/assign", approvalHandler.Assign)
admin.POST("/approvals/:id/remind", approvalHandler.Remind)
admin.POST("/approvals/overdue/scan", approvalHandler.ScanOverdue)
```

- [ ] **Step 3: 写 handler 测试**

覆盖：
- 学生创建 `leave` 成功
- 学生创建 `budget` 成功
- 学生撤回成功
- 非申请人访问 403
- 团干部审批 403
- 教师审批成功
- scope 外管理员访问 403
- 手动超时扫描成功

Run: `go test ./internal/http/handler ./internal/http/router -count=1`
Expected: PASS

## Task 5: 全量验证

**Files:**
- Modify: `docs/api/phase2-approvals-api.md`
- Optional Create: `scripts/dev/approvals_api_curl.sh`

- [ ] **Step 1: 确认 API 文档同步**

确保 `docs/api/phase2-approvals-api.md` 覆盖：
- 学生接口
- 管理员审批接口
- 转交接口
- 提醒接口
- 超时扫描接口
- 详情响应中的 `certificate_records`
- 枚举和错误响应
- 文档库边界

- [ ] **Step 2: 可选增加联调脚本**

如进入实现阶段，增加 `scripts/dev/approvals_api_curl.sh`，覆盖创建 leave、创建 budget、查看、审批、转交、提醒。

- [ ] **Step 3: 全量运行**

Run: `go test ./... -count=1`
Expected: PASS
