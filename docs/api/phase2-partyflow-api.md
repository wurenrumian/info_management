# Phase2 PartyFlow API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`

## Student Endpoint

- `GET /api/v1/partyflow/me`

权限：
- 登录学生

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "flow_type": "party",
      "current_stage": "activist",
      "stage_started_at": "2026-03-01T00:00:00Z",
      "next_action_hint": "满3个月后提交思想汇报",
      "reminder_rule_code": "party_activist_90d",
      "history": [
        {
          "id": 11,
          "from_stage": "applicant",
          "to_stage": "activist",
          "event_type": "update",
          "note": "支部确认",
          "happened_at": "2026-03-01T00:00:00Z"
        }
      ],
      "created_at": "2026-03-01T00:00:00Z",
      "updated_at": "2026-03-01T00:00:00Z"
    }
  ]
}
```

错误响应：

- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`

## Admin Endpoints

- `GET /api/v1/admin/partyflow/progress?flow_type=party&student_id=2023001&limit=20&offset=0`
- `GET /api/v1/admin/partyflow/progress/:id`
- `POST /api/v1/admin/partyflow/progress`
- `PATCH /api/v1/admin/partyflow/progress/:id`
- `POST /api/v1/admin/partyflow/progress/import`

权限：
- `role >= 2`

### GET /api/v1/admin/partyflow/progress

查询参数：

- `flow_type`：可选，`party` / `league`
- `student_id`：可选，按学号过滤
- `limit`：可选，默认 `20`
- `offset`：可选，默认 `0`

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "user_id": 100,
      "student_id": "2023001",
      "student_name": "张三",
      "flow_type": "party",
      "current_stage": "activist",
      "stage_started_at": "2026-03-01T00:00:00Z",
      "next_action_hint": "满3个月后提交思想汇报",
      "reminder_rule_code": "party_activist_90d",
      "created_at": "2026-03-01T00:00:00Z",
      "updated_at": "2026-03-10T00:00:00Z"
    }
  ],
  "total": 1
}
```

### GET /api/v1/admin/partyflow/progress/:id

成功响应：

```json
{
  "data": {
    "id": 1,
    "user_id": 100,
    "student_id": "2023001",
    "student_name": "张三",
    "flow_type": "party",
    "current_stage": "activist",
    "stage_started_at": "2026-03-01T00:00:00Z",
    "next_action_hint": "满3个月后提交思想汇报",
    "reminder_rule_code": "party_activist_90d",
    "history": [
      {
        "id": 11,
        "from_stage": "applicant",
        "to_stage": "activist",
        "event_type": "update",
        "note": "支部确认",
        "happened_at": "2026-03-01T00:00:00Z"
      }
    ],
    "created_at": "2026-03-01T00:00:00Z",
    "updated_at": "2026-03-10T00:00:00Z"
  }
}
```

### POST /api/v1/admin/partyflow/progress

请求体：

```json
{
  "user_id": 100,
  "flow_type": "party",
  "current_stage": "activist",
  "stage_started_at": "2026-03-01T00:00:00Z",
  "next_action_hint": "满3个月后提交思想汇报",
  "note": "初始化"
}
```

### PATCH /api/v1/admin/partyflow/progress/:id

请求体：

```json
{
  "current_stage": "development_target",
  "stage_started_at": "2026-06-01T00:00:00Z",
  "next_action_hint": "准备发展材料",
  "note": "进入发展对象阶段"
}
```

### POST /api/v1/admin/partyflow/progress/import

请求体：

```json
{
  "items": [
    {
      "student_id": "2023001",
      "flow_type": "party",
      "current_stage": "activist",
      "stage_started_at": "2026-03-01T00:00:00Z",
      "next_action_hint": "满3个月后提交思想汇报",
      "note": "班级批量导入"
    }
  ]
}
```

成功响应：

```json
{
  "data": {
    "success_count": 1,
    "failed_count": 0,
    "failed_items": []
  }
}
```

错误响应：

- `400 {"error":"invalid flow_type"}`
- `400 {"error":"invalid current_stage"}`
- `400 {"error":"invalid id"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"partyflow progress not found"}`

## Stage Enum

### `league`

- `applicant`
- `activist`
- `member`

### `party`

- `applicant`
- `activist`
- `development_target`
- `probationary_member`
- `full_member`

## Audit Log

写操作记录到 `admin_logs`：

- `partyflow.create`
- `partyflow.patch`
- `partyflow.import`

