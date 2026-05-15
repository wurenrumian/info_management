package repo

import (
	"manage/internal/model"

	"gorm.io/gorm"
)

type CertificateTemplateRepo struct {
	db *gorm.DB
}

func NewCertificateTemplateRepo(db *gorm.DB) *CertificateTemplateRepo {
	return &CertificateTemplateRepo{db: db}
}

func (r *CertificateTemplateRepo) Create(item *model.CertificateTemplate) error {
	return r.db.Create(item).Error
}

func (r *CertificateTemplateRepo) GetByID(id uint) (*model.CertificateTemplate, error) {
	var out model.CertificateTemplate
	if err := r.db.Where("id = ?", id).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *CertificateTemplateRepo) ListAll() ([]model.CertificateTemplate, error) {
	var out []model.CertificateTemplate
	if err := r.db.Order("id asc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *CertificateTemplateRepo) GetActiveByApprovalTypeAndStage(approvalType, stage string) (*model.CertificateTemplate, error) {
	var out model.CertificateTemplate
	if err := r.db.
		Where("approval_type = ? AND document_stage = ? AND status = ?", approvalType, stage, model.CertificateTemplateStatusActive).
		First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *CertificateTemplateRepo) Save(item *model.CertificateTemplate) error {
	return r.db.Save(item).Error
}
