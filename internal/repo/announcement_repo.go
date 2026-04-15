package repo

import (
	"manage/internal/model"

	"gorm.io/gorm"
)

type AnnouncementRepo struct {
	db *gorm.DB
}

func NewAnnouncementRepo(db *gorm.DB) *AnnouncementRepo {
	return &AnnouncementRepo{db: db}
}

func (r *AnnouncementRepo) Create(a *model.Announcement) error {
	return r.db.Create(a).Error
}

func (r *AnnouncementRepo) GetByID(id uint) (*model.Announcement, error) {
	var a model.Announcement
	if err := r.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AnnouncementRepo) ListWithTotal(status string, limit, offset int) ([]model.Announcement, int64, error) {
	q := r.db.Model(&model.Announcement{})

	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.Announcement
	err := q.Order("id desc").Limit(limit).Offset(offset).Find(&list).Error

	return list, total, err
}

func (r *AnnouncementRepo) UpdateByID(id uint, updates map[string]any) error {
	return UpdateByID(r.db.Model(&model.Announcement{}), id, updates)
}
