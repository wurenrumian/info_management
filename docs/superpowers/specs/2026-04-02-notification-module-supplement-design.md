# 通知模块补全设计文档（方案 A）

**日期**: 2026-04-02
**目标**: 补全通知模块三个关键缺失：Token缓存、订阅状态管理、事件推送接收

## 1. 概述

当前通知模块核心API调用正确，但缺少生产环境必需的 Token 缓存、用户订阅状态追踪和微信事件推送接收。本设计以最小改动补全这三项。

### 1.1 范围

- Access Token 内存缓存（避免频繁请求微信接口）
- 用户订阅记录表及上报接口
- 微信事件推送回调端点

### 1.2 不做的部分

- 不做分布式 Token 缓存（当前单实例部署）
- 不做长期订阅管理（一次性订阅场景足够）
- 不做消息重试（微信订阅消息不支持重试）

## 2. 新增文件

### 2.1 `internal/service/notification/token_cache.go`

Access Token 缓存，线程安全，提前5分钟过期。

```go
type TokenCache struct {
    mu        sync.RWMutex
    token     string
    expiresAt time.Time
    appID     string
    appSecret string
    client    *http.Client
}
```

- `GetToken()` — 返回有效token，过期则刷新
- `RefreshToken()` — 请求微信获取新token

### 2.2 `internal/model/user_subscribe.go`

用户订阅记录模型，映射表 `user_subscribes`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | uint (index) | 用户ID |
| template_code | string (index) | 模板业务标识 |
| wechat_template_id | string | 微信模板ID |
| status | string | subscribed / unsubscribed |
| subscribed_at | time | 订阅时间 |
| updated_at | time | 更新时间 |

### 2.3 `internal/http/handler/wechat_subscribe_handler.go`

两个端点：

**POST `/api/v1/user/subscribe/report`** — 前端上报

请求体：
```json
{
  "template_code": "deadline_remind",
  "wechat_template_id": "xxx",
  "status": "accept"  // accept / reject
}
```

**POST `/api/v1/wechat/callback`** — 微信事件推送

接收 XML/JSON 格式的订阅事件（popup_event、change_event、sent_event），解析后更新 `user_subscribes` 表。

## 3. 改动现有文件

| 文件 | 改动内容 |
|------|---------|
| `wechat_client.go` | `getAccessToken()` 改为接收 `TokenCache` 参数，从缓存获取 |
| `router.go` | 注册 `/user/subscribe/report` 和 `/wechat/callback` 路由 |
| `store/db.go` | AutoMigrate 加入 `UserSubscribe` |
| `.env.example` | 无需改动 |

## 4. 数据流

### 4.1 订阅流程

```
前端 wx.requestSubscribeMessage() → 用户选择 → 
  前端 POST /api/v1/user/subscribe/report → 
  后端写入 user_subscribes 表
```

### 4.2 发送流程（优化后）

```
业务模块 Send() → 查模板 → 查用户openid → 
  TokenCache.GetToken() → 调用微信API → 记录日志
```

### 4.3 事件推送流程

```
微信服务器 → POST /api/v1/wechat/callback → 
  解析事件 → 更新 user_subscribes 状态
```

## 5. 错误处理

- Token 刷新失败：返回原错误，不降级
- 订阅上报：幂等处理，重复上报更新状态
- 事件推送：微信要求返回 success 字符串确认接收，否则重试
