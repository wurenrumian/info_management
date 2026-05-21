package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"manage/internal/http/router"
	"manage/internal/model"
	"manage/internal/testutil"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCertificateHandlerRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	t.Setenv("DOCUMENT_UPLOAD_DIR", t.TempDir())

	err = db.AutoMigrate(
		&model.Class{},
		&model.User{},
		&model.AdminLog{},
		&model.Announcement{},
		&model.NotificationTemplate{},
		&model.NotificationLog{},
		&model.UserSubscribe{},
		&model.Approval{},
		&model.ApprovalAction{},
		&model.Document{},
		&model.CertificateTemplate{},
		&model.CertificateRecord{},
	)
	require.NoError(t, err)

	require.NoError(t, db.Create(&model.Class{
		ID:        1,
		ClassName: "信管1班",
		Grade:     "2023",
		Major:     "信息管理",
	}).Error)

	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "学生A",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
	}).Error)

	require.NoError(t, db.Create(&model.User{
		ID:        300,
		StudentID: "T300",
		Name:      "老师A",
		Role:      model.RoleTeacher,
		ClassID:   1,
		Grade:     "2023",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        200,
		StudentID: "C200",
		Name:      "干部A",
		Role:      model.RoleCadre,
		ClassID:   1,
		Grade:     "2023",
	}).Error)

	now := time.Now()
	require.NoError(t, db.Create(&model.Approval{
		ID:           1,
		ApplicantID:  100,
		ApprovalType: model.ApprovalTypeLeave,
		Status:       model.ApprovalStatusApproved,
		Title:        "请假申请",
		Semester:     "2026-1",
		SubmittedAt:  now,
		DecidedAt:    &now,
	}).Error)

	require.NoError(t, db.Create(&model.CertificateTemplate{
		ID:              1,
		Code:            "leave_approval_certificate",
		Name:            "Leave Certificate",
		ApprovalType:    model.ApprovalTypeLeave,
		DocumentStage:   model.CertificateDocumentStageApprovalCertificate,
		Status:          model.CertificateTemplateStatusActive,
		Renderer:        model.CertificateRendererTypst,
		TemplatePath:    "templates/certificates/leave_approval_certificate.typ",
		TemplateVersion: "v1",
	}).Error)
	require.NoError(t, db.Create(&model.CertificateTemplate{
		ID:              2,
		Code:            "leave_application_pdf",
		Name:            "Leave Application",
		ApprovalType:    model.ApprovalTypeLeave,
		DocumentStage:   model.CertificateDocumentStageApplication,
		Status:          model.CertificateTemplateStatusActive,
		Renderer:        model.CertificateRendererTypst,
		TemplatePath:    "templates/certificates/leave_application.typ",
		TemplateVersion: "v1",
	}).Error)

	require.NoError(t, db.Create(&model.CertificateRecord{
		ID:               1,
		ApprovalID:       1,
		ApplicantID:      100,
		TemplateID:       1,
		DocumentStage:    model.CertificateDocumentStageApprovalCertificate,
		CertificateNo:    "LEAVE-20260513-000001",
		VerificationCode: "VERIFY-CODE-001",
		VerificationHash: "hash-001",
		SealStatus:       model.CertificateSealStatusInternalSealApplied,
		Status:           model.CertificateRecordStatusGenerated,
		GeneratedAt:      &now,
	}).Error)

	return db, router.New(db)
}

func TestCertificateListMine(t *testing.T) {
	_, r := setupCertificateHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/certificates/me?limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"certificate_no":"LEAVE-20260513-000001"`)
}

func TestCertificateGetForbiddenWithoutToken(t *testing.T) {
	_, r := setupCertificateHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/certificates/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCertificateGetMine(t *testing.T) {
	_, r := setupCertificateHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/certificates/1", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"verification_code":"VERIFY-CODE-001"`)
}

func TestCertificateVerifyPublic(t *testing.T) {
	_, r := setupCertificateHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/certificates/verify?code=VERIFY-CODE-001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"certificate_no":"LEAVE-20260513-000001"`)
}

func TestCertificateAdminTemplates(t *testing.T) {
	_, r := setupCertificateHandlerRouter(t)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/certificates/templates", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(300, model.RoleTeacher, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"code":"leave_application_pdf"`)
}

func TestCertificateAdminRegenerateApplicationPDF(t *testing.T) {
	_, r := setupCertificateHandlerRouter(t)
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/approvals/1/application-pdf/regenerate", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(300, model.RoleTeacher, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"document_stage":"application"`)
	require.Contains(t, w.Body.String(), `"document_id":`)
}

func TestCertificateAdminRevoke(t *testing.T) {
	_, r := setupCertificateHandlerRouter(t)
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/certificates/1/revoke", strings.NewReader(`{"reason":"manual revoke"}`))
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(300, model.RoleTeacher, 1, "2023"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"status":"revoked"`)
}

func TestCertificateAdminRegenerateApprovalCertificate(t *testing.T) {
	_, r := setupCertificateHandlerRouter(t)
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/approvals/1/certificate/regenerate", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(300, model.RoleTeacher, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"document_stage":"approval_certificate"`)
	require.Contains(t, w.Body.String(), `"document_id":`)
}

func TestCertificateAdminRegenerateApprovalCertificateForbiddenForCadre(t *testing.T) {
	_, r := setupCertificateHandlerRouter(t)
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/approvals/1/certificate/regenerate", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(200, model.RoleCadre, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusForbidden, w.Code)
}
