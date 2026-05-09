package repo

import (
	"manage/internal/model"
	"manage/internal/service/authz"
	"time"

	"gorm.io/gorm"
)

type ApprovalRepo struct{ db *gorm.DB }

func NewApprovalRepo(db *gorm.DB) *ApprovalRepo { return &ApprovalRepo{db: db} }

func (r *ApprovalRepo) Create(item *model.Approval) error { return r.db.Create(item).Error }

func (r *ApprovalRepo) GetByID(id uint) (*model.Approval, error) {
	var out model.Approval
	if err := r.db.Where("id = ?", id).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ApprovalRepo) GetByIDInScope(scope authz.Scope, id uint) (*model.Approval, error) {
	var out model.Approval
	q := applyApprovalScope(r.db.Model(&model.Approval{}), scope).Where("id = ?", id)
	if err := q.First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ApprovalRepo) ListMine(userID uint, limit, offset int) ([]model.Approval, int64, error) {
	var total int64
	base := r.db.Model(&model.Approval{}).Where("applicant_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []model.Approval
	err := base.Order("id desc").Limit(limit).Offset(offset).Find(&out).Error
	return out, total, err
}

func (r *ApprovalRepo) ListByScope(scope authz.Scope, status, approvalType string, limit, offset int) ([]model.Approval, int64, error) {
	q := applyApprovalScope(r.db.Model(&model.Approval{}), scope)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if approvalType != "" {
		q = q.Where("approval_type = ?", approvalType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []model.Approval
	err := q.Order("id desc").Limit(limit).Offset(offset).Find(&out).Error
	return out, total, err
}

func (r *ApprovalRepo) Save(item *model.Approval) error {
	return r.db.Save(item).Error
}

func (r *ApprovalRepo) ListOverduePending(now time.Time, limit int) ([]model.Approval, error) {
	var out []model.Approval
	err := r.db.Where("status = ? AND due_at IS NOT NULL AND due_at <= ?", model.ApprovalStatusPending, now).
		Order("id asc").Limit(limit).Find(&out).Error
	return out, err
}

type ApprovalActionRepo struct{ db *gorm.DB }

func NewApprovalActionRepo(db *gorm.DB) *ApprovalActionRepo { return &ApprovalActionRepo{db: db} }

func (r *ApprovalActionRepo) Create(item *model.ApprovalAction) error { return r.db.Create(item).Error }

func (r *ApprovalActionRepo) ListByApprovalID(approvalID uint) ([]model.ApprovalAction, error) {
	var out []model.ApprovalAction
	err := r.db.Where("approval_id = ?", approvalID).Order("id asc").Find(&out).Error
	return out, err
}

func applyApprovalScope(q *gorm.DB, scope authz.Scope) *gorm.DB {
	switch {
	case scope.AllowAll:
	case scope.SelfUserID > 0:
		q = q.Where("applicant_id = ?", scope.SelfUserID)
	case scope.ClassID > 0 && scope.Grade != "":
		q = q.Where("applicant_id IN (SELECT id FROM users WHERE class_id = ? OR grade = ?)", scope.ClassID, scope.Grade)
	case scope.ClassID > 0:
		q = q.Where("applicant_id IN (SELECT id FROM users WHERE class_id = ?)", scope.ClassID)
	default:
		q = q.Where("1 = 0")
	}
	return q
}
