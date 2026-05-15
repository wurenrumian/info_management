package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	CertificateSealStatusNone                = "none"
	CertificateSealStatusInternalSealApplied = "internal_seal_applied"

	CertificateRecordStatusGenerated = "generated"
	CertificateRecordStatusFailed    = "failed"
	CertificateRecordStatusRevoked   = "revoked"
)

type CertificateRecord struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	ApprovalID       uint           `gorm:"index;not null" json:"approval_id"`
	ApplicantID      uint           `gorm:"index;not null" json:"applicant_id"`
	TemplateID       uint           `gorm:"index;not null" json:"template_id"`
	DocumentStage    string         `gorm:"type:varchar(30);index;not null" json:"document_stage"`
	CertificateNo    string         `gorm:"type:varchar(80);uniqueIndex" json:"certificate_no"`
	VerificationCode string         `gorm:"type:varchar(80);uniqueIndex" json:"verification_code"`
	VerificationHash string         `gorm:"type:varchar(128)" json:"verification_hash"`
	RenderedPayload  datatypes.JSON `gorm:"type:jsonb" json:"rendered_payload"`
	DocumentID       uint           `gorm:"index" json:"document_id"`
	SealStatus       string         `gorm:"type:varchar(30);index;not null" json:"seal_status"`
	SealAppliedBy    uint           `json:"seal_applied_by"`
	SealAppliedAt    *time.Time     `json:"seal_applied_at"`
	Status           string         `gorm:"type:varchar(20);index;not null" json:"status"`
	ErrorMessage     string         `gorm:"type:varchar(500)" json:"error_message"`
	GeneratedAt      *time.Time     `json:"generated_at"`
	RevokedAt        *time.Time     `json:"revoked_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}
