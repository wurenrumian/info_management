package announcements

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
	"manage/internal/service/notification"

	"gorm.io/gorm"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"

	AudienceAll      = "all"
	AudienceTargeted = "targeted"
)

var (
	ErrInvalidAudienceType = errors.New("invalid audience_type")
	ErrInvalidStatus       = errors.New("invalid status")
	ErrAnnouncementState   = errors.New("invalid announcement state")
)

type Service struct {
	repo     *repo.AnnouncementRepo
	userRepo *repo.UserRepo
	notifSvc *notification.Service
}

type TargetScope struct {
	Grades     []json.RawMessage `json:"grades"`
	Majors     []json.RawMessage `json:"majors"`
	ClassIDs   []uint            `json:"class_ids"`
	Roles      []int             `json:"roles"`
	StudentIDs []json.RawMessage `json:"student_ids"`
}

type CreateRequest struct {
	Title             string          `json:"title"`
	Content           string          `json:"content"`
	AudienceType      string          `json:"audience_type"`
	TargetScope       json.RawMessage `json:"target_scope"`
	Tags              json.RawMessage `json:"tags"`
	AttachmentFileIDs json.RawMessage `json:"attachment_file_ids"`
	ExternalLinks     json.RawMessage `json:"external_links"`
}

type ListRequest struct {
	Status string
	Limit  int
	Offset int
}

type PublishRequest struct {
	SendNotification bool
	TemplateCode     string
}

type NotificationFailure struct {
	UserID uint   `json:"user_id"`
	Error  string `json:"error"`
}

type PublishNotificationSummary struct {
	Attempted  int                   `json:"attempted"`
	Sent       int                   `json:"sent"`
	Failed     int                   `json:"failed"`
	FirstError string                `json:"first_error,omitempty"`
	Failures   []NotificationFailure `json:"failures,omitempty"`
}

func NewService(db *gorm.DB, notifSvc *notification.Service) *Service {
	return &Service{
		repo:     repo.NewAnnouncementRepo(db),
		userRepo: repo.NewUserRepo(db),
		notifSvc: notifSvc,
	}
}

func (s *Service) Create(actor auth.Actor, req CreateRequest) (*model.Announcement, error) {
	audienceType := strings.TrimSpace(req.AudienceType)
	if audienceType == "" {
		audienceType = AudienceAll
	}
	if audienceType != AudienceAll && audienceType != AudienceTargeted {
		return nil, ErrInvalidAudienceType
	}

	targetScope := normalizeJSON(req.TargetScope)
	if audienceType == AudienceAll {
		targetScope = []byte(`{}`)
	}

	item := &model.Announcement{
		Title:             strings.TrimSpace(req.Title),
		Content:           strings.TrimSpace(req.Content),
		Status:            StatusDraft,
		AudienceType:      audienceType,
		TargetScope:       targetScope,
		Tags:              normalizeJSON(req.Tags),
		AttachmentFileIDs: normalizeJSON(req.AttachmentFileIDs),
		ExternalLinks:     normalizeJSON(req.ExternalLinks),
		AuthorID:          actor.UserID,
	}
	if err := s.repo.Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) ListForStudent(actor auth.Actor, limit, offset int) ([]model.Announcement, int64, error) {
	// Current implementation filters in memory to apply audience scope matching.
	all, _, err := s.repo.ListWithTotal(StatusPublished, 10000, 0)
	if err != nil {
		return nil, 0, err
	}

	// Get fresh user data from database to ensure accurate grade, major, and student_id
	actorGrade, err := s.getActorGrade(actor.UserID)
	if err != nil {
		return nil, 0, err
	}
	actor.Grade = actorGrade

	actorMajor, err := s.getActorMajor(actor.UserID)
	if err != nil {
		return nil, 0, err
	}

	actorStudentID, _ := s.getActorStudentID(actor.UserID)
	actor.StudentID = actorStudentID

	filtered := make([]model.Announcement, 0, len(all))
	for _, item := range all {
		ok, err := matchAnnouncementForActor(item, actor, actorMajor)
		if err != nil {
			return nil, 0, err
		}
		if ok {
			filtered = append(filtered, item)
		}
	}

	total := int64(len(filtered))
	limit, offset = normalizeLimitOffset(limit, offset)
	if offset >= len(filtered) {
		return []model.Announcement{}, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

// ListAllPublished returns all published announcements without audience scope filtering.
// It is intended for privileged roles (teacher/super admin) that need full visibility.
func (s *Service) ListAllPublished(limit, offset int) ([]model.Announcement, int64, error) {
	return s.repo.ListWithTotal(StatusPublished, limit, offset)
}

// GetAllPublishedByID returns one published announcement without audience scope filtering.
// It is intended for privileged roles (teacher/super admin) that need full visibility.
func (s *Service) GetAllPublishedByID(id uint) (*model.Announcement, error) {
	item, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if item.Status != StatusPublished {
		return nil, gorm.ErrRecordNotFound
	}
	return item, nil
}

func (s *Service) GetForStudent(actor auth.Actor, id uint) (*model.Announcement, error) {
	item, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if item.Status != StatusPublished {
		return nil, gorm.ErrRecordNotFound
	}
	// Get fresh user data from database to ensure accurate grade, major, and student_id
	actorGrade, err := s.getActorGrade(actor.UserID)
	if err != nil {
		return nil, err
	}
	actor.Grade = actorGrade

	actorMajor, err := s.getActorMajor(actor.UserID)
	if err != nil {
		return nil, err
	}
	actorStudentID, _ := s.getActorStudentID(actor.UserID)
	actor.StudentID = actorStudentID
	ok, err := matchAnnouncementForActor(*item, actor, actorMajor)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return item, nil
}

func (s *Service) GetForAdmin(id uint) (*model.Announcement, error) {
	return s.repo.GetByID(id)
}

func (s *Service) ListForAdmin(req ListRequest) ([]model.Announcement, int64, error) {
	status := strings.TrimSpace(req.Status)
	if status != "" && status != StatusDraft && status != StatusPublished && status != StatusArchived {
		return nil, 0, ErrInvalidStatus
	}
	return s.repo.ListAdminWithTotal(status, req.Limit, req.Offset)
}

func (s *Service) Patch(id uint, updates map[string]any) error {
	return s.repo.Patch(id, updates)
}

func (s *Service) Publish(ctx context.Context, id uint, req PublishRequest) (*model.Announcement, *PublishNotificationSummary, error) {
	item, err := s.repo.GetByID(id)
	if err != nil {
		return nil, nil, err
	}
	if item.Status == StatusArchived {
		return nil, nil, fmt.Errorf("%w: archived item can not be published", ErrAnnouncementState)
	}

	now := time.Now()
	if err := s.repo.Publish(id, now); err != nil {
		return nil, nil, err
	}
	item.Status = StatusPublished
	item.PublishedAt = &now

	var summary *PublishNotificationSummary

	if req.SendNotification && s.notifSvc != nil {
		templateCode := strings.TrimSpace(req.TemplateCode)
		if templateCode == "" {
			templateCode = "announcement"
		}
		userIDs, err := s.pickRecipients(*item)
		if err != nil {
			return nil, nil, err
		}
		summary = &PublishNotificationSummary{
			Attempted: len(userIDs),
		}
		for _, userID := range userIDs {
			err := s.notifSvc.Send(ctx, notification.SendRequest{
				UserID:       userID,
				TemplateCode: templateCode,
				Page:         "/pages/announcement/detail?id=" + fmt.Sprintf("%d", item.ID),
				TemplateData: map[string]interface{}{
					"thing1": map[string]string{"value": truncateString(item.Content, 20)},
					"time3":  map[string]string{"value": now.Format("2006-01-02 15:04")},
					"thing4": map[string]string{"value": truncateString(item.Title, 20)},
				},
			})
			if err != nil {
				errMsg := err.Error()
				if summary.FirstError == "" {
					summary.FirstError = errMsg
				}
				summary.Failures = append(summary.Failures, NotificationFailure{
					UserID: userID,
					Error:  errMsg,
				})
				continue
			}
			summary.Sent++
		}
		summary.Failed = len(summary.Failures)
	}

	return item, summary, nil
}

func (s *Service) Archive(id uint) error {
	return s.repo.Archive(id)
}

func (s *Service) getActorMajor(userID uint) (string, error) {
	u, err := s.userRepo.GetByIDInScope(authz.Scope{AllowAll: true}, userID)
	if err != nil {
		return "", err
	}
	major := strings.TrimSpace(u.Major)
	if major == "" {
		major = strings.TrimSpace(u.Class.Major)
	}
	return major, nil
}

func (s *Service) getActorGrade(userID uint) (string, error) {
	u, err := s.userRepo.GetByIDInScope(authz.Scope{AllowAll: true}, userID)
	if err != nil {
		return "", err
	}
	// Use class.grade as the authoritative source per grade governance spec
	grade := strings.TrimSpace(u.Class.Grade)
	if grade == "" {
		grade = strings.TrimSpace(u.Grade)
	}
	return grade, nil
}

func (s *Service) getActorStudentID(userID uint) (string, error) {
	u, err := s.userRepo.GetByIDInScope(authz.Scope{AllowAll: true}, userID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(u.StudentID), nil
}

func (s *Service) pickRecipients(item model.Announcement) ([]uint, error) {
	users, _, err := s.userRepo.ListByScopeWithTotal(authz.Scope{AllowAll: true}, 10000, 0)
	if err != nil {
		return nil, err
	}

	out := make([]uint, 0, len(users))
	for _, u := range users {
		major := strings.TrimSpace(u.Major)
		if major == "" {
			major = strings.TrimSpace(u.Class.Major)
		}
		actor := auth.Actor{
			UserID:    u.ID,
			StudentID: u.StudentID,
			Role:      u.Role,
			ClassID:   u.ClassID,
			Grade:     u.Grade,
		}
		ok, err := matchAnnouncementForActor(item, actor, major)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, u.ID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func matchAnnouncementForActor(item model.Announcement, actor auth.Actor, major string) (bool, error) {
	switch item.AudienceType {
	case AudienceAll:
		return true, nil
	case AudienceTargeted:
		scope, err := parseTargetScope(item.TargetScope)
		if err != nil {
			return false, err
		}
		return matchTargetScope(actor, major, scope), nil
	default:
		return false, ErrInvalidAudienceType
	}
}

func parseTargetScope(raw []byte) (TargetScope, error) {
	raw = normalizeJSON(raw)
	if string(raw) == "{}" {
		return TargetScope{}, nil
	}
	var scope TargetScope
	if err := json.Unmarshal(raw, &scope); err != nil {
		return TargetScope{}, err
	}
	return scope, nil
}

func matchTargetScope(actor auth.Actor, major string, scope TargetScope) bool {
	return containsStringValue(scope.Grades, actor.Grade) &&
		containsStringValue(scope.Majors, strings.TrimSpace(major)) &&
		containsUint(scope.ClassIDs, actor.ClassID) &&
		containsInt(scope.Roles, actor.Role) &&
		containsStudentID(scope.StudentIDs, actor.StudentID)
}

// containsStringValue checks if the actor's value matches any value in the scope array.
// Supports both string and integer values in the scope array.
func containsStringValue(vals []json.RawMessage, actorValue string) bool {
	if len(vals) == 0 {
		return true
	}
	actorValue = strings.TrimSpace(actorValue)
	for _, v := range vals {
		// Try to parse as string first
		var strVal string
		if err := json.Unmarshal(v, &strVal); err == nil {
			if strings.TrimSpace(strVal) == actorValue {
				return true
			}
			continue
		}
		// Try to parse as int (for legacy data with integer values)
		var intVal int64
		if err := json.Unmarshal(v, &intVal); err == nil {
			if strconv.FormatInt(intVal, 10) == actorValue {
				return true
			}
		}
	}
	return false
}

// containsStudentID checks if actor's student ID is in the scope.
// Supports both string and integer values in the scope array.
func containsStudentID(vals []json.RawMessage, actorStudentID string) bool {
	if len(vals) == 0 {
		return true
	}
	actorStudentID = strings.TrimSpace(actorStudentID)
	for _, v := range vals {
		// Try to parse as string first
		var strVal string
		if err := json.Unmarshal(v, &strVal); err == nil {
			if strings.TrimSpace(strVal) == actorStudentID {
				return true
			}
			continue
		}
		// Try to parse as int (for legacy data with integer student_ids)
		var intVal int64
		if err := json.Unmarshal(v, &intVal); err == nil {
			if strconv.FormatInt(intVal, 10) == actorStudentID {
				return true
			}
		}
	}
	return false
}

func normalizeJSON(raw []byte) []byte {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return []byte("{}")
	}
	return raw
}

func normalizeLimitOffset(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func containsString(vals []string, v string) bool {
	if len(vals) == 0 {
		return true
	}
	for _, each := range vals {
		if each == v {
			return true
		}
	}
	return false
}

func containsUint(vals []uint, v uint) bool {
	if len(vals) == 0 {
		return true
	}
	for _, each := range vals {
		if each == v {
			return true
		}
	}
	return false
}

func containsInt(vals []int, v int) bool {
	if len(vals) == 0 {
		return true
	}
	for _, each := range vals {
		if each == v {
			return true
		}
	}
	return false
}

// truncateString truncates a string to at most maxLen runes (not bytes).
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
