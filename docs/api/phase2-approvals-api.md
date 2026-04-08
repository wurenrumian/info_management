# Phase2 Approvals API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`

## Student Endpoints

- `POST /api/v1/approvals`
- `GET /api/v1/approvals/me?status=pending&limit=20&offset=0`
- `GET /api/v1/approvals/:id`
- `POST /api/v1/approvals/:id/withdraw`

### POST /api/v1/approvals

请求体：

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

成功响应：

```json
{
  "data": {
    "id": 1,
    "approval_type": "leave",
    "status": "pending",
    "title": "五一前请假申请",
    "form_data": {
      "reason": "回家处理事务",
      "start_date": "2026-04-10",
      "end_date": "2026-04-12",
      "contact_phone": "13800000000"
    },
    "attachment_file_ids": [12, 18],
    "current_approver_id": 900,
    "semester": "2025-2026-2",
    "submitted_at": "2026-04-08T10:00:00Z",
    "created_at": "2026-04-08T10:00:00Z",
    "updated_at": "2026-04-08T10:00:00Z"
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
      "title": "五一前请假申请",
      "semester": "2025-2026-2",
      "submitted_at": "2026-04-08T10:00:00Z",
      "updated_at": "2026-04-08T10:00:00Z"
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
    "title": "五一前请假申请",
    "form_data": {
      "reason": "回家处理事务",
      "start_date": "2026-04-10",
      "end_date": "2026-04-12",
      "contact_phone": "13800000000"
    },
    "attachment_file_ids": [12, 18],
    "current_approver_id": 900,
    "semester": "2025-2026-2",
    "actions": [
      {
        "id": 10,
        "action_type": "submit",
        "operator_id": 100,
        "comment": "",
        "created_at": "2026-04-08T10:00:00Z"
      }
    ],
    "submitted_at": "2026-04-08T10:00:00Z",
    "created_at": "2026-04-08T10:00:00Z",
    "updated_at": "2026-04-08T10:00:00Z"
  }
}
```

### POST /api/v1/approvals/:id/withdraw

成功响应：

```json
{
  "data": {
    "id": 1,
    "status": "withdrawn",
    "updated_at": "2026-04-08T11:00:00Z"
  }
}
```

## Admin Endpoints

- `GET /api/v1/admin/approvals?status=pending&approval_type=leave&limit=20&offset=0`
- `GET /api/v1/admin/approvals/:id`
- `PATCH /api/v1/admin/approvals/:id`

权限：
- `role >= 2`

### PATCH /api/v1/admin/approvals/:id

请求体：

```json
{
  "action": "approve",
  "comment": "情况属实，准假"
}
```

成功响应：

```json
{
  "data": {
    "id": 1,
    "status": "approved",
    "decided_at": "2026-04-08T12:00:00Z",
    "updated_at": "2026-04-08T12:00:00Z"
  }
}
```

错误响应：

- `400 {"error":"invalid approval_type"}`
- `400 {"error":"invalid action"}`
- `400 {"error":"invalid id"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"approval not found"}`
- `409 {"error":"approval status is not pending"}`

## Approval Type Enum

- `leave`
- `stamp`

## Status Enum

- `pending`
- `approved`
- `rejected`
- `withdrawn`

## Audit Log

写操作记录到 `admin_logs`：

- `approvals.create`
- `approvals.withdraw`
- `approvals.review`

