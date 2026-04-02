package auth

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"manage/internal/model"
	"manage/internal/repo"
)

// DevLoginService handles development-only token issuance.
type DevLoginService struct {
	userRepo  *repo.UserRepo
	jwtSecret string
}

// NewDevLoginService creates a DevLoginService.
func NewDevLoginService(db *gorm.DB, jwtSecret string) *DevLoginService {
	return &DevLoginService{
		userRepo:  repo.NewUserRepo(db),
		jwtSecret: jwtSecret,
	}
}

// LoginByStudentID returns a JWT token and user for the given student ID.
func (s *DevLoginService) LoginByStudentID(studentID string) (string, *model.User, error) {
	studentID = strings.TrimSpace(studentID)
	if studentID == "" {
		return "", nil, errors.New("missing student_id")
	}

	user, err := s.userRepo.GetByStudentID(studentID)
	if err != nil {
		return "", nil, err
	}

	token, err := GenerateToken(user.ID, user.Role, user.ClassID, user.Grade, s.jwtSecret)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}
