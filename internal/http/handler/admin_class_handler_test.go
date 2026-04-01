package handler_test

import (
	"bytes"
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

func setupClassTestRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}, &model.AdminLog{}))

	require.NoError(t, db.Create(&model.Class{ID: 1, ClassName: "C1", Grade: "2023", Major: "CS"}).Error)
	require.NoError(t, db.Create(&model.Class{ID: 2, ClassName: "C2", Grade: "2022", Major: "SE"}).Error)
	require.NoError(t, db.Create(&model.AdminLog{AdminID: 999, Action: "seed", TargetType: "class", TargetID: 1, IPAddress: "127.0.0.1"}).Error)

	r := router.New(db)
	return db, r
}

func TestClassCreateOnlySuperAdmin(t *testing.T) {
	_, r := setupClassTestRouter(t)
	body := []byte(`{"class_name":"N1","grade":"2024","major":"CS"}`)

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/classes", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	token1 := testutil.GenerateTestToken(300, 3, 1, "2023")
	req1.Header.Set("Authorization", "Bearer "+token1)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusForbidden, w1.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/logs", nil)
	req2.Header.Set("Content-Type", "application/json")
	token2 := testutil.GenerateTestToken(999, 4, 1, "2023")
	req2.Header.Set("Authorization", "Bearer "+token2)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	require.Contains(t, w2.Body.String(), "seed")
}

func TestClassListRespectsScope(t *testing.T) {
	_, r := setupClassTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/classes", nil)
	token := testutil.GenerateTestToken(300, 3, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "C1")
	require.NotContains(t, w.Body.String(), "C2")
}

func TestAdminLogsOnlySuperAdmin(t *testing.T) {
	_, r := setupClassTestRouter(t)

	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/logs", nil)
	token1 := testutil.GenerateTestToken(300, 3, 1, "2023")
	req1.Header.Set("Authorization", "Bearer "+token1)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusForbidden, w1.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/logs", nil)
	token2 := testutil.GenerateTestToken(999, 4, 1, "2023")
	req2.Header.Set("Authorization", "Bearer "+token2)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	require.Contains(t, w2.Body.String(), "seed")
}
