# 审批流程模块设计（v1）

**日期**：2026-04-08
**更新日期**：2026-04-27
**阶段**：Phase 2-C

## 1. 目标

实现学院内部轻量审批的最小闭环：
- 学生提交请假/离校申请或活动预算申请
- 学生查看本人申请状态、详情和审批历史
- 学生可在待审批时撤回
- 管理员按 scope 查看、处理、转交和提醒
- 审批动作与管理员操作均留痕

## 2. 范围

### In Scope

- 固定申请类型：`leave`、`budget`
- 线性单当前审批人模型
- 学生发起、查看、撤回
- 管理员列表、详情、通过、驳回、转交、提醒
- 审批历史记录
- 附件引用复用统一文件服务
- 超时提醒复用现有通知能力
- 保存一学期数据，历史可查

### Out of Scope

- 奖学金/助学金评审流程
- 休学/复学正式审批
- 宿舍调整正式审批
- 校历和节假日维护
- 通用表单引擎
- 多节点并行审批
- 自动接管学校/学生处/研究生院/宿管/财务等正式系统流程
- 电子证明的模板管理、PDF 渲染、核验码生成与作废逻辑

说明：
- 奖学金助学金申请细则、休学复学细则、宿舍调整申请细则、校历节假日、请假条模板、预算申请模板均进入文档库/知识库。
- `leave` 可引用请假条模板文件，`budget` 可引用预算申请模板文件，但模板文件由文档库管理。
- 本模块定位为学院内部流转与留痕，不替代学校正式审批系统或线下流程。
- 电子证明相关能力由 `certificates` 模块负责；`approvals` 作为上游数据源，负责提供审批数据与状态，并在详情接口聚合返回关联 `certificate_records`。

## 3. 数据模型

### 3.1 `approvals`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| applicant_id | bigint FK | 申请人 |
| approval_type | varchar(20) | `leave` / `budget` |
| status | varchar(20) | `pending` / `approved` / `rejected` / `withdrawn` / `expired` |
| current_step | varchar(40) | 当前步骤，如 `review` / `budget_review` |
| title | varchar(200) | 摘要标题 |
| form_data | jsonb | 申请表单 |
| attachment_file_ids | jsonb | 附件文件 ID 列表 |
| template_file_id | bigint nullable | 使用的模板文件 ID |
| current_approver_id | bigint FK nullable | 当前审批人 |
| semester | varchar(20) | 所属学期 |
| due_at | timestamp nullable | 建议处理截止时间 |
| submitted_at | timestamp | 提交时间 |
| decided_at | timestamp nullable | 审结时间 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

说明：
- `form_data` 保存不同申请类型的结构化字段。
- `attachment_file_ids` 和 `template_file_id` 均只保存文件服务引用，不直接保存文件内容。
- `due_at` 用于超时提醒，不自动通过或驳回。

### 3.2 `approval_actions`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| approval_id | bigint FK | 所属申请 |
| action_type | varchar(20) | `submit` / `approve` / `reject` / `withdraw` / `assign` / `remind` / `expire` |
| operator_id | bigint FK | 操作人；系统任务可为 0 |
| from_status | varchar(20) | 动作前状态 |
| to_status | varchar(20) | 动作后状态 |
| comment | varchar(500) | 审批意见或备注 |
| snapshot | jsonb | 动作快照 |
| created_at | timestamp | 创建时间 |

## 4. 状态流转

```text
pending -> approved
pending -> rejected
pending -> withdrawn
pending -> expired
```

规则：
- 只有 `pending` 状态可被审批或转交。
- 只有申请人可撤回本人 `pending` 状态申请。
- 已审结申请不可二次审批。
- 超时只触发提醒，不自动通过或驳回。
- 管理员可手动将长期未处理申请标记为 `expired`。

## 5. 申请类型

### 5.1 请假/离校申请 `leave`

适用：
- 离校连续两天及以上
- 离开校区所在城市
- 涉及缺席教学活动的请假

`form_data` 示例：

```json
{
  "leave_type": "leave_city",
  "reason": "回家处理事务",
  "start_at": "2026-05-01T09:00:00+08:00",
  "end_at": "2026-05-03T18:00:00+08:00",
  "destination": "北京市外",
  "contact_phone": "13800000000",
  "emergency_contact": "13900000000",
  "miss_classes": false,
  "student_type": "undergraduate"
}
```

字段要求：
- `reason`、`start_at`、`end_at`、`contact_phone` 必填。
- `end_at` 必须晚于 `start_at`。
- `leave_type` 建议枚举：`short_leave` / `leave_city` / `miss_classes`。
- 研究生可在 `form_data` 中补充 `advisor_name`、`advisor_acknowledged`，v1 不做导师账号审批流。

### 5.2 活动/班级预算申请 `budget`

适用：
- 班级活动预算申请
- 团日活动预算申请
- 学院内部活动预算初审

`form_data` 示例：

```json
{
  "activity_name": "班级团日活动",
  "activity_date": "2026-05-20",
  "budget_amount": 1200,
  "purpose": "活动物料与场地费用",
  "items": [
    {"name": "物料", "amount": 500},
    {"name": "场地", "amount": 700}
  ]
}
```

字段要求：
- `activity_name`、`budget_amount`、`purpose` 必填。
- `budget_amount` 必须大于 0。
- `items` 的金额合计如提供，应与 `budget_amount` 一致或由管理员人工核验。

## 6. 审批人与分派

v1 不做复杂工作流，仅做单当前审批人。

默认分派建议：
- `leave`：优先分派给申请人班级 `counselor_id`；若为空，则由管理员手动转交。
- `budget`：优先分派给教师/超管角色的学院管理员；若无法自动确定，则由管理员手动转交。

权限边界：
- 学生只能创建、查看、撤回本人申请。
- 团干部可在 scope 内查看申请、发起提醒或协助转交，但默认不作为最终审批人。
- 教师/超管可在 scope 内审批、驳回、转交、标记过期。
- 若团队后续明确团干部可审批某类日常事项，可通过 authz 扩展，不改变数据模型。

## 7. 超时提醒

处理方式：
1. 创建申请时根据类型设置 `due_at`。
2. 定时任务每天扫描 `pending` 且超过 `due_at` 的申请。
3. 调用通知服务提醒当前审批人。
4. 写入 `approval_actions(action_type=remind)`。
5. 不自动通过、不自动驳回。

建议默认时限：
- `leave`：24 小时内处理。
- `budget`：3 天内处理。

调度实现：
- 使用 Go 标准库 `time.Timer` / `time.Ticker`。
- 提供管理员手动扫描接口用于联调和补偿。
- 使用环境变量控制后台任务开关。

## 8. API 设计

### 8.1 学生端

- `POST /api/v1/approvals`
- `GET /api/v1/approvals/me?status=pending&approval_type=leave&limit=20&offset=0`
- `GET /api/v1/approvals/:id`
- `POST /api/v1/approvals/:id/withdraw`

创建请求体：

```json
{
  "approval_type": "leave",
  "title": "五一离校请假申请",
  "form_data": {
    "leave_type": "leave_city",
    "reason": "回家处理事务",
    "start_at": "2026-05-01T09:00:00+08:00",
    "end_at": "2026-05-03T18:00:00+08:00",
    "destination": "北京市外",
    "contact_phone": "13800000000",
    "miss_classes": false,
    "student_type": "undergraduate"
  },
  "template_file_id": 12,
  "attachment_file_ids": [18]
}
```

### 8.2 管理端

- `GET /api/v1/admin/approvals?status=pending&approval_type=leave&limit=20&offset=0`
- `GET /api/v1/admin/approvals/:id`
- `PATCH /api/v1/admin/approvals/:id/review`
- `PATCH /api/v1/admin/approvals/:id/assign`
- `POST /api/v1/admin/approvals/:id/remind`
- `POST /api/v1/admin/approvals/overdue/scan`

审批请求体：

```json
{
  "action": "approve",
  "comment": "情况属实，同意"
}
```

转交请求体：

```json
{
  "current_approver_id": 900,
  "comment": "转交负责老师处理"
}
```

详情接口统一约定：
- `GET /api/v1/approvals/:id` 与 `GET /api/v1/admin/approvals/:id` 返回同一详情结构。
- 接入 `certificates` 模块后，详情中返回 `certificate_records` 字段。
- 当尚无可用 PDF 记录时，`certificate_records` 返回空数组。

## 9. 权限与 Scope

新增 authz 动作建议：

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

权限规则：
- 学生：`create` / `my:list` / `get` / `withdraw`，且只能作用于本人申请。
- 团干部：scope 内 `list` / `get` / `remind`，可协助查看和提醒。
- 教师：scope 内 `list` / `get` / `review` / `assign` / `remind` / `overdue:scan`。
- 超管：全部审批动作。

## 10. 代码结构

```text
internal/
├── model/
│   ├── approval.go
│   └── approval_action.go
├── repo/
│   ├── approval_repo.go
│   ├── approval_repo_test.go
│   ├── approval_action_repo.go
│   └── approval_action_repo_test.go
├── service/
│   └── approvals/
│       ├── reminder.go
│       ├── scheduler.go
│       ├── service.go
│       └── service_test.go
└── http/
    └── handler/
        ├── approval_handler.go
        └── approval_handler_test.go
```

## 11. 测试策略

- handler 测试：
  - 学生创建 `leave` 成功
  - 学生创建 `budget` 成功
  - 学生撤回 pending 申请成功
  - 非申请人访问 403
  - 团干部审批返回 403
  - 教师审批成功
  - scope 外管理员访问 403
  - 已撤回申请不可审批
- repo / service 测试：
  - scope 列表过滤
  - 状态流转合法性
  - 表单字段校验
  - 每次动作自动写 `approval_actions`
  - 审结后禁止再次处理
  - 超时扫描写 `remind` 动作且不改变审批状态
  - 详情接口可稳定返回 `certificate_records`

## 12. 验收标准

- `leave` 和 `budget` 两类申请可完整走通提交、查看、审批、驳回、撤回链路
- 学生可以查看历史与撤回待审批申请
- 管理员可在 scope 内查询和处理
- 团干部无法最终审批
- 教师/超管可审批、转交、提醒
- 历史动作与 `admin_logs` 都有记录
- 文档库承载其它细则与模板，审批模块不重复管理模板文件
- 审批详情接口为 `certificates` 模块预留并返回 `certificate_records`
