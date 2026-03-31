package repo

import (
	"manage/internal/model"
	"manage/internal/service/authz"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) ListByScope(scope authz.Scope, limit, offset int) ([]model.User, error) {
	q := r.db.Model(&model.User{}).Preload("Class").Limit(limit).Offset(offset)
	switch {
	case scope.AllowAll:
	case scope.SelfUserID > 0:
		q = q.Where("id = ?", scope.SelfUserID)
	case scope.ClassID > 0 && scope.Grade != "":
		q = q.Where("class_id = ? OR grade = ?", scope.ClassID, scope.Grade)
	case scope.ClassID > 0:
		q = q.Where("class_id = ?", scope.ClassID)
	default:
		q = q.Where("1 = 0")
	}

	var out []model.User
	err := q.Find(&out).Error
	return out, err
}

