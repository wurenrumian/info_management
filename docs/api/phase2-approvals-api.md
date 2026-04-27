# Phase2 Approvals API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`

## Student Endpoints

- `POST /api/v1/approvals`
- `GET /api/v1/approvals/me?status=pending&approval_type=leave&limit=20&offset=0`
- `GET /api/v1/approvals/:id`
- `POST /api/v1/approvals/:id/withdraw`

### POST /api/v1/approvals

#### Leave Request

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
    "emergency_contact": "13900000000",
    "miss_classes": false,
    "student_type": "undergraduate"
  },
  "template_file_id": 12,
  "attachment_file_ids": [18]
}
```

#### Budget Request

```json
{
  "approval_type": "budget",
  "title": "班级团日活动预算申请",
  "form_data": {
    "activity_name": "班级团日活动",
    "activity_date": "2026-05-20",
    "budget_amount": 1200,
    "purpose": "活动物料与场地费用",
    "items": [
      {"name": "物料", "amount": 500},
      {"name": "场地", "amount": 700}
    ]
  },
  "template_file_id": 18,
  "attachment_file_ids": [19]
}
```

成功响应：

```json
{
  "data": {
    "id": 1,
    "approval_type": "leave",
    "status": "pending",
    "current_step": "review",
    "title": "五一离校请假申请",
    "form_data": {
      "leave_type": "leave_city",
      "reason": "回家处理事务",
      "start_at": "2026-05-01T09:00:00+08:00",
      "end_at": "2026-05-03T18:00:00+08:00",
      "contact_phone": "13800000000"
    },
    "template_file_id": 12,
    "attachment_file_ids": [18],
    "current_approver_id": 900,
    "semester": "2025-2026-2",
    "due_at": "2026-04-28T10:00:00Z",
    "submitted_at": "2026-04-27T10:00:00Z",
    "created_at": "2026-04-27T10:00:00Z",
    "updated_at": "2026-04-27T10:00:00Z"
  }
}
```

### GET /api/v1/approvals/me

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "approval_type": "leave",
      "status": "pending",
      "current_step": "review",
      "title": "五一离校请假申请",
      "semester": "2025-2026-2",
      "due_at": "2026-04-28T10:00:00Z",
      "submitted_at": "2026-04-27T10:00:00Z",
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
    "id": 1,
    "approval_type": "leave",
    "status": "pending",
    "current_step": "review",
    "title": "五一离校请假申请",
    "form_data": {
      "leave_type": "leave_city",
      "reason": "回家处理事务",
      "start_at": "2026-05-01T09:00:00+08:00",
      "end_at": "2026-05-03T18:00:00+08:00",
      "contact_phone": "13800000000"
    },
    "template_file_id": 12,
    "attachment_file_ids": [18],
    "current_approver_id": 900,
    "semester": "2025-2026-2",
    "actions": [
      {
        "id": 10,
        "action_type": "submit",
        "operator_id": 100,
        "from_status": "",
        "to_status": "pending",
        "comment": "",
        "created_at": "2026-04-27T10:00:00Z"
      }
    ],
    "certificate_records": [],
    "submitted_at": "2026-04-27T10:00:00Z",
    "created_at": "2026-04-27T10:00:00Z",
    "updated_at": "2026-04-27T10:00:00Z"
  }
}
```

说明：
- `GET /api/v1/admin/approvals/:id` 返回相同详情结构，但按管理员 scope 校验访问范围。
- `certificate_records` 为审批与电子证明的统一集成字段；当尚无可用 PDF 记录时返回空数组。

### POST /api/v1/approvals/:id/withdraw

成功响应：

```json
{
  "data": {
    "id": 1,
    "status": "withdrawn",
    "updated_at": "2026-04-27T11:00:00Z"
  }
}
```

## Admin Endpoints

- `GET /api/v1/admin/approvals?status=pending&approval_type=leave&limit=20&offset=0`
- `GET /api/v1/admin/approvals/:id`
- `PATCH /api/v1/admin/approvals/:id/review`
- `PATCH /api/v1/admin/approvals/:id/assign`
- `POST /api/v1/admin/approvals/:id/remind`
- `POST /api/v1/admin/approvals/overdue/scan`

权限：
- 团干部可在 scope 内查看和提醒，但不可最终审批
- 教师/超管可在 scope 内审批、转交、提醒

### PATCH /api/v1/admin/approvals/:id/review

请求体：

```json
{
  "action": "approve",
  "comment": "情况属实，同意"
}
```

成功响应：

```json
{
  "data": {
    "id": 1,
    "status": "approved",
    "decided_at": "2026-04-27T12:00:00Z",
    "updated_at": "2026-04-27T12:00:00Z"
  }
}
```

### PATCH /api/v1/admin/approvals/:id/assign

请求体：

```json
{
  "current_approver_id": 900,
  "comment": "转交负责老师处理"
}
```

成功响应：

```json
{
  "data": {
    "id": 1,
    "current_approver_id": 900,
    "updated_at": "2026-04-27T12:00:00Z"
  }
}
```

### POST /api/v1/admin/approvals/:id/remind

成功响应：

```json
{
  "data": {
    "id": 1,
    "reminded": true
  }
}
```

### POST /api/v1/admin/approvals/overdue/scan

手动扫描超时 pending 申请并提醒当前审批人。

成功响应：

```json
{
  "data": {
    "scanned_count": 5,
    "reminded_count": 2,
    "failed_count": 0
  }
}
```

## Approval Type Enum

- `leave`
- `budget`

## Status Enum

- `pending`
- `approved`
- `rejected`
- `withdrawn`
- `expired`

## Action Type Enum

- `submit`
- `approve`
- `reject`
- `withdraw`
- `assign`
- `remind`
- `expire`

## Error Responses

- `400 {"error":"invalid approval_type"}`
- `400 {"error":"invalid action"}`
- `400 {"error":"invalid form_data"}`
- `400 {"error":"invalid id"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"approval not found"}`
- `409 {"error":"approval status is not pending"}`

## Audit Log

写操作记录到 `admin_logs`：

- `approvals.create`
- `approvals.withdraw`
- `approvals.review`
- `approvals.assign`
- `approvals.remind`
- `approvals.overdue_scan`

## Document Library Boundary

以下内容不进入审批流，由文档库/知识库承载：

- 奖学金助学金申请细则
- 休学复学细则
- 宿舍调整申请细则
- 校历和节假日具体时间
- 请假条模板
- 预算申请模板

## Certificates Integration

- `GET /api/v1/approvals/:id` 与 `GET /api/v1/admin/approvals/:id` 统一返回 `certificate_records`。
- 申请材料 PDF 与审批结果凭证由 `certificates` 模块生成和维护。
- 若审批尚未生成任何可用 PDF，`certificate_records` 返回 `[]`。
