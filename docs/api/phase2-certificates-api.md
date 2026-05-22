# Phase2 Certificates API

电子证明模块在 v1 中服务审批流程，实际返回的是 `model.CertificateTemplate`、`model.CertificateRecord` 和核验结果结构。

## Base URL

`/api/v1`

## Required Headers

- 除核验接口外，均需要 `Authorization: Bearer <token>`
- `GET /api/v1/certificates/verify` 不要求登录

## Template Codes

当前仓库使用的模板 code 仍按以下约定：

| code | approval_type | document_stage |
|------|---------------|----------------|
| `leave_application_pdf` | `leave` | `application` |
| `leave_approval_certificate` | `leave` | `approval_certificate` |
| `budget_application_pdf` | `budget` | `application` |
| `budget_approval_certificate` | `budget` | `approval_certificate` |

## Student Endpoints

- `GET /api/v1/certificates/me?approval_type=leave&limit=20&offset=0`
- `GET /api/v1/certificates/:id`

### GET /api/v1/certificates/me

说明：

- 返回当前用户名下的证书记录列表。
- 可以用 `approval_type` 过滤。
- 结果使用 `{"data":[...],"total":N}` 包装。

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "approval_id": 10,
      "applicant_id": 100,
      "template_id": 1,
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
      "generated_at": "2026-04-27T12:00:00+08:00",
      "revoked_at": null,
      "created_at": "2026-04-27T12:00:00+08:00",
      "updated_at": "2026-04-27T12:00:00+08:00"
    },
    {
      "id": 2,
      "approval_id": 10,
      "applicant_id": 100,
      "template_id": 2,
      "document_stage": "approval_certificate",
      "certificate_no": "LEAVE-20260427-000010",
      "verification_code": "LEAVE-10-1745751600000000000",
      "verification_hash": "9f0f...",
      "rendered_payload": {},
      "document_id": 89,
      "seal_status": "internal_seal_applied",
      "seal_applied_by": 0,
      "seal_applied_at": "2026-04-27T13:00:00+08:00",
      "status": "generated",
      "error_message": "",
      "generated_at": "2026-04-27T13:00:00+08:00",
      "revoked_at": null,
      "created_at": "2026-04-27T13:00:00+08:00",
      "updated_at": "2026-04-27T13:00:00+08:00"
    }
  ],
  "total": 2
}
```

### GET /api/v1/certificates/:id

说明：

- 返回单条 `model.CertificateRecord`。
- 访问范围按审批人/班级/年级 scope 校验。

成功响应同上单条记录。

## Public Verification Endpoint

- `GET /api/v1/certificates/verify?code=xxx`

### GET /api/v1/certificates/verify

Query：

- `code`：核验码

说明：

- 该接口只返回最小必要信息。
- 不是 `valid/disclaimer` 风格，也不会返回完整 `rendered_payload`。

成功响应：

```json
{
  "data": {
    "record_id": 2,
    "approval_id": 10,
    "applicant_id": 100,
    "approval_type": "leave",
    "document_stage": "approval_certificate",
    "certificate_no": "LEAVE-20260427-000010",
    "verification_code": "LEAVE-10-1745751600000000000",
    "status": "generated",
    "generated_at": "2026-04-27T13:00:00+08:00"
  }
}
```

说明：

- 空 `code` 返回 `400 invalid verification code`。
- 找不到记录返回 `404 certificate not found`。
- 如果记录已作废，仍会返回该记录的元数据，但 `status` 会是 `revoked`。

## Admin Endpoints

- `GET /api/v1/admin/certificates/templates`
- `POST /api/v1/admin/certificates/templates/:id/activate`
- `POST /api/v1/admin/certificates/templates/:id/deactivate`
- `POST /api/v1/admin/approvals/:id/application-pdf/regenerate`
- `POST /api/v1/admin/approvals/:id/certificate/regenerate`
- `POST /api/v1/admin/certificates/:id/revoke`

权限：

- 教师/超管可管理模板、重试生成和作废。
- 团干部只能在 scope 内查看审批相关内容，不可做这些写操作。

### GET /api/v1/admin/certificates/templates

说明：

- 返回 `model.CertificateTemplate` 列表。
- `total` 等于当前返回数组长度。

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
      "template_path": "templates/certificates/leave_application.typ",
      "template_version": "v1",
      "field_mapping": {},
      "disclaimer": "",
      "created_by": 0,
      "updated_by": 0,
      "created_at": "2026-04-01T08:00:00Z",
      "updated_at": "2026-04-01T08:00:00Z"
    }
  ],
  "total": 4
}
```

### POST /api/v1/admin/certificates/templates/:id/activate

### POST /api/v1/admin/certificates/templates/:id/deactivate

说明：

- 这两个接口共用同一个 handler。
- 路径以 `/activate` 结尾时设置为 `active`，否则设置为 `inactive`。
- 返回完整的模板记录。

成功响应：

```json
{
  "data": {
    "id": 1,
    "code": "leave_application_pdf",
    "name": "请假申请材料 PDF",
    "approval_type": "leave",
    "document_stage": "application",
    "status": "inactive",
    "renderer": "typst",
    "template_path": "templates/certificates/leave_application.typ",
    "template_version": "v1",
    "field_mapping": {},
    "disclaimer": "",
    "created_by": 0,
    "updated_by": 0,
    "created_at": "2026-04-01T08:00:00Z",
    "updated_at": "2026-04-01T08:00:00Z"
  }
}
```

### POST /api/v1/admin/approvals/:id/application-pdf/regenerate

说明：

- 手动重试申请材料 PDF 生成。
- 返回完整的 `model.CertificateRecord`。

成功响应：

```json
{
  "data": {
    "id": 3,
    "approval_id": 10,
    "applicant_id": 100,
    "template_id": 1,
    "document_stage": "application",
    "certificate_no": "",
    "verification_code": "",
    "verification_hash": "",
    "rendered_payload": {},
    "document_id": 90,
    "seal_status": "none",
    "seal_applied_by": 0,
    "seal_applied_at": null,
    "status": "generated",
    "error_message": "",
    "generated_at": "2026-04-27T14:00:00+08:00",
    "revoked_at": null,
    "created_at": "2026-04-27T14:00:00+08:00",
    "updated_at": "2026-04-27T14:00:00+08:00"
  }
}
```

### POST /api/v1/admin/approvals/:id/certificate/regenerate

说明：

- 只允许 `approved` 状态的审批生成审批结果凭证。
- 会生成 `certificate_no`、`verification_code`、`verification_hash` 和内部章状态。

成功响应：

```json
{
  "data": {
    "id": 4,
    "approval_id": 10,
    "applicant_id": 100,
    "template_id": 2,
    "document_stage": "approval_certificate",
    "certificate_no": "LEAVE-20260427-000010",
    "verification_code": "LEAVE-10-1745751600000000000",
    "verification_hash": "9f0f...",
    "rendered_payload": {},
    "document_id": 91,
    "seal_status": "internal_seal_applied",
    "seal_applied_by": 0,
    "seal_applied_at": "2026-04-27T14:00:00+08:00",
    "status": "generated",
    "error_message": "",
    "generated_at": "2026-04-27T14:00:00+08:00",
    "revoked_at": null,
    "created_at": "2026-04-27T14:00:00+08:00",
    "updated_at": "2026-04-27T14:00:00+08:00"
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

说明：

- `reason` 可为空字符串，但如果传了非空内容，会写入 `error_message`。
- 作废后记录状态变为 `revoked`，并写入 `revoked_at`。

成功响应：

```json
{
  "data": {
    "id": 4,
    "approval_id": 10,
    "applicant_id": 100,
    "template_id": 2,
    "document_stage": "approval_certificate",
    "certificate_no": "LEAVE-20260427-000010",
    "verification_code": "LEAVE-10-1745751600000000000",
    "verification_hash": "9f0f...",
    "rendered_payload": {},
    "document_id": 91,
    "seal_status": "internal_seal_applied",
    "seal_applied_by": 0,
    "seal_applied_at": "2026-04-27T14:00:00+08:00",
    "status": "revoked",
    "error_message": "审批结果被撤销或重新生成",
    "generated_at": "2026-04-27T14:00:00+08:00",
    "revoked_at": "2026-04-27T15:00:00+08:00",
    "created_at": "2026-04-27T14:00:00+08:00",
    "updated_at": "2026-04-27T15:00:00+08:00"
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
- `400 {"error":"invalid verification code"}`
- `400 {"error":"approval not approved"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"certificate not found"}`
- `500 {"error":"query failed"}`
- `500 {"error":"operation failed"}`

## Notes

- `GET /api/v1/certificates/me` 和 `GET /api/v1/certificates/:id` 都返回完整 `model.CertificateRecord`。
- 公开核验接口只回传最小必要字段，便于前端展示和审计，不会暴露 `rendered_payload`。
- 模板列表和切换接口都返回完整模板对象，而不是只返回 `id/status`。
