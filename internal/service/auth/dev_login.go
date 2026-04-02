package auth

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
)

// DevLoginService handles development-only token issuance.
type DevLoginService struct {
	userRepo  *repo.UserRepo
	jwtSecret string
}

const (
	defaultDevClassID = 10
	defaultDevGrade   = "2020"
	defaultDevMajor   = "信息管理"
)

// NewDevLoginService creates a DevLoginService.
func NewDevLoginService(db *gorm.DB, jwtSecret string) *DevLoginService {
	return &DevLoginService{
		userRepo:  repo.NewUserRepo(db),
		jwtSecret: jwtSecret,
	}
}

// RegisterOrLogin returns a JWT token and dev user for the given student ID.
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
		if !authz.Authorize(*role, authz.ActionGetMe) && *role != model.RoleSuperAdmin {
			return 0, errors.New("invalid role")
		}
		return *role, nil
	default:
		return 0, errors.New("invalid role")
	}
}
