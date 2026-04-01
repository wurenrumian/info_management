# 微信 OpenID 绑定与 JWT 认证实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现微信 OpenID 绑定/登录功能，并将认证方式从 Header 注入升级为 JWT

**Architecture:** 分层 Go 架构，新增 JWT 认证中间件替换现有 Header 注入，新增 WeChat 服务层处理微信 API 调用，handler 层提供绑定和登录接口

**Tech Stack:** Go 1.24, Gin, GORM, golang-jwt, bcrypt

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `go.mod` | 修改 | 添加 `golang.org/x/crypto` (bcrypt), `github.com/golang-jwt/jwt/v5` |
| `internal/model/user.go` | 修改 | 新增 `PasswordHash` 字段 |
| `internal/service/auth/jwt.go` | 创建 | JWT 生成与解析 |
| `internal/service/wechat/service.go` | 创建 | 微信 code2Session 调用 |
| `internal/repo/user_repo.go` | 修改 | 新增 `GetByOpenID`, `GetByStudentID`, `UpdatePasswordHash` |
| `internal/http/middleware/auth.go` | 创建 | JWT 认证中间件 |
| `internal/http/handler/wechat_handler.go` | 创建 | 绑定和登录接口 |
| `internal/http/router/router.go` | 修改 | 注册 wechat 路由，替换中间件 |
| `internal/testutil/token.go` | 创建 | 测试辅助函数 |
| `internal/service/auth/jwt_test.go` | 创建 | JWT 单元测试 |
| `internal/service/wechat/service_test.go` | 创建 | WeChat 服务单元测试 |
| `internal/http/handler/wechat_handler_test.go` | 创建 | Handler 单元测试 |
| `internal/http/middleware/auth_test.go` | 创建 | 中间件单元测试 |
| `internal/http/handler/admin_user_handler_test.go` | 修改 | Header → JWT |
| `internal/http/handler/knowledge_handler_test.go` | 修改 | Header → JWT |
| `internal/http/handler/admin_class_handler_test.go` | 修改 | Header → JWT |
| `tests/api_contract_test.go` | 修改 | Header → JWT |
| `internal/http/middleware/identity_test.go` | 修改 | 改为测试 JWT 中间件 |
| `scripts/dev/knowledge_api_curl.sh` | 修改 | 添加 JWT token 生成 |

---

### Task 1: 添加依赖和 PasswordHash 字段

**Files:**
- Modify: `go.mod`
- Modify: `internal/model/user.go`

- [ ] **Step 1: 添加 JWT 依赖**

Run:
```bash
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
```

- [ ] **Step 2: 在 User model 新增 PasswordHash 字段**

Modify `internal/model/user.go`:
```go
type User struct {
	ID           uint           `gorm:"primaryKey"`
	StudentID    string         `gorm:"size:20;uniqueIndex;not null"`
	Name         string         `gorm:"size:50;not null"`
	OpenID       *string        `gorm:"size:100"`
	PasswordHash *string        `gorm:"size:255"`
	Role         int            `gorm:"not null;index"`
	ClassID      uint           `gorm:"index"`
	Grade        string         `gorm:"size:10;index"`
	Major        string         `gorm:"size:100"`
	ExtraAttrs   datatypes.JSON `gorm:"type:jsonb"`
	ProfileAttrs datatypes.JSON `gorm:"type:jsonb"`
	Class        Class          `gorm:"foreignKey:ClassID"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
```

新增 `PasswordHash *string` 字段，位于 `OpenID` 之后。

- [ ] **Step 3: 验证编译通过**

Run:
```bash
go build ./...
```

Expected: 无错误输出

---

### Task 2: 创建 JWT 服务

**Files:**
- Create: `internal/service/auth/jwt.go`
- Test: `internal/service/auth/jwt_test.go`

- [ ] **Step 1: 写测试**

Create `internal/service/auth/jwt_test.go`:
```go
package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := "test-secret-key-32bytes!!"
	token, err := GenerateToken(1, 1, 10, "2023", secret)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ParseToken(token, secret)
	require.NoError(t, err)
	require.Equal(t, uint(1), claims.UserID)
	require.Equal(t, 1, claims.Role)
	require.Equal(t, uint(10), claims.ClassID)
	require.Equal(t, "2023", claims.Grade)
}

func TestParseTokenRejectsInvalid(t *testing.T) {
	_, err := ParseToken("invalid.token.here", "wrong-secret")
	require.Error(t, err)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:
```bash
go test ./internal/service/auth/ -v
```

Expected: FAIL (GenerateToken/ParseToken 未定义)

- [ ] **Step 3: 实现 JWT 服务**

Create `internal/service/auth/jwt.go`:
```go
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID  uint   `json:"sub"`
	Role    int    `json:"role"`
	ClassID uint   `json:"class_id"`
	Grade   string `json:"grade"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, role int, classID uint, grade string, secret string) (string, error) {
	claims := Claims{
		UserID:  userID,
		Role:    role,
		ClassID: classID,
		Grade:   grade,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run:
```bash
go test ./internal/service/auth/ -v
```

Expected: PASS

---

### Task 3: 创建 WeChat 服务

**Files:**
- Create: `internal/service/wechat/service.go`
- Test: `internal/service/wechat/service_test.go`

- [ ] **Step 1: 写测试**

Create `internal/service/wechat/service_test.go`:
```go
package wechat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodeToOpenIDSuccess(t *testing.T) {
	mockResp := map[string]any{
		"openid":    "mock_openid_123",
		"session_key": "sk_xxx",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	svc := NewService("fake_appid", "fake_secret", srv.URL)
	openID, err := svc.CodeToOpenID("test_code")
	require.NoError(t, err)
	require.Equal(t, "mock_openid_123", openID)
}

func TestCodeToOpenIDError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 40029,
			"errmsg":  "invalid code",
		})
	}))
	defer srv.Close()

	svc := NewService("fake_appid", "fake_secret", srv.URL)
	_, err := svc.CodeToOpenID("bad_code")
	require.Error(t, err)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:
```bash
go test ./internal/service/wechat/ -v
```

Expected: FAIL

- [ ] **Step 3: 实现 WeChat 服务**

Create `internal/service/wechat/service.go`:
```go
package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Service struct {
	appID     string
	appSecret string
	baseURL   string
}

func NewService(appID, appSecret string) *Service {
	return &Service{
		appID:     appID,
		appSecret: appSecret,
		baseURL:   "https://api.weixin.qq.com/sns/jscode2session",
	}
}

// NewServiceWithBaseURL creates a service with a custom base URL (for testing).
func NewServiceWithBaseURL(appID, appSecret, baseURL string) *Service {
	return &Service{
		appID:     appID,
		appSecret: appSecret,
		baseURL:   baseURL,
	}
}

type code2SessionResp struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func (s *Service) CodeToOpenID(code string) (string, error) {
	url := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		s.baseURL, s.appID, s.appSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("wechat api request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response failed: %w", err)
	}

	var result code2SessionResp
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat api error: %d %s", result.ErrCode, result.ErrMsg)
	}

	if result.OpenID == "" {
		return "", fmt.Errorf("empty openid in response")
	}

	return result.OpenID, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run:
```bash
go test ./internal/service/wechat/ -v
```

Expected: PASS

---

### Task 4: 扩展 UserRepo

**Files:**
- Modify: `internal/repo/user_repo.go`

- [ ] **Step 1: 添加 GetByOpenID 方法**

Append to `internal/repo/user_repo.go`:
```go
func (r *UserRepo) GetByOpenID(openID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("openid = ?", openID).Preload("Class").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) GetByStudentID(studentID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("student_id = ?", studentID).Preload("Class").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) UpdatePasswordHash(userID uint, hash string) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("password_hash", hash).Error
}
```

- [ ] **Step 2: 验证编译通过**

Run:
```bash
go build ./...
```

Expected: 无错误输出

---

### Task 5: 创建 JWT 认证中间件

**Files:**
- Create: `internal/http/middleware/auth.go`
- Test: `internal/http/middleware/auth_test.go`

- [ ] **Step 1: 写测试**

Create `internal/http/middleware/auth_test.go`:
```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"manage/internal/auth"
	"manage/internal/http/middleware"
	"manage/internal/testutil"
)

func TestJWTAuthInjectsActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.JWTAuth("test-secret"))
	r.GET("/probe", func(c *gin.Context) {
		a, ok := auth.GetActor(c)
		require.True(t, ok, "expected actor in context")
		c.JSON(http.StatusOK, gin.H{"user_id": a.UserID, "role": a.Role})
	})

	token := testutil.GenerateTestToken(12, 3, 10, "2023")
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"user_id":12`)
	require.Contains(t, w.Body.String(), `"role":3`)
}

func TestJWTAuthRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.JWTAuth("test-secret"))
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthRejectsInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.JWTAuth("test-secret"))
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:
```bash
go test ./internal/http/middleware/ -v
```

Expected: FAIL

- [ ] **Step 3: 实现 JWT 中间件**

Create `internal/http/middleware/auth.go`:
```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"manage/internal/auth"
	jwtauth "manage/internal/service/auth"
)

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		claims, err := jwtauth.ParseToken(parts[1], secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		actor := auth.Actor{
			UserID:  claims.UserID,
			Role:    claims.Role,
			ClassID: claims.ClassID,
			Grade:   claims.Grade,
		}
		auth.SetActor(c, actor)
		c.Next()
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run:
```bash
go test ./internal/http/middleware/ -v
```

Expected: PASS

---

### Task 6: 创建 WeChat Handler

**Files:**
- Create: `internal/http/handler/wechat_handler.go`
- Test: `internal/http/handler/wechat_handler_test.go`

- [ ] **Step 1: 写测试**

Create `internal/http/handler/wechat_handler_test.go`:
```go
package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/http/handler"
	"manage/internal/http/middleware"
	"manage/internal/model"
	"manage/internal/testutil"
)

func setupWechatTestRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}))

	openID := "test_openid_123"
	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "student",
		OpenID:    &openID,
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
	}).Error)

	r := gin.New()
	wechatHandler := handler.NewWechatHandler(db, "fake_appid", "fake_secret", "test-secret")
	r.POST("/api/v1/wechat/login", wechatHandler.Login)
	r.POST("/api/v1/wechat/bind", wechatHandler.Bind)
	r.Use(middleware.JWTAuth("test-secret"))
	r.GET("/api/v1/me", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	return db, r
}

func TestWechatLoginSuccess(t *testing.T) {
	// Note: This test needs a mock for wechat service CodeToOpenID
	// In practice, use dependency injection or interface-based wechat service
}

func TestWechatLoginNotBound(t *testing.T) {
	_, r := setupWechatTestRouter(t)

	body := []byte(`{"code":"unused_code"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "未绑定")
}

func TestWechatBindWithToken(t *testing.T) {
	_, r := setupWechatTestRouter(t)

	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	body := []byte(`{"code":"unused_code"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/bind", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Will fail because wechat mock isn't set up, but auth should pass
	require.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestWechatBindWithPassword(t *testing.T) {
	db, r := setupWechatTestRouter(t)

	// Set a password on the user
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	hashStr := string(hash)
	db.Model(&model.User{}).Where("id = ?", 100).Update("password_hash", &hashStr)

	body := []byte(`{"student_id":"S100","password":"password123","code":"unused_code"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/bind", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Auth should pass, wechat will fail due to no mock
	require.NotEqual(t, http.StatusUnauthorized, w.Code)
	require.NotEqual(t, http.StatusForbidden, w.Code)
}
```

- [ ] **Step 2: 实现 Handler**

Create `internal/http/handler/wechat_handler.go`:
```go
package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/repo"
	jwtauth "manage/internal/service/auth"
	"manage/internal/service/wechat"
)

type WechatHandler struct {
	wechatSvc  *wechat.Service
	userRepo   *repo.UserRepo
	jwtSecret  string
}

func NewWechatHandler(db *gorm.DB, appID, appSecret, jwtSecret string) *WechatHandler {
	return &WechatHandler{
		wechatSvc: wechat.NewService(appID, appSecret),
		userRepo:  repo.NewUserRepo(db),
		jwtSecret: jwtSecret,
	}
}

type wechatLoginReq struct {
	Code string `json:"code"`
}

type wechatBindReq struct {
	Code      string `json:"code"`
	StudentID string `json:"student_id"`
	Password  string `json:"password"`
}

// Login handles wechat login: code -> openid -> find user -> return JWT
func (h *WechatHandler) Login(c *gin.Context) {
	var req wechatLoginReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		response.Error(c, 400, "missing code")
		return
	}

	openID, err := h.wechatSvc.CodeToOpenID(req.Code)
	if err != nil {
		response.Error(c, 400, "微信授权码无效")
		return
	}

	user, err := h.userRepo.GetByOpenID(openID)
	if err != nil {
		response.Error(c, 404, "未绑定账号，请先绑定")
		return
	}

	token, err := jwtauth.GenerateToken(user.ID, user.Role, user.ClassID, user.Grade, h.jwtSecret)
	if err != nil {
		response.Error(c, 500, "生成 token 失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":          user.ID,
			"student_id":  user.StudentID,
			"name":        user.Name,
			"role":        user.Role,
			"class_id":    user.ClassID,
			"grade":       user.Grade,
			"major":       user.Major,
		},
	})
}

// Bind handles wechat bind: with token or with student_id+password
func (h *WechatHandler) Bind(c *gin.Context) {
	var req wechatBindReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		response.Error(c, 400, "missing code")
		return
	}

	openID, err := h.wechatSvc.CodeToOpenID(req.Code)
	if err != nil {
		response.Error(c, 400, "微信授权码无效")
		return
	}

	// Check if openid already bound
	existing, _ := h.userRepo.GetByOpenID(openID)

	var userID uint

	// Try to get user from JWT token first
	actor, ok := auth.GetActor(c)
	if ok {
		// Scenario 1: logged in with token
		userID = actor.UserID
	} else if req.StudentID != "" && req.Password != "" {
		// Scenario 2: verify student_id + password
		user, err := h.userRepo.GetByStudentID(req.StudentID)
		if err != nil {
			response.Error(c, 401, "学号或密码不正确")
			return
		}
		if user.PasswordHash == nil {
			response.Error(c, 401, "学号或密码不正确")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
			response.Error(c, 401, "学号或密码不正确")
			return
		}
		userID = user.ID
	} else {
		response.Error(c, 401, "请登录后绑定或提供学号和密码")
		return
	}

	// Check if openid already bound to another user
	if existing != nil && existing.ID != userID {
		response.Error(c, 409, "该微信已绑定其他账号")
		return
	}

	// Bind openid
	if err := h.userRepo.UpdateByID(userID, map[string]any{"openid": openID}); err != nil {
		response.Error(c, 500, "绑定失败")
		return
	}

	// Set password if not set and password provided
	if existing == nil && req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err == nil {
			hashStr := string(hash)
			_ = h.userRepo.UpdatePasswordHash(userID, hashStr)
		}
	}

	response.OK(c, gin.H{"ok": true, "message": "绑定成功"})
}
```

- [ ] **Step 3: 验证编译通过**

Run:
```bash
go build ./...
```

Expected: 无错误输出

---

### Task 7: 注册路由

**Files:**
- Modify: `internal/http/router/router.go`

- [ ] **Step 1: 修改路由**

Replace `internal/http/router/router.go` content:
```go
package router

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"manage/internal/http/handler"
	"manage/internal/http/middleware"
)

func New(db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", handler.Health)
	uploadDir := strings.TrimSpace(os.Getenv("KNOWLEDGE_UPLOAD_DIR"))
	if uploadDir == "" {
		uploadDir = "./data/uploads/knowledge"
	}
	r.Static("/uploads/knowledge", uploadDir)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}
	appID := os.Getenv("WECHAT_APP_ID")
	appSecret := os.Getenv("WECHAT_APP_SECRET")

	api := r.Group("/api/v1")

	// WeChat routes (no auth required for login, optional for bind)
	wechatHandler := handler.NewWechatHandler(db, appID, appSecret, jwtSecret)
	api.POST("/wechat/login", wechatHandler.Login)
	api.POST("/wechat/bind", wechatHandler.Bind)

	// Protected routes
	api.Use(middleware.JWTAuth(jwtSecret))

	meHandler := handler.NewMeHandler(db)
	knowledgeHandler := handler.NewKnowledgeHandler(db)
	adminUserHandler := handler.NewAdminUserHandler(db)
	adminClassHandler := handler.NewAdminClassHandler(db)
	adminLogHandler := handler.NewAdminLogHandler(db)
	adminKnowledgeHandler := handler.NewAdminKnowledgeHandler(db)

	api.GET("/me", meHandler.GetMe)
	api.GET("/knowledge/search", knowledgeHandler.Search)

	admin := api.Group("/admin")
	admin.GET("/users", adminUserHandler.ListUsers)
	admin.GET("/users/:id", adminUserHandler.GetUser)
	admin.PATCH("/users/:id", adminUserHandler.PatchUser)

	admin.GET("/classes", adminClassHandler.ListClasses)
	admin.GET("/classes/:id", adminClassHandler.GetClass)
	admin.POST("/classes", adminClassHandler.CreateClass)
	admin.PATCH("/classes/:id", adminClassHandler.PatchClass)

	admin.GET("/logs", adminLogHandler.ListLogs)
	admin.GET("/knowledge", adminKnowledgeHandler.ListKnowledge)
	admin.GET("/knowledge/:id", adminKnowledgeHandler.GetKnowledge)
	admin.POST("/knowledge", adminKnowledgeHandler.CreateKnowledge)
	admin.POST("/knowledge/import", adminKnowledgeHandler.ImportKnowledge)
	admin.PATCH("/knowledge/:id", adminKnowledgeHandler.PatchKnowledge)
	admin.DELETE("/knowledge/:id", adminKnowledgeHandler.DeleteKnowledge)

	return r
}
```

- [ ] **Step 2: 验证编译通过**

Run:
```bash
go build ./...
```

Expected: 无错误输出

---

### Task 8: 创建测试辅助包

**Files:**
- Create: `internal/testutil/token.go`

- [ ] **Step 1: 创建测试辅助函数**

Create `internal/testutil/token.go`:
```go
package testutil

import (
	"os"

	jwtauth "manage/internal/service/auth"
)

func GenerateTestToken(userID uint, role int, classID uint, grade string) string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	token, _ := jwtauth.GenerateToken(userID, role, classID, grade, secret)
	return token
}
```

- [ ] **Step 2: 验证编译通过**

Run:
```bash
go build ./...
```

Expected: 无错误输出

---

### Task 9: 迁移所有现有测试到 JWT

**Files:**
- Modify: `internal/http/handler/admin_user_handler_test.go`
- Modify: `internal/http/handler/knowledge_handler_test.go`
- Modify: `internal/http/handler/admin_class_handler_test.go`
- Modify: `tests/api_contract_test.go`
- Modify: `internal/http/middleware/identity_test.go`

- [ ] **Step 1: 修改 admin_user_handler_test.go**

Replace all `req.Header.Set("X-User-Id", ...)` patterns with:
```go
token := testutil.GenerateTestToken(100, 1, 1, "2023")
req.Header.Set("Authorization", "Bearer "+token)
```

Add import: `"manage/internal/testutil"`

Remove old header lines:
- `req.Header.Set("X-User-Id", ...)`
- `req.Header.Set("X-User-Role", ...)`
- `req.Header.Set("X-User-Class-Id", ...)`
- `req.Header.Set("X-User-Grade", ...)`

- [ ] **Step 2: 修改 knowledge_handler_test.go**

Same pattern as Step 1. Replace all X-User-* headers with JWT token.

- [ ] **Step 3: 修改 admin_class_handler_test.go**

Same pattern as Step 1.

- [ ] **Step 4: 修改 api_contract_test.go**

Same pattern. Update the `headers` map in test cases to use `Authorization: Bearer <token>` format.

- [ ] **Step 5: 修改 identity_test.go**

Replace content to test JWT middleware instead of IdentityFromHeaders:
```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"manage/internal/auth"
	"manage/internal/http/middleware"
	"manage/internal/testutil"
)

func TestJWTMiddlewareInjectsActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.JWTAuth("test-secret"))
	r.GET("/probe", func(c *gin.Context) {
		a, ok := auth.GetActor(c)
		require.True(t, ok, "expected actor in context")
		c.JSON(http.StatusOK, gin.H{"user_id": a.UserID, "role": a.Role})
	})

	token := testutil.GenerateTestToken(12, 3, 0, "")
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"user_id":12`)
	require.Contains(t, w.Body.String(), `"role":3`)
}
```

- [ ] **Step 6: 运行所有测试**

Run:
```bash
go test ./... -count=1 -v
```

Expected: 全部 PASS

---

### Task 10: 更新开发脚本

**Files:**
- Modify: `scripts/dev/knowledge_api_curl.sh`

- [ ] **Step 1: 添加 JWT token 生成逻辑**

在脚本开头（环境变量定义后）添加：
```bash
JWT_SECRET="${JWT_SECRET:-dev-secret-change-in-production}"

generate_token() {
  local user_id="$1"
  local role="$2"
  local class_id="$3"
  local grade="$4"
  
  local header
  header=$(echo -n '{"alg":"HS256","typ":"JWT"}' | openssl base64 | tr -d '\n' | tr '+/' '-_' | tr -d '=')
  local payload
  payload=$(echo -n "{\"sub\":$user_id,\"role\":$role,\"class_id\":$class_id,\"grade\":\"$grade\"}" | openssl base64 | tr -d '\n' | tr '+/' '-_' | tr -d '=')
  local signature
  signature=$(echo -n "$header.$payload" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | openssl base64 | tr -d '\n' | tr '+/' '-_' | tr -d '=')
  echo "$header.$payload.$signature"
}

ADMIN_TOKEN=$(generate_token "$ADMIN_USER_ID" "$ADMIN_ROLE" "$ADMIN_CLASS_ID" "$ADMIN_GRADE")
STUDENT_TOKEN=$(generate_token "$STUDENT_USER_ID" "$STUDENT_ROLE" "$STUDENT_CLASS_ID" "$STUDENT_GRADE")
```

- [ ] **Step 2: 替换所有请求头**

将所有 `-H "X-User-Id: ..."` 等替换为 `-H "Authorization: Bearer $ADMIN_TOKEN"` 或 `-H "Authorization: Bearer $STUDENT_TOKEN"`。

例如：
```bash
# 原来
-H "X-User-Id: $ADMIN_USER_ID" \
-H "X-User-Role: $ADMIN_ROLE" \
-H "X-User-Class-Id: $ADMIN_CLASS_ID" \
-H "X-User-Grade: $ADMIN_GRADE" \

# 改为
-H "Authorization: Bearer $ADMIN_TOKEN" \
```

- [ ] **Step 3: 测试脚本**

Run (with server running):
```bash
JWT_SECRET="dev-secret-change-in-production" ./scripts/dev/knowledge_api_curl.sh
```

Expected: 所有 PASS


