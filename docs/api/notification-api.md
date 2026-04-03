# 通知模块 API 文档

## 概述

通知模块提供微信小程序订阅消息的模板管理、发送记录查询与用户未读数量查询功能。

## 基础信息

- API 前缀：`/api/v1`
- 认证方式：JWT Token（`Authorization: Bearer <token>`）
- 成功响应：`{"data": ...}`
- 失败响应：`{"error": "..."}`

## 接口列表

### 1. 创建通知模板

```
POST /api/v1/admin/notification/templates
```

**权限**：Cadre / Teacher / SuperAdmin

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `code` | string | 是 | 模板唯一标识，如 `deadline_remind` |
| `wechat_template_id` | string | 是 | 微信后台配置的模板 ID |
| `name` | string | 是 | 模板名称（中文） |
| `fields` | string | 否 | JSON 格式的字段定义 |

**响应**：

- `200 OK`：创建成功，返回模板对象
- `400 Bad Request`：参数缺失
- `403 Forbidden`：无权限
- `500 Internal Server Error`：服务器错误

**示例**：

```bash
curl -X POST http://localhost:8080/api/v1/admin/notification/templates \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"code":"deadline_remind","wechat_template_id":"tmpl_123","name":"截止提醒","fields":"{\"thing1\":\"事项\"}"}'
```

---

### 2. 获取通知模板

```
GET /api/v1/admin/notification/templates/:code
```

**权限**：Cadre / Teacher / SuperAdmin

**路径参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `code` | string | 模板唯一标识 |

**响应**：

- `200 OK`：返回模板对象
- `403 Forbidden`：无权限
- `404 Not Found`：模板不存在

**示例**：

```bash
curl http://localhost:8080/api/v1/admin/notification/templates/deadline_remind \
  -H "Authorization: Bearer <token>"
```

---

### 3. 查询发送记录

```
GET /api/v1/admin/notification/logs
```

**权限**：Cadre / Teacher / SuperAdmin

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `user_id` | uint | 否 | 按用户 ID 过滤 |
| `template_code` | string | 否 | 按模板标识过滤 |
| `status` | string | 否 | 按状态过滤（`pending`/`sent`/`failed`） |
| `limit` | int | 否 | 每页条数（默认 20） |
| `offset` | int | 否 | 偏移量（默认 0） |

**响应**：

- `200 OK`：返回记录列表与总数
- `403 Forbidden`：无权限
- `500 Internal Server Error`：服务器错误

**响应体**：

```json
{
  "data": [
    {
      "id": 1,
      "user_id": 100,
      "template_code": "deadline_remind",
      "template_data": "{\"thing1\":{\"value\":\"测试通知\"}}",
      "status": "sent",
      "error_msg": "",
      "sent_at": "2026-04-02T10:00:00Z",
      "created_at": "2026-04-02T10:00:00Z"
    }
  ],
  "total": 1
}
```

**示例**：

```bash
curl "http://localhost:8080/api/v1/admin/notification/logs?status=sent&limit=10" \
  -H "Authorization: Bearer <token>"
```

---

### 4. 上报订阅结果

```
POST /api/v1/user/subscribe/report
```

**权限**：已登录用户（JWT 认证）

**说明**：前端调用 `wx.requestSubscribeMessage` 后，将用户订阅结果上报给后端。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `template_code` | string | 是 | 模板业务标识，如 `deadline_remind` |
| `wechat_template_id` | string | 是 | 微信后台配置的模板 ID |
| `status` | string | 是 | `accept`（同意）、`reject`（拒绝）、`ban`（后台封禁）、`filter`（同名模板被过滤） |

**响应**：

- `200 OK`：`{"data": {"ok": true}}`
- `400 Bad Request`：参数错误或 status 值非法（仅支持 `accept/reject/ban/filter`）
- `401 Unauthorized`：未登录

**示例**：

```bash
curl -X POST http://localhost:8080/api/v1/user/subscribe/report \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"template_code":"deadline_remind","wechat_template_id":"tmpl_123","status":"accept"}'
```

---

### 5. 微信事件推送回调

```
POST /api/v1/wechat/callback
```

**权限**：无需认证（微信服务器回调）

**说明**：接收微信服务器推送的订阅事件，包括用户弹窗订阅结果、订阅状态变更等。

**请求体**：XML 格式（由微信服务器推送）

**响应**：

- `200 OK`：返回 `success` 字符串确认接收
- `400 Bad Request`：请求体读取失败

**事件类型**：

| Event | 说明 |
|-------|------|
| `subscribe_msg_popup_event` | 用户弹窗选择订阅/拒绝 |
| `subscribe_msg_change_event` | 用户在设置中修改订阅状态 |
| `subscribe_msg_sent_event` | 订阅消息下发结果事件（异步推送） |

---

### 6. 获取未读消息数量

```
GET /api/v1/notifications/unread/count
```

**权限**：已登录用户（JWT 认证）

**说明**：当前实现将 `notification_logs.status = pending` 视为未读消息。

**响应**：

- `200 OK`：`{"data":{"count":2}}`
- `401 Unauthorized`：未登录
- `403 Forbidden`：无权限
- `500 Internal Server Error`：服务器错误

**示例**：

```bash
curl http://localhost:8080/api/v1/notifications/unread/count \
  -H "Authorization: Bearer <token>"
```

## 权限规则

| 角色 | 创建模板 | 获取模板 | 查询记录 |
|------|----------|----------|----------|
| Student (1) | ❌ | ❌ | ❌ |
| Cadre (2) | ✅ | ✅ | ✅ |
| Teacher (3) | ✅ | ✅ | ✅ |
| SuperAdmin (4) | ✅ | ✅ | ✅ |

## 测试命令

```bash
go test ./internal/http/handler/... -count=1 -run TestNotification
go test ./internal/service/notification/... -count=1
```
