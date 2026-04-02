package auth

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"manage/internal/model"
	"manage/internal/repo"
)

const (
	defaultDevClassID = 10
	defaultDevGrade   = "2020"
	defaultDevMajor   = "信息管理"
)

// DevLoginService handles development-only test user issuance.
type DevLoginService struct {
	userRepo  *repo.UserRepo
	classRepo *repo.ClassRepo
	jwtSecret string
}

// NewDevLoginService creates a DevLoginService.
func NewDevLoginService(db *gorm.DB, jwtSecret string) *DevLoginService {
	return &DevLoginService{
		userRepo:  repo.NewUserRepo(db),
		classRepo: repo.NewClassRepo(db),
		jwtSecret: jwtSecret,
	}
}

// RegisterOrLogin returns a JWT token for an existing user or creates a dev user first.
func (s *DevLoginService) RegisterOrLogin(studentID string, role *int) (string, *model.User, error) {
	studentID = strings.TrimSpace(studentID)
	if studentID == "" {
		return "", nil, errors.New("missing student_id")
	}

	user, err := s.userRepo.GetByStudentID(studentID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, err
		}

		devRole, err := normalizeDevRole(role)
		if err != nil {
			return "", nil, err
		}
		if err := s.ensureDefaultDevClass(); err != nil {
			return "", nil, err
		}

		user = &model.User{
			StudentID: studentID,
			Name:      "Dev-" + studentID,
			Role:      devRole,
			ClassID:   defaultDevClassID,
			Grade:     defaultDevGrade,
			Major:     defaultDevMajor,
		}
		if err := s.userRepo.Create(user); err != nil {
			return "", nil, err
		}
	}

	token, err := GenerateToken(user.ID, user.Role, user.ClassID, user.Grade, s.jwtSecret)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func normalizeDevRole(role *int) (int, error) {
	if role == nil {
		return model.RoleStudent, nil
	}
	switch *role {
	case model.RoleStudent, model.RoleCadre, model.RoleTeacher, model.RoleSuperAdmin:
		return *role, nil
	default:
		return 0, errors.New("invalid role")
	}
}

func (s *DevLoginService) ensureDefaultDevClass() error {
	_, err := s.classRepo.GetByID(defaultDevClassID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return s.classRepo.Create(&model.Class{
		ID:        defaultDevClassID,
		ClassName: "Dev Class 10",
		Grade:     defaultDevGrade,
		Major:     defaultDevMajor,
	})
}
