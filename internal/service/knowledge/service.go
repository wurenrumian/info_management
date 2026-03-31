package knowledge

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"manage/internal/model"
	"manage/internal/repo"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Service encapsulates knowledge-search and maintenance business rules.
type Service struct {
	repo *repo.KnowledgeRepo
}

// NewService creates a knowledge service.
func NewService(db *gorm.DB) *Service {
	return &Service{repo: repo.NewKnowledgeRepo(db)}
}

// Search returns knowledge items for student-facing query.
func (s *Service) Search(query string, limit, offset int) ([]model.KnowledgeItem, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("missing q")
	}
	limit, offset = normalizePage(limit, offset)
	return s.repo.Search(query, limit, offset)
}

// List returns knowledge items for admin management.
func (s *Service) List(query string, limit, offset int) ([]model.KnowledgeItem, error) {
	limit, offset = normalizePage(limit, offset)
	return s.repo.List(query, limit, offset)
}

// Create stores a knowledge item.
func (s *Service) Create(item *model.KnowledgeItem) error {
	if item == nil {
		return errors.New("invalid item")
	}
	if strings.TrimSpace(item.Question) == "" || strings.TrimSpace(item.Answer) == "" {
		return errors.New("missing fields")
	}
	return s.repo.Create(item)
}

// Patch updates a knowledge item by id.
func (s *Service) Patch(id uint, updates map[string]any) error {
	if id == 0 {
		return errors.New("invalid id")
	}
	if len(updates) == 0 {
		return errors.New("empty patch")
	}
	return s.repo.UpdateByID(id, updates)
}

func normalizePage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
