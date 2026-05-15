package repo_test

import (
	"testing"
	"time"

	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCertificateRepoDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.User{},
		&model.Approval{},
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
		ID:        101,
		StudentID: "S101",
		Name:      "student-2",
		Role:      model.RoleStudent,
		ClassID:   2,
		Grade:     "2022",
	}).Error)

	now := time.Now()
	require.NoError(t, db.Create(&model.Approval{
		ID:           1,
		ApplicantID:  100,
		ApprovalType: model.ApprovalTypeLeave,
		Status:       model.ApprovalStatusApproved,
		Title:        "leave-1",
		Semester:     "2026-1",
		SubmittedAt:  now,
	}).Error)
	require.NoError(t, db.Create(&model.Approval{
		ID:           2,
		ApplicantID:  101,
		ApprovalType: model.ApprovalTypeBudget,
		Status:       model.ApprovalStatusApproved,
		Title:        "budget-1",
		Semester:     "2026-1",
		SubmittedAt:  now,
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

	return db
}

func TestCertificateRecordRepoGetByIDInScope(t *testing.T) {
	db := setupCertificateRepoDB(t)
	r := repo.NewCertificateRecordRepo(db)

	require.NoError(t, r.Create(&model.CertificateRecord{
		ID:               1,
		ApprovalID:       1,
		ApplicantID:      100,
		TemplateID:       1,
		DocumentStage:    model.CertificateDocumentStageApplication,
		CertificateNo:    "CERT-001",
		VerificationCode: "VERIFY-001",
		SealStatus:       model.CertificateSealStatusNone,
		Status:           model.CertificateRecordStatusGenerated,
	}))
	require.NoError(t, r.Create(&model.CertificateRecord{
		ID:               2,
		ApprovalID:       2,
		ApplicantID:      101,
		TemplateID:       1,
		DocumentStage:    model.CertificateDocumentStageApplication,
		CertificateNo:    "CERT-002",
		VerificationCode: "VERIFY-002",
		SealStatus:       model.CertificateSealStatusNone,
		Status:           model.CertificateRecordStatusGenerated,
	}))

	_, err := r.GetByIDInScope(authz.Scope{SelfUserID: 100}, 1)
	require.NoError(t, err)

	_, err = r.GetByIDInScope(authz.Scope{SelfUserID: 100}, 2)
	require.Error(t, err)
}

func TestCertificateRecordRepoListMine(t *testing.T) {
	db := setupCertificateRepoDB(t)
	r := repo.NewCertificateRecordRepo(db)

	require.NoError(t, r.Create(&model.CertificateRecord{
		ApprovalID:       1,
		ApplicantID:      100,
		TemplateID:       1,
		DocumentStage:    model.CertificateDocumentStageApplication,
		CertificateNo:    "CERT-101",
		VerificationCode: "VERIFY-101",
		SealStatus:       model.CertificateSealStatusNone,
		Status:           model.CertificateRecordStatusGenerated,
	}))
	require.NoError(t, r.Create(&model.CertificateRecord{
		ApprovalID:       2,
		ApplicantID:      101,
		TemplateID:       1,
		DocumentStage:    model.CertificateDocumentStageApplication,
		CertificateNo:    "CERT-102",
		VerificationCode: "VERIFY-102",
		SealStatus:       model.CertificateSealStatusNone,
		Status:           model.CertificateRecordStatusGenerated,
	}))

	list, total, err := r.ListMine(100, model.ApprovalTypeLeave, 20, 0)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, list, 1)
	require.EqualValues(t, 100, list[0].ApplicantID)
}
