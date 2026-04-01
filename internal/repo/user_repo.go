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

func (r *UserRepo) GetByIDInScope(scope authz.Scope, id uint) (*model.User, error) {
	q := r.db.Model(&model.User{}).Preload("Class").Where("id = ?", id)
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

	var out model.User
	if err := q.First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *UserRepo) UpdateByID(id uint, updates map[string]any) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *UserRepo) GetByOpenID(openID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("openid = ?", openID).Preload("Class").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) GetByStudentID(studentID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("student_id = ?", studentID).Preload("Class").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) UpdatePasswordHash(userID uint, hash string) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("password_hash", hash).Error
}
