# PartyFlow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现学生党团状态查询、管理员维护、批量导入、里程碑记录、规则目录管理与轻量定时提醒。

**Architecture:** 使用 `student_political_statuses` 保存当前党团状态事实，`student_political_status_events` 保存历史事件和提醒发送留痕，`political_reminder_rules` 保存可启停的提醒规则目录；提醒扫描用 Go 标准库 `time.Timer/time.Ticker`，通知发送复用现有 `notification.Service`。

**Tech Stack:** Go 1.25, Gin, GORM, SQLite (testing), PostgreSQL/Kingbase (production)

---

## File Structure

| 操作 | 文件路径 | 说明 |
|------|----------|------|
| 创建 | `internal/model/political_status.go` | 学生党团状态事实表模型 |
| 创建 | `internal/model/political_status_event.go` | 状态/里程碑/提醒事件模型 |
| 创建 | `internal/model/political_reminder_rule.go` | 提醒规则模型 |
| 创建 | `internal/repo/political_status_repo.go` | 状态查询与写入 |
| 创建 | `internal/repo/political_status_event_repo.go` | 事件写入与查询 |
| 创建 | `internal/repo/political_reminder_rule_repo.go` | 规则查询、seed 与更新 |
| 创建 | `internal/repo/political_status_repo_test.go` | repo scope 与唯一约束测试 |
| 创建 | `internal/repo/political_reminder_rule_repo_test.go` | 规则 seed 幂等测试 |
| 创建 | `internal/service/partyflow/service.go` | 状态创建、更新、导入、学生视图 |
| 创建 | `internal/service/partyflow/reminder.go` | 规则扫描、周期计算、去重、发送 |
| 创建 | `internal/service/partyflow/rules_seed.go` | 默认规则种子 |
| 创建 | `internal/service/partyflow/scheduler.go` | Go 标准库定时扫描 |
| 创建 | `internal/service/partyflow/service_test.go` | service 测试 |
| 创建 | `internal/http/handler/partyflow_handler.go` | PartyFlow API |
| 创建 | `internal/http/handler/partyflow_handler_test.go` | handler 测试 |
| 修改 | `internal/service/authz/actions.go` | 新增 partyflow 动作 |
| 修改 | `internal/service/authz/authorize.go` | 新增权限规则 |
| 修改 | `internal/store/db.go` | AutoMigrate 新模型 |
| 修改 | `internal/app` 相关启动文件 | 按环境变量启动 scheduler |
| 修改 | `internal/http/router/router.go` | 注册 PartyFlow 路由 |
| 修改 | `docs/api/phase2-partyflow-api.md` | 同步正式 API 文档 |

## Task 1: 模型、authz 与迁移

**Files:**
- Create: `internal/model/political_status.go`
- Create: `internal/model/political_status_event.go`
- Create: `internal/model/political_reminder_rule.go`
- Modify: `internal/service/authz/actions.go`
- Modify: `internal/service/authz/authorize.go`
- Modify: `internal/store/db.go`

- [ ] **Step 1: 创建模型**

`PoliticalStatus` 字段：

```go
type PoliticalStatus struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UserID          uint           `gorm:"index:idx_political_user_org,unique;not null" json:"user_id"`
	OrgType         string         `gorm:"type:varchar(20);index:idx_political_user_org,unique;not null" json:"org_type"`
	Status          string         `gorm:"type:varchar(40);index;not null" json:"status"`
	StatusStartedAt time.Time      `json:"status_started_at"`
	JoinedAt        *time.Time     `json:"joined_at"`
	NextActionHint  string         `gorm:"type:varchar(200)" json:"next_action_hint"`
	Metadata        datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedBy       uint           `json:"created_by"`
	UpdatedBy       uint           `json:"updated_by"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
```

`PoliticalStatusEvent` 字段：

```go
type PoliticalStatusEvent struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	StatusID   uint           `gorm:"index;not null" json:"status_id"`
	UserID     uint           `gorm:"index;not null" json:"user_id"`
	OrgType    string         `gorm:"type:varchar(20);index;not null" json:"org_type"`
	FromStatus string         `gorm:"type:varchar(40)" json:"from_status"`
	ToStatus   string         `gorm:"type:varchar(40)" json:"to_status"`
	EventType  string         `gorm:"type:varchar(30);index;not null" json:"event_type"`
	EventCode  string         `gorm:"type:varchar(80);index" json:"event_code"`
	Note       string         `gorm:"type:varchar(500)" json:"note"`
	OperatorID uint           `json:"operator_id"`
	HappenedAt time.Time      `json:"happened_at"`
	Metadata   datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
}
```

`PoliticalReminderRule` 字段：

```go
type PoliticalReminderRule struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	RuleCode           string         `gorm:"type:varchar(80);uniqueIndex;not null" json:"rule_code"`
	OrgType            string         `gorm:"type:varchar(20);index;not null" json:"org_type"`
	Status             string         `gorm:"type:varchar(40);index;not null" json:"status"`
	TriggerType        string         `gorm:"type:varchar(30);not null" json:"trigger_type"`
	TriggerEventCode   string         `gorm:"type:varchar(80)" json:"trigger_event_code"`
	OffsetDays         int            `json:"offset_days"`
	RepeatIntervalDays int            `json:"repeat_interval_days"`
	Title              string         `gorm:"type:varchar(100);not null" json:"title"`
	MessageTemplate    string         `gorm:"type:varchar(500);not null" json:"message_template"`
	Audience           string         `gorm:"type:varchar(20);not null" json:"audience"`
	Enabled            bool           `gorm:"index;not null;default:true" json:"enabled"`
	Metadata           datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}
```

- [ ] **Step 2: 新增 authz 动作**

```go
ActionPartyflowMeGet          = "partyflow:me:get"
ActionPartyflowStatusList     = "partyflow:status:list"
ActionPartyflowStatusGet      = "partyflow:status:get"
ActionPartyflowStatusCreate   = "partyflow:status:create"
ActionPartyflowStatusPatch    = "partyflow:status:patch"
ActionPartyflowStatusImport   = "partyflow:status:import"
ActionPartyflowEventCreate    = "partyflow:event:create"
ActionPartyflowRuleList       = "partyflow:rule:list"
ActionPartyflowRulePatch      = "partyflow:rule:patch"
ActionPartyflowReminderScan   = "partyflow:reminder:scan"
```

- [ ] **Step 3: 新增权限规则**

- 学生允许 `ActionPartyflowMeGet`
- 团干部允许本人查询、scope 内状态查询/创建/更新/导入、事件创建、规则列表
- 教师允许团干部能力，额外允许规则修改和手动扫描
- 超管允许全部 partyflow 动作
- 修改 `authorize.go` 时只追加新 action，不替换已有权限分支

- [ ] **Step 4: 加入迁移**

在 `internal/store/db.go` 的 `AutoMigrate` 中追加：

```go
&model.PoliticalStatus{},
&model.PoliticalStatusEvent{},
&model.PoliticalReminderRule{},
```

- [ ] **Step 5: 运行测试**

Run: `go test ./internal/service/authz ./internal/model ./internal/store -count=1`
Expected: PASS

## Task 2: Repo 与规则种子

**Files:**
- Create: `internal/repo/political_status_repo.go`
- Create: `internal/repo/political_status_event_repo.go`
- Create: `internal/repo/political_reminder_rule_repo.go`
- Create: `internal/repo/political_status_repo_test.go`
- Create: `internal/repo/political_reminder_rule_repo_test.go`
- Create: `internal/service/partyflow/rules_seed.go`

- [ ] **Step 1: 写状态 repo**

必须支持：
- `ListByScope(scope, filters, limit, offset)`
- `GetByIDWithScope(id, scope)`
- `GetByUserOrg(userID, orgType)`
- `Create`
- `Update`
- `UpsertByUserOrg`

查询必须通过 `ApplyUserScope(..., "user_id")` 应用 scope。

- [ ] **Step 2: 写事件 repo**

必须支持：
- `Create`
- `ListByStatusID(statusID, limit, offset)`
- `ReminderSentExists(statusID, ruleCode, periodIndex)`

`ReminderSentExists` 查询 `event_type = reminder_sent`、`event_code = ruleCode`，并在 metadata 中匹配 `period_index`。
SQLite 测试可先采用 JSON 字符串包含或 service 层解析兜底，Kingbase/PostgreSQL 后续可用 jsonb 查询优化。

- [ ] **Step 3: 写规则 repo**

必须支持：
- `SeedDefaults(defaultRules)` 幂等写入
- `List(filters)`
- `ListEnabled()`
- `GetByID`
- `Patch`

Seed 规则时以 `rule_code` 做幂等键，不覆盖管理员已修改的启停和时间配置，除非后续明确需要强制同步。

- [ ] **Step 4: 写默认规则种子**

`rules_seed.go` 提供默认规则列表：

```text
league_applicant_talk_30d
league_activist_train_90d
league_development_publicity_5workdays
league_approval_30d
league_member_archive_30d
party_applicant_talk_30d
party_activist_report_every_90d
party_development_training_reminder
party_probationary_transfer_365d
party_probationary_report_every_90d
```

默认启用：
- `league_applicant_talk_30d`
- `league_activist_train_90d`
- `party_applicant_talk_30d`
- `party_activist_report_every_90d`
- `party_probationary_transfer_365d`

其他规则 seed 但默认关闭。

- [ ] **Step 5: 写 repo 测试**

覆盖：
- `user_id + org_type` 唯一
- scope 过滤
- seed 幂等
- seed 不覆盖已修改规则

Run: `go test ./internal/repo -count=1`
Expected: PASS

## Task 3: Service 核心行为

**Files:**
- Create: `internal/service/partyflow/service.go`
- Create: `internal/service/partyflow/service_test.go`

- [ ] **Step 1: 写输入/输出结构**

核心输入：
- `UpsertStatusInput`
- `PatchStatusInput`
- `ImportStatusInput`
- `CreateEventInput`
- `PatchRuleInput`

核心输出：
- `StudentStatusView`
- `AdminStatusView`
- `ImportResult`

- [ ] **Step 2: 写状态创建/更新/导入**

规则：
- `org_type` 只允许 `league` / `party`
- `league.status` 只允许 `none/applicant/activist/development_target/member`
- `party.status` 只允许 `none/applicant/activist/development_target/probationary_member/full_member`
- 创建写 `create` 事件
- 状态变化写 `status_change` 事件
- 导入写 `import` 事件
- 写操作记录 `admin_logs`

- [ ] **Step 3: 写学生视图**

`GetMine(actor)` 返回该学生所有 `PoliticalStatus` 和对应 history。

- [ ] **Step 4: 写里程碑事件**

`CreateEvent(actor, statusID, input)`：
- 校验 status 在 actor scope 内
- 允许 `event_type = milestone/manual_adjust`
- 写 `PoliticalStatusEvent`
- 写 `admin_logs`

- [ ] **Step 5: 写 service 测试**

覆盖：
- 创建状态写事件
- 更新状态写事件
- 重复导入同一学生同一 org 走更新
- 学生只能看本人
- scope 外管理员访问 403 风格错误

Run: `go test ./internal/service/partyflow -count=1`
Expected: PASS

## Task 4: 提醒扫描与 scheduler

**Files:**
- Create: `internal/service/partyflow/reminder.go`
- Create: `internal/service/partyflow/scheduler.go`
- Modify: `internal/service/partyflow/service_test.go`
- Modify: `internal/app` 相关启动文件

- [ ] **Step 1: 写提醒扫描入口**

```go
func (s *Service) ScanAndSendReminders(ctx context.Context, now time.Time) (ScanResult, error)
```

扫描逻辑：
- 读取 enabled rules
- 查找匹配 `org_type + status` 的状态
- 根据 `trigger_type` 计算 base time
- 应用 `metadata.reminder_overrides` 中的 `next_due_at`
- 判断是否 due
- 周期规则计算 `period_index`
- 查重，避免同一期重复发送
- 调用 notification service 发送
- 写 `reminder_sent` 事件，metadata 记录 `period_index`、发送结果、rule_code

- [ ] **Step 2: 写周期提醒计算**

测试用例：
- 89 天不提醒
- 90 天生成第 1 期
- 180 天生成第 2 期
- 重复扫描不重复生成第 1/2 期
- 状态进入 stop_statuses 后停止提醒

- [ ] **Step 3: 写 Go 标准库 scheduler**

要求：
- 使用 `time.Timer` 计算下一次扫描时间
- 使用 `time.Ticker` 或循环 timer 做每日扫描
- `PARTYFLOW_REMINDER_SCHEDULER_ENABLED=false` 时不启动
- `PARTYFLOW_REMINDER_SCAN_HOUR=8` 控制每日扫描小时
- ctx 取消时正常退出
- 不引入 cron、队列或工作流引擎

- [ ] **Step 4: 写提醒测试**

使用 fake notification sender，验证：
- 发送被调用
- `reminder_sent` 事件写入
- 失败时返回统计，并保留错误信息

Run: `go test ./internal/service/partyflow -count=1`
Expected: PASS

## Task 5: Handler、路由与 API 文档

**Files:**
- Create: `internal/http/handler/partyflow_handler.go`
- Create: `internal/http/handler/partyflow_handler_test.go`
- Modify: `internal/http/router/router.go`
- Modify: `docs/api/phase2-partyflow-api.md`

- [ ] **Step 1: 写 handler**

接口：
- `GetMine`
- `ListStatuses`
- `GetStatus`
- `CreateStatus`
- `PatchStatus`
- `ImportStatuses`
- `CreateEvent`
- `ListRules`
- `PatchRule`
- `ScanReminders`

handler 只做：
- 解析参数
- `auth.GetActor(c)`
- `authz.Authorize`
- 调 service
- 用 `response.OK` / `response.Error` 返回

- [ ] **Step 2: 注册路由**

```go
api.GET("/partyflow/me", partyflowHandler.GetMine)

admin.GET("/partyflow/statuses", partyflowHandler.ListStatuses)
admin.GET("/partyflow/statuses/:id", partyflowHandler.GetStatus)
admin.POST("/partyflow/statuses", partyflowHandler.CreateStatus)
admin.PATCH("/partyflow/statuses/:id", partyflowHandler.PatchStatus)
admin.POST("/partyflow/statuses/import", partyflowHandler.ImportStatuses)
admin.POST("/partyflow/statuses/:id/events", partyflowHandler.CreateEvent)
admin.GET("/partyflow/reminder-rules", partyflowHandler.ListRules)
admin.PATCH("/partyflow/reminder-rules/:id", partyflowHandler.PatchRule)
admin.POST("/partyflow/reminders/scan", partyflowHandler.ScanReminders)
```

- [ ] **Step 3: 写 handler 测试**

覆盖：
- 学生查看本人成功
- 管理员导入成功
- 管理员记录里程碑成功
- 规则 patch 成功
- 手动扫描成功
- 非法 org/status 参数 400
- 未认证 401
- 越权 403
- scope 外 403

Run: `go test ./internal/http/handler ./internal/http/router -count=1`
Expected: PASS

## Task 6: 全量验证

**Files:**
- Modify: `docs/api/phase2-partyflow-api.md`
- Optional Create: `scripts/dev/partyflow_api_curl.sh`

- [ ] **Step 1: 确认 API 文档同步**

确保 `docs/api/phase2-partyflow-api.md` 覆盖：
- 学生接口
- 管理员状态接口
- 里程碑事件接口
- 提醒规则接口
- 手动扫描接口
- 枚举和错误响应

- [ ] **Step 2: 可选增加联调脚本**

如进入实现阶段，增加 `scripts/dev/partyflow_api_curl.sh`，覆盖登录、创建状态、导入、查看、规则列表、手动扫描。

- [ ] **Step 3: 全量运行**

Run: `go test ./... -count=1`
Expected: PASS
