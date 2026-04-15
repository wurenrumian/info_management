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

func setupAnnouncementHandlerRouter(t *testing.T) http.Handler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Class{},
		&model.User{},
		&model.AdminLog{},
		&model.Announcement{},
		&model.NotificationTemplate{},
		&model.NotificationLog{},
		&model.UserSubscribe{},
	))
	return router.New(db)
}

func TestAdminAnnouncementsListForbiddenForStudent(t *testing.T) {
	r := setupAnnouncementHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/announcements?status=draft&limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "forbidden")
}
