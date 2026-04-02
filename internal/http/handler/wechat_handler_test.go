package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/http/router"
	"manage/internal/model"
)

func setupWechatTestRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Class{}))

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

func TestDevLoginReturnsTokenWhenEnabled(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	_, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S100"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"token"`)
	require.Contains(t, w.Body.String(), `"student_id":"S100"`)
}

func TestDevLoginForbiddenWhenDisabled(t *testing.T) {
	original, had := os.LookupEnv("APP_ENV")
	if had {
		t.Cleanup(func() { _ = os.Setenv("APP_ENV", original) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv("APP_ENV") })
	}
	_ = os.Unsetenv("APP_ENV")

	_, r := setupWechatTestRouter(t)

	body := []byte(`{"student_id":"S100"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "dev login is disabled")
}
