package repo

import (
	"manage/internal/model"

	"gorm.io/gorm"
)

type AdminLogRepo struct {
	db *gorm.DB
}

func NewAdminLogRepo(db *gorm.DB) *AdminLogRepo {
	return &AdminLogRepo{db: db}
}

func (r *AdminLogRepo) Create(log *model.AdminLog) error {
	return r.db.Create(log).Error
}

func (r *AdminLogRepo) List(limit, offset int) ([]model.AdminLog, error) {
	var out []model.AdminLog
	err := r.db.Order("id desc").Limit(limit).Offset(offset).Find(&out).Error
	return out, err
}
