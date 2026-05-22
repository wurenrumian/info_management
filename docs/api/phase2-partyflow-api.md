# Phase2 PartyFlow API

## Base URL

`/api/v1`

## Required Headers

- `Authorization: Bearer <token>`

## Overview

- `GET /api/v1/partyflow/me` 返回当前用户的党/团流程状态列表，包含历史事件。
- 管理端按 scope 管理状态、事件、提醒规则。
- 规则扫描和事件创建的返回体都和当前 handler 一致，直接返回模型或汇总对象。

## Student Endpoint

- `GET /api/v1/partyflow/me`

权限：

- 登录用户且拥有 `partyflow:me:get` 权限。

成功响应：

```json
{
  "data": [
    {
      "id": 1,
      "user_id": 100,
      "org_type": "party",
      "status": "activist",
      "status_started_at": "2026-03-01T00:00:00Z",
      "joined_at": null,
      "next_action_hint": "每满3个月提交一次思想汇报",
      "metadata": {},
      "history": [
        {
          "id": 11,
          "partyflow_status_id": 1,
          "event_type": "status_change",
          "event_code": "status_patched",
          "from_status": "applicant",
          "to_status": "activist",
          "note": "支部确认",
          "happened_at": "2026-03-01T00:00:00Z",
          "metadata": {},
          "created_at": "2026-03-01T00:00:00Z",
          "updated_at": "2026-03-01T00:00:00Z"
        }
      ],
      "created_at": "2026-03-01T00:00:00Z",
      "updated_at": "2026-03-01T00:00:00Z"
    }
  ]
}
```

## Admin Status Endpoints

- `GET /api/v1/admin/partyflow/statuses?org_type=party&student_id=2023001&limit=20&offset=0`
- `GET /api/v1/admin/partyflow/statuses/:id`
- `POST /api/v1/admin/partyflow/statuses`
- `PATCH /api/v1/admin/partyflow/statuses/:id`
- `POST /api/v1/admin/partyflow/statuses/import`
- `POST /api/v1/admin/partyflow/statuses/:id/events`

权限：

- 通过 `authz` scope 校验。
- 团干部、教师、超管能看到的范围不同，但接口形态一致。

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
      "org_type": "party",
      "status": "activist",
      "status_started_at": "2026-03-01T00:00:00Z",
      "joined_at": null,
      "next_action_hint": "每满3个月提交一次思想汇报",
      "metadata": {
        "contact_person": "李老师"
      },
      "created_at": "2026-03-01T00:00:00Z",
      "updated_at": "2026-03-10T00:00:00Z",
      "student_id": "2023001",
      "student_name": "张三"
    }
  ],
  "total": 1
}
```

说明：

- 列表项是 `PartyflowAdminListItem`，也就是 `PartyflowStatus` 外加 `student_id` 和 `student_name`。
- 列表不包含 `history`，详情才会返回历史。

### GET /api/v1/admin/partyflow/statuses/:id

成功响应：

```json
{
  "data": {
    "id": 1,
    "user_id": 100,
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
        "partyflow_status_id": 1,
        "event_type": "status_change",
        "event_code": "status_patched",
        "from_status": "applicant",
        "to_status": "activist",
        "note": "支部确认",
        "happened_at": "2026-03-01T00:00:00Z",
        "metadata": {},
        "created_at": "2026-03-01T00:00:00Z",
        "updated_at": "2026-03-01T00:00:00Z"
      }
    ],
    "created_at": "2026-03-01T00:00:00Z",
    "updated_at": "2026-03-10T00:00:00Z",
    "student_id": "2023001",
    "student_name": "张三"
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

说明：

- `user_id` 必填。
- `org_type` 只支持 `party` 和 `league`。
- `status` 必须符合该 `org_type` 的状态集合。
- `status_started_at` 为空时默认当前时间。
- `note` 为空时后端会自动生成初始化说明。
- 成功后返回 `GET /api/v1/admin/partyflow/statuses/:id` 同样的完整详情结构。

### PATCH /api/v1/admin/partyflow/statuses/:id

请求体：

```json
{
  "status": "development_target",
  "status_started_at": "2026-06-01T00:00:00Z",
  "joined_at": null,
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

说明：

- 所有字段都是可选的。
- 如果只改了部分字段，其他字段保持不变。
- 当 `status` 真正发生变化时，会自动追加一条 `status_change` 事件。
- 如果没有任何可更新字段，接口会直接返回当前详情。
- 成功后返回 `GET /api/v1/admin/partyflow/statuses/:id` 同样的完整详情结构。

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

说明：

- 这里按 `student_id` 找人，不是按 `user_id`。
- 失败项会写入 `failed_items`，不会中断整批导入。
- 已存在的 `(user_id, org_type)` 会走更新逻辑，不存在则创建。

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

说明：

- 这里创建的是人工事件，不是状态自动历史。
- `event_type` 只允许 `milestone` 或 `manual_adjust`。
- `event_code` 可填业务事件码，`note` 为空时后端会自动生成。
- 成功后返回更新后的完整详情结构。

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

成功响应：

```json
{
  "data": {
    "id": 1,
    "user_id": 100,
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
        "id": 12,
        "partyflow_status_id": 1,
        "event_type": "milestone",
        "event_code": "party_publicity_done",
        "from_status": "activist",
        "to_status": "activist",
        "note": "公示完成",
        "happened_at": "2026-04-01T00:00:00Z",
        "metadata": {
          "file_ids": [12]
        },
        "created_at": "2026-04-01T00:00:00Z",
        "updated_at": "2026-04-01T00:00:00Z"
      },
      {
        "id": 11,
        "partyflow_status_id": 1,
        "event_type": "status_change",
        "event_code": "status_patched",
        "from_status": "applicant",
        "to_status": "activist",
        "note": "支部确认",
        "happened_at": "2026-03-01T00:00:00Z",
        "metadata": {},
        "created_at": "2026-03-01T00:00:00Z",
        "updated_at": "2026-03-01T00:00:00Z"
      }
    ],
    "created_at": "2026-03-01T00:00:00Z",
    "updated_at": "2026-04-01T00:00:00Z",
    "student_id": "2023001",
    "student_name": "张三"
  }
}
```

## Admin Reminder Rule Endpoints

- `GET /api/v1/admin/partyflow/reminder-rules?org_type=party&enabled=true`
- `PATCH /api/v1/admin/partyflow/reminder-rules/:id`
- `POST /api/v1/admin/partyflow/reminders/scan`

权限：

- 教师/超管可维护规则和手动扫描。

### GET /api/v1/admin/partyflow/reminder-rules

说明：

- 返回 `model.PartyflowReminderRule` 数组。
- 结果使用 `response.OK` 包装，也就是 `{"data":[...]}`。

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
      "trigger_event_code": "",
      "offset_days": 90,
      "repeat_interval_days": 90,
      "title": "思想汇报提醒",
      "message_template": "积极分子每满 90 天提交思想汇报",
      "audience": "student",
      "enabled": true,
      "metadata": {},
      "created_at": "2026-04-01T08:00:00Z",
      "updated_at": "2026-04-01T08:00:00Z"
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
  "message_template": "积极分子每满 90 天提交思想汇报"
}
```

说明：

- 所有字段都是可选的。
- 返回完整规则对象，不只是 `id/enabled`。

成功响应：

```json
{
  "data": {
    "id": 1,
    "rule_code": "party_activist_report_every_90d",
    "org_type": "party",
    "status": "activist",
    "trigger_type": "status_started",
    "trigger_event_code": "",
    "offset_days": 90,
    "repeat_interval_days": 90,
    "title": "思想汇报提醒",
    "message_template": "积极分子每满 90 天提交思想汇报",
    "audience": "student",
    "enabled": true,
    "metadata": {},
    "created_at": "2026-04-01T08:00:00Z",
    "updated_at": "2026-04-27T08:00:00Z"
  }
}
```

### POST /api/v1/admin/partyflow/reminders/scan

请求体可为空；如果要指定扫描时刻，可以传：

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
    "sent_count": 3,
    "skipped_count": 7,
    "failed_count": 0
  }
}
```

说明：

- `generated_count` 和 `sent_count` 在当前实现中会一起递增。
- `skipped_count` 表示未到触发时间或已存在相同提醒事件。

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

### `history event_type`

系统历史里可能出现以下类型：

- `create`
- `status_change`
- `milestone`
- `import`
- `manual_adjust`
- `reminder_sent`

### `request event_type`

`POST /api/v1/admin/partyflow/statuses/:id/events` 只接受：

- `milestone`
- `manual_adjust`

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
- `400 {"error":"invalid event_type"}`
- `400 {"error":"invalid id"}`
- `400 {"error":"invalid body"}`
- `401 {"error":"unauthorized"}`
- `403 {"error":"forbidden"}`
- `404 {"error":"partyflow status not found"}`
- `404 {"error":"partyflow reminder rule not found"}`
- `404 {"error":"user not found"}`
- `500 {"error":"query partyflow statuses failed"}`
- `500 {"error":"create partyflow status failed"}`
- `500 {"error":"patch partyflow status failed"}`
- `500 {"error":"import partyflow statuses failed"}`
- `500 {"error":"create partyflow event failed"}`
- `500 {"error":"query partyflow reminder rules failed"}`
- `500 {"error":"patch partyflow reminder rule failed"}`
- `500 {"error":"scan partyflow reminders failed"}`

## Audit Log

写操作记录到 `admin_logs`：

- `partyflow.status_create`
- `partyflow.status_patch`
- `partyflow.status_import`
- `partyflow.event_create`
- `partyflow.rule_patch`
- `partyflow.reminder_scan`
