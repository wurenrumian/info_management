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

func setupMeTestRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Class{}))

	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "张三",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        101,
		StudentID: "S101",
		Name:      "李四",
		Role:      model.RoleStudent,
		ClassID:   2,
		Grade:     "2023",
	}).Error)

	r := router.New(db)
	return db, r
}

func TestGetMeReturnsSelf(t *testing.T) {
	_, r := setupMeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "S100")
	require.NotContains(t, w.Body.String(), "S101")
}

func TestGetMeForbiddenForUnknownRole(t *testing.T) {
	_, r := setupMeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	token := testutil.GenerateTestToken(999, 0, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetMeNotFound(t *testing.T) {
	_, r := setupMeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	token := testutil.GenerateTestToken(999, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "user not found")
}
