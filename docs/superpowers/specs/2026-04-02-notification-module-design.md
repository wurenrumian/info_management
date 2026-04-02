# 共享通知模块设计文档

**日期**: 2026-04-02  
**目标**: 为 info_management 项目创建共享的通知模块，支持微信小程序订阅消息发送与发送记录追踪

## 1. 概述

通知模块作为共享服务，供项目中各业务模块（党团事务、审批流程、信息推送等）调用，统一处理微信小程序订阅消息的发送逻辑与发送记录管理。

### 1.1 范围

- 通知模板管理（CRUD）
- 统一发送接口（单条/批量）
- 微信订阅消息 API 对接
- 发送记录存储与查询
- 错误分类与处理

### 1.2 不做的部分

- 不做站内消息系统
- 不做邮件/短信通知
- 不做自动重试（微信订阅消息需用户主动授权，重试无意义）
- 不做定时任务调度（由调用方负责触发时机）

## 2. 架构概览

### 2.1 目录结构

```
internal/service/notification/
├── model.go          # 数据模型定义
├── wechat_client.go  # 微信订阅消息 API 客户端
├── service.go        # 核心发送逻辑（统一入口）
└── repo.go           # 数据库操作（模板 CRUD + 记录存储）
```

### 2.2 调用方式

业务模块通过注入 `notification.Service` 发送通知：

```go
// 示例：党团模块触发提醒
notifSvc.Send(ctx, notification.SendRequest{
    UserID:       studentID,
    TemplateCode: "deadline_remind",
    Page:         "/pages/party-progress/index",
    TemplateData: map[string]interface{}{
        "thing1": map[string]string{"value": "思想汇报提交截止"},
        "time2":  map[string]string{"value": "2026年4月10日"},
    },
})
```

### 2.3 数据流

1. 业务模块调用 `Send()` →
2. Service 按 `TemplateCode` 查询模板 →
3. 查询用户 openid →
4. 调用微信 API 发送 →
5. 记录发送结果到 `notification_logs` 表

## 3. 数据模型

### 3.1 通知模板表 `notification_templates`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| code | string (unique) | 模板唯一标识（如 `deadline_remind`） |
| wechat_template_id | string | 微信小程序后台配置的模板 ID |
| name | string | 模板名称（中文，管理后台显示） |
| fields | datatypes.JSON | 模板字段定义（如 `{"thing1":"事项","time2":"时间"}`） |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

### 3.2 发送记录表 `notification_logs`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | uint | 接收用户 ID |
| template_code | string | 模板标识 |
| template_data | datatypes.JSON | 发送时使用的模板数据 |
| status | string | `pending` / `sent` / `failed` |
| error_msg | string | 失败时的错误信息 |
| sent_at | time | 发送成功时间 |
| created_at | time | 记录创建时间 |

## 4. 核心接口

### 4.1 Service 层

```go
type SendRequest struct {
    UserID       uint
    TemplateCode string
    Page         string                    // 小程序跳转页面路径
    TemplateData map[string]interface{}   // 模板数据
}

type SendResult struct {
    UserID uint
    Err    error
}

type LogFilter struct {
    UserID       *uint
    TemplateCode *string
    Status       *string
    StartAt      *time.Time
    EndAt        *time.Time
    Offset       int
    Limit        int
}

type Service interface {
    // Send 发送单条通知
    Send(ctx context.Context, req SendRequest) error

    // SendBatch 批量发送（同一模板，不同用户）
    SendBatch(ctx context.Context, templateCode string, users []uint, dataFn func(userID uint) map[string]interface{}) []SendResult

    // GetTemplate 按 code 获取模板
    GetTemplate(code string) (*Template, error)

    // CreateTemplate 创建模板
    CreateTemplate(t *Template) error

    // GetLogs 查询发送记录
    GetLogs(filter LogFilter) ([]NotificationLog, int64, error)
}
```

### 4.2 微信客户端

```go
type WechatClient interface {
    // SendSubscribeMessage 调用微信订阅消息 API
    SendSubscribeMessage(openid, templateID, page string, data map[string]interface{}) error
}
```

### 4.3 环境变量

| 变量 | 说明 |
|------|------|
| `WECHAT_SUBSCRIBE_MSG_ENABLED` | 是否启用订阅消息（开发环境可设为 false） |

## 5. 错误处理

### 5.1 微信 API 错误分类

| 错误码 | 含义 | 处理方式 |
|--------|------|----------|
| 43101 | 用户拒绝接收 | 记录 `failed`，标记"用户未订阅" |
| 43004 | 用户未订阅 | 记录 `failed`，标记"需要用户先订阅" |
| 47003 | 模板参数不准确 | 记录 `failed`，标记"模板数据错误" |
| 其他 | 网络/系统异常 | 记录 `failed`，保留错误信息 |

### 5.2 边界情况

1. **用户不存在 openid**: 跳过发送，记录 `failed`
2. **模板不存在**: 返回错误，不发送
3. **批量发送部分失败**: 不整体回滚，逐条记录结果
4. **开发环境**: `WECHAT_SUBSCRIBE_MSG_ENABLED=false` 时仅记录日志，不调用微信 API

## 6. 与现有代码集成

### 6.1 初始化

在 `internal/app/app.go` 中：

```go
wechatClient := notification.NewWechatClient(httpClient, cfg.WechatAppID, cfg.WechatAppSecret)
notifRepo := notification.NewRepo(db)
notifSvc := notification.NewService(wechatClient, notifRepo)
```

### 6.2 注入业务模块

```go
partySvc := party.NewService(repo, notifSvc)
approvalSvc := approval.NewService(repo, notifSvc)
```

### 6.3 遵循现有模式

- 使用 GORM 操作数据库
- 使用 `datatypes.JSON` 存储 JSON 字段
- 错误处理使用标准 `error` 返回值
- 单元测试放在同目录下 `*_test.go`
