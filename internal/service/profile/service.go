package profile

import (
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
)

// HomeData is the aggregated homepage response payload.
type HomeData struct {
	Basic      model.User   `json:"basic"`
	QuickEntry QuickEntry   `json:"quick_entry"`
	Account    AccountState `json:"account"`
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
func (s *Service) GetMe(actor auth.Actor) (*model.User, error) {
	return s.userRepo.GetByIDInScope(authz.Scope{SelfUserID: actor.UserID}, actor.UserID)
}

// GetHome returns the aggregated homepage payload for the current user.
func (s *Service) GetHome(actor auth.Actor) (*HomeData, error) {
	user, err := s.GetMe(actor)
	if err != nil {
		return nil, err
	}

	knowledgeCount, err := s.knowledgeRepo.CountAll()
	if err != nil {
		return nil, err
	}

	return &HomeData{
		Basic: *user,
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
func (s *Service) PatchMe(userID uint, avatarURL, bio *string) error {
	if userID == 0 {
		return errors.New("invalid user id")
	}
	if avatarURL == nil && bio == nil {
		return ErrEmptyPatch
	}

	user, err := s.userRepo.GetByIDInScope(authz.Scope{SelfUserID: userID}, userID)
	if err != nil {
		return err
	}

	attrs := map[string]any{}
	if len(user.ProfileAttrs) > 0 {
		if err := json.Unmarshal([]byte(user.ProfileAttrs), &attrs); err != nil {
			return err
		}
	}

	if avatarURL != nil {
		attrs["avatar_url"] = strings.TrimSpace(*avatarURL)
	}
	if bio != nil {
		attrs["bio"] = strings.TrimSpace(*bio)
	}

	raw, err := json.Marshal(attrs)
	if err != nil {
		return err
	}

	return s.userRepo.UpdateByID(userID, map[string]any{
		"profile_attrs": datatypes.JSON(raw),
	})
}
