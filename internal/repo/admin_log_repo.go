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
	out, _, err := r.ListWithTotal(limit, offset)
	return out, err
}

func (r *AdminLogRepo) ListWithTotal(limit, offset int) ([]model.AdminLog, int64, error) {
	var total int64
	if err := r.db.Model(&model.AdminLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []model.AdminLog
	err := r.db.Order("id desc").Limit(limit).Offset(offset).Find(&out).Error
	return out, total, err
}
