package repo

import (
	"manage/internal/model"

	"gorm.io/gorm"
)

// DocumentRepo provides data access for documents.
type DocumentRepo struct {
	db *gorm.DB
}

// NewDocumentRepo creates a document repository.
func NewDocumentRepo(db *gorm.DB) *DocumentRepo {
	return &DocumentRepo{db: db}
}

// Create inserts one document record.
func (r *DocumentRepo) Create(doc *model.Document) error {
	return r.db.Create(doc).Error
}

// GetByID returns one document by id.
func (r *DocumentRepo) GetByID(id uint) (*model.Document, error) {
	var doc model.Document
	if err := r.db.First(&doc, id).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// ListWithTotal returns documents with pagination and total count.
func (r *DocumentRepo) ListWithTotal(limit, offset int) ([]model.Document, int64, error) {
	var total int64
	if err := r.db.Model(&model.Document{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var docs []model.Document
	err := r.db.Model(&model.Document{}).Order("id desc").Limit(limit).Offset(offset).Find(&docs).Error
	return docs, total, err
}

// DeleteByID deletes one document by id.
func (r *DocumentRepo) DeleteByID(id uint) error {
	tx := r.db.Delete(&model.Document{}, id)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
