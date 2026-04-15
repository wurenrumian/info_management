package repo

import (
	"manage/internal/model"
	"strings"
	"time"

	"gorm.io/gorm"
)

// AnnouncementRepo provides data access for announcements.
type AnnouncementRepo struct {
	db *gorm.DB
}

// NewAnnouncementRepo creates a new announcement repository.
func NewAnnouncementRepo(db *gorm.DB) *AnnouncementRepo {
	return &AnnouncementRepo{db: db}
}

// Create inserts one announcement row.
func (r *AnnouncementRepo) Create(a *model.Announcement) error {
	return r.db.Create(a).Error
}

// GetByID returns one announcement by id.
func (r *AnnouncementRepo) GetByID(id uint) (*model.Announcement, error) {
	var a model.Announcement
	if err := r.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func sanitizeLimitOffset(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func applyAnnouncementStatusFilter(q *gorm.DB, status string) *gorm.DB {
	status = strings.TrimSpace(status)
	if status == "" {
		return q
	}
	return q.Where("status = ?", status)
}

func listWithTotal(base *gorm.DB, limit, offset int) ([]model.Announcement, int64, error) {
	limit, offset = sanitizeLimitOffset(limit, offset)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Announcement
	err := base.Order("id desc").Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

// ListWithTotal returns student-side list with pagination and optional status filter.
func (r *AnnouncementRepo) ListWithTotal(status string, limit, offset int) ([]model.Announcement, int64, error) {
	base := applyAnnouncementStatusFilter(r.db.Model(&model.Announcement{}), status)
	return listWithTotal(base, limit, offset)
}

// ListAdminWithTotal returns admin-side list with pagination and optional status filter.
func (r *AnnouncementRepo) ListAdminWithTotal(status string, limit, offset int) ([]model.Announcement, int64, error) {
	base := applyAnnouncementStatusFilter(r.db.Model(&model.Announcement{}), status)
	return listWithTotal(base, limit, offset)
}

// Patch updates announcement fields by id.
func (r *AnnouncementRepo) Patch(id uint, updates map[string]any) error {
	return UpdateByID(r.db.Model(&model.Announcement{}), id, updates)
}

// Publish marks an announcement as published and sets published time.
func (r *AnnouncementRepo) Publish(id uint, publishedAt time.Time) error {
	return UpdateByID(r.db.Model(&model.Announcement{}), id, map[string]any{
		"status":       "published",
		"published_at": publishedAt,
	})
}

// Archive marks an announcement as archived.
func (r *AnnouncementRepo) Archive(id uint) error {
	return UpdateByID(r.db.Model(&model.Announcement{}), id, map[string]any{
		"status": "archived",
	})
}

// UpdateByID keeps backward compatibility with existing caller sites.
func (r *AnnouncementRepo) UpdateByID(id uint, updates map[string]any) error {
	return r.Patch(id, updates)
}

// DeleteByID deletes an announcement by id.
func (r *AnnouncementRepo) DeleteByID(id uint) error {
	return r.db.Delete(&model.Announcement{}, id).Error
}

// UpdateStatus keeps backward compatibility with existing caller sites.
func (r *AnnouncementRepo) UpdateStatus(id uint, status string) error {
	return r.Patch(id, map[string]any{"status": status})
}
