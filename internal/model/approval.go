package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	ApprovalTypeLeave  = "leave"
	ApprovalTypeBudget = "budget"

	ApprovalStatusPending   = "pending"
	ApprovalStatusApproved  = "approved"
	ApprovalStatusRejected  = "rejected"
	ApprovalStatusWithdrawn = "withdrawn"

	ApprovalStepReview       = "review"
	ApprovalStepBudgetReview = "budget_review"

	ApprovalActionSubmit   = "submit"
	ApprovalActionWithdraw = "withdraw"
	ApprovalActionApprove  = "approve"
	ApprovalActionReject   = "reject"
	ApprovalActionAssign   = "assign"
	ApprovalActionRemind   = "remind"
)

type Approval struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	ApplicantID       uint           `gorm:"index;not null" json:"applicant_id"`
	ApprovalType      string         `gorm:"type:varchar(20);index;not null" json:"approval_type"`
	Status            string         `gorm:"type:varchar(20);index;not null" json:"status"`
	CurrentStep       string         `gorm:"type:varchar(40);index" json:"current_step"`
	Title             string         `gorm:"type:varchar(200);not null" json:"title"`
	FormData          datatypes.JSON `gorm:"type:jsonb" json:"form_data"`
	AttachmentFileIDs datatypes.JSON `gorm:"type:jsonb" json:"attachment_file_ids"`
	TemplateFileID    *uint          `gorm:"index" json:"template_file_id"`
	CurrentApproverID *uint          `gorm:"index" json:"current_approver_id"`
	Semester          string         `gorm:"type:varchar(20);index;not null" json:"semester"`
	DueAt             *time.Time     `gorm:"index" json:"due_at"`
	SubmittedAt       time.Time      `json:"submitted_at"`
	DecidedAt         *time.Time     `json:"decided_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type ApprovalAction struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	ApprovalID uint           `gorm:"index;not null" json:"approval_id"`
	ActionType string         `gorm:"type:varchar(20);not null" json:"action_type"`
	OperatorID uint           `gorm:"index;not null" json:"operator_id"`
	FromStatus string         `gorm:"type:varchar(20)" json:"from_status"`
	ToStatus   string         `gorm:"type:varchar(20)" json:"to_status"`
	Comment    string         `gorm:"type:varchar(500)" json:"comment"`
	Snapshot   datatypes.JSON `gorm:"type:jsonb" json:"snapshot"`
	CreatedAt  time.Time      `json:"created_at"`
}
