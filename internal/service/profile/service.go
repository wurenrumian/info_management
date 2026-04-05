package profile

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
)

// HomeData is the aggregated homepage response payload.
type HomeData struct {
	Basic      MeData       `json:"basic"`
	QuickEntry QuickEntry   `json:"quick_entry"`
	Account    AccountState `json:"account"`
}

type MeData struct {
	ID             uint      `json:"id"`
	StudentID      string    `json:"student_id"`
	RealName       string    `json:"real_name"`
	Nickname       string    `json:"nickname"`
	Role           int       `json:"role"`
	Major          string    `json:"major"`
	College        string    `json:"college"`
	EnrollmentYear int       `json:"enrollment_year"`
	Bio            string    `json:"bio"`
	AvatarURL      string    `json:"avatar_url"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// QuickEntry contains quick-access counters for the homepage.
type QuickEntry struct {
	AnnouncementsCount int64 `json:"announcements_count"`
	ApprovalsCount     int64 `json:"approvals_count"`
	KnowledgeCount     int64 `json:"knowledge_count"`
}

// AccountState contains account-level flags.
type AccountState struct {
	WechatBound bool `json:"wechat_bound"`
}

var ErrEmptyPatch = errors.New("empty patch")

type PatchMeInput struct {
	Nickname       *string
	Major          *string
	College        *string
	EnrollmentYear *int
	Bio            *string
	AvatarURL      *string
}

// Service encapsulates profile-related business logic.
type Service struct {
	userRepo      *repo.UserRepo
	knowledgeRepo *repo.KnowledgeRepo
}

// NewService creates a profile service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		userRepo:      repo.NewUserRepo(db),
		knowledgeRepo: repo.NewKnowledgeRepo(db),
	}
}

// GetMe returns the current user according to auth scope.
func (s *Service) GetMe(actor auth.Actor) (*MeData, error) {
	user, err := s.userRepo.GetByIDInScope(authz.Scope{SelfUserID: actor.UserID}, actor.UserID)
	if err != nil {
		return nil, err
	}
	me, err := toMeData(user)
	if err != nil {
		return nil, err
	}
	return me, nil
}

// GetHome returns the aggregated homepage payload for the current user.
func (s *Service) GetHome(actor auth.Actor) (*HomeData, error) {
	user, err := s.userRepo.GetByIDInScope(authz.Scope{SelfUserID: actor.UserID}, actor.UserID)
	if err != nil {
		return nil, err
	}
	meData, err := toMeData(user)
	if err != nil {
		return nil, err
	}

	knowledgeCount, err := s.knowledgeRepo.CountAll()
	if err != nil {
		return nil, err
	}

	return &HomeData{
		Basic: *meData,
		QuickEntry: QuickEntry{
			AnnouncementsCount: 0,
			ApprovalsCount:     0,
			KnowledgeCount:     knowledgeCount,
		},
		Account: AccountState{
			WechatBound: user.OpenID != nil && strings.TrimSpace(*user.OpenID) != "",
		},
	}, nil
}

// PatchMe updates editable profile fields stored in profile_attrs.
func (s *Service) PatchMe(userID uint, input PatchMeInput) (*MeData, error) {
	if userID == 0 {
		return nil, errors.New("invalid user id")
	}
	if input.Nickname == nil &&
		input.Major == nil &&
		input.College == nil &&
		input.EnrollmentYear == nil &&
		input.AvatarURL == nil &&
		input.Bio == nil {
		return nil, ErrEmptyPatch
	}

	user, err := s.userRepo.GetByIDInScope(authz.Scope{SelfUserID: userID}, userID)
	if err != nil {
		return nil, err
	}

	attrs := map[string]any{}
	if len(user.ProfileAttrs) > 0 {
		if err := json.Unmarshal([]byte(user.ProfileAttrs), &attrs); err != nil {
			return nil, err
		}
	}

	updates := map[string]any{}
	if input.Major != nil {
		updates["major"] = strings.TrimSpace(*input.Major)
	}
	if input.College != nil {
		updates["college"] = strings.TrimSpace(*input.College)
	}
	if input.EnrollmentYear != nil {
		updates["enrollment_year"] = *input.EnrollmentYear
	}
	if input.Nickname != nil {
		attrs["nickname"] = strings.TrimSpace(*input.Nickname)
	}
	if input.AvatarURL != nil {
		attrs["avatar_url"] = strings.TrimSpace(*input.AvatarURL)
	}
	if input.Bio != nil {
		attrs["bio"] = strings.TrimSpace(*input.Bio)
	}

	raw, err := json.Marshal(attrs)
	if err != nil {
		return nil, err
	}
	updates["profile_attrs"] = datatypes.JSON(raw)

	if err := s.userRepo.UpdateByID(userID, updates); err != nil {
		return nil, err
	}

	user, err = s.userRepo.GetByIDInScope(authz.Scope{SelfUserID: userID}, userID)
	if err != nil {
		return nil, err
	}
	me, err := toMeData(user)
	if err != nil {
		return nil, err
	}
	return me, nil
}

func toMeData(user *model.User) (*MeData, error) {
	attrs := map[string]any{}
	if len(user.ProfileAttrs) > 0 {
		if err := json.Unmarshal([]byte(user.ProfileAttrs), &attrs); err != nil {
			return nil, err
		}
	}
	return &MeData{
		ID:             user.ID,
		StudentID:      user.StudentID,
		RealName:       user.Name,
		Nickname:       readString(attrs["nickname"]),
		Role:           user.Role,
		Major:          user.Major,
		College:        user.College,
		EnrollmentYear: user.EnrollmentYear,
		Bio:            readString(attrs["bio"]),
		AvatarURL:      normalizeAvatarURL(readString(attrs["avatar_url"])),
		UpdatedAt:      user.UpdatedAt,
	}, nil
}

func readString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

var legacyUploadsPrefixPattern = regexp.MustCompile(`^/uploads/documents/(avatars|images|knowledge|announcements)/`)

func normalizeAvatarURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Backward compatibility for legacy paths generated under /uploads/documents/<category>/...
	return legacyUploadsPrefixPattern.ReplaceAllString(raw, "/uploads/$1/")
}
