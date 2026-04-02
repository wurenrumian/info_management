# 共享通知模块补全 Implementation Plan（方案 A）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补全通知模块三个关键缺失：Access Token 缓存、用户订阅状态管理、微信事件推送接收

**Architecture:** 在现有 `internal/service/notification/` 下新增 token_cache.go，新增 user_subscribe 模型和订阅上报/事件回调 handler。

**Tech Stack:** Go, GORM, Gin, sync (并发安全)

**Design Doc:** `docs/superpowers/specs/2026-04-02-notification-module-supplement-design.md`

---

### Task 1: 创建 Access Token 缓存

**Files:**
- Create: `internal/service/notification/token_cache.go`

- [ ] **Step 1: 创建 TokenCache**

```go
// internal/service/notification/token_cache.go
package notification

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type TokenCache struct {
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
	appID     string
	appSecret string
	client    *http.Client
	baseURL   string
}

func NewTokenCache(appID, appSecret string, client *http.Client) *TokenCache {
	if client == nil {
		client = &http.Client{}
	}
	return &TokenCache{
		appID:     appID,
		appSecret: appSecret,
		client:    client,
		baseURL:   "https://api.weixin.qq.com",
	}
}

func (c *TokenCache) GetToken() (string, error) {
	c.mu.RLock()
	if time.Now().Before(c.expiresAt) {
		token := c.token
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	return c.RefreshToken()
}

func (c *TokenCache) RefreshToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Now().Before(c.expiresAt) {
		return c.token, nil
	}

	url := fmt.Sprintf("%s/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		c.baseURL, c.appID, c.appSecret)

	resp, err := c.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("get token error %d: %s", result.ErrCode, result.ErrMsg)
	}

	c.token = result.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	return c.token, nil
}
```

- [ ] **Step 2: 构建验证**

```bash
go build ./...
```

Expected: 无编译错误

- [ ] **Step 3: Commit**

```bash
git add internal/service/notification/token_cache.go
git commit -m "feat: add access token cache for wechat subscribe messages"
```

---

### Task 2: 创建用户订阅记录模型

**Files:**
- Create: `internal/model/user_subscribe.go`
- Modify: `internal/store/db.go` — 添加 UserSubscribe 到 AutoMigrate

- [ ] **Step 1: 创建 UserSubscribe 模型**

```go
// internal/model/user_subscribe.go
package model

import "time"

type UserSubscribe struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	TemplateCode     string    `gorm:"size:64;index;not null" json:"template_code"`
	WechatTemplateID string    `gorm:"size:100;not null" json:"wechat_template_id"`
	Status           string    `gorm:"size:20;not null;default:subscribed" json:"status"`
	SubscribedAt     time.Time `json:"subscribed_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (UserSubscribe) TableName() string {
	return "user_subscribes"
}
```

- [ ] **Step 2: 更新 AutoMigrate**

修改 `internal/store/db.go` 第 15 行，添加 `&model.UserSubscribe{}`：

```go
if err := db.AutoMigrate(
	&model.User{}, &model.Class{}, &model.AdminLog{},
	&model.KnowledgeItem{}, &model.Document{},
	&model.NotificationTemplate{}, &model.NotificationLog{},
	&model.UserSubscribe{},
); err != nil {
```

- [ ] **Step 3: 构建验证**

```bash
go build ./...
```

Expected: 无编译错误

- [ ] **Step 4: Commit**

```bash
git add internal/model/user_subscribe.go internal/store/db.go
git commit -m "feat: add user subscribe model for tracking subscription status"
```

---

### Task 3: 创建订阅上报和事件回调 Handler

**Files:**
- Create: `internal/http/handler/wechat_subscribe_handler.go`

- [ ] **Step 1: 创建 Handler**

```go
// internal/http/handler/wechat_subscribe_handler.go
package handler

import (
	"encoding/xml"
	"io"
	"net/http"
	"time"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/http/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SubscribeHandler struct {
	db *gorm.DB
}

func NewSubscribeHandler(db *gorm.DB) *SubscribeHandler {
	return &SubscribeHandler{db: db}
}

type subscribeReportReq struct {
	TemplateCode     string `json:"template_code" binding:"required"`
	WechatTemplateID string `json:"wechat_template_id" binding:"required"`
	Status           string `json:"status" binding:"required"`
}

func (h *SubscribeHandler) ReportSubscribe(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req subscribeReportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	if req.Status != "accept" && req.Status != "reject" {
		response.Error(c, http.StatusBadRequest, "status must be accept or reject")
		return
	}

	status := "subscribed"
	if req.Status == "reject" {
		status = "unsubscribed"
	}

	var existing model.UserSubscribe
	err := h.db.Where("user_id = ? AND template_code = ?", actor.UserID, req.TemplateCode).First(&existing).Error

	now := time.Now()
	sub := model.UserSubscribe{
		UserID:           actor.UserID,
		TemplateCode:     req.TemplateCode,
		WechatTemplateID: req.WechatTemplateID,
		Status:           status,
		UpdatedAt:        now,
	}

	if err == gorm.ErrRecordNotFound {
		sub.SubscribedAt = now
		if err := h.db.Create(&sub).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "failed to record subscription")
			return
		}
	} else {
		if err := h.db.Model(&model.UserSubscribe{}).Where("id = ?", existing.ID).Updates(sub).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "failed to update subscription")
			return
		}
	}

	response.OK(c, gin.H{"ok": true})
}

type wechatEventXML struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   string   `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Event        string   `xml:"Event"`
	List         []struct {
		TemplateID            string `xml:"TemplateId"`
		SubscribeStatusString string `xml:"SubscribeStatusString"`
		PopupScene            string `xml:"PopupScene"`
	} `xml:"SubscribeMsgPopupEvent>List"`
	ChangeEventList []struct {
		TemplateID            string `xml:"TemplateId"`
		SubscribeStatusString string `xml:"SubscribeStatusString"`
	} `xml:"SubscribeMsgChangeEvent>List"`
}

func (h *SubscribeHandler) WechatCallback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	defer c.Request.Body.Close()

	var event wechatEventXML
	if err := xml.Unmarshal(body, &event); err != nil {
		c.String(http.StatusOK, "success")
		return
	}

	switch event.Event {
	case "subscribe_msg_popup_event":
		for _, item := range event.List {
			h.updateSubscribeByOpenID(event.FromUserName, item.TemplateID, item.SubscribeStatusString)
		}
	case "subscribe_msg_change_event":
		for _, item := range event.ChangeEventList {
			h.updateSubscribeByOpenID(event.FromUserName, item.TemplateID, item.SubscribeStatusString)
		}
	}

	c.String(http.StatusOK, "success")
}

func (h *SubscribeHandler) updateSubscribeByOpenID(openID, templateID, statusStr string) {
	var user model.User
	if err := h.db.Where("openid = ?", openID).First(&user).Error; err != nil {
		return
	}

	status := "subscribed"
	if statusStr == "reject" {
		status = "unsubscribed"
	}

	h.db.Model(&model.UserSubscribe{}).
		Where("user_id = ? AND wechat_template_id = ?", user.ID, templateID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})
}
```

- [ ] **Step 2: 构建验证**

```bash
go build ./...
```

Expected: 无编译错误

- [ ] **Step 3: Commit**

```bash
git add internal/http/handler/wechat_subscribe_handler.go
git commit -m "feat: add subscribe report and wechat event callback handlers"
```

---

### Task 4: 更新 WechatClient 使用 TokenCache + 注册路由

**Files:**
- Modify: `internal/service/notification/wechat_client.go` — 改用 TokenCache
- Modify: `internal/http/router/router.go` — 注册新路由

- [ ] **Step 1: 修改 WechatClient 接收 TokenCache**

修改 `wechat_client.go`，将 `getAccessToken()` 替换为使用 TokenCache：

```go
// 修改 WechatClient 结构体
type WechatClient struct {
	tokenCache *TokenCache
	httpClient *http.Client
	baseURL    string
}

func NewWechatClient(httpClient *http.Client, tokenCache *TokenCache) *WechatClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &WechatClient{
		tokenCache: tokenCache,
		httpClient: httpClient,
		baseURL:    "https://api.weixin.qq.com",
	}
}
```

修改 `SendSubscribeMessage` 中的 `getAccessToken()` 调用：

```go
// 将:
accessToken, err := c.getAccessToken()
// 改为:
accessToken, err := c.tokenCache.GetToken()
```

删除旧的 `getAccessToken()` 方法和 `tokenResp` 结构体。

- [ ] **Step 2: 更新 router.go**

1. 修改 `initNotificationSvc`：

```go
func initNotificationSvc(db *gorm.DB) *notification.Service {
	appID := os.Getenv("WECHAT_APP_ID")
	appSecret := os.Getenv("WECHAT_APP_SECRET")
	tokenCache := notification.NewTokenCache(appID, appSecret, nil)
	wechatClient := notification.NewWechatClient(nil, tokenCache)
	repo := notification.NewRepo(db)
	userRepo := notification.NewGormUserRepo(db)
	return notification.NewService(wechatClient, repo, userRepo)
}
```

2. 在 `api.Use(middleware.JWTAuth(jwtSecret))` 之后、admin group 之前添加：

```go
subscribeHandler := handler.NewSubscribeHandler(db)
api.POST("/user/subscribe/report", subscribeHandler.ReportSubscribe)
```

3. 在 `return r` 之前（不受 JWT 保护，微信服务器回调）：

```go
api.POST("/wechat/callback", subscribeHandler.WechatCallback)
```

注意：`/wechat/callback` 不应受 JWT 中间件保护。需将其放在 JWT 中间件作用域之外，或单独创建无认证路由。

- [ ] **Step 3: 构建验证**

```bash
go build ./...
```

Expected: 无编译错误

- [ ] **Step 4: Commit**

```bash
git add internal/service/notification/wechat_client.go internal/http/router/router.go
git commit -m "refactor: use token cache in wechat client and register new routes"
```

---

### Task 5: 运行全部测试

- [ ] **Step 1: 运行全部测试**

```bash
go test ./... -count=1
```

Expected: 全部通过（现有测试不应因重构而失败）

- [ ] **Step 2: 最终提交**

```bash
git add .
git commit -m "chore: verify notification supplement changes pass all tests"
```
