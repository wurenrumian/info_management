package grade

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"manage/internal/model"
	"manage/internal/repo"
)

// Service handles grade governance logic.
type Service struct {
	userRepo  *repo.UserRepo
	classRepo *repo.ClassRepo
}

// NewService creates a grade governance service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		userRepo:  repo.NewUserRepo(db),
		classRepo: repo.NewClassRepo(db),
	}
}

// ResolveEffectiveGrade resolves the grade used by auth/jwt.
// It prefers class.grade and falls back to user.grade when class is missing.
func (s *Service) ResolveEffectiveGrade(user *model.User) (string, error) {
	if user == nil {
		return "", errors.New("nil user")
	}
	if user.ClassID > 0 {
		class, err := s.classRepo.GetByID(user.ClassID)
		if err == nil {
			return strings.TrimSpace(class.Grade), nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}
	return strings.TrimSpace(user.Grade), nil
}

// SyncUserGradeByClassID updates one user's grade from class.grade.
func (s *Service) SyncUserGradeByClassID(userID uint, classID uint) error {
	if userID == 0 || classID == 0 {
		return nil
	}
	class, err := s.classRepo.GetByID(classID)
	if err != nil {
		return err
	}
	return s.userRepo.UpdateByID(userID, map[string]any{
		"grade": strings.TrimSpace(class.Grade),
	})
}

// SyncUsersGradeByClassID updates all users in one class to class.grade.
func (s *Service) SyncUsersGradeByClassID(classID uint) (int64, error) {
	if classID == 0 {
		return 0, nil
	}
	class, err := s.classRepo.GetByID(classID)
	if err != nil {
		return 0, err
	}
	return s.userRepo.BulkUpdateGradeByClassID(classID, strings.TrimSpace(class.Grade))
}
