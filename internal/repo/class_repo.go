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
	q := r.db.Model(&model.Class{}).Limit(limit).Offset(offset)
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

	var out []model.Class
	err := q.Find(&out).Error
	return out, err
}

func (r *ClassRepo) GetByIDInScope(scope authz.Scope, id uint) (*model.Class, error) {
	q := r.db.Model(&model.Class{}).Where("id = ?", id)
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

	var out model.Class
	if err := q.First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ClassRepo) Create(item *model.Class) error {
	return r.db.Create(item).Error
}

func (r *ClassRepo) UpdateByID(id uint, updates map[string]any) error {
	result := r.db.Model(&model.Class{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
