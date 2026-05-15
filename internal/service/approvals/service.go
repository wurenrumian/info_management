package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/repo"
	approvalscert "manage/internal/service/certificates"
	"manage/internal/service/authz"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrInvalidApprovalType = errors.New("invalid approval_type")
	ErrInvalidFormData     = errors.New("invalid form_data")
	ErrInvalidState        = errors.New("invalid approval state")
	ErrForbidden           = errors.New("forbidden")
)

type Service struct {
	approvalRepo          *repo.ApprovalRepo
	actionRepo            *repo.ApprovalActionRepo
	certificateRecordRepo *repo.CertificateRecordRepo
	certificateService    *approvalscert.Service
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		approvalRepo:          repo.NewApprovalRepo(db),
		actionRepo:            repo.NewApprovalActionRepo(db),
		certificateRecordRepo: repo.NewCertificateRecordRepo(db),
		certificateService:    approvalscert.NewService(db),
	}
}

type CreateRequest struct {
	ApprovalType      string          `json:"approval_type"`
	Title             string          `json:"title"`
	FormData          json.RawMessage `json:"form_data"`
	AttachmentFileIDs []uint          `json:"attachment_file_ids"`
	TemplateFileID    *uint           `json:"template_file_id"`
	Semester          string          `json:"semester"`
}

type ReviewRequest struct {
	Action  string `json:"action"`
	Comment string `json:"comment"`
}

type AssignRequest struct {
	ApproverID uint   `json:"approver_id"`
	Comment    string `json:"comment"`
}

type ListAdminRequest struct {
	Status       string
	ApprovalType string
	Limit        int
	Offset       int
}

type ApprovalDetail struct {
	Approval           *model.Approval          `json:"approval"`
	Actions            []model.ApprovalAction   `json:"actions"`
	CertificateRecords []model.CertificateRecord `json:"certificate_records"`
}

func (s *Service) Create(actor auth.Actor, req CreateRequest) (*model.Approval, error) {
	tp := strings.TrimSpace(req.ApprovalType)
	if tp != model.ApprovalTypeLeave && tp != model.ApprovalTypeBudget {
		return nil, ErrInvalidApprovalType
	}
	if strings.TrimSpace(req.Title) == "" || len(req.FormData) == 0 {
		return nil, ErrInvalidFormData
	}
	if err := validateForm(tp, req.FormData); err != nil {
		return nil, err
	}
	now := time.Now()
	step, due := defaultStepAndDue(tp, now)
	item := &model.Approval{
		ApplicantID:       actor.UserID,
		ApprovalType:      tp,
		Status:            model.ApprovalStatusPending,
		CurrentStep:       step,
		Title:             strings.TrimSpace(req.Title),
		FormData:          datatypes.JSON(req.FormData),
		AttachmentFileIDs: mustJSON(req.AttachmentFileIDs),
		TemplateFileID:    req.TemplateFileID,
		Semester:          defaultSemester(req.Semester, now),
		DueAt:             &due,
		SubmittedAt:       now,
	}
	if err := s.approvalRepo.Create(item); err != nil {
		return nil, err
	}
	if err := s.actionRepo.Create(&model.ApprovalAction{
		ApprovalID: item.ID, ActionType: model.ApprovalActionSubmit, OperatorID: actor.UserID,
		ToStatus: model.ApprovalStatusPending, Snapshot: mustJSON(map[string]any{"submitted_at": now}),
	}); err != nil {
		return nil, err
	}
	_, _ = s.certificateService.GenerateApplicationPDF(context.Background(), item.ID)
	return item, nil
}

func (s *Service) ListMine(actor auth.Actor, limit, offset int) ([]model.Approval, int64, error) {
	return s.approvalRepo.ListMine(actor.UserID, limit, offset)
}

func (s *Service) Get(actor auth.Actor, id uint) (*ApprovalDetail, error) {
	item, err := s.approvalRepo.GetByIDInScope(authz.BuildScope(actor), id)
	if err != nil {
		return nil, err
	}
	actions, err := s.actionRepo.ListByApprovalID(item.ID)
	if err != nil {
		return nil, err
	}
	records, err := s.certificateRecordRepo.ListByApprovalID(item.ID)
	if err != nil {
		return nil, err
	}
	return &ApprovalDetail{Approval: item, Actions: actions, CertificateRecords: records}, nil
}

func (s *Service) Withdraw(actor auth.Actor, id uint) error {
	item, err := s.approvalRepo.GetByIDInScope(authz.Scope{SelfUserID: actor.UserID}, id)
	if err != nil {
		return err
	}
	if item.Status != model.ApprovalStatusPending {
		return ErrInvalidState
	}
	from := item.Status
	item.Status = model.ApprovalStatusWithdrawn
	item.DecidedAt = ptrTime(time.Now())
	item.CurrentStep = ""
	item.DueAt = nil
	if err := s.approvalRepo.Save(item); err != nil {
		return err
	}
	return s.actionRepo.Create(&model.ApprovalAction{
		ApprovalID: item.ID, ActionType: model.ApprovalActionWithdraw, OperatorID: actor.UserID,
		FromStatus: from, ToStatus: item.Status,
	})
}

func (s *Service) ListAdmin(actor auth.Actor, req ListAdminRequest) ([]model.Approval, int64, error) {
	return s.approvalRepo.ListByScope(authz.BuildScope(actor), strings.TrimSpace(req.Status), strings.TrimSpace(req.ApprovalType), req.Limit, req.Offset)
}

func (s *Service) Review(actor auth.Actor, id uint, req ReviewRequest) error {
	item, err := s.approvalRepo.GetByIDInScope(authz.BuildScope(actor), id)
	if err != nil {
		return err
	}
	if item.Status != model.ApprovalStatusPending {
		return ErrInvalidState
	}
	action := strings.TrimSpace(req.Action)
	toStatus := ""
	actionType := ""
	switch action {
	case "approve":
		toStatus, actionType = model.ApprovalStatusApproved, model.ApprovalActionApprove
	case "reject":
		toStatus, actionType = model.ApprovalStatusRejected, model.ApprovalActionReject
	default:
		return ErrInvalidFormData
	}
	from := item.Status
	item.Status = toStatus
	item.DecidedAt = ptrTime(time.Now())
	item.CurrentStep = ""
	item.DueAt = nil
	if err := s.approvalRepo.Save(item); err != nil {
		return err
	}
	if err := s.actionRepo.Create(&model.ApprovalAction{
		ApprovalID: item.ID, ActionType: actionType, OperatorID: actor.UserID,
		FromStatus: from, ToStatus: toStatus, Comment: strings.TrimSpace(req.Comment),
	}); err != nil {
		return err
	}
	if toStatus == model.ApprovalStatusApproved {
		_, _ = s.certificateService.GenerateApprovalCertificate(context.Background(), item.ID)
	}
	return nil
}

func (s *Service) Assign(actor auth.Actor, id uint, req AssignRequest) error {
	item, err := s.approvalRepo.GetByIDInScope(authz.BuildScope(actor), id)
	if err != nil {
		return err
	}
	if item.Status != model.ApprovalStatusPending {
		return ErrInvalidState
	}
	if req.ApproverID == 0 {
		return ErrInvalidFormData
	}
	item.CurrentApproverID = &req.ApproverID
	if err := s.approvalRepo.Save(item); err != nil {
		return err
	}
	return s.actionRepo.Create(&model.ApprovalAction{
		ApprovalID: item.ID, ActionType: model.ApprovalActionAssign, OperatorID: actor.UserID,
		FromStatus: item.Status, ToStatus: item.Status, Comment: strings.TrimSpace(req.Comment),
		Snapshot: mustJSON(map[string]any{"current_approver_id": req.ApproverID}),
	})
}

func (s *Service) Remind(actor auth.Actor, id uint, comment string) error {
	item, err := s.approvalRepo.GetByIDInScope(authz.BuildScope(actor), id)
	if err != nil {
		return err
	}
	if item.Status != model.ApprovalStatusPending {
		return ErrInvalidState
	}
	return s.actionRepo.Create(&model.ApprovalAction{
		ApprovalID: item.ID, ActionType: model.ApprovalActionRemind, OperatorID: actor.UserID,
		FromStatus: item.Status, ToStatus: item.Status, Comment: strings.TrimSpace(comment),
		Snapshot: mustJSON(map[string]any{"manual": true, "at": time.Now()}),
	})
}

type OverdueScanResult struct {
	Scanned  int `json:"scanned"`
	Reminded int `json:"reminded"`
}

func (s *Service) ScanAndRemindOverdue(ctx context.Context, now time.Time) (OverdueScanResult, error) {
	_ = ctx
	items, err := s.approvalRepo.ListOverduePending(now, 500)
	if err != nil {
		return OverdueScanResult{}, err
	}
	out := OverdueScanResult{Scanned: len(items)}
	for _, item := range items {
		_ = s.actionRepo.Create(&model.ApprovalAction{
			ApprovalID: item.ID, ActionType: model.ApprovalActionRemind, OperatorID: 0,
			FromStatus: item.Status, ToStatus: item.Status,
			Snapshot: mustJSON(map[string]any{"scan_at": now}),
		})
		out.Reminded++
	}
	return out, nil
}

func validateForm(tp string, raw json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ErrInvalidFormData
	}
	switch tp {
	case model.ApprovalTypeLeave:
		if blank(m["reason"]) || blank(m["start_at"]) || blank(m["end_at"]) || blank(m["contact_phone"]) {
			return ErrInvalidFormData
		}
	case model.ApprovalTypeBudget:
		if blank(m["activity_name"]) || blank(m["purpose"]) {
			return ErrInvalidFormData
		}
	}
	return nil
}

func defaultStepAndDue(tp string, now time.Time) (string, time.Time) {
	if tp == model.ApprovalTypeBudget {
		return model.ApprovalStepBudgetReview, now.Add(72 * time.Hour)
	}
	return model.ApprovalStepReview, now.Add(24 * time.Hour)
}

func defaultSemester(v string, now time.Time) string {
	v = strings.TrimSpace(v)
	if v != "" {
		return v
	}
	half := "1"
	if now.Month() >= 8 {
		half = "2"
	}
	return now.Format("2006") + "-" + half
}

func blank(v any) bool {
	if v == nil {
		return true
	}
	return strings.TrimSpace(toString(v)) == ""
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func mustJSON(v any) datatypes.JSON {
	b, _ := json.Marshal(v)
	return datatypes.JSON(b)
}
func ptrTime(t time.Time) *time.Time { return &t }
