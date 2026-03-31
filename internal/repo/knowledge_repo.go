package repo

import (
	"manage/internal/model"
	"regexp"
	"strings"
	"unicode/utf8"

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
	out, _, err := r.SearchWithTotal(query, limit, offset)
	return out, err
}

// SearchWithTotal returns keyword-matched knowledge items and total count.
func (r *KnowledgeRepo) SearchWithTotal(query string, limit, offset int) ([]model.KnowledgeItem, int64, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []model.KnowledgeItem{}, 0, nil
	}
	tokens := tokenizeSearchQuery(query)
	if len(tokens) == 0 {
		tokens = []string{query}
	}
	if r.db.Dialector.Name() == "postgres" {
		// Stage 1: prefer full-text query path (fast when index/lexeme matches).
		var out []model.KnowledgeItem
		var total int64
		tsQueryText := strings.Join(tokens, " ")
		countSQL := `
SELECT COUNT(1)
FROM knowledge_items
WHERE to_tsvector('simple', coalesce(question, '') || ' ' || coalesce(answer, '') || ' ' || coalesce(content_text, '') || ' ' || coalesce(keywords::text, ''))
      @@ plainto_tsquery('simple', ?)`
		if err := r.db.Raw(countSQL, tsQueryText).Scan(&total).Error; err != nil {
			return nil, 0, err
		}
		sql := `
SELECT *
FROM knowledge_items
WHERE to_tsvector('simple', coalesce(question, '') || ' ' || coalesce(answer, '') || ' ' || coalesce(content_text, '') || ' ' || coalesce(keywords::text, ''))
      @@ plainto_tsquery('simple', ?)
ORDER BY ts_rank(
	to_tsvector('simple', coalesce(question, '') || ' ' || coalesce(answer, '') || ' ' || coalesce(content_text, '') || ' ' || coalesce(keywords::text, '')),
	plainto_tsquery('simple', ?)
) DESC, id DESC
LIMIT ? OFFSET ?`
		if total > 0 {
			err := r.db.Raw(sql, tsQueryText, tsQueryText, limit, offset).Scan(&out).Error
			return out, total, err
		}
		// Stage 2: fallback to tokenized LIKE for Chinese or mixed text where lexeme split is weak.
		return r.searchLikeWithTokens(tokens, limit, offset)
	}

	return r.searchLikeWithTokens(tokens, limit, offset)
}

func (r *KnowledgeRepo) searchLikeWithTokens(tokens []string, limit, offset int) ([]model.KnowledgeItem, int64, error) {
	base := r.db.Model(&model.KnowledgeItem{})
	keywordExpr := "keywords LIKE ?"
	if r.db.Dialector.Name() == "postgres" {
		keywordExpr = "coalesce(keywords::text, '') LIKE ?"
	}
	for _, token := range tokens {
		like := "%" + token + "%"
		base = base.Where(
			"(question LIKE ? OR answer LIKE ? OR content_text LIKE ? OR "+keywordExpr+")",
			like, like, like, like,
		)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []model.KnowledgeItem
	err := base.Order("id desc").Limit(limit).Offset(offset).Find(&out).Error
	return out, total, err
}

var splitter = regexp.MustCompile(`[\s\p{P}\p{S}]+`)

func tokenizeSearchQuery(query string) []string {
	parts := splitter.Split(strings.TrimSpace(query), -1)
	uniq := map[string]struct{}{}
	out := make([]string, 0, len(parts)*2)
	appendToken := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" {
			return
		}
		if _, ok := uniq[t]; ok {
			return
		}
		uniq[t] = struct{}{}
		out = append(out, t)
	}

	for _, p := range parts {
		if p == "" {
			continue
		}
		appendToken(strings.ToLower(p))
		rs := []rune(p)
		if len(rs) >= 2 {
			for i := 0; i < len(rs)-1; i++ {
				bigram := string(rs[i : i+2])
				appendToken(strings.ToLower(bigram))
			}
		}
	}
	// Guard against pathological tokenization generating empty byte sequences.
	filtered := out[:0]
	for _, t := range out {
		if utf8.ValidString(t) && t != "" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// List returns knowledge items for admin management.
func (r *KnowledgeRepo) List(query string, limit, offset int) ([]model.KnowledgeItem, error) {
	out, _, err := r.ListWithTotal(query, limit, offset)
	return out, err
}

// ListWithTotal returns knowledge items for admin management with total count.
func (r *KnowledgeRepo) ListWithTotal(query string, limit, offset int) ([]model.KnowledgeItem, int64, error) {
	query = strings.TrimSpace(query)
	if query != "" {
		return r.SearchWithTotal(query, limit, offset)
	}

	var total int64
	if err := r.db.Model(&model.KnowledgeItem{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []model.KnowledgeItem
	err := r.db.Model(&model.KnowledgeItem{}).Order("id desc").Limit(limit).Offset(offset).Find(&out).Error
	return out, total, err
}

// Create inserts one knowledge item.
func (r *KnowledgeRepo) Create(item *model.KnowledgeItem) error {
	return r.db.Create(item).Error
}

// UpdateByID updates one knowledge item by id.
func (r *KnowledgeRepo) UpdateByID(id uint, updates map[string]any) error {
	tx := r.db.Model(&model.KnowledgeItem{}).Where("id = ?", id).Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetByID returns one knowledge item by id.
func (r *KnowledgeRepo) GetByID(id uint) (*model.KnowledgeItem, error) {
	var item model.KnowledgeItem
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// DeleteByID deletes one knowledge item by id.
func (r *KnowledgeRepo) DeleteByID(id uint) error {
	tx := r.db.Delete(&model.KnowledgeItem{}, id)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
