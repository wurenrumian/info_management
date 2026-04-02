package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/http/router"
	"manage/internal/model"
	"manage/internal/testutil"
)

func setupNotificationRouterTest(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Class{}, &model.AdminLog{},
		&model.KnowledgeItem{}, &model.Document{},
		&model.NotificationTemplate{}, &model.NotificationLog{},
	))
	r := router.New(db)
	return db, r
}

func TestNotificationTemplatesForbiddenForStudent(t *testing.T) {
	_, r := setupNotificationRouterTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/notification/templates", nil)
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestNotificationLogsForbiddenForStudent(t *testing.T) {
	_, r := setupNotificationRouterTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/notification/logs", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestNotificationGetTemplateForbiddenForStudent(t *testing.T) {
	_, r := setupNotificationRouterTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/notification/templates/test", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}
