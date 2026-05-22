package repo

import (
	"manage/internal/model"
	"manage/internal/service/authz"

	"gorm.io/gorm"
)

type CertificateRecordRepo struct {
	db *gorm.DB
}

func NewCertificateRecordRepo(db *gorm.DB) *CertificateRecordRepo {
	return &CertificateRecordRepo{db: db}
}

func (r *CertificateRecordRepo) Create(item *model.CertificateRecord) error {
	return r.db.Create(item).Error
}

func (r *CertificateRecordRepo) Save(item *model.CertificateRecord) error {
	return r.db.Save(item).Error
}

func (r *CertificateRecordRepo) GetByID(id uint) (*model.CertificateRecord, error) {
	var out model.CertificateRecord
	if err := r.db.Where("id = ?", id).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *CertificateRecordRepo) GetByIDInScope(scope authz.Scope, id uint) (*model.CertificateRecord, error) {
	var out model.CertificateRecord
	if err := applyCertificateScope(r.db.Model(&model.CertificateRecord{}), scope).
		Where("id = ?", id).
		First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *CertificateRecordRepo) ListMine(applicantID uint, approvalType string, limit, offset int) ([]model.CertificateRecord, int64, error) {
	q := r.db.Model(&model.CertificateRecord{}).Where("applicant_id = ?", applicantID)
	if approvalType != "" {
		q = q.Where(
			"approval_id IN (SELECT id FROM approvals WHERE approval_type = ?)",
			approvalType,
		)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []model.CertificateRecord
	err := q.Order("id desc").Limit(limit).Offset(offset).Find(&out).Error
	return out, total, err
}

func (r *CertificateRecordRepo) ListByApprovalID(approvalID uint) ([]model.CertificateRecord, error) {
	var out []model.CertificateRecord
	err := r.db.Where("approval_id = ?", approvalID).Order("id asc").Find(&out).Error
	return out, err
}

func (r *CertificateRecordRepo) GetByVerificationCode(code string) (*model.CertificateRecord, error) {
	var out model.CertificateRecord
	if err := r.db.Where("verification_code = ?", code).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func applyCertificateScope(q *gorm.DB, scope authz.Scope) *gorm.DB {
	switch {
	case scope.AllowAll:
		return q
	case scope.SelfUserID > 0:
		return q.Where("applicant_id = ?", scope.SelfUserID)
	case scope.ClassID > 0 && scope.Grade != "":
		return q.Where(
			"applicant_id IN (SELECT id FROM users WHERE class_id = ? OR grade = ?)",
			scope.ClassID,
			scope.Grade,
		)
	case scope.ClassID > 0:
		return q.Where(
			"applicant_id IN (SELECT id FROM users WHERE class_id = ?)",
			scope.ClassID,
		)
	default:
		return q.Where("1 = 0")
	}
}
