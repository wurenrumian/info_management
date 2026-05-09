package approvals_test

import (
	"encoding/json"
	"testing"

	"manage/internal/auth"
	"manage/internal/model"
	approvals "manage/internal/service/approvals"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupApprovalService(t *testing.T) (*gorm.DB, *approvals.Service) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Approval{}, &model.ApprovalAction{}))
	require.NoError(t, db.Create(&model.User{ID: 100, StudentID: "S100", Name: "stu", Role: model.RoleStudent, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 300, StudentID: "T300", Name: "tea", Role: model.RoleTeacher, ClassID: 1, Grade: "2023"}).Error)
	return db, approvals.NewService(db)
}

func TestCreateLeaveAndWithdraw(t *testing.T) {
	db, svc := setupApprovalService(t)
	actor := auth.Actor{UserID: 100, Role: model.RoleStudent, ClassID: 1, Grade: "2023"}

	form := map[string]any{
		"reason":        "回家",
		"start_at":      "2026-05-01T09:00:00+08:00",
		"end_at":        "2026-05-02T18:00:00+08:00",
		"contact_phone": "13800000000",
	}
	raw, _ := json.Marshal(form)

	item, err := svc.Create(actor, approvals.CreateRequest{
		ApprovalType: model.ApprovalTypeLeave,
		Title:        "请假申请",
		FormData:     raw,
	})
	require.NoError(t, err)
	require.Equal(t, model.ApprovalStatusPending, item.Status)
	require.Equal(t, model.ApprovalStepReview, item.CurrentStep)

	err = svc.Withdraw(actor, item.ID)
	require.NoError(t, err)

	var got model.Approval
	require.NoError(t, db.First(&got, item.ID).Error)
	require.Equal(t, model.ApprovalStatusWithdrawn, got.Status)
}

func TestTeacherReviewApprove(t *testing.T) {
	db, svc := setupApprovalService(t)
	student := auth.Actor{UserID: 100, Role: model.RoleStudent, ClassID: 1, Grade: "2023"}
	teacher := auth.Actor{UserID: 300, Role: model.RoleTeacher, ClassID: 1, Grade: "2023"}

	raw := []byte(`{"reason":"回家","start_at":"2026-05-01T09:00:00+08:00","end_at":"2026-05-02T18:00:00+08:00","contact_phone":"13800000000"}`)
	item, err := svc.Create(student, approvals.CreateRequest{
		ApprovalType: model.ApprovalTypeLeave,
		Title:        "请假申请2",
		FormData:     raw,
	})
	require.NoError(t, err)

	err = svc.Review(teacher, item.ID, approvals.ReviewRequest{Action: "approve", Comment: "通过"})
	require.NoError(t, err)

	var got model.Approval
	require.NoError(t, db.First(&got, item.ID).Error)
	require.Equal(t, model.ApprovalStatusApproved, got.Status)
	require.NotNil(t, got.DecidedAt)
}
