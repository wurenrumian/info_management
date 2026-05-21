package certificates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"manage/internal/auth"
	"manage/internal/config"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrInvalidCertificateStage = errors.New("invalid certificate stage")
	ErrApprovalNotApproved     = errors.New("approval not approved")
	ErrInvalidVerificationCode = errors.New("invalid verification code")
)

type Service struct {
	approvalRepo *repo.ApprovalRepo
	templateRepo *repo.CertificateTemplateRepo
	recordRepo   *repo.CertificateRecordRepo
	documentRepo *repo.DocumentRepo
	renderer     Renderer
	uploadDir    string
}

func NewService(db *gorm.DB) *Service {
	return NewServiceWithRenderer(db, NewNoopRenderer())
}

func NewServiceWithRenderer(db *gorm.DB, renderer Renderer) *Service {
	if renderer == nil {
		renderer = NewNoopRenderer()
	}
	return &Service{
		approvalRepo: repo.NewApprovalRepo(db),
		templateRepo: repo.NewCertificateTemplateRepo(db),
		recordRepo:   repo.NewCertificateRecordRepo(db),
		documentRepo: repo.NewDocumentRepo(db),
		renderer:     renderer,
		uploadDir:    config.DocumentUploadDir(),
	}
}

type VerifyResult struct {
	RecordID         uint       `json:"record_id"`
	ApprovalID       uint       `json:"approval_id"`
	ApplicantID      uint       `json:"applicant_id"`
	ApprovalType     string     `json:"approval_type"`
	DocumentStage    string     `json:"document_stage"`
	CertificateNo    string     `json:"certificate_no"`
	VerificationCode string     `json:"verification_code"`
	Status           string     `json:"status"`
	GeneratedAt      *time.Time `json:"generated_at"`
}

func (s *Service) ListTemplates() ([]model.CertificateTemplate, error) {
	return s.templateRepo.ListAll()
}

func (s *Service) ToggleTemplate(id uint, active bool) (*model.CertificateTemplate, error) {
	item, err := s.templateRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if active {
		item.Status = model.CertificateTemplateStatusActive
	} else {
		item.Status = model.CertificateTemplateStatusInactive
	}
	if err := s.templateRepo.Save(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) RegenerateApplicationPDF(ctx context.Context, approvalID uint) (*model.CertificateRecord, error) {
	return s.GenerateApplicationPDF(ctx, approvalID)
}

func (s *Service) RegenerateApprovalCertificate(ctx context.Context, approvalID uint) (*model.CertificateRecord, error) {
	return s.GenerateApprovalCertificate(ctx, approvalID)
}

func (s *Service) GenerateApplicationPDF(ctx context.Context, approvalID uint) (*model.CertificateRecord, error) {
	approval, err := s.approvalRepo.GetByID(approvalID)
	if err != nil {
		return nil, err
	}
	tpl, err := s.templateRepo.GetActiveByApprovalTypeAndStage(approval.ApprovalType, model.CertificateDocumentStageApplication)
	if err != nil {
		return nil, err
	}
	payload := buildPayload(approval, tpl, nil)
	return s.persistRenderedRecord(ctx, approval, tpl, model.CertificateDocumentStageApplication, "", "", "", model.CertificateSealStatusNone, nil, payload)
}

func (s *Service) GenerateApprovalCertificate(ctx context.Context, approvalID uint) (*model.CertificateRecord, error) {
	approval, err := s.approvalRepo.GetByID(approvalID)
	if err != nil {
		return nil, err
	}
	if approval.Status != model.ApprovalStatusApproved {
		return nil, ErrApprovalNotApproved
	}
	tpl, err := s.templateRepo.GetActiveByApprovalTypeAndStage(approval.ApprovalType, model.CertificateDocumentStageApprovalCertificate)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	certificateNo, code, err := s.nextApprovalCertificateIdentifiers(approval, now)
	if err != nil {
		return nil, err
	}
	payload := buildPayload(approval, tpl, map[string]any{"certificate_no": certificateNo, "verification_code": code, "decided_at": approval.DecidedAt})
	return s.persistRenderedRecord(ctx, approval, tpl, model.CertificateDocumentStageApprovalCertificate, certificateNo, code, buildVerificationHash(code), model.CertificateSealStatusInternalSealApplied, &now, payload)
}

func (s *Service) ListMine(actor auth.Actor, approvalType string, limit, offset int) ([]model.CertificateRecord, int64, error) {
	return s.recordRepo.ListMine(actor.UserID, strings.TrimSpace(approvalType), limit, offset)
}

func (s *Service) Get(actor auth.Actor, id uint) (*model.CertificateRecord, error) {
	return s.recordRepo.GetByIDInScope(authz.BuildScope(actor), id)
}

func (s *Service) VerifyByCode(code string) (*VerifyResult, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvalidVerificationCode
	}

	record, err := s.recordRepo.GetByVerificationCode(code)
	if err != nil {
		return nil, err
	}
	approval, err := s.approvalRepo.GetByID(record.ApprovalID)
	if err != nil {
		return nil, err
	}

	return &VerifyResult{
		RecordID:         record.ID,
		ApprovalID:       record.ApprovalID,
		ApplicantID:      record.ApplicantID,
		ApprovalType:     approval.ApprovalType,
		DocumentStage:    record.DocumentStage,
		CertificateNo:    record.CertificateNo,
		VerificationCode: record.VerificationCode,
		Status:           record.Status,
		GeneratedAt:      record.GeneratedAt,
	}, nil
}

func (s *Service) Revoke(_ context.Context, id uint, reason string) (*model.CertificateRecord, error) {
	item, err := s.recordRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	item.Status = model.CertificateRecordStatusRevoked
	item.RevokedAt = &now
	if trimmed := strings.TrimSpace(reason); trimmed != "" {
		item.ErrorMessage = trimmed
	}
	if err := s.recordRepo.Save(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) persistRenderedRecord(ctx context.Context, approval *model.Approval, tpl *model.CertificateTemplate, stage, certificateNo, verificationCode, verificationHash, sealStatus string, sealAppliedAt *time.Time, payload map[string]any) (*model.CertificateRecord, error) {
	now := time.Now()
	record := &model.CertificateRecord{ApprovalID: approval.ID, ApplicantID: approval.ApplicantID, TemplateID: tpl.ID, DocumentStage: stage, CertificateNo: certificateNo, VerificationCode: verificationCode, VerificationHash: verificationHash, RenderedPayload: buildRenderedPayload(payload), DocumentID: 0, SealStatus: sealStatus, Status: model.CertificateRecordStatusGenerated, GeneratedAt: &now, SealAppliedAt: sealAppliedAt}
	rendered, err := s.renderer.Render(ctx, tpl.TemplatePath, payload)
	if err != nil {
		record.Status = model.CertificateRecordStatusFailed
		record.GeneratedAt = nil
		record.SealAppliedAt = nil
		record.ErrorMessage = err.Error()
		if err2 := s.recordRepo.Create(record); err2 != nil {
			return nil, err2
		}
		return nil, err
	}
	doc, err := s.createDocument(approval, stage, rendered, now)
	if err != nil {
		record.Status = model.CertificateRecordStatusFailed
		record.GeneratedAt = nil
		record.SealAppliedAt = nil
		record.ErrorMessage = err.Error()
		if err2 := s.recordRepo.Create(record); err2 != nil {
			return nil, err2
		}
		return nil, err
	}
	record.DocumentID = doc.ID
	if err := s.recordRepo.Create(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) createDocument(approval *model.Approval, stage string, content []byte, now time.Time) (*model.Document, error) {
	relPath := filepath.ToSlash(filepath.Join("certificates", fmt.Sprintf("%s_%d_%d.pdf", stage, approval.ID, now.UnixNano())))
	fullPath := filepath.Join(s.uploadDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil { return nil, err }
	if err := os.WriteFile(fullPath, content, 0o644); err != nil { return nil, err }
	doc := &model.Document{Title: fmt.Sprintf("approval_%d_%s.pdf", approval.ID, stage), FilePath: relPath, FileSize: int64(len(content)), ContentType: "application/pdf", UploaderID: approval.ApplicantID}
	if err := s.documentRepo.Create(doc); err != nil { return nil, err }
	return doc, nil
}

func (s *Service) nextApprovalCertificateIdentifiers(approval *model.Approval, now time.Time) (string, string, error) {
	records, err := s.recordRepo.ListByApprovalID(approval.ID)
	if err != nil {
		return "", "", err
	}
	count := 0
	for _, record := range records {
		if record.DocumentStage == model.CertificateDocumentStageApprovalCertificate {
			count++
		}
	}
	certificateNo := buildCertificateNo(approval.ApprovalType, approval.ID, now)
	code := buildVerificationCode(approval.ApprovalType, approval.ID, now)
	if count == 0 {
		return certificateNo, code, nil
	}
	suffix := "-R" + strconv.Itoa(count+1)
	return certificateNo + suffix, code + suffix, nil
}

func buildPayload(approval *model.Approval, tpl *model.CertificateTemplate, extra map[string]any) map[string]any {
	var formData any = map[string]any{}
	if len(approval.FormData) > 0 {
		var decoded map[string]any
		if err := json.Unmarshal(approval.FormData, &decoded); err == nil {
			formData = decoded
		}
	}
	payload := map[string]any{"approval_id": approval.ID, "applicant_id": approval.ApplicantID, "approval_type": approval.ApprovalType, "title": approval.Title, "status": approval.Status, "semester": approval.Semester, "submitted_at": approval.SubmittedAt, "decided_at": approval.DecidedAt, "form_data": formData, "template_code": tpl.Code, "template_version": tpl.TemplateVersion, "document_stage": tpl.DocumentStage}
	for k, v := range extra {
		payload[k] = v
	}
	return payload
}

func buildRenderedPayload(payload map[string]any) datatypes.JSON {
	b, _ := json.Marshal(payload)
	return datatypes.JSON(b)
}
