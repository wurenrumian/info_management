package repo

import (
	"strconv"
	"strings"
	"time"

	"manage/internal/model"
	"manage/internal/service/authz"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PartyflowStatusRepo handles partyflow status persistence.
type PartyflowStatusRepo struct {
	db *gorm.DB
}

// NewPartyflowStatusRepo creates repo instance.
func NewPartyflowStatusRepo(db *gorm.DB) *PartyflowStatusRepo {
	return &PartyflowStatusRepo{db: db}
}

// PartyflowAdminListItem is admin-facing row with joined student info.
type PartyflowAdminListItem struct {
	model.PartyflowStatus
	StudentID   string `json:"student_id"`
	StudentName string `json:"student_name"`
}

func normalizePage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func applyPartyflowScope(q *gorm.DB, scope authz.Scope) *gorm.DB {
	switch {
	case scope.AllowAll:
	case scope.ClassID > 0 && scope.Grade != "":
		q = q.Where("u.class_id = ? OR u.grade = ?", scope.ClassID, scope.Grade)
	case scope.ClassID > 0:
		q = q.Where("u.class_id = ?", scope.ClassID)
	case scope.Grade != "":
		q = q.Where("u.grade = ?", scope.Grade)
	default:
		q = q.Where("1 = 0")
	}
	return q
}

// ListMine returns statuses owned by one user.
func (r *PartyflowStatusRepo) ListMine(userID uint) ([]model.PartyflowStatus, error) {
	var out []model.PartyflowStatus
	err := r.db.
		Where("user_id = ?", userID).
		Preload("History", func(tx *gorm.DB) *gorm.DB { return tx.Order("happened_at desc") }).
		Order("id desc").
		Find(&out).Error
	return out, err
}

// AdminListFilter defines list filters for admin statuses.
type AdminListFilter struct {
	OrgType   string
	Status    string
	StudentID string
	Limit     int
	Offset    int
}

// PartyflowRuleListFilter defines rule list filters.
type PartyflowRuleListFilter struct {
	OrgType  string
	Enabled  *bool
	RuleCode string
}

// ListAdmin returns scoped admin list with student fields.
func (r *PartyflowStatusRepo) ListAdmin(scope authz.Scope, filter AdminListFilter) ([]PartyflowAdminListItem, int64, error) {
	limit, offset := normalizePage(filter.Limit, filter.Offset)

	base := r.db.
		Table("partyflow_statuses s").
		Joins("JOIN users u ON u.id = s.user_id")

	base = applyPartyflowScope(base, scope)

	if v := strings.TrimSpace(filter.OrgType); v != "" {
		base = base.Where("s.org_type = ?", v)
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		base = base.Where("s.status = ?", v)
	}
	if v := strings.TrimSpace(filter.StudentID); v != "" {
		base = base.Where("u.student_id = ?", v)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []PartyflowAdminListItem
	err := base.
		Select("s.*, u.student_id AS student_id, u.name AS student_name").
		Order("s.id desc").
		Limit(limit).
		Offset(offset).
		Scan(&out).Error
	return out, total, err
}

// GetByIDInScope returns one status detail with history and user basic fields.
func (r *PartyflowStatusRepo) GetByIDInScope(scope authz.Scope, id uint) (*PartyflowAdminListItem, error) {
	base := r.db.
		Table("partyflow_statuses s").
		Joins("JOIN users u ON u.id = s.user_id").
		Where("s.id = ?", id)
	base = applyPartyflowScope(base, scope)

	var out PartyflowAdminListItem
	if err := base.Select("s.*, u.student_id AS student_id, u.name AS student_name").Scan(&out).Error; err != nil {
		return nil, err
	}
	if out.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var history []model.PartyflowEvent
	if err := r.db.Where("partyflow_status_id = ?", out.ID).Order("happened_at desc").Find(&history).Error; err != nil {
		return nil, err
	}
	out.History = history
	return &out, nil
}

// Create inserts a new status row.
func (r *PartyflowStatusRepo) Create(item *model.PartyflowStatus) error {
	return r.db.Create(item).Error
}

// Patch updates status row by id.
func (r *PartyflowStatusRepo) Patch(id uint, updates map[string]any) error {
	return UpdateByID(r.db.Model(&model.PartyflowStatus{}), id, updates)
}

// CreateEvent inserts one event.
func (r *PartyflowStatusRepo) CreateEvent(item *model.PartyflowEvent) error {
	return r.db.Create(item).Error
}

// GetByID returns one status by primary key.
func (r *PartyflowStatusRepo) GetByID(id uint) (*model.PartyflowStatus, error) {
	var out model.PartyflowStatus
	if err := r.db.Where("id = ?", id).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// ListByOrgStatus returns status rows by org and status for reminder scanning.
func (r *PartyflowStatusRepo) ListByOrgStatus(orgType, status string) ([]model.PartyflowStatus, error) {
	var out []model.PartyflowStatus
	err := r.db.Where("org_type = ? AND status = ?", strings.TrimSpace(orgType), strings.TrimSpace(status)).
		Order("id asc").
		Find(&out).Error
	return out, err
}

// GetByUserAndOrg returns one status by logical unique key.
func (r *PartyflowStatusRepo) GetByUserAndOrg(userID uint, orgType string) (*model.PartyflowStatus, error) {
	var out model.PartyflowStatus
	if err := r.db.Where("user_id = ? AND org_type = ?", userID, strings.TrimSpace(orgType)).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEnabledRules returns all enabled reminder rules.
func (r *PartyflowStatusRepo) ListEnabledRules() ([]model.PartyflowReminderRule, error) {
	var out []model.PartyflowReminderRule
	err := r.db.Where("enabled = ?", true).Order("id asc").Find(&out).Error
	return out, err
}

// ListRules returns reminder rules with optional filters.
func (r *PartyflowStatusRepo) ListRules(filter PartyflowRuleListFilter) ([]model.PartyflowReminderRule, error) {
	q := r.db.Model(&model.PartyflowReminderRule{})
	if v := strings.TrimSpace(filter.OrgType); v != "" {
		q = q.Where("org_type = ?", v)
	}
	if filter.Enabled != nil {
		q = q.Where("enabled = ?", *filter.Enabled)
	}
	if v := strings.TrimSpace(filter.RuleCode); v != "" {
		q = q.Where("rule_code = ?", v)
	}
	var out []model.PartyflowReminderRule
	if err := q.Order("id asc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// GetRuleByID returns one reminder rule.
func (r *PartyflowStatusRepo) GetRuleByID(id uint) (*model.PartyflowReminderRule, error) {
	var out model.PartyflowReminderRule
	if err := r.db.Where("id = ?", id).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// PatchRule updates one reminder rule.
func (r *PartyflowStatusRepo) PatchRule(id uint, updates map[string]any) error {
	return UpdateByID(r.db.Model(&model.PartyflowReminderRule{}), id, updates)
}

// SeedReminderRules inserts default rules idempotently by rule_code.
func (r *PartyflowStatusRepo) SeedReminderRules(items []model.PartyflowReminderRule) error {
	for _, item := range items {
		var found model.PartyflowReminderRule
		err := r.db.Where("rule_code = ?", item.RuleCode).First(&found).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		if err := r.db.Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

// ReminderEventExists checks one reminder period has already been emitted.
func (r *PartyflowStatusRepo) ReminderEventExists(statusID uint, ruleCode string, periodIndex int) (bool, error) {
	needle := `"period_index":` + strconv.Itoa(periodIndex)
	var count int64
	err := r.db.Model(&model.PartyflowEvent{}).
		Where("partyflow_status_id = ? AND event_type = ? AND event_code = ?", statusID, "reminder_sent", strings.TrimSpace(ruleCode)).
		Where("metadata LIKE ?", "%"+needle+"%").
		Count(&count).Error
	return count > 0, err
}

// NewStatusFromCreate builds status model with normalized defaults.
func NewStatusFromCreate(userID uint, orgType, status string, statusStartedAt time.Time, joinedAt *time.Time, nextActionHint string, metadata datatypes.JSON) *model.PartyflowStatus {
	return &model.PartyflowStatus{
		UserID:          userID,
		OrgType:         orgType,
		Status:          status,
		StatusStartedAt: statusStartedAt,
		JoinedAt:        joinedAt,
		NextActionHint:  nextActionHint,
		Metadata:        metadata,
	}
}
