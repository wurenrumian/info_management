# 共享通知模块 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 创建共享通知模块，支持微信小程序订阅消息的发送、模板管理与发送记录追踪

**Architecture:** 在 `internal/service/notification/` 下创建通知服务，包含数据模型、微信客户端、Service 层和 Repo 层。通过依赖注入供各业务模块调用。

**Tech Stack:** Go, GORM, Gin, testify (testing), httptest (mocking)

---

### Task 1: 创建数据模型 (notification_template & notification_log)

**Files:**
- Create: `internal/model/notification_template.go`
- Create: `internal/model/notification_log.go`
- Modify: `internal/store/db.go` — 添加新模型到 AutoMigrate

- [ ] **Step 1: 创建 notification_template 模型**

```go
// internal/model/notification_template.go
package model

import "time"

type NotificationTemplate struct {
	ID               uint      `gorm:"primaryKey"`
	Code             string    `gorm:"size:64;uniqueIndex;not null"`
	WechatTemplateID string    `gorm:"size:100;not null"`
	Name             string    `gorm:"size:100;not null"`
	Fields           string    `gorm:"type:jsonb"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (NotificationTemplate) TableName() string {
	return "notification_templates"
}
```

- [ ] **Step 2: 创建 notification_log 模型**

```go
// internal/model/notification_log.go
package model

import "time"

type NotificationLog struct {
	ID           uint       `gorm:"primaryKey"`
	UserID       uint       `gorm:"index;not null"`
	TemplateCode string     `gorm:"size:64;index;not null"`
	TemplateData string     `gorm:"type:jsonb"`
	Status       string     `gorm:"size:16;not null;default:pending"`
	ErrorMsg     string     `gorm:"size:500"`
	SentAt       *time.Time
	CreatedAt    time.Time
}

func (NotificationLog) TableName() string {
	return "notification_logs"
}
```

- [ ] **Step 3: 更新 AutoMigrate**

修改 `internal/store/db.go` 第 15 行，添加两个新模型：

```go
if err := db.AutoMigrate(
	&model.User{}, &model.Class{}, &model.AdminLog{},
	&model.KnowledgeItem{}, &model.Document{},
	&model.NotificationTemplate{}, &model.NotificationLog{},
); err != nil {
```

- [ ] **Step 4: 运行测试验证无编译错误**

```bash
go build ./...
```
Expected: 无编译错误

- [ ] **Step 5: Commit**

```bash
git add internal/model/notification_template.go internal/model/notification_log.go internal/store/db.go
git commit -m "feat: add notification template and log models"
```

---

### Task 2: 创建 Repo 层 (数据库操作)

**Files:**
- Create: `internal/service/notification/repo.go`
- Create: `internal/service/notification/repo_test.go`

- [ ] **Step 1: 创建 Repo**

```go
// internal/service/notification/repo.go
package notification

import (
	"manage/internal/model"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) GetTemplateByCode(code string) (*model.NotificationTemplate, error) {
	var t model.NotificationTemplate
	if err := r.db.Where("code = ?", code).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) CreateTemplate(t *model.NotificationTemplate) error {
	return r.db.Create(t).Error
}

func (r *Repo) CreateLog(log *model.NotificationLog) error {
	return r.db.Create(log).Error
}

type LogFilter struct {
	UserID       *uint
	TemplateCode *string
	Status       *string
	Offset       int
	Limit        int
}

func (r *Repo) ListLogs(filter LogFilter) ([]model.NotificationLog, int64, error) {
	q := r.db.Model(&model.NotificationLog{})
	if filter.UserID != nil {
		q = q.Where("user_id = ?", *filter.UserID)
	}
	if filter.TemplateCode != nil {
		q = q.Where("template_code = ?", *filter.TemplateCode)
	}
	if filter.Status != nil {
		q = q.Where("status = ?", *filter.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	var logs []model.NotificationLog
	if err := q.Order("created_at desc").Offset(filter.Offset).Limit(filter.Limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
```

- [ ] **Step 2: 创建 Repo 测试**

```go
// internal/service/notification/repo_test.go
package notification

import (
	"testing"
	"time"

	"manage/internal/model"
	"manage/internal/testutil"

	"github.com/stretchr/testify/require"
)

func TestGetTemplateByCode(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "test_remind",
		WechatTemplateID: "tmpl_123",
		Name:             "测试提醒",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	result, err := repo.GetTemplateByCode("test_remind")
	require.NoError(t, err)
	require.Equal(t, "test_remind", result.Code)
	require.Equal(t, "tmpl_123", result.WechatTemplateID)
}

func TestGetTemplateByCodeNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	_, err := repo.GetTemplateByCode("nonexistent")
	require.Error(t, err)
}

func TestCreateLog(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	log := &model.NotificationLog{
		UserID:       1,
		TemplateCode: "test_remind",
		TemplateData: `{"thing1":"测试"}`,
		Status:       "pending",
	}
	require.NoError(t, repo.CreateLog(log))
	require.NotZero(t, log.ID)
}

func TestListLogs(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	for i := 0; i < 5; i++ {
		repo.CreateLog(&model.NotificationLog{
			UserID:       1,
			TemplateCode: "test_remind",
			Status:       "sent",
			CreatedAt:    time.Now(),
		})
	}

	logs, total, err := repo.ListLogs(LogFilter{
		UserID: ptrUint(1),
		Limit:  10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(5), total)
	require.Len(t, logs, 5)
}

func TestListLogsFilterByStatus(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	repo.CreateLog(&model.NotificationLog{
		UserID:       1,
		TemplateCode: "test_remind",
		Status:       "sent",
		CreatedAt:    time.Now(),
	})
	repo.CreateLog(&model.NotificationLog{
		UserID:       1,
		TemplateCode: "test_remind",
		Status:       "failed",
		CreatedAt:    time.Now(),
	})

	status := "failed"
	logs, total, err := repo.ListLogs(LogFilter{
		UserID: ptrUint(1),
		Status: &status,
		Limit:  10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	require.Equal(t, "failed", logs[0].Status)
}

func ptrUint(v uint) *uint {
	return &v
}
```

- [ ] **Step 3: 检查 testutil 是否存在 NewTestDB**

```bash
ls internal/testutil/
```

如果不存在 `NewTestDB` 辅助函数，需要先创建它（参考项目现有测试模式）。

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/service/notification/... -v -count=1
```
Expected: 全部通过

- [ ] **Step 5: Commit**

```bash
git add internal/service/notification/repo.go internal/service/notification/repo_test.go
git commit -m "feat: add notification repo layer with tests"
```

---

### Task 3: 创建微信订阅消息客户端

**Files:**
- Create: `internal/service/notification/wechat_client.go`
- Create: `internal/service/notification/wechat_client_test.go`

- [ ] **Step 1: 创建 WechatClient**

```go
// internal/service/notification/wechat_client.go
package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type WechatClient struct {
	appID      string
	appSecret  string
	httpClient *http.Client
	baseURL    string
}

func NewWechatClient(httpClient *http.Client, appID, appSecret string) *WechatClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &WechatClient{
		appID:      appID,
		appSecret:  appSecret,
		httpClient: httpClient,
		baseURL:    "https://api.weixin.qq.com",
	}
}

type subscribeMsgRequest struct {
	ToUser           string                 `json:"touser"`
	TemplateID       string                 `json:"template_id"`
	Page             string                 `json:"page"`
	Data             map[string]interface{} `json:"data"`
	MiniprogramState string                 `json:"miniprogram_state"`
}

type subscribeMsgResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (c *WechatClient) SendSubscribeMessage(openid, templateID, page string, data map[string]interface{}) error {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	url := fmt.Sprintf("%s/cgi-bin/message/subscribe/send?access_token=%s", c.baseURL, accessToken)

	reqBody := subscribeMsgRequest{
		ToUser:           openid,
		TemplateID:       templateID,
		Page:             page,
		Data:             data,
		MiniprogramState: "formal",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var result subscribeMsgResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("wechat subscribe msg error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

type tokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

func (c *WechatClient) getAccessToken() (string, error) {
	url := fmt.Sprintf("%s/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		c.baseURL, c.appID, c.appSecret)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result tokenResp
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("get token error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return result.AccessToken, nil
}
```

- [ ] **Step 2: 创建 WechatClient 测试**

```go
// internal/service/notification/wechat_client_test.go
package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSendSubscribeMessageSuccess(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/cgi-bin/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fake_token",
			"expires_in":   7200,
		})
	})

	mux.HandleFunc("/cgi-bin/message/subscribe/send", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 0,
			"errmsg":  "ok",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewWechatClient(nil, "fake_appid", "fake_secret")
	client.baseURL = srv.URL

	err := client.SendSubscribeMessage("openid123", "tmpl_123", "/pages/index", map[string]interface{}{
		"thing1": map[string]string{"value": "测试通知"},
	})
	require.NoError(t, err)
}

func TestSendSubscribeMessageWechatError(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/cgi-bin/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fake_token",
			"expires_in":   7200,
		})
	})

	mux.HandleFunc("/cgi-bin/message/subscribe/send", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 43004,
			"errmsg":  "require subscribe",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewWechatClient(nil, "fake_appid", "fake_secret")
	client.baseURL = srv.URL

	err := client.SendSubscribeMessage("openid123", "tmpl_123", "/pages/index", map[string]interface{}{
		"thing1": map[string]string{"value": "测试通知"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "43004")
}

func TestSendSubscribeMessageTokenError(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/cgi-bin/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 40013,
			"errmsg":  "invalid appid",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewWechatClient(nil, "fake_appid", "fake_secret")
	client.baseURL = srv.URL

	err := client.SendSubscribeMessage("openid123", "tmpl_123", "/pages/index", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "get access token")
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/service/notification/... -v -count=1
```
Expected: 全部通过（包含 Task 2 的测试）

- [ ] **Step 4: Commit**

```bash
git add internal/service/notification/wechat_client.go internal/service/notification/wechat_client_test.go
git commit -m "feat: add wechat subscribe message client with tests"
```

---

### Task 4: 创建 Service 层 (核心发送逻辑)

**Files:**
- Create: `internal/service/notification/service.go`
- Create: `internal/service/notification/service_test.go`

- [ ] **Step 1: 创建 Service**

```go
// internal/service/notification/service.go
package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"manage/internal/model"
)

type WechatClientInterface interface {
	SendSubscribeMessage(openid, templateID, page string, data map[string]interface{}) error
}

type RepoInterface interface {
	GetTemplateByCode(code string) (*model.NotificationTemplate, error)
	CreateTemplate(t *model.NotificationTemplate) error
	CreateLog(log *model.NotificationLog) error
	ListLogs(filter LogFilter) ([]model.NotificationLog, int64, error)
}

type UserRepo interface {
	GetUserOpenID(userID uint) (string, error)
}

type Service struct {
	wechatClient WechatClientInterface
	repo         RepoInterface
	userRepo     UserRepo
}

func NewService(wechatClient WechatClientInterface, repo RepoInterface, userRepo UserRepo) *Service {
	return &Service{
		wechatClient: wechatClient,
		repo:         repo,
		userRepo:     userRepo,
	}
}

type SendRequest struct {
	UserID       uint
	TemplateCode string
	Page         string
	TemplateData map[string]interface{}
}

type SendResult struct {
	UserID uint
	Err    error
}

func (s *Service) Send(ctx context.Context, req SendRequest) error {
	if !s.isEnabled() {
		return nil
	}

	tmpl, err := s.repo.GetTemplateByCode(req.TemplateCode)
	if err != nil {
		return fmt.Errorf("get template %s: %w", req.TemplateCode, err)
	}

	openID, err := s.userRepo.GetUserOpenID(req.UserID)
	if err != nil {
		s.recordLog(req, "failed", fmt.Sprintf("get openid: %v", err))
		return fmt.Errorf("get user openid: %w", err)
	}

	if openID == "" {
		s.recordLog(req, "failed", "user has no openid")
		return fmt.Errorf("user %d has no openid", req.UserID)
	}

	err = s.wechatClient.SendSubscribeMessage(openID, tmpl.WechatTemplateID, req.Page, req.TemplateData)

	if err != nil {
		s.recordLog(req, "failed", err.Error())
		return fmt.Errorf("send subscribe message: %w", err)
	}

	s.recordLog(req, "sent", "")
	return nil
}

func (s *Service) SendBatch(ctx context.Context, templateCode string, users []uint, dataFn func(userID uint) map[string]interface{}) []SendResult {
	results := make([]SendResult, 0, len(users))
	for _, uid := range users {
		req := SendRequest{
			UserID:       uid,
			TemplateCode: templateCode,
			Page:         "",
			TemplateData: dataFn(uid),
		}
		err := s.Send(ctx, req)
		results = append(results, SendResult{UserID: uid, Err: err})
	}
	return results
}

func (s *Service) GetTemplate(code string) (*model.NotificationTemplate, error) {
	return s.repo.GetTemplateByCode(code)
}

func (s *Service) CreateTemplate(t *model.NotificationTemplate) error {
	return s.repo.CreateTemplate(t)
}

func (s *Service) GetLogs(filter LogFilter) ([]model.NotificationLog, int64, error) {
	return s.repo.ListLogs(filter)
}

func (s *Service) recordLog(req SendRequest, status string, errMsg string) {
	dataBytes, _ := json.Marshal(req.TemplateData)
	log := &model.NotificationLog{
		UserID:       req.UserID,
		TemplateCode: req.TemplateCode,
		TemplateData: string(dataBytes),
		Status:       status,
		ErrorMsg:     errMsg,
		CreatedAt:    time.Now(),
	}
	if status == "sent" {
		now := time.Now()
		log.SentAt = &now
	}
	s.repo.CreateLog(log)
}

func (s *Service) isEnabled() bool {
	v := os.Getenv("WECHAT_SUBSCRIBE_MSG_ENABLED")
	return v != "false"
}
```

- [ ] **Step 2: 创建 Service 测试**

```go
// internal/service/notification/service_test.go
package notification

import (
	"context"
	"errors"
	"testing"

	"manage/internal/model"
	"manage/internal/testutil"

	"github.com/stretchr/testify/require"
)

type mockWechatClient struct {
	sendErr error
}

func (m *mockWechatClient) SendSubscribeMessage(openid, templateID, page string, data map[string]interface{}) error {
	return m.sendErr
}

type mockUserRepo struct {
	openID string
	err    error
}

func (m *mockUserRepo) GetUserOpenID(userID uint) (string, error) {
	return m.openID, m.err
}

func TestSendSuccess(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "test_remind",
		WechatTemplateID: "tmpl_123",
		Name:             "测试提醒",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	mockWechat := &mockWechatClient{sendErr: nil}
	mockUsers := &mockUserRepo{openID: "openid_123", err: nil}

	svc := NewService(mockWechat, repo, mockUsers)

	err := svc.Send(context.Background(), SendRequest{
		UserID:       1,
		TemplateCode: "test_remind",
		Page:         "/pages/index",
		TemplateData: map[string]interface{}{
			"thing1": map[string]string{"value": "测试通知"},
		},
	})
	require.NoError(t, err)

	logs, total, err := repo.ListLogs(LogFilter{UserID: ptrUint(1), Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "sent", logs[0].Status)
}

func TestSendNoOpenID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "test_remind",
		WechatTemplateID: "tmpl_123",
		Name:             "测试提醒",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	mockWechat := &mockWechatClient{}
	mockUsers := &mockUserRepo{openID: "", err: nil}

	svc := NewService(mockWechat, repo, mockUsers)

	err := svc.Send(context.Background(), SendRequest{
		UserID:       1,
		TemplateCode: "test_remind",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no openid")
}

func TestSendTemplateNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	mockWechat := &mockWechatClient{}
	mockUsers := &mockUserRepo{openID: "openid_123"}

	svc := NewService(mockWechat, repo, mockUsers)

	err := svc.Send(context.Background(), SendRequest{
		UserID:       1,
		TemplateCode: "nonexistent",
	})
	require.Error(t, err)
}

func TestSendWechatError(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "test_remind",
		WechatTemplateID: "tmpl_123",
		Name:             "测试提醒",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	mockWechat := &mockWechatClient{sendErr: errors.New("wechat error 43004")}
	mockUsers := &mockUserRepo{openID: "openid_123"}

	svc := NewService(mockWechat, repo, mockUsers)

	err := svc.Send(context.Background(), SendRequest{
		UserID:       1,
		TemplateCode: "test_remind",
	})
	require.Error(t, err)

	logs, _, _ := repo.ListLogs(LogFilter{UserID: ptrUint(1), Limit: 10})
	require.Equal(t, "failed", logs[0].Status)
	require.Contains(t, logs[0].ErrorMsg, "43004")
}

func TestSendBatch(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "test_remind",
		WechatTemplateID: "tmpl_123",
		Name:             "测试提醒",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	mockWechat := &mockWechatClient{}
	mockUsers := &mockUserRepo{openID: "openid_123"}

	svc := NewService(mockWechat, repo, mockUsers)

	results := svc.SendBatch(context.Background(), "test_remind", []uint{1, 2, 3}, func(uid uint) map[string]interface{} {
		return map[string]interface{}{
			"thing1": map[string]string{"value": "通知"},
		}
	})
	require.Len(t, results, 3)
	for _, r := range results {
		require.NoError(t, r.Err)
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/service/notification/... -v -count=1
```
Expected: 全部通过

- [ ] **Step 4: Commit**

```bash
git add internal/service/notification/service.go internal/service/notification/service_test.go
git commit -m "feat: add notification service with send logic and tests"
```

---

### Task 5: 集成到应用 (user_repo + handler + router)

**Files:**
- Create: `internal/service/notification/user_repo.go`
- Create: `internal/http/handler/notification_handler.go`
- Modify: `internal/http/router/router.go` — 注册通知管理路由

- [ ] **Step 1: 创建 UserRepo 适配器**

```go
// internal/service/notification/user_repo.go
package notification

import "gorm.io/gorm"

type GormUserRepo struct {
	db *gorm.DB
}

func NewGormUserRepo(db *gorm.DB) *GormUserRepo {
	return &GormUserRepo{db: db}
}

func (r *GormUserRepo) GetUserOpenID(userID uint) (string, error) {
	var openID *string
	err := r.db.Model(&struct {
		OpenID *string
	}{
		OpenID: openID,
	}).Table("users").Select("openid").Where("id = ?", userID).Scan(&openID).Error
	if err != nil {
		return "", err
	}
	if openID == nil {
		return "", nil
	}
	return *openID, nil
}
```

- [ ] **Step 2: 创建 Notification Handler**

```go
// internal/http/handler/notification_handler.go
package handler

import (
	"net/http"
	"strconv"

	"manage/internal/model"
	"manage/internal/service/notification"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	svc *notification.Service
}

func NewNotificationHandler(svc *notification.Service) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) CreateTemplate(c *gin.Context) {
	var req struct {
		Code             string `json:"code" binding:"required"`
		WechatTemplateID string `json:"wechat_template_id" binding:"required"`
		Name             string `json:"name" binding:"required"`
		Fields           string `json:"fields"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tmpl := &model.NotificationTemplate{
		Code:             req.Code,
		WechatTemplateID: req.WechatTemplateID,
		Name:             req.Name,
		Fields:           req.Fields,
	}

	if err := h.svc.CreateTemplate(tmpl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": tmpl})
}

func (h *NotificationHandler) GetTemplate(c *gin.Context) {
	code := c.Param("code")
	tmpl, err := h.svc.GetTemplate(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tmpl})
}

func (h *NotificationHandler) ListLogs(c *gin.Context) {
	filter := notification.LogFilter{
		Limit: 20,
	}

	if uid := c.Query("user_id"); uid != "" {
		id, err := strconv.ParseUint(uid, 10, 32)
		if err == nil {
			uidUint := uint(id)
			filter.UserID = &uidUint
		}
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if tmpl := c.Query("template_code"); tmpl != "" {
		filter.TemplateCode = &tmpl
	}
	if offset := c.Query("offset"); offset != "" {
		o, _ := strconv.Atoi(offset)
		filter.Offset = o
	}
	if limit := c.Query("limit"); limit != "" {
		l, _ := strconv.Atoi(limit)
		if l > 0 {
			filter.Limit = l
		}
	}

	logs, total, err := h.svc.GetLogs(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
	})
}
```

- [ ] **Step 3: 更新 router.go**

在 `internal/http/router/router.go` 中：

1. 在 import 中添加：
```go
"manage/internal/service/notification"
```

2. 在 `admin` group 末尾、`return r` 之前添加：

```go
// Notification routes
notifSvc := initNotificationSvc(db)
notifHandler := handler.NewNotificationHandler(notifSvc)

admin.POST("/notification/templates", notifHandler.CreateTemplate)
admin.GET("/notification/templates/:code", notifHandler.GetTemplate)
admin.GET("/notification/logs", notifHandler.ListLogs)
```

3. 在文件末尾添加辅助函数：

```go
func initNotificationSvc(db *gorm.DB) *notification.Service {
	appID := os.Getenv("WECHAT_APP_ID")
	appSecret := os.Getenv("WECHAT_APP_SECRET")
	wechatClient := notification.NewWechatClient(nil, appID, appSecret)
	repo := notification.NewRepo(db)
	userRepo := notification.NewGormUserRepo(db)
	return notification.NewService(wechatClient, repo, userRepo)
}
```

- [ ] **Step 4: 构建验证**

```bash
go build ./...
```
Expected: 无编译错误

- [ ] **Step 5: Commit**

```bash
git add internal/service/notification/user_repo.go internal/http/handler/notification_handler.go internal/http/router/router.go
git commit -m "feat: integrate notification module into app with admin routes"
```

---

### Task 6: 添加 Handler 测试

**Files:**
- Create: `internal/http/handler/notification_handler_test.go`

- [ ] **Step 1: 创建 Handler 测试**

```go
// internal/http/handler/notification_handler_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"manage/internal/model"
	"manage/internal/service/notification"
	"manage/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupNotificationTest(t *testing.T) (*NotificationHandler, *notification.Repo) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := notification.NewRepo(db)
	mockWechat := &mockNotifWechatClient{}
	mockUsers := &mockNotifUserRepo{openID: "test_openid"}
	svc := notification.NewService(mockWechat, repo, mockUsers)
	return NewNotificationHandler(svc), repo
}

type mockNotifWechatClient struct{}

func (m *mockNotifWechatClient) SendSubscribeMessage(openid, templateID, page string, data map[string]interface{}) error {
	return nil
}

type mockNotifUserRepo struct {
	openID string
}

func (m *mockNotifUserRepo) GetUserOpenID(userID uint) (string, error) {
	return m.openID, nil
}

func TestCreateTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := setupNotificationTest(t)

	body := map[string]string{
		"code":               "test_tpl",
		"wechat_template_id": "tmpl_abc",
		"name":               "测试模板",
		"fields":             `{"thing1":"事项"}`,
	}
	data, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/notification/templates", bytes.NewReader(data))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateTemplate(c)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateTemplateMissingField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := setupNotificationTest(t)

	body := map[string]string{
		"code": "test_tpl",
	}
	data, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/notification/templates", bytes.NewReader(data))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateTemplate(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, repo := setupNotificationTest(t)

	tmpl := &model.NotificationTemplate{
		Code:             "test_tpl",
		WechatTemplateID: "tmpl_abc",
		Name:             "测试模板",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notification/templates/test_tpl", nil)
	c.AddParam("code", "test_tpl")

	handler.GetTemplate(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	require.Equal(t, "test_tpl", data["code"])
}

func TestGetTemplateNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := setupNotificationTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notification/templates/nonexistent", nil)
	c.AddParam("code", "nonexistent")

	handler.GetTemplate(c)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, repo := setupNotificationTest(t)

	repo.CreateLog(&model.NotificationLog{
		UserID:       1,
		TemplateCode: "test_tpl",
		Status:       "sent",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notification/logs", nil)

	handler.ListLogs(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.Equal(t, float64(1), resp["total"])
}
```

- [ ] **Step 2: 运行全部测试**

```bash
go test ./... -count=1 -v
```
Expected: 全部通过

- [ ] **Step 3: Commit**

```bash
git add internal/http/handler/notification_handler_test.go
git commit -m "test: add notification handler tests"
```

---

### Task 7: 更新 .env.example 与文档

**Files:**
- Modify: `.env.example`
- Modify: `README.md`

- [ ] **Step 1: 更新 .env.example**

在现有环境变量后添加：

```
WECHAT_SUBSCRIBE_MSG_ENABLED=true
```

- [ ] **Step 2: 更新 README.md**

在 Environment Variables 表格中添加一行：

| `WECHAT_SUBSCRIBE_MSG_ENABLED` | Enable WeChat subscribe message sending (default: true) |

- [ ] **Step 3: 最终验证**

```bash
go build ./...
go test ./... -count=1
```
Expected: 构建成功，全部测试通过

- [ ] **Step 4: Commit**

```bash
git add .env.example README.md
git commit -m "docs: add notification module env var and update README"
```
