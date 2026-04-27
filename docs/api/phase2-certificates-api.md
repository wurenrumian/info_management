# Phase2 Certificates API

电子证明模块在 v1 中服务审批流程，只生成两类 PDF：
- 申请材料 PDF：审批前/审批中查看，不盖章，不带核验码。
- 审批结果凭证 PDF：审批通过后生成，带编号、核验码和内部审批章/系统生成章。

## Base URL

`/api/v1`

## Required Headers

- 除核验接口外，均需要 `Authorization: Bearer <token>`
- `GET /api/v1/certificates/verify` 不要求登录，但必须只返回最小必要信息

## Template Codes

| code | approval_type | document_stage | 说明 |
|------|---------------|----------------|------|
| `leave_application_pdf` | `leave` | `application` | 请假申请材料 PDF |
| `leave_approval_certificate` | `leave` | `approval_certificate` | 请假审批通过凭证 |
| `budget_application_pdf` | `budget` | `application` | 预算申请材料 PDF |
| `budget_approval_certificate` | `budget` | `approval_certificate` | 预算审批通过凭证 |

## Student Endpoints

- `GET /api/v1/certificates/me?approval_type=leave&limit=20&offset=0`
- `GET /api/v1/certificates/:id`

学生通常不直接调用生成接口。申请材料 PDF 和审批结果凭证由审批流程自动生成，并通过审批详情返回。

### GET /api/v1/certificates/me

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "approval_id": 10,
      "approval_type": "leave",
      "document_stage": "application",
      "certificate_no": "",
      "document_id": 88,
      "download_url": "/api/v1/files/88/download",
      "seal_status": "none",
      "status": "generated",
      "generated_at": "2026-04-27T12:00:00+08:00"
    },
    {
      "id": 2,
      "approval_id": 10,
      "approval_type": "leave",
      "document_stage": "approval_certificate",
      "certificate_no": "LEAVE-20260427-000010",
      "document_id": 89,
      "download_url": "/api/v1/files/89/download",
      "seal_status": "internal_seal_applied",
      "status": "generated",
      "generated_at": "2026-04-27T13:00:00+08:00"
    }
  ],
  "total": 2
}
```

### GET /api/v1/certificates/:id

成功响应：

```json
{
  "data": {
    "id": 2,
    "approval_id": 10,
    "approval_type": "leave",
    "document_stage": "approval_certificate",
    "certificate_no": "LEAVE-20260427-000010",
    "verification_code": "9X4K-2M7Q-P8FD",
    "document_id": 89,
    "download_url": "/api/v1/files/89/download",
    "seal_status": "internal_seal_applied",
    "status": "generated",
    "generated_at": "2026-04-27T13:00:00+08:00"
  }
}
```

## Public Verification Endpoint

- `GET /api/v1/certificates/verify?code=xxx`

### GET /api/v1/certificates/verify

Query:

- `code`: 审批结果凭证核验码

成功响应：

```json
{
  "data": {
    "valid": true,
    "status": "generated",
    "certificate_no": "LEAVE-20260427-000010",
    "approval_type": "leave",
    "document_stage": "approval_certificate",
    "applicant_name": "张三",
    "generated_at": "2026-04-27T13:00:00+08:00",
    "disclaimer": "本文件由学院信息管理系统自动生成，仅用于学院内部审批留痕与流转，不等同于学校正式公章文件。"
  }
}
```

说明：
- 核验接口只返回最小必要信息。
- 不返回身份证号、联系方式、详细请假原因、预算明细等敏感字段。
- 已作废凭证返回 `valid=false` 或 `status=revoked`。

## Approval Detail Integration

统一约定：
- `GET /api/v1/approvals/:id` 与 `GET /api/v1/admin/approvals/:id` 返回相同的 `certificate_records` 字段。
- 当尚无可用 PDF 记录时，`certificate_records` 返回空数组。

示例：

```json
{
  "data": {
    "id": 10,
    "approval_type": "leave",
    "status": "approved",
    "certificate_records": [
      {
        "id": 1,
        "document_stage": "application",
        "document_id": 88,
        "download_url": "/api/v1/files/88/download",
        "seal_status": "none",
        "status": "generated"
      },
      {
        "id": 2,
        "document_stage": "approval_certificate",
        "certificate_no": "LEAVE-20260427-000010",
        "document_id": 89,
        "download_url": "/api/v1/files/89/download",
        "seal_status": "internal_seal_applied",
        "status": "generated"
      }
    ]
  }
}
```

## Admin Endpoints

- `GET /api/v1/admin/certificates/templates`
- `POST /api/v1/admin/certificates/templates/:id/activate`
- `POST /api/v1/admin/certificates/templates/:id/deactivate`
- `POST /api/v1/admin/approvals/:id/application-pdf/regenerate`
- `POST /api/v1/admin/approvals/:id/certificate/regenerate`
- `POST /api/v1/admin/certificates/:id/revoke`

权限：
- 教师/超管：可管理模板启停、重试生成、作废凭证。
- 团干部：可查看 scope 内审批材料和提醒，不可生成或作废审批结果凭证。

### GET /api/v1/admin/certificates/templates

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "code": "leave_application_pdf",
      "name": "请假申请材料 PDF",
      "approval_type": "leave",
      "document_stage": "application",
      "status": "active",
      "renderer": "typst",
      "template_version": "v1"
    }
  ],
  "total": 4
}
```

### POST /api/v1/admin/certificates/templates/:id/activate

成功响应：

```json
{
  "data": {
    "id": 1,
    "status": "active"
  }
}
```

### POST /api/v1/admin/certificates/templates/:id/deactivate

成功响应：

```json
{
  "data": {
    "id": 1,
    "status": "inactive"
  }
}
```

### POST /api/v1/admin/approvals/:id/application-pdf/regenerate

说明：
- 用于申请材料 PDF 生成失败或模板更新后的手动重试。
- 可用于 `pending`、`approved`、`rejected`、`withdrawn`、`expired` 状态的审批。
- 不生成编号、核验码和内部章。

成功响应：

```json
{
  "data": {
    "record_id": 3,
    "approval_id": 10,
    "document_stage": "application",
    "document_id": 90,
    "download_url": "/api/v1/files/90/download",
    "seal_status": "none",
    "status": "generated"
  }
}
```

### POST /api/v1/admin/approvals/:id/certificate/regenerate

说明：
- 仅 `approved` 状态允许生成审批结果凭证。
- 生成结果带 `certificate_no`、`verification_code` 和内部章。

成功响应：

```json
{
  "data": {
    "record_id": 4,
    "approval_id": 10,
    "document_stage": "approval_certificate",
    "certificate_no": "LEAVE-20260427-000010",
    "verification_code": "9X4K-2M7Q-P8FD",
    "document_id": 91,
    "download_url": "/api/v1/files/91/download",
    "seal_status": "internal_seal_applied",
    "status": "generated"
  }
}
```

### POST /api/v1/admin/certificates/:id/revoke

请求体：

```json
{
  "reason": "审批结果被撤销或重新生成"
}
```

成功响应：

```json
{
  "data": {
    "id": 4,
    "status": "revoked",
    "revoked_at": "2026-04-27T15:00:00+08:00"
  }
}
```

## Status Enum

Template status:
- `active`
- `inactive`

Record status:
- `generated`
- `failed`
- `revoked`

Document stage:
- `application`
- `approval_certificate`

Seal status:
- `none`
- `internal_seal_applied`

## Error Responses

- `400 {"error":"invalid id"}`
- `400 {"error":"missing verification code"}`
- `400 {"error":"approval is not approved"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"certificate record not found"}`
- `404 {"error":"certificate template not found"}`
- `500 {"error":"failed to render pdf"}`

## Audit Log

写操作记录到 `admin_logs`：

- `certificates.template_toggle`
- `certificates.application_pdf_regenerate`
- `certificates.approval_certificate_regenerate`
- `certificates.revoke`

自动生成也应写入业务日志或 `certificate_records`，便于追踪生成失败和重试历史。
