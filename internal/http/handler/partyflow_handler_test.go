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

func setupPartyflowHandlerRouter(t *testing.T) http.Handler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Class{},
		&model.User{},
		&model.AdminLog{},
		&model.PartyflowStatus{},
		&model.PartyflowEvent{},
		&model.PartyflowReminderRule{},
	))

	require.NoError(t, db.Create(&model.Class{ID: 1, ClassName: "信管1班", Grade: "2023", Major: "信息管理"}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "张三",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
		Major:     "信息管理",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        999,
		StudentID: "A999",
		Name:      "管理员",
		Role:      model.RoleSuperAdmin,
		ClassID:   1,
		Grade:     "2023",
		Major:     "信息管理",
	}).Error)
	return router.New(db)
}

func TestPartyflowAdminListForbiddenForStudent(t *testing.T) {
	r := setupPartyflowHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/partyflow/statuses?limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "forbidden")
}

func TestPartyflowListRulesForAdmin(t *testing.T) {
	r := setupPartyflowHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/partyflow/reminder-rules?org_type=party&enabled=true", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "party_activist_report_every_90d")
}
