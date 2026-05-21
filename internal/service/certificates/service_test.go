package certificates_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"manage/internal/auth"
	"manage/internal/model"
	certificates "manage/internal/service/certificates"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCertificateService(t *testing.T) (*gorm.DB, *certificates.Service) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	t.Setenv("DOCUMENT_UPLOAD_DIR", t.TempDir())
	err = db.AutoMigrate(
		&model.User{},
		&model.Approval{},
		&model.Document{},
		&model.CertificateTemplate{},
		&model.CertificateRecord{},
	)
	require.NoError(t, err)

	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "student-1",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        300,
		StudentID: "T300",
		Name:      "teacher-1",
		Role:      model.RoleTeacher,
		ClassID:   1,
		Grade:     "2023",
	}).Error)

	now := time.Now()
	require.NoError(t, db.Create(&model.Approval{
		ID:           1,
		ApplicantID:  100,
		ApprovalType: model.ApprovalTypeLeave,
		Status:       model.ApprovalStatusPending,
		Title:        "leave-pending",
		Semester:     "2026-1",
		SubmittedAt:  now,
	}).Error)
	require.NoError(t, db.Create(&model.Approval{
		ID:           2,
		ApplicantID:  100,
		ApprovalType: model.ApprovalTypeLeave,
		Status:       model.ApprovalStatusApproved,
		Title:        "leave-approved",
		Semester:     "2026-1",
		SubmittedAt:  now,
		DecidedAt:    &now,
	}).Error)

	require.NoError(t, db.Create(&model.CertificateTemplate{
		ID:              1,
		Code:            "leave_application_pdf",
		Name:            "Leave Application",
		ApprovalType:    model.ApprovalTypeLeave,
		DocumentStage:   model.CertificateDocumentStageApplication,
		Status:          model.CertificateTemplateStatusActive,
		Renderer:        model.CertificateRendererTypst,
		TemplatePath:    "templates/certificates/leave_application.typ",
		TemplateVersion: "v1",
	}).Error)
	require.NoError(t, db.Create(&model.CertificateTemplate{
		ID:              2,
		Code:            "leave_approval_certificate",
		Name:            "Leave Approval Certificate",
		ApprovalType:    model.ApprovalTypeLeave,
		DocumentStage:   model.CertificateDocumentStageApprovalCertificate,
		Status:          model.CertificateTemplateStatusActive,
		Renderer:        model.CertificateRendererTypst,
		TemplatePath:    "templates/certificates/leave_approval_certificate.typ",
		TemplateVersion: "v1",
	}).Error)

	return db, certificates.NewService(db)
}

func TestGenerateApplicationPDF(t *testing.T) {
	db, svc := setupCertificateService(t)

	record, err := svc.GenerateApplicationPDF(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, model.CertificateDocumentStageApplication, record.DocumentStage)
	require.Equal(t, model.CertificateRecordStatusGenerated, record.Status)
	require.Equal(t, model.CertificateSealStatusNone, record.SealStatus)
	require.NotZero(t, record.DocumentID)

	var stored model.CertificateRecord
	require.NoError(t, db.First(&stored, record.ID).Error)
	require.Equal(t, uint(1), stored.ApprovalID)
	require.NotZero(t, stored.DocumentID)

	var doc model.Document
	require.NoError(t, db.First(&doc, stored.DocumentID).Error)
	require.Equal(t, "application/pdf", doc.ContentType)
}

func TestGenerateApprovalCertificate(t *testing.T) {
	_, svc := setupCertificateService(t)

	record, err := svc.GenerateApprovalCertificate(context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, model.CertificateDocumentStageApprovalCertificate, record.DocumentStage)
	require.NotEmpty(t, record.CertificateNo)
	require.NotEmpty(t, record.VerificationCode)
	require.NotEmpty(t, record.VerificationHash)
	require.Equal(t, model.CertificateSealStatusInternalSealApplied, record.SealStatus)
	require.NotZero(t, record.DocumentID)

	record2, err := svc.GenerateApprovalCertificate(context.Background(), 2)
	require.NoError(t, err)
	require.NotEqual(t, record.CertificateNo, record2.CertificateNo)
}

func TestGenerateApprovalCertificateRequiresApprovedStatus(t *testing.T) {
	_, svc := setupCertificateService(t)

	_, err := svc.GenerateApprovalCertificate(context.Background(), 1)
	require.Error(t, err)
	require.ErrorIs(t, err, certificates.ErrApprovalNotApproved)
}

func TestListMineAndGet(t *testing.T) {
	_, svc := setupCertificateService(t)

	_, err := svc.GenerateApplicationPDF(context.Background(), 1)
	require.NoError(t, err)

	actor := auth.Actor{
		UserID:  100,
		Role:    model.RoleStudent,
		ClassID: 1,
		Grade:   "2023",
	}

	list, total, err := svc.ListMine(actor, model.ApprovalTypeLeave, 20, 0)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, list, 1)

	got, err := svc.Get(actor, list[0].ID)
	require.NoError(t, err)
	require.Equal(t, list[0].ID, got.ID)
}

func TestVerifyByCode(t *testing.T) {
	_, svc := setupCertificateService(t)

	record, err := svc.GenerateApprovalCertificate(context.Background(), 2)
	require.NoError(t, err)

	out, err := svc.VerifyByCode(record.VerificationCode)
	require.NoError(t, err)
	require.Equal(t, record.ID, out.RecordID)
	require.Equal(t, record.CertificateNo, out.CertificateNo)
	require.Equal(t, model.ApprovalTypeLeave, out.ApprovalType)
}

func TestToggleTemplateAndRevoke(t *testing.T) {
	db, svc := setupCertificateService(t)

	tpl, err := svc.ToggleTemplate(1, false)
	require.NoError(t, err)
	require.Equal(t, model.CertificateTemplateStatusInactive, tpl.Status)
	_, err = svc.GenerateApplicationPDF(context.Background(), 1)
	require.Error(t, err)

	record, err := svc.GenerateApprovalCertificate(context.Background(), 2)
	require.NoError(t, err)

	revoked, err := svc.Revoke(context.Background(), record.ID, "manual revoke")
	require.NoError(t, err)
	require.Equal(t, model.CertificateRecordStatusRevoked, revoked.Status)
	require.NotNil(t, revoked.RevokedAt)
	
	var stored model.CertificateRecord
	require.NoError(t, db.First(&stored, record.ID).Error)
	require.Equal(t, model.CertificateRecordStatusRevoked, stored.Status)
	require.Equal(t, "manual revoke", stored.ErrorMessage)
}

func TestGenerateApplicationPDFRecordsFailureOnRenderError(t *testing.T) {
	db, _ := setupCertificateService(t)
	svc := certificates.NewServiceWithRenderer(db, certificates.FailingRenderer{Err: errors.New("render failed")})

	_, err := svc.GenerateApplicationPDF(context.Background(), 1)
	require.Error(t, err)

	var stored model.CertificateRecord
	require.NoError(t, db.Order("id desc").First(&stored).Error)
	require.Equal(t, model.CertificateRecordStatusFailed, stored.Status)
	require.Equal(t, "render failed", stored.ErrorMessage)
	require.Equal(t, model.CertificateDocumentStageApplication, stored.DocumentStage)
}
