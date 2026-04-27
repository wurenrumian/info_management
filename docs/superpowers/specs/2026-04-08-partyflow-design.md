# 党团流程模块设计（v1）

**日期**：2026-04-08
**更新日期**：2026-04-27
**阶段**：Phase 2-A

## 1. 目标

实现党团流程的轻量闭环：
- 学生查看本人当前党团状态、主阶段、历史动作和下一步提示
- 管理员按 scope 查询、创建、更新、导入学生党团状态
- 管理员记录谈话、公示、审批、备案、归档等流程动作
- 系统按启用的提醒规则定时生成提醒，并复用现有通知能力发送
- 所有状态变化、里程碑和提醒发送均可审计

## 2. 范围

### In Scope

- 统一承载“入团 + 入党”两条主线
- 使用独立状态事实表保存学生当前党团状态
- 使用事件表保存状态变化、里程碑、导入、提醒发送历史
- 使用规则表保存可启停、可调整的提醒规则目录
- 支持一次性提醒和周期性提醒
- 支持管理员批量导入当前状态
- 支持 Go 标准库定时扫描和管理员手动触发扫描
- 写操作记录 `admin_logs`

### Out of Scope

- 理论自测
- 通用工作流引擎
- 复杂规则表达式引擎
- 多渠道通知编排
- 流程附件审核
- 独立提醒实例表和复杂批量 override 表

说明：
- v1 重点是“状态管理 + 可配置提醒”，不是把党团制度的每个步骤都做成强状态机。
- 临时调整日期 v1 先放入 `student_political_statuses.metadata.reminder_overrides`，等批量调整需求明确后再拆独立表。

## 3. 数据模型

### 3.1 `student_political_statuses`

表示每个学生在某条党团主线上的当前事实状态。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| user_id | bigint FK | 学生用户 |
| org_type | varchar(20) | `league` / `party` |
| status | varchar(40) | 当前状态编码 |
| status_started_at | timestamp | 当前状态开始时间 |
| joined_at | timestamp nullable | 正式入团/入党时间 |
| next_action_hint | varchar(200) | 下一步提示 |
| metadata | jsonb | 培养联系人、介绍人、材料状态、临时提醒覆盖等补充信息 |
| created_by | bigint FK | 创建人 |
| updated_by | bigint FK | 更新人 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

约束：
- `user_id + org_type` 唯一
- `status_started_at` 是大多数提醒规则的默认起算时间

`metadata.reminder_overrides` 示例：

```json
{
  "reminder_overrides": {
    "party_activist_report_every_90d": {
      "next_due_at": "2026-06-01T00:00:00Z",
      "reason": "本学期统一材料提交提前"
    }
  }
}
```

### 3.2 `student_political_status_events`

表示状态变化、里程碑和提醒发送历史。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| status_id | bigint FK | 关联当前状态 |
| user_id | bigint FK | 学生用户 |
| org_type | varchar(20) | `league` / `party` |
| from_status | varchar(40) | 变更前状态 |
| to_status | varchar(40) | 变更后状态 |
| event_type | varchar(30) | `create` / `status_change` / `milestone` / `import` / `manual_adjust` / `reminder_sent` |
| event_code | varchar(80) | 里程碑编码或提醒规则编码 |
| note | varchar(500) | 备注 |
| operator_id | bigint FK | 操作人；系统任务可为 0 |
| happened_at | timestamp | 事件发生时间 |
| metadata | jsonb | 周期号、发送结果、材料快照等补充信息 |
| created_at | timestamp | 创建时间 |

提醒去重：
- 对周期提醒，`metadata.period_index` 表示第几期
- 发送前检查同一 `status_id + event_type(reminder_sent) + event_code(rule_code) + period_index` 是否已存在

### 3.3 `political_reminder_rules`

表示可启停的提醒规则目录。规则种子由后端初始化生成，不要求管理员手写规则文件。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| rule_code | varchar(80) unique | 稳定规则编码 |
| org_type | varchar(20) | `league` / `party` |
| status | varchar(40) | 适用状态 |
| trigger_type | varchar(30) | `status_started` / `event_completed` / `fixed_date` |
| trigger_event_code | varchar(80) | `event_completed` 起算事件，可为空 |
| offset_days | int | 起算点后多少天提醒 |
| repeat_interval_days | int nullable | 周期提醒间隔；空或 0 表示一次性提醒 |
| title | varchar(100) | 提醒标题 |
| message_template | varchar(500) | 提醒内容模板 |
| audience | varchar(20) | `student` / `admin` / `both` |
| enabled | bool | 是否启用 |
| metadata | jsonb | 停止状态、说明、模板参数等 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

`metadata.stop_statuses` 示例：

```json
{
  "stop_statuses": ["development_target", "probationary_member", "full_member"]
}
```

## 4. 状态建模

### 4.1 `league` 主线

- `none`
- `applicant`
- `activist`
- `development_target`
- `member`

### 4.2 `party` 主线

- `none`
- `applicant`
- `activist`
- `development_target`
- `probationary_member`
- `full_member`

说明：
- 后端只把主身份/主阶段做成稳定状态。
- 谈话、公示、审批、备案、智慧团建建档、归档等动作记录为 `milestone` 事件。
- 前端可用状态表展示步骤条，用事件表展示时间线。

## 5. 提醒规则

### 5.1 规则来源

- 后端提供默认规则种子，由初始化/迁移后的 seed 方法写入 `political_reminder_rules`。
- 管理员通过 API 启用、停用、调整 `offset_days`、`repeat_interval_days`、标题和内容。
- 不要求业务人员维护本地规则文件；文档中的规则列表是初始种子说明。

### 5.2 初始规则目录

| rule_code | 类型 | 默认启用 | 说明 |
|-----------|------|----------|------|
| `league_applicant_talk_30d` | 一次性 | true | 入团申请后 30 天内谈话 |
| `league_activist_train_90d` | 一次性 | true | 入团积极分子培养满 90 天可推荐发展对象 |
| `league_development_publicity_5workdays` | 一次性 | false | 发展对象公示不少于 5 个工作日，v1 先按 5 天处理 |
| `league_approval_30d` | 一次性 | false | 支部大会后 30 天内审批 |
| `league_member_archive_30d` | 一次性 | false | 审批同意后 30 天内建立智慧团建电子档案 |
| `party_applicant_talk_30d` | 一次性 | true | 入党申请提交后 30 天内谈话 |
| `party_activist_report_every_90d` | 周期性 | true | 入党积极分子每满 90 天提交一次思想汇报 |
| `party_development_training_reminder` | 一次性 | false | 发展对象阶段提醒短期集中培训 |
| `party_probationary_transfer_365d` | 一次性 | true | 预备党员满 365 天提醒转正申请 |
| `party_probationary_report_every_90d` | 周期性 | false | 预备党员教育考察期间按季度提醒材料或思想汇报 |

说明：
- 默认启用项覆盖原始需求中的“关键节点提醒”。
- 其他可能规则先 seed 但默认关闭，避免打扰过多。
- “5 个工作日”v1 不做工作日历，先按 5 天处理；如后续需要，再扩展校历/工作日计算。

### 5.3 周期提醒去重

周期提醒计算：

```text
base_time = status_started_at 或 trigger_event happened_at
period_index = floor((now - base_time - offset_days) / repeat_interval_days) + 1
```

示例：
- 入党积极分子进入 `activist` 后第 90 天生成 `period_index=1`
- 第 180 天生成 `period_index=2`
- 第 270 天生成 `period_index=3`

只要状态进入 `metadata.stop_statuses`，该规则停止继续生成提醒。

## 6. 技术方案

- 后端：Go + Gin + GORM
- 数据库：Kingbase/PostgreSQL，测试使用 SQLite in-memory
- 弹性字段：`jsonb`
- 定时任务：Go 标准库 `time.Timer` / `time.Ticker`
- 通知发送：复用现有 `internal/service/notification`
- 导入：v1 采用 JSON 批量导入，Excel 由前端或脚本转换

调度方式：
1. 管理端手动触发 `POST /api/v1/admin/partyflow/reminders/scan`，用于联调和补偿
2. 应用启动后按环境变量开启每日定时扫描

建议环境变量：

```text
PARTYFLOW_REMINDER_SCHEDULER_ENABLED=false
PARTYFLOW_REMINDER_SCAN_HOUR=8
```

## 7. API 设计

### 7.1 学生端

- `GET /api/v1/partyflow/me`

返回本人所有党团状态：

```json
{
  "data": [
    {
      "id": 1,
      "org_type": "party",
      "status": "activist",
      "status_started_at": "2026-03-01T00:00:00Z",
      "joined_at": null,
      "next_action_hint": "每满3个月提交一次思想汇报",
      "history": [
        {
          "id": 11,
          "event_type": "status_change",
          "event_code": "party_set_activist",
          "from_status": "applicant",
          "to_status": "activist",
          "note": "支部确认",
          "happened_at": "2026-03-01T00:00:00Z"
        }
      ]
    }
  ]
}
```

### 7.2 管理端

- `GET /api/v1/admin/partyflow/statuses?org_type=party&student_id=2023001&limit=20&offset=0`
- `GET /api/v1/admin/partyflow/statuses/:id`
- `POST /api/v1/admin/partyflow/statuses`
- `PATCH /api/v1/admin/partyflow/statuses/:id`
- `POST /api/v1/admin/partyflow/statuses/import`
- `POST /api/v1/admin/partyflow/statuses/:id/events`
- `GET /api/v1/admin/partyflow/reminder-rules`
- `PATCH /api/v1/admin/partyflow/reminder-rules/:id`
- `POST /api/v1/admin/partyflow/reminders/scan`

`POST /statuses/import` 请求体：

```json
{
  "items": [
    {
      "student_id": "2023001",
      "org_type": "party",
      "status": "activist",
      "status_started_at": "2026-03-01T00:00:00Z",
      "next_action_hint": "每满3个月提交一次思想汇报",
      "note": "班级批量导入"
    }
  ]
}
```

`POST /statuses/:id/events` 请求体：

```json
{
  "event_type": "milestone",
  "event_code": "party_publicity_done",
  "note": "公示完成",
  "happened_at": "2026-04-01T00:00:00Z"
}
```

`PATCH /reminder-rules/:id` 请求体：

```json
{
  "enabled": true,
  "offset_days": 90,
  "repeat_interval_days": 90,
  "title": "思想汇报提醒",
  "message_template": "你已进入入党积极分子阶段满 {period_days} 天，请按要求提交思想汇报。"
}
```

## 8. 权限与 Scope

- 学生：仅可查看本人 `GET /partyflow/me`
- 团干部/教师/超管：可按 scope 查询、创建、更新、导入学生党团状态
- 团干部/教师/超管：可记录里程碑事件
- 超管或教师：可维护提醒规则并手动触发扫描
- 所有管理员写操作记录 `admin_logs`

新增 authz 动作建议：

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

## 9. 代码结构

```text
internal/
├── model/
│   ├── political_status.go
│   ├── political_status_event.go
│   └── political_reminder_rule.go
├── repo/
│   ├── political_status_repo.go
│   ├── political_status_repo_test.go
│   ├── political_status_event_repo.go
│   ├── political_reminder_rule_repo.go
│   └── political_reminder_rule_repo_test.go
├── service/
│   └── partyflow/
│       ├── reminder.go
│       ├── rules_seed.go
│       ├── scheduler.go
│       ├── service.go
│       └── service_test.go
└── http/
    └── handler/
        ├── partyflow_handler.go
        └── partyflow_handler_test.go
```

## 10. 测试策略

- handler 测试：
  - 学生查看本人状态成功
  - 管理员导入成功
  - 管理员记录里程碑成功
  - 非法 org_type / status 参数返回 400
  - 非本人访问 403
  - 跨班/跨年级 scope 访问 403
- repo / service 测试：
  - `user_id + org_type` 唯一
  - scope 下分页查询
  - 更新状态自动写事件
  - 重复导入同一 `user_id + org_type` 走更新逻辑
  - 默认规则 seed 幂等
  - 入党积极分子 90/180/270 天周期提醒去重
  - 状态变更后周期提醒停止
  - 规则关闭后不生成提醒
  - `metadata.reminder_overrides` 能覆盖下一次提醒时间

## 11. 验收标准

- 学生端能稳定查询本人党团状态与历史事件
- 管理端能在 scope 内进行查询、创建、更新、导入
- 管理端能记录里程碑事件
- 提醒规则可查询、启停和调整默认时间
- 入党积极分子每满 90 天提醒能跑通完整链路
- 提醒发送写事件留痕，重复扫描不会重复提醒同一期
