# 党团流程模块设计（v0）

**日期**：2026-04-08  
**阶段**：Phase 2-A

## 1. 目标

实现学生党团流程的最小闭环：
- 学生查看本人当前阶段、历史节点、下一步提示
- 管理员按 scope 查询与更新学生进度
- 每次阶段变化留痕
- 对明确规则的关键节点生成提醒任务，并复用现有通知能力发送

## 2. 范围

### In Scope

- 统一承载“入团 + 入党”两条主线
- 当前阶段查询
- 历史事件记录
- 管理员批量导入当前阶段
- 固定规则提醒生成与发送
- 写操作记录 `admin_logs`

### Out of Scope

- 理论自测
- 任意自定义状态机配置
- 复杂提醒编排与多渠道通知
- 流程附件审核

## 3. 数据模型

### 3.1 `party_progresses`

表示每个学生在某条党团主线上的当前状态。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| user_id | bigint FK | 学生用户 |
| flow_type | varchar(20) | `league` / `party` |
| current_stage | varchar(40) | 当前阶段编码 |
| stage_started_at | timestamp | 当前阶段开始时间 |
| next_action_hint | varchar(200) | 下一步提示 |
| reminder_rule_code | varchar(40) | 当前阶段适用提醒规则 |
| metadata | jsonb | 补充信息 |
| created_by | bigint FK | 创建人 |
| updated_by | bigint FK | 更新人 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

约束：
- `user_id + flow_type` 唯一

### 3.2 `party_progress_events`

表示阶段变化与导入留痕。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| progress_id | bigint FK | 关联当前进度 |
| user_id | bigint FK | 学生用户 |
| flow_type | varchar(20) | `league` / `party` |
| from_stage | varchar(40) | 变更前阶段 |
| to_stage | varchar(40) | 变更后阶段 |
| event_type | varchar(20) | `create` / `update` / `import` / `reminder_sent` |
| note | varchar(500) | 备注 |
| operator_id | bigint FK | 操作人 |
| happened_at | timestamp | 事件时间 |
| created_at | timestamp | 创建时间 |

## 4. 流程建模

### 4.1 `league` 主线

- `applicant`
- `activist`
- `member`

### 4.2 `party` 主线

- `applicant`
- `activist`
- `development_target`
- `probationary_member`
- `full_member`

说明：
- v0 使用固定枚举，不做数据库配置化
- 前端展示文案可以映射中文，后端统一使用稳定英文编码

## 5. 提醒规则

v0 只支持固定规则：
- `league_activist_90d`
- `party_activist_90d`
- `party_development_target_180d`
- `party_probationary_member_365d`

处理方式：
1. 定时任务按规则扫描 `party_progresses`
2. 对满足条件且未发送过同规则提醒的记录生成一次提醒
3. 调用现有 `notification.Service` 发送订阅消息
4. 同时写入 `party_progress_events(event_type=reminder_sent)`

## 6. API 设计

### 6.1 学生端

- `GET /api/v1/partyflow/me`

返回本人所有流程：

```json
{
  "data": [
    {
      "id": 1,
      "flow_type": "party",
      "current_stage": "activist",
      "stage_started_at": "2026-03-01T00:00:00Z",
      "next_action_hint": "满3个月后提交思想汇报",
      "history": [
        {
          "id": 11,
          "from_stage": "applicant",
          "to_stage": "activist",
          "event_type": "update",
          "note": "支部确认",
          "happened_at": "2026-03-01T00:00:00Z"
        }
      ]
    }
  ]
}
```

### 6.2 管理端

- `GET /api/v1/admin/partyflow/progress?flow_type=party&student_id=2023001&limit=20&offset=0`
- `GET /api/v1/admin/partyflow/progress/:id`
- `POST /api/v1/admin/partyflow/progress`
- `PATCH /api/v1/admin/partyflow/progress/:id`
- `POST /api/v1/admin/partyflow/progress/import`

`POST /import` 请求体：

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

说明：
- v0 先采用 JSON 批量导入接口
- Excel 导入由后续管理端或脚本转换为该结构，不在本轮后端直接解析

## 7. 权限与 Scope

- 学生：仅可查看本人 `GET /partyflow/me`
- 团干部/教师/超管：可按 scope 查询、创建、更新、导入
- 所有管理员写操作记录 `admin_logs`

新增 authz 动作建议：

```go
ActionPartyflowMeGet     = "partyflow:me:get"
ActionPartyflowList      = "partyflow:list"
ActionPartyflowGet       = "partyflow:get"
ActionPartyflowCreate    = "partyflow:create"
ActionPartyflowPatch     = "partyflow:patch"
ActionPartyflowImport    = "partyflow:import"
ActionPartyflowRemindRun = "partyflow:remind:run"
```

## 8. 代码结构

```text
internal/
├── model/
│   ├── party_progress.go
│   └── party_progress_event.go
├── repo/
│   ├── party_progress_repo.go
│   ├── party_progress_repo_test.go
│   ├── party_progress_event_repo.go
│   └── party_progress_event_repo_test.go
├── service/
│   └── partyflow/
│       ├── service.go
│       ├── reminder.go
│       └── service_test.go
└── http/
    └── handler/
        ├── partyflow_handler.go
        └── partyflow_handler_test.go
```

## 9. 测试策略

- handler 测试：
  - 学生查看本人流程成功
  - 非法 id / 非法阶段参数
  - 403 越权访问
  - 管理员导入成功
- repo / service 测试：
  - scope 下分页查询
  - 更新阶段自动写事件
  - 重复导入同一 `user_id + flow_type` 走更新逻辑
  - 提醒扫描不会重复发送同规则提醒

## 10. 验收标准

- 学生端能稳定查询本人党团流程与历史
- 管理端能在 scope 内进行查询、创建、更新、导入
- 阶段变化与提醒发送均有审计留痕
- 至少 1 条固定规则提醒能跑通完整链路

