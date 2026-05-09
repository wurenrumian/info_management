package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"manage/internal/http/router"
	"manage/internal/model"
	"manage/internal/testutil"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupApprovalHandlerRouter(t *testing.T) (*gorm.DB, http.Handler) {
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
		&model.Approval{},
		&model.ApprovalAction{},
	))
	require.NoError(t, db.Create(&model.Class{ID: 1, ClassName: "信管1班", Grade: "2023", Major: "信息管理"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 100, StudentID: "S100", Name: "学生", Role: model.RoleStudent, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 200, StudentID: "C200", Name: "团干部", Role: model.RoleCadre, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 300, StudentID: "T300", Name: "老师", Role: model.RoleTeacher, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 999, StudentID: "A999", Name: "超管", Role: model.RoleSuperAdmin, ClassID: 1, Grade: "2023"}).Error)
	return db, router.New(db)
}

func TestApprovalCreateAndListMine(t *testing.T) {
	_, r := setupApprovalHandlerRouter(t)

	body := []byte(`{
		"approval_type":"leave",
		"title":"五一请假",
		"form_data":{"reason":"回家","start_at":"2026-05-01T09:00:00+08:00","end_at":"2026-05-02T18:00:00+08:00","contact_phone":"13800000000"}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"status":"pending"`)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/approvals/me?limit=20&offset=0", nil)
	req2.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	require.Contains(t, w2.Body.String(), "五一请假")
}

func TestCadreReviewForbidden(t *testing.T) {
	db, r := setupApprovalHandlerRouter(t)
	require.NoError(t, db.Create(&model.Approval{
		ApplicantID: 100, ApprovalType: model.ApprovalTypeLeave, Status: model.ApprovalStatusPending,
		CurrentStep: "review", Title: "待审批", Semester: "2026-1",
	}).Error)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/approvals/1/review", bytes.NewReader([]byte(`{"action":"approve"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(200, model.RoleCadre, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestTeacherReviewSuccess(t *testing.T) {
	db, r := setupApprovalHandlerRouter(t)
	require.NoError(t, db.Create(&model.Approval{
		ApplicantID: 100, ApprovalType: model.ApprovalTypeLeave, Status: model.ApprovalStatusPending,
		CurrentStep: "review", Title: "待审批", Semester: "2026-1",
	}).Error)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/approvals/1/review", bytes.NewReader([]byte(`{"action":"approve","comment":"通过"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(300, model.RoleTeacher, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var got model.Approval
	require.NoError(t, db.First(&got, 1).Error)
	require.Equal(t, model.ApprovalStatusApproved, got.Status)
}
