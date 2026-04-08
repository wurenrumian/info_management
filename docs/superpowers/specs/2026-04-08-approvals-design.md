# 审批流程模块设计（v0）

**日期**：2026-04-08  
**阶段**：Phase 2-C

## 1. 目标

实现学生事务审批的最小闭环：
- 学生提交申请并查看本人状态
- 管理员按 scope 审批通过或驳回
- 学生可在待审批时撤回
- 保留一学期审批历史

## 2. 范围

### In Scope

- 固定申请类型：`leave`、`stamp`
- 线性单当前审批人模型
- 学生发起、查看、撤回
- 管理员列表、详情、审批
- 审批历史记录
- 逾期提醒占位并复用现有通知能力

### Out of Scope

- 多节点并行审批
- 自定义流程编排
- 通用表单引擎
- 与电子证明的强耦合联动

## 3. 数据模型

### 3.1 `approvals`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| applicant_id | bigint FK | 申请人 |
| approval_type | varchar(20) | `leave` / `stamp` |
| status | varchar(20) | `pending` / `approved` / `rejected` / `withdrawn` |
| title | varchar(200) | 摘要标题 |
| form_data | jsonb | 申请表单 |
| attachment_file_ids | jsonb | 附件文件 ID 列表 |
| current_approver_id | bigint FK | 当前审批人 |
| semester | varchar(20) | 所属学期 |
| submitted_at | timestamp | 提交时间 |
| decided_at | timestamp | 审结时间 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

### 3.2 `approval_actions`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| approval_id | bigint FK | 所属申请 |
| action_type | varchar(20) | `submit` / `approve` / `reject` / `withdraw` / `assign` |
| operator_id | bigint FK | 操作人 |
| comment | varchar(500) | 审批意见 |
| snapshot | jsonb | 动作快照 |
| created_at | timestamp | 创建时间 |

## 4. 状态流转

```text
pending -> approved
pending -> rejected
pending -> withdrawn
```

规则：
- 只有 `pending` 状态可被审批
- 只有申请人可撤回 `pending` 状态申请
- 已审结申请不可二次处理

## 5. 表单结构

### 5.1 请假申请 `leave`

```json
{
  "reason": "回家处理事务",
  "start_date": "2026-04-10",
  "end_date": "2026-04-12",
  "contact_phone": "13800000000"
}
```

### 5.2 盖章申请 `stamp`

```json
{
  "document_name": "实习证明",
  "copies": 2,
  "purpose": "实习单位提交"
}
```

## 6. API 设计

### 6.1 学生端

- `POST /api/v1/approvals`
- `GET /api/v1/approvals/me?status=pending&limit=20&offset=0`
- `GET /api/v1/approvals/:id`
- `POST /api/v1/approvals/:id/withdraw`

创建请求体：

```json
{
  "approval_type": "leave",
  "title": "五一前请假申请",
  "form_data": {
    "reason": "回家处理事务",
    "start_date": "2026-04-10",
    "end_date": "2026-04-12",
    "contact_phone": "13800000000"
  },
  "attachment_file_ids": [12, 18]
}
```

### 6.2 管理端

- `GET /api/v1/admin/approvals?status=pending&approval_type=leave&limit=20&offset=0`
- `GET /api/v1/admin/approvals/:id`
- `PATCH /api/v1/admin/approvals/:id`

审批请求体：

```json
{
  "action": "approve",
  "comment": "情况属实，准假"
}
```

## 7. 权限与 Scope

- 学生：创建、查看本人、撤回本人
- 管理员：在 scope 内查看与审批
- 审批人选择规则：
  - v0 按申请人班级的 `counselor_id`
  - 若无班主任，则回退至教师角色管理员人工维护

新增 authz 动作建议：

```go
ActionApprovalsCreate   = "approvals:create"
ActionApprovalsMyList   = "approvals:my:list"
ActionApprovalsGet      = "approvals:get"
ActionApprovalsWithdraw = "approvals:withdraw"
ActionApprovalsList     = "approvals:list"
ActionApprovalsReview   = "approvals:review"
```

## 8. 代码结构

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
│       ├── service.go
│       └── service_test.go
└── http/
    └── handler/
        ├── approval_handler.go
        └── approval_handler_test.go
```

## 9. 测试策略

- handler 测试：
  - 学生创建成功
  - 学生撤回成功
  - 非申请人访问 403
  - 管理员审批成功
  - 已撤回申请不可审批
- repo / service 测试：
  - scope 列表过滤
  - 状态流转合法性
  - 每次动作自动写 `approval_actions`
  - 审结后禁止再次处理

## 10. 验收标准

- 两类申请可完整走通提交与审批链路
- 学生可以查看历史与撤回待审批申请
- 管理员可在 scope 内查询和处理
- 历史动作与 `admin_logs` 都有记录

