package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/model"
	"manage/internal/http/router"
)

func setupTestRouter(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}, &model.AdminLog{}))
	require.NoError(t, db.Create(&model.Class{ID: 1, ClassName: "C1", Grade: "2023", Major: "CS"}).Error)
	require.NoError(t, db.Create(&model.Class{ID: 2, ClassName: "C2", Grade: "2023", Major: "CS"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 100, StudentID: "S100", Name: "u100", Role: model.RoleStudent, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 101, StudentID: "S101", Name: "u101", Role: model.RoleStudent, ClassID: 2, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 999, StudentID: "A999", Name: "admin", Role: model.RoleSuperAdmin, ClassID: 1, Grade: "2023"}).Error)
	_ = router.New(db)
	return db
}

func TestGetMeOnlyReturnsSelf(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("X-User-Id", "100")
	req.Header.Set("X-User-Role", "1")
	req.Header.Set("X-User-Class-Id", "1")
	req.Header.Set("X-User-Grade", "2023")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "S100")
	require.NotContains(t, w.Body.String(), "S101")
}

func TestAdminUsersListRespectsScope(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("X-User-Id", "200")
	req.Header.Set("X-User-Role", "2")
	req.Header.Set("X-User-Class-Id", "1")
	req.Header.Set("X-User-Grade", "2023")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "S100")
	require.NotContains(t, w.Body.String(), "S101")
}

func TestPatchUserWritesAdminLog(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	body := []byte(`{"name":"new-name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/100", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "999")
	req.Header.Set("X-User-Role", "4")
	req.Header.Set("X-User-Class-Id", "1")
	req.Header.Set("X-User-Grade", "2023")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var logs []model.AdminLog
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, uint(999), logs[0].AdminID)
	require.Equal(t, uint(100), logs[0].TargetID)
}

