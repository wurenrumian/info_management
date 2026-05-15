package approvals_test

import (
	"encoding/json"
	"testing"
	"time"

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
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Approval{}, &model.ApprovalAction{}, &model.CertificateTemplate{}, &model.CertificateRecord{}))
	require.NoError(t, db.Create(&model.User{ID: 100, StudentID: "S100", Name: "stu", Role: model.RoleStudent, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 300, StudentID: "T300", Name: "tea", Role: model.RoleTeacher, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.CertificateTemplate{ID: 1, Code: "leave_application_pdf", Name: "Leave Application", ApprovalType: model.ApprovalTypeLeave, DocumentStage: model.CertificateDocumentStageApplication, Status: model.CertificateTemplateStatusActive, Renderer: model.CertificateRendererTypst, TemplatePath: "templates/certificates/leave_application.typ", TemplateVersion: "v1"}).Error)
	require.NoError(t, db.Create(&model.CertificateTemplate{ID: 2, Code: "leave_approval_certificate", Name: "Leave Approval Certificate", ApprovalType: model.ApprovalTypeLeave, DocumentStage: model.CertificateDocumentStageApprovalCertificate, Status: model.CertificateTemplateStatusActive, Renderer: model.CertificateRendererTypst, TemplatePath: "templates/certificates/leave_approval_certificate.typ", TemplateVersion: "v1"}).Error)
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
	var records []model.CertificateRecord
	require.NoError(t, db.Where("approval_id = ?", item.ID).Order("id asc").Find(&records).Error)
	require.Len(t, records, 1)
	require.Equal(t, model.CertificateDocumentStageApplication, records[0].DocumentStage)
}

func TestCreateBudgetWritesSubmitAction(t *testing.T) {
	db, svc := setupApprovalService(t)
	actor := auth.Actor{UserID: 100, Role: model.RoleStudent, ClassID: 1, Grade: "2023"}

	raw := []byte(`{"activity_name":"班级团日活动","purpose":"活动物料与场地费用","budget_amount":1200}`)
	item, err := svc.Create(actor, approvals.CreateRequest{
		ApprovalType: model.ApprovalTypeBudget,
		Title:        "预算申请",
		FormData:     raw,
	})
	require.NoError(t, err)
	require.Equal(t, model.ApprovalStatusPending, item.Status)
	require.Equal(t, model.ApprovalStepBudgetReview, item.CurrentStep)
	require.NotNil(t, item.DueAt)

	var actions []model.ApprovalAction
	require.NoError(t, db.Where("approval_id = ?", item.ID).Find(&actions).Error)
	require.Len(t, actions, 1)
	require.Equal(t, model.ApprovalActionSubmit, actions[0].ActionType)
	require.Equal(t, actor.UserID, actions[0].OperatorID)
	require.Equal(t, model.ApprovalStatusPending, actions[0].ToStatus)
}

func TestCreateRejectsInvalidFormData(t *testing.T) {
	_, svc := setupApprovalService(t)
	actor := auth.Actor{UserID: 100, Role: model.RoleStudent, ClassID: 1, Grade: "2023"}

	_, err := svc.Create(actor, approvals.CreateRequest{
		ApprovalType: model.ApprovalTypeLeave,
		Title:        "缺字段请假",
		FormData:     []byte(`{"reason":"回家"}`),
	})
	require.ErrorIs(t, err, approvals.ErrInvalidFormData)
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
	var records []model.CertificateRecord
	require.NoError(t, db.Where("approval_id = ?", item.ID).Order("id asc").Find(&records).Error)
	require.Len(t, records, 2)
	require.Equal(t, model.CertificateDocumentStageApplication, records[0].DocumentStage)
	require.Equal(t, model.CertificateDocumentStageApprovalCertificate, records[1].DocumentStage)
}

func TestGetIncludesCertificateRecords(t *testing.T) {
	db, svc := setupApprovalService(t)
	actor := auth.Actor{UserID: 100, Role: model.RoleStudent, ClassID: 1, Grade: "2023"}
	now := time.Now()

	require.NoError(t, db.Create(&model.Approval{
		ID:           1,
		ApplicantID:  100,
		ApprovalType: model.ApprovalTypeLeave,
		Status:       model.ApprovalStatusApproved,
		Title:        "请假申请-详情",
		Semester:     "2026-1",
		SubmittedAt:  now,
		DecidedAt:    &now,
	}).Error)
	require.NoError(t, db.Create(&model.CertificateRecord{
		ApprovalID: 1, ApplicantID: 100, TemplateID: 1, DocumentStage: model.CertificateDocumentStageApplication,
		CertificateNo: "CERT-1", VerificationCode: "VERIFY-1", SealStatus: model.CertificateSealStatusNone,
		Status: model.CertificateRecordStatusGenerated, GeneratedAt: &now,
	}).Error)

	detail, err := svc.Get(actor, 1)
	require.NoError(t, err)
	require.Len(t, detail.CertificateRecords, 1)
	require.Equal(t, "CERT-1", detail.CertificateRecords[0].CertificateNo)
}

func TestReviewedApprovalCannotBeReviewedAgain(t *testing.T) {
	_, svc := setupApprovalService(t)
	student := auth.Actor{UserID: 100, Role: model.RoleStudent, ClassID: 1, Grade: "2023"}
	teacher := auth.Actor{UserID: 300, Role: model.RoleTeacher, ClassID: 1, Grade: "2023"}

	raw := []byte(`{"reason":"回家","start_at":"2026-05-01T09:00:00+08:00","end_at":"2026-05-02T18:00:00+08:00","contact_phone":"13800000000"}`)
	item, err := svc.Create(student, approvals.CreateRequest{
		ApprovalType: model.ApprovalTypeLeave,
		Title:        "请假申请",
		FormData:     raw,
	})
	require.NoError(t, err)

	require.NoError(t, svc.Review(teacher, item.ID, approvals.ReviewRequest{Action: "approve"}))
	err = svc.Review(teacher, item.ID, approvals.ReviewRequest{Action: "reject"})
	require.ErrorIs(t, err, approvals.ErrInvalidState)
}

func TestAssignUpdatesApproverAndWritesAction(t *testing.T) {
	db, svc := setupApprovalService(t)
	student := auth.Actor{UserID: 100, Role: model.RoleStudent, ClassID: 1, Grade: "2023"}
	teacher := auth.Actor{UserID: 300, Role: model.RoleTeacher, ClassID: 1, Grade: "2023"}

	raw := []byte(`{"reason":"回家","start_at":"2026-05-01T09:00:00+08:00","end_at":"2026-05-02T18:00:00+08:00","contact_phone":"13800000000"}`)
	item, err := svc.Create(student, approvals.CreateRequest{
		ApprovalType: model.ApprovalTypeLeave,
		Title:        "请假申请",
		FormData:     raw,
	})
	require.NoError(t, err)

	require.NoError(t, svc.Assign(teacher, item.ID, approvals.AssignRequest{ApproverID: 300, Comment: "转交老师"}))

	var got model.Approval
	require.NoError(t, db.First(&got, item.ID).Error)
	require.NotNil(t, got.CurrentApproverID)
	require.Equal(t, uint(300), *got.CurrentApproverID)

	var action model.ApprovalAction
	require.NoError(t, db.Where("approval_id = ? AND action_type = ?", item.ID, model.ApprovalActionAssign).First(&action).Error)
	require.Equal(t, teacher.UserID, action.OperatorID)
	require.Equal(t, "转交老师", action.Comment)
}

func TestScanAndRemindOverdueWritesRemindAction(t *testing.T) {
	db, svc := setupApprovalService(t)
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	require.NoError(t, db.Create(&model.Approval{
		ApplicantID: 100, ApprovalType: model.ApprovalTypeLeave, Status: model.ApprovalStatusPending,
		CurrentStep: model.ApprovalStepReview, Title: "超时请假", Semester: "2026-1", SubmittedAt: now, DueAt: &past,
	}).Error)
	require.NoError(t, db.Create(&model.Approval{
		ApplicantID: 100, ApprovalType: model.ApprovalTypeLeave, Status: model.ApprovalStatusPending,
		CurrentStep: model.ApprovalStepReview, Title: "未超时请假", Semester: "2026-1", SubmittedAt: now, DueAt: &future,
	}).Error)

	out, err := svc.ScanAndRemindOverdue(t.Context(), now)
	require.NoError(t, err)
	require.Equal(t, 1, out.Scanned)
	require.Equal(t, 1, out.Reminded)

	var actions []model.ApprovalAction
	require.NoError(t, db.Where("action_type = ?", model.ApprovalActionRemind).Find(&actions).Error)
	require.Len(t, actions, 1)
	require.Equal(t, uint(0), actions[0].OperatorID)
}
