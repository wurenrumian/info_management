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

func setupAdminLogHandlerTestRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AdminLog{}, &model.User{}, &model.Class{}))

	require.NoError(t, db.Create(&model.AdminLog{
		AdminID:    200,
		Action:     "knowledge.create",
		TargetType: "knowledge",
		TargetID:   1,
		IPAddress:  "10.0.0.1",
	}).Error)
	require.NoError(t, db.Create(&model.AdminLog{
		AdminID:    200,
		Action:     "user.patch",
		TargetType: "user",
		TargetID:   100,
		IPAddress:  "10.0.0.2",
	}).Error)

	r := router.New(db)
	return db, r
}

func TestAdminLogsForbiddenForTeacher(t *testing.T) {
	_, r := setupAdminLogHandlerTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/logs", nil)
	token := testutil.GenerateTestToken(300, 3, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminLogsForbiddenForStudent(t *testing.T) {
	_, r := setupAdminLogHandlerTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/logs", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminLogsForbiddenForCadre(t *testing.T) {
	_, r := setupAdminLogHandlerTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/logs", nil)
	token := testutil.GenerateTestToken(200, 2, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminLogsReturnsLogsForSuperAdmin(t *testing.T) {
	_, r := setupAdminLogHandlerTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/logs", nil)
	token := testutil.GenerateTestToken(999, 4, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "knowledge.create")
	require.Contains(t, w.Body.String(), "user.patch")
}
