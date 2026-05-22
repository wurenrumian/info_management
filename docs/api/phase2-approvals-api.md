# Phase2 Approvals API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`

## Overview

- 学生端提交、查询、撤回审批。
- 管理端按 scope 查看、审核、转交、提醒和扫描超时审批。
- `GET /api/v1/approvals/:id` 与 `GET /api/v1/admin/approvals/:id` 返回同一结构：
  - `approval`: 审批主表 `model.Approval`
  - `actions`: 审批动作列表
  - `certificate_records`: 关联的证书/申请材料记录

## Student Endpoints

- `POST /api/v1/approvals`
- `GET /api/v1/approvals/me?status=pending&approval_type=leave&limit=20&offset=0`
- `GET /api/v1/approvals/:id`
- `POST /api/v1/approvals/:id/withdraw`

### POST /api/v1/approvals

请求体支持两类 `approval_type`：

#### Leave Request

```json
{
  "approval_type": "leave",
  "title": "五一离校请假申请",
  "form_data": {
    "reason": "回家处理事务",
    "start_at": "2026-05-01T09:00:00+08:00",
    "end_at": "2026-05-03T18:00:00+08:00",
    "contact_phone": "13800000000"
  },
  "template_file_id": 12,
  "attachment_file_ids": [18],
  "semester": "2025-2026-2"
}
```

#### Budget Request

```json
{
  "approval_type": "budget",
  "title": "班级团日活动预算申请",
  "form_data": {
    "activity_name": "班级团日活动",
    "purpose": "活动物料与场地费用",
    "budget_amount": 1200,
    "activity_date": "2026-05-20",
    "items": [
      { "name": "物料", "amount": 500 },
      { "name": "场地", "amount": 700 }
    ]
  },
  "template_file_id": 18,
  "attachment_file_ids": [19]
}
```

说明：

- `approval_type` 只支持 `leave` 和 `budget`。
- `form_data` 至少要满足类型校验：
  - `leave`：需要 `reason`、`start_at`、`end_at`、`contact_phone`
  - `budget`：需要 `activity_name`、`purpose`
- `semester` 为空时后端会自动补默认值。
- 创建后状态固定为 `pending`，并生成一条 `submit` 动作。
- 创建审批时会自动尝试生成申请材料 PDF。

成功响应：

```json
{
  "data": {
    "id": 1,
    "applicant_id": 100,
    "approval_type": "leave",
    "status": "pending",
    "current_step": "review",
    "title": "五一离校请假申请",
    "form_data": {
      "reason": "回家处理事务",
      "start_at": "2026-05-01T09:00:00+08:00",
      "end_at": "2026-05-03T18:00:00+08:00",
      "contact_phone": "13800000000"
    },
    "attachment_file_ids": [18],
    "template_file_id": 12,
    "current_approver_id": null,
    "semester": "2025-2026-2",
    "due_at": "2026-04-28T10:00:00Z",
    "submitted_at": "2026-04-27T10:00:00Z",
    "decided_at": null,
    "created_at": "2026-04-27T10:00:00Z",
    "updated_at": "2026-04-27T10:00:00Z"
  }
}
```

### GET /api/v1/approvals/me

说明：

- 返回当前用户提交的审批列表。
- 列表是 `model.Approval` 数组，外层仍然是 `{"data":[...],"total":N}`。

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "applicant_id": 100,
      "approval_type": "leave",
      "status": "pending",
      "current_step": "review",
      "title": "五一离校请假申请",
      "form_data": {
        "reason": "回家处理事务",
        "start_at": "2026-05-01T09:00:00+08:00",
        "end_at": "2026-05-03T18:00:00+08:00",
        "contact_phone": "13800000000"
      },
      "attachment_file_ids": [18],
      "template_file_id": 12,
      "current_approver_id": null,
      "semester": "2025-2026-2",
      "due_at": "2026-04-28T10:00:00Z",
      "submitted_at": "2026-04-27T10:00:00Z",
      "decided_at": null,
      "created_at": "2026-04-27T10:00:00Z",
      "updated_at": "2026-04-27T10:00:00Z"
    }
  ],
  "total": 1
}
```

### GET /api/v1/approvals/:id

成功响应：

```json
{
  "data": {
    "approval": {
      "id": 1,
      "applicant_id": 100,
      "approval_type": "leave",
      "status": "pending",
      "current_step": "review",
      "title": "五一离校请假申请",
      "form_data": {
        "reason": "回家处理事务",
        "start_at": "2026-05-01T09:00:00+08:00",
        "end_at": "2026-05-03T18:00:00+08:00",
        "contact_phone": "13800000000"
      },
      "attachment_file_ids": [18],
      "template_file_id": 12,
      "current_approver_id": null,
      "semester": "2025-2026-2",
      "due_at": "2026-04-28T10:00:00Z",
      "submitted_at": "2026-04-27T10:00:00Z",
      "decided_at": null,
      "created_at": "2026-04-27T10:00:00Z",
      "updated_at": "2026-04-27T10:00:00Z"
    },
    "actions": [
      {
        "id": 10,
        "approval_id": 1,
        "action_type": "submit",
        "operator_id": 100,
        "from_status": "",
        "to_status": "pending",
        "comment": "",
        "snapshot": {
          "submitted_at": "2026-04-27T10:00:00Z"
        },
        "created_at": "2026-04-27T10:00:00Z"
      }
    ],
    "certificate_records": [
      {
        "id": 3,
        "approval_id": 1,
        "applicant_id": 100,
        "template_id": 12,
        "document_stage": "application",
        "certificate_no": "",
        "verification_code": "",
        "verification_hash": "",
        "rendered_payload": {},
        "document_id": 88,
        "seal_status": "none",
        "seal_applied_by": 0,
        "seal_applied_at": null,
        "status": "generated",
        "error_message": "",
        "generated_at": "2026-04-27T10:00:00Z",
        "revoked_at": null,
        "created_at": "2026-04-27T10:00:00Z",
        "updated_at": "2026-04-27T10:00:00Z"
      }
    ]
  }
}
```

说明：

- 学生端只要满足 scope，就可以看自己的审批详情。
- `certificate_records` 会包含申请材料 PDF 和审批结果凭证记录，具体数量取决于审批阶段。

### POST /api/v1/approvals/:id/withdraw

说明：

- 只允许 `pending` 状态撤回。
- 撤回后状态变为 `withdrawn`，会清空 `current_step` 和 `due_at`。

成功响应：

```json
{
  "data": {
    "withdrawn": true
  }
}
```

## Admin Endpoints

- `GET /api/v1/admin/approvals?status=pending&approval_type=leave&limit=20&offset=0`
- `GET /api/v1/admin/approvals/:id`
- `POST /api/v1/admin/approvals/:id/review`
- `POST /api/v1/admin/approvals/:id/assign`
- `POST /api/v1/admin/approvals/:id/remind`
- `POST /api/v1/admin/approvals/scan-overdue`

权限：

- 团干部、教师、超管都通过 `authz` 做 scope 校验。
- 团干部只能 scope 内查看和提醒，不能越权处理。

### GET /api/v1/admin/approvals

说明：

- 返回 `model.Approval` 列表。
- 支持 `status` 和 `approval_type` 过滤。

### GET /api/v1/admin/approvals/:id

说明：

- 返回结构与学生端详情一致。
- scope 校验按管理员角色和班级/年级范围执行。

### POST /api/v1/admin/approvals/:id/review

请求体：

```json
{
  "action": "approve",
  "comment": "情况属实，同意"
}
```

说明：

- `action` 只支持 `approve` 和 `reject`。
- 只允许对 `pending` 审批做最终处理。
- 审批通过后会自动尝试生成审批结果凭证。

成功响应：

```json
{
  "data": {
    "reviewed": true
  }
}
```

### POST /api/v1/admin/approvals/:id/assign

请求体：

```json
{
  "approver_id": 900,
  "comment": "转交负责老师处理"
}
```

说明：

- 请求字段名是 `approver_id`，不是 `current_approver_id`。
- 只能在 `pending` 状态转交。

成功响应：

```json
{
  "data": {
    "assigned": true
  }
}
```

### POST /api/v1/admin/approvals/:id/remind

请求体可为空：

```json
{
  "comment": "请尽快处理"
}
```

成功响应：

```json
{
  "data": {
    "reminded": true
  }
}
```

### POST /api/v1/admin/approvals/scan-overdue

说明：

- 扫描当前时间之前已超时的 `pending` 审批。
- 不需要请求体，扫描时间由后端当前时间决定。

成功响应：

```json
{
  "data": {
    "scanned": 5,
    "reminded": 2
  }
}
```

## Error Responses

- `400 {"error":"invalid approval_type"}`
- `400 {"error":"invalid form_data"}`
- `400 {"error":"invalid approval state"}`
- `400 {"error":"invalid id"}`
- `400 {"error":"invalid request"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"approval not found"}`
- `500 {"error":"query failed"}`
- `500 {"error":"operation failed"}`

## Notes

- `GET /api/v1/approvals/:id` 和 `GET /api/v1/admin/approvals/:id` 都是嵌套结构，不是把 `approval` 字段拍平。
- `certificate_records` 由证书模块维护，审批模块只负责带出关联记录。
- `ListMine` / `ListAdmin` 返回的都是 `model.Approval` 数组。
