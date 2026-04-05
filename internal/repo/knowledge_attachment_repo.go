package repo

import (
	"manage/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// KnowledgeAttachmentRepo provides data access for knowledge-document relations.
type KnowledgeAttachmentRepo struct {
	db *gorm.DB
}

// NewKnowledgeAttachmentRepo creates a relation repository.
func NewKnowledgeAttachmentRepo(db *gorm.DB) *KnowledgeAttachmentRepo {
	return &KnowledgeAttachmentRepo{db: db}
}

// CreateBatch inserts relation rows and ignores duplicates.
func (r *KnowledgeAttachmentRepo) CreateBatch(rows []model.KnowledgeAttachment) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "knowledge_id"}, {Name: "file_id"}},
		DoNothing: true,
	}).Create(&rows).Error
}

// ListByKnowledgeID returns relations for one knowledge item.
func (r *KnowledgeAttachmentRepo) ListByKnowledgeID(knowledgeID uint) ([]model.KnowledgeAttachment, error) {
	var out []model.KnowledgeAttachment
	err := r.db.Where("knowledge_id = ?", knowledgeID).Order("id asc").Find(&out).Error
	return out, err
}

// ListByKnowledgeIDs returns relations for multiple knowledge ids.
func (r *KnowledgeAttachmentRepo) ListByKnowledgeIDs(knowledgeIDs []uint) ([]model.KnowledgeAttachment, error) {
	if len(knowledgeIDs) == 0 {
		return []model.KnowledgeAttachment{}, nil
	}
	var out []model.KnowledgeAttachment
	err := r.db.Where("knowledge_id IN ?", knowledgeIDs).Order("id asc").Find(&out).Error
	return out, err
}

// DeleteByKnowledgeAndFileID deletes one relation row.
func (r *KnowledgeAttachmentRepo) DeleteByKnowledgeAndFileID(knowledgeID, fileID uint) error {
	tx := r.db.Where("knowledge_id = ? AND file_id = ?", knowledgeID, fileID).Delete(&model.KnowledgeAttachment{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
