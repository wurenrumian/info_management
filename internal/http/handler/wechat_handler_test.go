package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"manage/internal/http/handler"
	"manage/internal/http/router"
	"manage/internal/model"
	"manage/internal/service/notification"
)

type mockDevSubscribeWechatClient struct{}

func (m *mockDevSubscribeWechatClient) SendSubscribeMessage(openid, templateID, page string, data map[string]interface{}) error {
	return nil
}

func setupWechatTestRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Class{}, &model.NotificationTemplate{}, &model.NotificationLog{}, &model.UserSubscribe{}))
	require.NoError(t, db.Create(&model.Class{
		ID:        1,
		ClassName: "Class 1",
		Grade:     "2023",
		Major:     "信息管理",
	}).Error)

	pwd := "hashed_password"
	require.NoError(t, db.Create(&model.User{
		ID:           100,
		StudentID:    "S100",
		Name:         "张三",
		Role:         model.RoleStudent,
		ClassID:      1,
		Grade:        "2023",
		PasswordHash: &pwd,
	}).Error)

	r := router.New(db)
	return db, r
}

func TestWechatLoginMissingCode(t *testing.T) {
	_, r := setupWechatTestRouter(t)

	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "missing code")
}

func TestWechatLoginInvalidCode(t *testing.T) {
	_, r := setupWechatTestRouter(t)

	body := []byte(`{"code":"invalid_code"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid authorization code")
}

func TestWechatBindMissingCode(t *testing.T) {
	_, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S100","password":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/bind", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "missing code")
}

func TestWechatBindInvalidCode(t *testing.T) {
	_, r := setupWechatTestRouter(t)

	body := []byte(`{"code":"bad_code","student_id":"S100","password":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/bind", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid authorization code")
}

func TestDevRegisterOrLoginReturnsTokenForExistingUser(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	_, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S100"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/register-or-login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"token"`)
	require.Contains(t, w.Body.String(), `"student_id":"S100"`)
}

func TestDevRegisterOrLoginCreatesUserWhenMissing(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	db, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S200","role":2}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/register-or-login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"student_id":"S200"`)
	require.Contains(t, w.Body.String(), `"name":"Dev-S200"`)
	require.Contains(t, w.Body.String(), `"role":2`)

	var user model.User
	require.NoError(t, db.Where("student_id = ?", "S200").First(&user).Error)
	require.Equal(t, "Dev-S200", user.Name)
	require.Equal(t, model.RoleCadre, user.Role)
	require.Equal(t, "2020", user.Grade)
	require.Equal(t, "信息管理", user.Major)
	require.Equal(t, uint(10), user.ClassID)
}

func TestDevRegisterOrLoginForbiddenWhenDisabled(t *testing.T) {
	original, had := os.LookupEnv("APP_ENV")
	if had {
		t.Cleanup(func() { _ = os.Setenv("APP_ENV", original) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv("APP_ENV") })
	}
	_ = os.Unsetenv("APP_ENV")

	_, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S100"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/register-or-login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "dev register-or-login is disabled")
}

func TestDevLoginAndSendSubscribeCheckForbiddenWhenDisabled(t *testing.T) {
	original, had := os.LookupEnv("APP_ENV")
	if had {
		t.Cleanup(func() { _ = os.Setenv("APP_ENV", original) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv("APP_ENV") })
	}
	_ = os.Unsetenv("APP_ENV")

	_, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S100"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/login-and-send-subscribe-check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "dev login-and-send-subscribe-check is disabled")
}

func TestDevLoginAndSendSubscribeCheckCreatesSubscription(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	t.Setenv("WECHAT_SUBSCRIBE_MSG_ENABLED", "false")

	db, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S100","template_code":"dev_login_check","wechat_template_id":"tmpl_dev_check","status":"accept","open_id":"dev-openid-s100"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/login-and-send-subscribe-check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"token"`)
	require.Contains(t, w.Body.String(), `"send_ok":true`)
	require.Contains(t, w.Body.String(), `"granted_count":1`)
	require.Contains(t, w.Body.String(), `"consumed_count":0`)
	require.Contains(t, w.Body.String(), `"remaining_count":1`)

	var tmpl model.NotificationTemplate
	require.NoError(t, db.Where("code = ?", "dev_login_check").First(&tmpl).Error)
	require.Equal(t, "tmpl_dev_check", tmpl.WechatTemplateID)

	var sub model.UserSubscribe
	require.NoError(t, db.Where("user_id = ? AND template_code = ?", 100, "dev_login_check").First(&sub).Error)
	require.Equal(t, "subscribed", sub.Status)
	require.Equal(t, 1, sub.GrantedCount)
	require.Equal(t, 0, sub.ConsumedCount)

	var user model.User
	require.NoError(t, db.Where("id = ?", 100).First(&user).Error)
	require.NotNil(t, user.OpenID)
	require.Equal(t, "dev-openid-s100", *user.OpenID)
}

func TestDevLoginAndSendSubscribeCheckFillsLoginTemplateDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	t.Setenv("WECHAT_SUBSCRIBE_MSG_ENABLED", "false")

	_, r := setupWechatTestRouter(t)

	body := []byte(`{
		"student_id":"S100",
		"template_code":"loging_notification",
		"wechat_template_id":"tmpl_login_notification",
		"status":"accept",
		"open_id":"dev-openid-s100",
		"template_data":{
			"thing1":{"value":"登录提醒"},
			"time2":{"value":""}
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/login-and-send-subscribe-check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.10:4567"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"send_ok":true`)
}

func TestDevLoginAndSendSubscribeCheckDoesNotIncreaseGrantedCountOnSuccessfulSend(t *testing.T) {
	t.Setenv("APP_ENV", "dev")

	db, _ := setupWechatTestRouter(t)
	require.NoError(t, db.Create(&model.NotificationTemplate{
		Code:             "dev_login_check",
		WechatTemplateID: "tmpl_dev_check",
		Name:             "Dev Login Check",
	}).Error)
	require.NoError(t, db.Create(&model.UserSubscribe{
		UserID:           100,
		TemplateCode:     "dev_login_check",
		WechatTemplateID: "tmpl_dev_check",
		Status:           "subscribed",
		GrantedCount:     1,
		ConsumedCount:    0,
	}).Error)

	notifSvc := notification.NewService(
		&mockDevSubscribeWechatClient{},
		notification.NewRepo(db),
		notification.NewGormUserRepo(db),
	)
	wechatHandler := handler.NewWechatHandler(db, "", "", "test-secret", notifSvc)

	body := []byte(`{"student_id":"S100","template_code":"dev_login_check","wechat_template_id":"tmpl_dev_check","status":"accept","open_id":"dev-openid-s100"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/dev/login-and-send-subscribe-check", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	wechatHandler.DevLoginAndSendSubscribeCheck(c)

	require.Equal(t, http.StatusOK, w.Code)

	var sub model.UserSubscribe
	require.NoError(t, db.Where("user_id = ? AND template_code = ?", 100, "dev_login_check").First(&sub).Error)
	require.Equal(t, 1, sub.GrantedCount)
	require.Equal(t, 1, sub.ConsumedCount)
}

func TestPublicRegisterCreatesUserWhenMissing(t *testing.T) {
	db, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S300","name":"李四"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/public-register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"token"`)
	require.Contains(t, w.Body.String(), `"student_id":"S300"`)
	require.Contains(t, w.Body.String(), `"name":"李四"`)

	var user model.User
	require.NoError(t, db.Where("student_id = ?", "S300").First(&user).Error)
	require.Equal(t, "李四", user.Name)
	require.Equal(t, model.RoleStudent, user.Role)
	require.NotZero(t, user.ClassID)

	var class model.Class
	require.NoError(t, db.First(&class, user.ClassID).Error)
	require.Equal(t, "未绑定班级", class.ClassName)
}

func TestPublicRegisterReturnsTokenForExistingUser(t *testing.T) {
	_, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S100","name":"张三"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/public-register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"token"`)
	require.Contains(t, w.Body.String(), `"student_id":"S100"`)
}

func TestPublicRegisterNameMismatch(t *testing.T) {
	_, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S100","name":"王五"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/public-register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "student id and name do not match")
}

func TestPublicRegisterMissingFields(t *testing.T) {
	_, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S100"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/public-register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "missing student_id or name")
}
