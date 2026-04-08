# Phase2 Certificates API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`

## Student Endpoints

- `GET /api/v1/certificates/templates`
- `POST /api/v1/certificates/generate`
- `GET /api/v1/certificates/me?limit=20&offset=0`

### GET /api/v1/certificates/templates

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "code": "student_status",
      "name": "在读证明",
      "status": "active"
    }
  ]
}
```

### POST /api/v1/certificates/generate

请求体：

```json
{
  "template_code": "student_status"
}
```

成功响应：

```json
{
  "data": {
    "record_id": 1,
    "template_code": "student_status",
    "document_id": 88,
    "download_url": "/api/v1/files/88/download",
    "created_at": "2026-04-08T12:00:00Z"
  }
}
```

### GET /api/v1/certificates/me

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "template_code": "student_status",
      "document_id": 88,
      "status": "generated",
      "created_at": "2026-04-08T12:00:00Z"
    }
  ],
  "total": 1
}
```

## Admin Endpoints

- `GET /api/v1/admin/certificates/templates`
- `POST /api/v1/admin/certificates/templates`
- `PATCH /api/v1/admin/certificates/templates/:id`
- `POST /api/v1/admin/certificates/templates/:id/activate`
- `POST /api/v1/admin/certificates/templates/:id/deactivate`

权限：
- `role >= 2`

### POST /api/v1/admin/certificates/templates

请求体：

```json
{
  "code": "student_status",
  "name": "在读证明",
  "template_schema": {
    "title": "在读证明",
    "body_lines": [
      "兹证明 {student_name}，学号 {student_id}，系 {grade} 级 {major} 学生。",
      "特此证明。"
    ],
    "footer": "信息学院"
  },
  "field_mapping": {
    "student_name": "user.name",
    "student_id": "user.student_id",
    "grade": "user.grade",
    "major": "class.major"
  }
}
```

### PATCH /api/v1/admin/certificates/templates/:id

请求体：

```json
{
  "name": "本科生在读证明"
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

错误响应：

- `400 {"error":"missing template_code"}`
- `400 {"error":"invalid id"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"certificate template not found"}`

## Status Enum

- `active`
- `inactive`
- `generated`
- `failed`

## Audit Log

写操作记录到 `admin_logs`：

- `certificates.template_create`
- `certificates.template_patch`
- `certificates.template_toggle`
- `certificates.generate`

