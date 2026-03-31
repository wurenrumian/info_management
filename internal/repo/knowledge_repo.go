package repo

import (
	"manage/internal/model"
	"strings"

	"gorm.io/gorm"
)

// KnowledgeRepo provides data access for knowledge items.
type KnowledgeRepo struct {
	db *gorm.DB
}

// NewKnowledgeRepo creates a knowledge repository.
func NewKnowledgeRepo(db *gorm.DB) *KnowledgeRepo {
	return &KnowledgeRepo{db: db}
}

// Search returns keyword-matched knowledge items.
func (r *KnowledgeRepo) Search(query string, limit, offset int) ([]model.KnowledgeItem, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []model.KnowledgeItem{}, nil
	}
	if r.db.Dialector.Name() == "postgres" {
		var out []model.KnowledgeItem
		sql := `
SELECT *
FROM knowledge_items
WHERE to_tsvector('simple', coalesce(question, '') || ' ' || coalesce(answer, '') || ' ' || coalesce(keywords::text, ''))
      @@ plainto_tsquery('simple', ?)
ORDER BY ts_rank(
	to_tsvector('simple', coalesce(question, '') || ' ' || coalesce(answer, '') || ' ' || coalesce(keywords::text, '')),
	plainto_tsquery('simple', ?)
) DESC, id DESC
LIMIT ? OFFSET ?`
		err := r.db.Raw(sql, query, query, limit, offset).Scan(&out).Error
		return out, err
	}

	like := "%" + query + "%"
	var out []model.KnowledgeItem
	err := r.db.Model(&model.KnowledgeItem{}).
		Where("question LIKE ? OR answer LIKE ? OR keywords LIKE ?", like, like, like).
		Order("id desc").
		Limit(limit).
		Offset(offset).
		Find(&out).Error
	return out, err
}

// List returns knowledge items for admin management.
func (r *KnowledgeRepo) List(query string, limit, offset int) ([]model.KnowledgeItem, error) {
	query = strings.TrimSpace(query)
	if query != "" {
		return r.Search(query, limit, offset)
	}
	var out []model.KnowledgeItem
	err := r.db.Model(&model.KnowledgeItem{}).Order("id desc").Limit(limit).Offset(offset).Find(&out).Error
	return out, err
}

// Create inserts one knowledge item.
func (r *KnowledgeRepo) Create(item *model.KnowledgeItem) error {
	return r.db.Create(item).Error
}

// UpdateByID updates one knowledge item by id.
func (r *KnowledgeRepo) UpdateByID(id uint, updates map[string]any) error {
	return r.db.Model(&model.KnowledgeItem{}).Where("id = ?", id).Updates(updates).Error
}

