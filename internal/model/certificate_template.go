package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	CertificateDocumentStageApplication         = "application"
	CertificateDocumentStageApprovalCertificate = "approval_certificate"

	CertificateTemplateStatusActive   = "active"
	CertificateTemplateStatusInactive = "inactive"

	CertificateRendererTypst = "typst"
)

type CertificateTemplate struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Code            string         `gorm:"type:varchar(60);uniqueIndex;not null" json:"code"`
	Name            string         `gorm:"type:varchar(100);not null" json:"name"`
	ApprovalType    string         `gorm:"type:varchar(20);index;not null" json:"approval_type"`
	DocumentStage   string         `gorm:"type:varchar(30);index;not null" json:"document_stage"`
	Status          string         `gorm:"type:varchar(20);index;not null" json:"status"`
	Renderer        string         `gorm:"type:varchar(20);not null" json:"renderer"`
	TemplatePath    string         `gorm:"type:varchar(255);not null" json:"template_path"`
	TemplateVersion string         `gorm:"type:varchar(40);not null" json:"template_version"`
	FieldMapping    datatypes.JSON `gorm:"type:jsonb" json:"field_mapping"`
	Disclaimer      string         `gorm:"type:varchar(500)" json:"disclaimer"`
	CreatedBy       uint           `json:"created_by"`
	UpdatedBy       uint           `json:"updated_by"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
