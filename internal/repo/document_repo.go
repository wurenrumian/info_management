package repo

import (
	"manage/internal/model"
	"regexp"
	"strings"
	"unicode/utf8"

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

// ListByIDs returns documents by ids.
func (r *DocumentRepo) ListByIDs(ids []uint) ([]model.Document, error) {
	if len(ids) == 0 {
		return []model.Document{}, nil
	}
	var docs []model.Document
	err := r.db.Where("id IN ?", ids).Find(&docs).Error
	return docs, err
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

// SearchWithTotal returns documents by title/content query with pagination and total count.
func (r *DocumentRepo) SearchWithTotal(query string, limit, offset int) ([]model.Document, int64, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []model.Document{}, 0, nil
	}
	tokens := tokenizeDocumentSearchQuery(query)
	if len(tokens) == 0 {
		tokens = []string{query}
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	if r.db.Dialector.Name() == "postgres" {
		var out []model.Document
		var total int64
		tsQueryText := strings.Join(tokens, " ")
		countSQL := `
SELECT COUNT(1)
FROM documents
WHERE to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(content_text, ''))
      @@ plainto_tsquery('simple', ?)`
		if err := r.db.Raw(countSQL, tsQueryText).Scan(&total).Error; err != nil {
			return nil, 0, err
		}
		sql := `
SELECT *
FROM documents
WHERE to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(content_text, ''))
      @@ plainto_tsquery('simple', ?)
ORDER BY ts_rank(
	to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(content_text, '')),
	plainto_tsquery('simple', ?)
) DESC, id DESC
LIMIT ? OFFSET ?`
		if total > 0 {
			err := r.db.Raw(sql, tsQueryText, tsQueryText, limit, offset).Scan(&out).Error
			return out, total, err
		}
	}

	base := r.db.Model(&model.Document{})
	for _, token := range tokens {
		like := "%" + token + "%"
		base = base.Where("(title LIKE ? OR content_text LIKE ?)", like, like)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []model.Document
	err := base.Order("id desc").Limit(limit).Offset(offset).Find(&out).Error
	return out, total, err
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

var documentSplitter = regexp.MustCompile(`[\s\p{P}\p{S}]+`)

func tokenizeDocumentSearchQuery(query string) []string {
	parts := documentSplitter.Split(strings.TrimSpace(query), -1)
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
				appendToken(strings.ToLower(string(rs[i : i+2])))
			}
		}
	}
	filtered := out[:0]
	for _, t := range out {
		if utf8.ValidString(t) && t != "" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
