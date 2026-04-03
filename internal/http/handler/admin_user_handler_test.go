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
	"manage/internal/http/router"
	"manage/internal/model"
	"manage/internal/testutil"
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
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
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
	token := testutil.GenerateTestToken(200, 2, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "S100")
	require.NotContains(t, w.Body.String(), "S101")

	var resp struct {
		Total int64 `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, int64(2), resp.Total)
}

func TestPatchUserWritesAdminLog(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	body := []byte(`{"name":"new-name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/100", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(999, 4, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var logs []model.AdminLog
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, uint(999), logs[0].AdminID)
	require.Equal(t, uint(100), logs[0].TargetID)
}
