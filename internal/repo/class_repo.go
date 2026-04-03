package repo

import (
	"manage/internal/model"
	"manage/internal/service/authz"

	"gorm.io/gorm"
)

type ClassRepo struct {
	db *gorm.DB
}

func NewClassRepo(db *gorm.DB) *ClassRepo {
	return &ClassRepo{db: db}
}

func (r *ClassRepo) ListByScope(scope authz.Scope, limit, offset int) ([]model.Class, error) {
	out, _, err := r.ListByScopeWithTotal(scope, limit, offset)
	return out, err
}

func (r *ClassRepo) ListByScopeWithTotal(scope authz.Scope, limit, offset int) ([]model.Class, int64, error) {
	base := applyClassScope(r.db.Model(&model.Class{}), scope)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []model.Class
	err := applyClassScope(r.db.Model(&model.Class{}), scope).
		Limit(limit).
		Offset(offset).
		Find(&out).Error
	return out, total, err
}

func (r *ClassRepo) GetByIDInScope(scope authz.Scope, id uint) (*model.Class, error) {
	q := applyClassScope(r.db.Model(&model.Class{}), scope).Where("id = ?", id)

	var out model.Class
	if err := q.First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ClassRepo) GetByID(id uint) (*model.Class, error) {
	var out model.Class
	if err := r.db.First(&out, id).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ClassRepo) Create(item *model.Class) error {
	return r.db.Create(item).Error
}

func (r *ClassRepo) UpdateByID(id uint, updates map[string]any) error {
	return UpdateByID(r.db.Model(&model.Class{}), id, updates)
}

func applyClassScope(q *gorm.DB, scope authz.Scope) *gorm.DB {
	switch {
	case scope.AllowAll:
	case scope.ClassID > 0 && scope.Grade != "":
		q = q.Where("id = ? OR grade = ?", scope.ClassID, scope.Grade)
	case scope.ClassID > 0:
		q = q.Where("id = ?", scope.ClassID)
	case scope.Grade != "":
		q = q.Where("grade = ?", scope.Grade)
	default:
		q = q.Where("1 = 0")
	}
	return q
}
