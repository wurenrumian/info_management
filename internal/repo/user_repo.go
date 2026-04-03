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
	out, _, err := r.ListByScopeWithTotal(scope, limit, offset)
	return out, err
}

func (r *UserRepo) ListByScopeWithTotal(scope authz.Scope, limit, offset int) ([]model.User, int64, error) {
	base := applyUserScope(r.db.Model(&model.User{}), scope)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []model.User
	err := applyUserScope(r.db.Model(&model.User{}).Preload("Class"), scope).
		Limit(limit).
		Offset(offset).
		Find(&out).Error
	return out, total, err
}

func (r *UserRepo) GetByIDInScope(scope authz.Scope, id uint) (*model.User, error) {
	q := applyUserScope(r.db.Model(&model.User{}).Preload("Class"), scope).Where("id = ?", id)

	var out model.User
	if err := q.First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *UserRepo) UpdateByID(id uint, updates map[string]any) error {
	return UpdateByID(r.db.Model(&model.User{}), id, updates)
}

func (r *UserRepo) BulkUpdateGradeByClassID(classID uint, grade string) (int64, error) {
	tx := r.db.Model(&model.User{}).Where("class_id = ?", classID).Update("grade", grade)
	return tx.RowsAffected, tx.Error
}

func (r *UserRepo) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepo) GetByOpenID(openID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("open_id = ?", openID).Preload("Class").First(&user).Error
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

func applyUserScope(q *gorm.DB, scope authz.Scope) *gorm.DB {
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
	return q
}
