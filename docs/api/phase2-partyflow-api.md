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
        },
        {
          "id": 12,
          "event_type": "reminder_sent",
          "event_code": "party_activist_report_every_90d",
          "note": "思想汇报提醒已发送",
          "happened_at": "2026-05-30T08:00:00Z",
          "metadata": {
            "period_index": 1
          }
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

## Admin Status Endpoints

- `GET /api/v1/admin/partyflow/statuses?org_type=party&student_id=2023001&limit=20&offset=0`
- `GET /api/v1/admin/partyflow/statuses/:id`
- `POST /api/v1/admin/partyflow/statuses`
- `PATCH /api/v1/admin/partyflow/statuses/:id`
- `POST /api/v1/admin/partyflow/statuses/import`
- `POST /api/v1/admin/partyflow/statuses/:id/events`

权限：
- 团干部/教师/超管按 scope 管理

### GET /api/v1/admin/partyflow/statuses

查询参数：

- `org_type`：可选，`party` / `league`
- `status`：可选，按当前状态过滤
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
      "org_type": "party",
      "status": "activist",
      "status_started_at": "2026-03-01T00:00:00Z",
      "joined_at": null,
      "next_action_hint": "每满3个月提交一次思想汇报",
      "created_at": "2026-03-01T00:00:00Z",
      "updated_at": "2026-03-10T00:00:00Z"
    }
  ],
  "total": 1
}
```

### GET /api/v1/admin/partyflow/statuses/:id

成功响应：

```json
{
  "data": {
    "id": 1,
    "user_id": 100,
    "student_id": "2023001",
    "student_name": "张三",
    "org_type": "party",
    "status": "activist",
    "status_started_at": "2026-03-01T00:00:00Z",
    "joined_at": null,
    "next_action_hint": "每满3个月提交一次思想汇报",
    "metadata": {
      "contact_person": "李老师"
    },
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
    ],
    "created_at": "2026-03-01T00:00:00Z",
    "updated_at": "2026-03-10T00:00:00Z"
  }
}
```

### POST /api/v1/admin/partyflow/statuses

请求体：

```json
{
  "user_id": 100,
  "org_type": "party",
  "status": "activist",
  "status_started_at": "2026-03-01T00:00:00Z",
  "joined_at": null,
  "next_action_hint": "每满3个月提交一次思想汇报",
  "metadata": {
    "contact_person": "李老师"
  },
  "note": "初始化"
}
```

### PATCH /api/v1/admin/partyflow/statuses/:id

请求体：

```json
{
  "status": "development_target",
  "status_started_at": "2026-06-01T00:00:00Z",
  "next_action_hint": "准备发展材料",
  "metadata": {
    "reminder_overrides": {
      "party_activist_report_every_90d": {
        "next_due_at": "2026-06-01T00:00:00Z",
        "reason": "统一材料提交时间提前"
      }
    }
  },
  "note": "进入发展对象阶段"
}
```

### POST /api/v1/admin/partyflow/statuses/import

请求体：

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

### POST /api/v1/admin/partyflow/statuses/:id/events

记录谈话、公示、审批、备案、归档等里程碑事件。

请求体：

```json
{
  "event_type": "milestone",
  "event_code": "party_publicity_done",
  "note": "公示完成",
  "happened_at": "2026-04-01T00:00:00Z",
  "metadata": {
    "file_ids": [12]
  }
}
```

## Admin Reminder Rule Endpoints

- `GET /api/v1/admin/partyflow/reminder-rules?org_type=party&enabled=true`
- `PATCH /api/v1/admin/partyflow/reminder-rules/:id`
- `POST /api/v1/admin/partyflow/reminders/scan`

权限：
- 教师/超管维护规则和手动扫描

### GET /api/v1/admin/partyflow/reminder-rules

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "rule_code": "party_activist_report_every_90d",
      "org_type": "party",
      "status": "activist",
      "trigger_type": "status_started",
      "offset_days": 90,
      "repeat_interval_days": 90,
      "audience": "student",
      "enabled": true,
      "title": "思想汇报提醒",
      "message_template": "你已进入入党积极分子阶段满 {period_days} 天，请按要求提交思想汇报。"
    }
  ]
}
```

### PATCH /api/v1/admin/partyflow/reminder-rules/:id

请求体：

```json
{
  "enabled": true,
  "offset_days": 90,
  "repeat_interval_days": 90,
  "audience": "student",
  "title": "思想汇报提醒",
  "message_template": "你已进入入党积极分子阶段满 {period_days} 天，请按要求提交思想汇报。"
}
```

成功响应：

```json
{
  "data": {
    "id": 1,
    "enabled": true,
    "updated_at": "2026-04-27T08:00:00Z"
  }
}
```

### POST /api/v1/admin/partyflow/reminders/scan

手动触发提醒扫描，用于联调和补偿。

请求体：

```json
{
  "now": "2026-05-30T08:00:00Z"
}
```

成功响应：

```json
{
  "data": {
    "scanned_count": 10,
    "generated_count": 3,
    "sent_count": 2,
    "skipped_count": 1,
    "failed_count": 0
  }
}
```

## Enum

### `org_type`

- `league`
- `party`

### `league.status`

- `none`
- `applicant`
- `activist`
- `development_target`
- `member`

### `party.status`

- `none`
- `applicant`
- `activist`
- `development_target`
- `probationary_member`
- `full_member`

### `event_type`

- `create`
- `status_change`
- `milestone`
- `import`
- `manual_adjust`
- `reminder_sent`

### `trigger_type`

- `status_started`
- `event_completed`
- `fixed_date`

### `audience`

- `student`
- `admin`
- `both`

## Initial Reminder Rules

默认规则由后端 seed 生成，管理员通过 API 启停和调整。

- `league_applicant_talk_30d`
- `league_activist_train_90d`
- `league_development_publicity_5workdays`
- `league_approval_30d`
- `league_member_archive_30d`
- `party_applicant_talk_30d`
- `party_activist_report_every_90d`
- `party_development_training_reminder`
- `party_probationary_transfer_365d`
- `party_probationary_report_every_90d`

## Error Responses

- `400 {"error":"invalid org_type"}`
- `400 {"error":"invalid status"}`
- `400 {"error":"invalid id"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"partyflow status not found"}`
- `404 {"error":"partyflow reminder rule not found"}`

## Audit Log

写操作记录到 `admin_logs`：

- `partyflow.status_create`
- `partyflow.status_patch`
- `partyflow.status_import`
- `partyflow.event_create`
- `partyflow.rule_patch`
- `partyflow.reminder_scan`
