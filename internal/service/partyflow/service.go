package partyflow

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrInvalidOrgType   = errors.New("invalid org_type")
	ErrInvalidStatus    = errors.New("invalid status")
	ErrInvalidEventType = errors.New("invalid event_type")
)

var (
	leagueStatuses = map[string]struct{}{
		"none": {}, "applicant": {}, "activist": {}, "development_target": {}, "member": {},
	}
	partyStatuses = map[string]struct{}{
		"none": {}, "applicant": {}, "activist": {}, "development_target": {}, "probationary_member": {}, "full_member": {},
	}
	statusLabels = map[string]string{
		"none":                "未开始",
		"applicant":           "申请人",
		"activist":            "积极分子",
		"development_target":  "发展对象",
		"probationary_member": "预备党员",
		"full_member":         "正式党员",
		"member":              "团员",
	}
)

// Service contains partyflow business logic.
type Service struct {
	statusRepo *repo.PartyflowStatusRepo
	userRepo   *repo.UserRepo
}

// NewService creates partyflow service.
func NewService(db *gorm.DB) *Service {
	s := &Service{
		statusRepo: repo.NewPartyflowStatusRepo(db),
		userRepo:   repo.NewUserRepo(db),
	}
	if db != nil && db.Migrator().HasTable(&model.PartyflowReminderRule{}) {
		_ = s.statusRepo.SeedReminderRules(defaultReminderRules())
	}
	return s
}

// ListMine returns current user's partyflow statuses.
func (s *Service) ListMine(actor auth.Actor) ([]model.PartyflowStatus, error) {
	return s.statusRepo.ListMine(actor.UserID)
}

// AdminListParams defines list filters.
type AdminListParams struct {
	OrgType   string
	Status    string
	StudentID string
	Limit     int
	Offset    int
}

// ListAdmin returns scoped admin status list.
func (s *Service) ListAdmin(actor auth.Actor, params AdminListParams) ([]repo.PartyflowAdminListItem, int64, error) {
	orgType := strings.TrimSpace(params.OrgType)
	if orgType != "" && !isValidOrgType(orgType) {
		return nil, 0, ErrInvalidOrgType
	}
	if v := strings.TrimSpace(params.Status); v != "" && !isValidStatus(orgType, v) {
		return nil, 0, ErrInvalidStatus
	}
	return s.statusRepo.ListAdmin(authz.BuildScope(actor), repo.AdminListFilter{
		OrgType:   orgType,
		Status:    strings.TrimSpace(params.Status),
		StudentID: strings.TrimSpace(params.StudentID),
		Limit:     params.Limit,
		Offset:    params.Offset,
	})
}

// GetAdmin returns one scoped admin status detail.
func (s *Service) GetAdmin(actor auth.Actor, id uint) (*repo.PartyflowAdminListItem, error) {
	return s.statusRepo.GetByIDInScope(authz.BuildScope(actor), id)
}

// CreateStatusRequest is create payload.
type CreateStatusRequest struct {
	UserID          uint            `json:"user_id"`
	OrgType         string          `json:"org_type"`
	Status          string          `json:"status"`
	StatusStartedAt time.Time       `json:"status_started_at"`
	JoinedAt        *time.Time      `json:"joined_at"`
	NextActionHint  string          `json:"next_action_hint"`
	Metadata        json.RawMessage `json:"metadata"`
	Note            string          `json:"note"`
}

// CreateStatus creates a status and an init event.
func (s *Service) CreateStatus(actor auth.Actor, req CreateStatusRequest) (*repo.PartyflowAdminListItem, error) {
	orgType := strings.TrimSpace(req.OrgType)
	status := strings.TrimSpace(req.Status)
	if !isValidOrgType(orgType) {
		return nil, ErrInvalidOrgType
	}
	if !isValidStatus(orgType, status) {
		return nil, ErrInvalidStatus
	}

	// Check target user is visible under caller scope.
	if _, err := s.userRepo.GetByIDInScope(authz.BuildScope(actor), req.UserID); err != nil {
		return nil, err
	}

	startedAt := req.StatusStartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	item := repo.NewStatusFromCreate(
		req.UserID,
		orgType,
		status,
		startedAt,
		req.JoinedAt,
		strings.TrimSpace(req.NextActionHint),
		normalizeJSON(req.Metadata),
	)
	if err := s.statusRepo.Create(item); err != nil {
		return nil, err
	}

	note := strings.TrimSpace(req.Note)
	if note == "" {
		note = "初始化：" + statusLabel(status)
	}
	_ = s.statusRepo.CreateEvent(&model.PartyflowEvent{
		PartyflowStatusID: item.ID,
		EventType:         "create",
		EventCode:         "status_initialized",
		ToStatus:          item.Status,
		Note:              note,
		HappenedAt:        item.StatusStartedAt,
	})

	return s.GetAdmin(actor, item.ID)
}

// PatchStatusRequest is patch payload.
type PatchStatusRequest struct {
	Status          *string         `json:"status"`
	StatusStartedAt *time.Time      `json:"status_started_at"`
	JoinedAt        *time.Time      `json:"joined_at"`
	NextActionHint  *string         `json:"next_action_hint"`
	Metadata        json.RawMessage `json:"metadata"`
	Note            *string         `json:"note"`
}

// PatchStatus updates one status and appends event when status changed.
func (s *Service) PatchStatus(actor auth.Actor, id uint, req PatchStatusRequest) (*repo.PartyflowAdminListItem, error) {
	before, err := s.statusRepo.GetByIDInScope(authz.BuildScope(actor), id)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if !isValidStatus(before.OrgType, status) {
			return nil, ErrInvalidStatus
		}
		updates["status"] = status
	}
	if req.StatusStartedAt != nil {
		updates["status_started_at"] = *req.StatusStartedAt
	}
	if req.JoinedAt != nil {
		updates["joined_at"] = *req.JoinedAt
	}
	if req.NextActionHint != nil {
		updates["next_action_hint"] = strings.TrimSpace(*req.NextActionHint)
	}
	if len(req.Metadata) > 0 {
		updates["metadata"] = normalizeJSON(req.Metadata)
	}
	if len(updates) == 0 {
		return s.GetAdmin(actor, id)
	}

	if err := s.statusRepo.Patch(id, updates); err != nil {
		return nil, err
	}

	if req.Status != nil && strings.TrimSpace(*req.Status) != before.Status {
		eventNote := ""
		if req.Note != nil {
			eventNote = strings.TrimSpace(*req.Note)
		}
		if eventNote == "" {
			eventNote = "阶段变更：" + statusLabel(before.Status) + " → " + statusLabel(strings.TrimSpace(*req.Status))
		}
		happenedAt := time.Now()
		if req.StatusStartedAt != nil {
			happenedAt = *req.StatusStartedAt
		}
		_ = s.statusRepo.CreateEvent(&model.PartyflowEvent{
			PartyflowStatusID: id,
			EventType:         "status_change",
			EventCode:         "status_patched",
			FromStatus:        before.Status,
			ToStatus:          strings.TrimSpace(*req.Status),
			Note:              eventNote,
			HappenedAt:        happenedAt,
		})
	}

	return s.GetAdmin(actor, id)
}

func normalizeJSON(raw []byte) datatypes.JSON {
	if len(strings.TrimSpace(string(raw))) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(raw)
}

// ImportStatusItem is one row in bulk import.
type ImportStatusItem struct {
	StudentID       string          `json:"student_id"`
	OrgType         string          `json:"org_type"`
	Status          string          `json:"status"`
	StatusStartedAt time.Time       `json:"status_started_at"`
	JoinedAt        *time.Time      `json:"joined_at"`
	NextActionHint  string          `json:"next_action_hint"`
	Metadata        json.RawMessage `json:"metadata"`
	Note            string          `json:"note"`
}

// ImportStatusesRequest is bulk import payload.
type ImportStatusesRequest struct {
	Items []ImportStatusItem `json:"items"`
}

// ImportFailedItem keeps per-row import error.
type ImportFailedItem struct {
	StudentID string `json:"student_id"`
	Reason    string `json:"reason"`
}

// ImportResult is bulk import summary.
type ImportResult struct {
	SuccessCount int                `json:"success_count"`
	FailedCount  int                `json:"failed_count"`
	FailedItems  []ImportFailedItem `json:"failed_items"`
}

// ImportStatuses upserts statuses by student_id + org_type.
func (s *Service) ImportStatuses(actor auth.Actor, req ImportStatusesRequest) (ImportResult, error) {
	result := ImportResult{FailedItems: make([]ImportFailedItem, 0)}
	for _, item := range req.Items {
		studentID := strings.TrimSpace(item.StudentID)
		orgType := strings.TrimSpace(item.OrgType)
		status := strings.TrimSpace(item.Status)
		if studentID == "" || !isValidOrgType(orgType) || !isValidStatus(orgType, status) {
			result.FailedItems = append(result.FailedItems, ImportFailedItem{StudentID: studentID, Reason: "invalid row"})
			continue
		}
		user, err := s.userRepo.GetByStudentID(studentID)
		if err != nil {
			result.FailedItems = append(result.FailedItems, ImportFailedItem{StudentID: studentID, Reason: "user not found"})
			continue
		}
		if _, err := s.userRepo.GetByIDInScope(authz.BuildScope(actor), user.ID); err != nil {
			result.FailedItems = append(result.FailedItems, ImportFailedItem{StudentID: studentID, Reason: "out of scope"})
			continue
		}
		existing, err := s.statusRepo.GetByUserAndOrg(user.ID, orgType)
		if err != nil && err != gorm.ErrRecordNotFound {
			result.FailedItems = append(result.FailedItems, ImportFailedItem{StudentID: studentID, Reason: "query failed"})
			continue
		}
		happenedAt := item.StatusStartedAt
		if happenedAt.IsZero() {
			happenedAt = time.Now()
		}
		if err == gorm.ErrRecordNotFound {
			created := repo.NewStatusFromCreate(user.ID, orgType, status, happenedAt, item.JoinedAt, strings.TrimSpace(item.NextActionHint), normalizeJSON(item.Metadata))
			if e := s.statusRepo.Create(created); e != nil {
				result.FailedItems = append(result.FailedItems, ImportFailedItem{StudentID: studentID, Reason: "create failed"})
				continue
			}
			note := strings.TrimSpace(item.Note)
			if note == "" {
				note = "批量导入：" + statusLabel(status)
			}
			_ = s.statusRepo.CreateEvent(&model.PartyflowEvent{
				PartyflowStatusID: created.ID,
				EventType:         "import",
				EventCode:         "status_import",
				ToStatus:          status,
				Note:              note,
				HappenedAt:        happenedAt,
			})
			result.SuccessCount++
			continue
		}
		updates := map[string]any{
			"status":            status,
			"status_started_at": happenedAt,
			"joined_at":         item.JoinedAt,
			"next_action_hint":  strings.TrimSpace(item.NextActionHint),
			"metadata":          normalizeJSON(item.Metadata),
		}
		if e := s.statusRepo.Patch(existing.ID, updates); e != nil {
			result.FailedItems = append(result.FailedItems, ImportFailedItem{StudentID: studentID, Reason: "update failed"})
			continue
		}
		note := strings.TrimSpace(item.Note)
		if note == "" {
			note = "批量导入：" + statusLabel(existing.Status) + " → " + statusLabel(status)
		}
		_ = s.statusRepo.CreateEvent(&model.PartyflowEvent{
			PartyflowStatusID: existing.ID,
			EventType:         "import",
			EventCode:         "status_import",
			FromStatus:        existing.Status,
			ToStatus:          status,
			Note:              note,
			HappenedAt:        happenedAt,
		})
		result.SuccessCount++
	}
	result.FailedCount = len(result.FailedItems)
	return result, nil
}

// CreateEventRequest is payload for milestone/manual event.
type CreateEventRequest struct {
	EventType  string          `json:"event_type"`
	EventCode  string          `json:"event_code"`
	Note       string          `json:"note"`
	HappenedAt time.Time       `json:"happened_at"`
	Metadata   json.RawMessage `json:"metadata"`
}

// CreateEvent adds one manual event on one status in scope.
func (s *Service) CreateEvent(actor auth.Actor, statusID uint, req CreateEventRequest) (*repo.PartyflowAdminListItem, error) {
	eventType := strings.TrimSpace(req.EventType)
	if eventType != "milestone" && eventType != "manual_adjust" {
		return nil, ErrInvalidEventType
	}
	row, err := s.statusRepo.GetByIDInScope(authz.BuildScope(actor), statusID)
	if err != nil {
		return nil, err
	}
	happenedAt := req.HappenedAt
	if happenedAt.IsZero() {
		happenedAt = time.Now()
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		if eventType == "milestone" {
			note = "里程碑"
		} else {
			note = "人工调整"
		}
		if strings.TrimSpace(req.EventCode) != "" {
			note = note + "：" + strings.TrimSpace(req.EventCode)
		}
	}
	if err := s.statusRepo.CreateEvent(&model.PartyflowEvent{
		PartyflowStatusID: row.ID,
		EventType:         eventType,
		EventCode:         strings.TrimSpace(req.EventCode),
		FromStatus:        row.Status,
		ToStatus:          row.Status,
		Note:              note,
		HappenedAt:        happenedAt,
		Metadata:          normalizeJSON(req.Metadata),
	}); err != nil {
		return nil, err
	}
	return s.GetAdmin(actor, statusID)
}

// RuleListParams filters reminder rules.
type RuleListParams struct {
	OrgType string
	Enabled *bool
}

// ListRules returns reminder rules.
func (s *Service) ListRules(_ auth.Actor, params RuleListParams) ([]model.PartyflowReminderRule, error) {
	orgType := strings.TrimSpace(params.OrgType)
	if orgType != "" && !isValidOrgType(orgType) {
		return nil, ErrInvalidOrgType
	}
	return s.statusRepo.ListRules(repo.PartyflowRuleListFilter{OrgType: orgType, Enabled: params.Enabled})
}

// PatchRuleRequest is payload for rule patch.
type PatchRuleRequest struct {
	Enabled            *bool   `json:"enabled"`
	OffsetDays         *int    `json:"offset_days"`
	RepeatIntervalDays *int    `json:"repeat_interval_days"`
	Audience           *string `json:"audience"`
	Title              *string `json:"title"`
	MessageTemplate    *string `json:"message_template"`
}

// PatchRule updates one reminder rule.
func (s *Service) PatchRule(_ auth.Actor, id uint, req PatchRuleRequest) (*model.PartyflowReminderRule, error) {
	updates := map[string]any{}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.OffsetDays != nil {
		updates["offset_days"] = *req.OffsetDays
	}
	if req.RepeatIntervalDays != nil {
		updates["repeat_interval_days"] = *req.RepeatIntervalDays
	}
	if req.Audience != nil {
		updates["audience"] = strings.TrimSpace(*req.Audience)
	}
	if req.Title != nil {
		updates["title"] = strings.TrimSpace(*req.Title)
	}
	if req.MessageTemplate != nil {
		updates["message_template"] = strings.TrimSpace(*req.MessageTemplate)
	}
	if len(updates) > 0 {
		if err := s.statusRepo.PatchRule(id, updates); err != nil {
			return nil, err
		}
	}
	return s.statusRepo.GetRuleByID(id)
}

// ScanResult is manual scan summary.
type ScanResult struct {
	ScannedCount   int `json:"scanned_count"`
	GeneratedCount int `json:"generated_count"`
	SentCount      int `json:"sent_count"`
	SkippedCount   int `json:"skipped_count"`
	FailedCount    int `json:"failed_count"`
}

// ScanRemindersRequest allows optional scan time.
type ScanRemindersRequest struct {
	Now *time.Time `json:"now"`
}

// ScanReminders performs one manual reminder scan.
func (s *Service) ScanReminders(req ScanRemindersRequest) (ScanResult, error) {
	now := time.Now()
	if req.Now != nil {
		now = *req.Now
	}
	result := ScanResult{}
	rules, err := s.statusRepo.ListEnabledRules()
	if err != nil {
		return result, err
	}
	for _, rule := range rules {
		rows, err := s.statusRepo.ListByOrgStatus(rule.OrgType, rule.Status)
		if err != nil {
			result.FailedCount++
			continue
		}
		for _, st := range rows {
			result.ScannedCount++
			dueAt := st.StatusStartedAt.Add(time.Duration(rule.OffsetDays) * 24 * time.Hour)
			if override := nextDueOverride(st.Metadata, rule.RuleCode); override != nil {
				dueAt = *override
			}
			if now.Before(dueAt) {
				result.SkippedCount++
				continue
			}
			periodIndex := 1
			if rule.RepeatIntervalDays > 0 {
				elapsedDays := int(now.Sub(dueAt).Hours() / 24)
				periodIndex = elapsedDays/rule.RepeatIntervalDays + 1
			}
			exists, err := s.statusRepo.ReminderEventExists(st.ID, rule.RuleCode, periodIndex)
			if err != nil {
				result.FailedCount++
				continue
			}
			if exists {
				result.SkippedCount++
				continue
			}
			raw, _ := json.Marshal(map[string]any{"period_index": periodIndex, "rule_code": rule.RuleCode})
			note := strings.TrimSpace(rule.Title)
			if note != "" {
				note = note + "已发送"
			} else {
				note = "reminder sent"
			}
			if err := s.statusRepo.CreateEvent(&model.PartyflowEvent{
				PartyflowStatusID: st.ID,
				EventType:         "reminder_sent",
				EventCode:         rule.RuleCode,
				FromStatus:        st.Status,
				ToStatus:          st.Status,
				Note:              note,
				HappenedAt:        now,
				Metadata:          datatypes.JSON(raw),
			}); err != nil {
				result.FailedCount++
				continue
			}
			result.GeneratedCount++
			result.SentCount++
		}
	}
	return result, nil
}

func isValidOrgType(v string) bool {
	v = strings.TrimSpace(v)
	return v == "party" || v == "league"
}

func isValidStatus(orgType, status string) bool {
	status = strings.TrimSpace(status)
	if status == "" {
		return false
	}
	orgType = strings.TrimSpace(orgType)
	if orgType == "party" {
		_, ok := partyStatuses[status]
		return ok
	}
	if orgType == "league" {
		_, ok := leagueStatuses[status]
		return ok
	}
	_, ok1 := leagueStatuses[status]
	_, ok2 := partyStatuses[status]
	return ok1 || ok2
}

func nextDueOverride(raw datatypes.JSON, ruleCode string) *time.Time {
	if len(raw) == 0 {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	overrides, ok := obj["reminder_overrides"].(map[string]any)
	if !ok {
		return nil
	}
	item, ok := overrides[ruleCode].(map[string]any)
	if !ok {
		return nil
	}
	v, ok := item["next_due_at"].(string)
	if !ok || strings.TrimSpace(v) == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil
	}
	return &t
}

func statusLabel(status string) string {
	status = strings.TrimSpace(status)
	if v, ok := statusLabels[status]; ok {
		return v
	}
	if status == "" {
		return "-"
	}
	return status
}

func defaultReminderRules() []model.PartyflowReminderRule {
	emptyJSON := datatypes.JSON([]byte("{}"))
	return []model.PartyflowReminderRule{
		{RuleCode: "league_applicant_talk_30d", OrgType: "league", Status: "applicant", TriggerType: "status_started", OffsetDays: 30, RepeatIntervalDays: 0, Title: "入团谈话提醒", MessageTemplate: "入团申请后 30 天内应完成谈话", Audience: "admin", Enabled: true, Metadata: emptyJSON},
		{RuleCode: "league_activist_train_90d", OrgType: "league", Status: "activist", TriggerType: "status_started", OffsetDays: 90, RepeatIntervalDays: 0, Title: "积极分子培养提醒", MessageTemplate: "积极分子培养满 90 天可推荐发展对象", Audience: "admin", Enabled: true, Metadata: emptyJSON},
		{RuleCode: "league_development_publicity_5workdays", OrgType: "league", Status: "development_target", TriggerType: "status_started", OffsetDays: 5, RepeatIntervalDays: 0, Title: "公示提醒", MessageTemplate: "发展对象公示不少于 5 天", Audience: "admin", Enabled: false, Metadata: emptyJSON},
		{RuleCode: "league_approval_30d", OrgType: "league", Status: "development_target", TriggerType: "status_started", OffsetDays: 30, RepeatIntervalDays: 0, Title: "审批提醒", MessageTemplate: "支部大会后 30 天内审批", Audience: "admin", Enabled: false, Metadata: emptyJSON},
		{RuleCode: "league_member_archive_30d", OrgType: "league", Status: "member", TriggerType: "status_started", OffsetDays: 30, RepeatIntervalDays: 0, Title: "归档提醒", MessageTemplate: "审批后 30 天内建立电子档案", Audience: "admin", Enabled: false, Metadata: emptyJSON},
		{RuleCode: "party_applicant_talk_30d", OrgType: "party", Status: "applicant", TriggerType: "status_started", OffsetDays: 30, RepeatIntervalDays: 0, Title: "入党谈话提醒", MessageTemplate: "入党申请后 30 天内谈话", Audience: "admin", Enabled: true, Metadata: emptyJSON},
		{RuleCode: "party_activist_report_every_90d", OrgType: "party", Status: "activist", TriggerType: "status_started", OffsetDays: 90, RepeatIntervalDays: 90, Title: "思想汇报提醒", MessageTemplate: "积极分子每满 90 天提交思想汇报", Audience: "student", Enabled: true, Metadata: emptyJSON},
		{RuleCode: "party_development_training_reminder", OrgType: "party", Status: "development_target", TriggerType: "status_started", OffsetDays: 30, RepeatIntervalDays: 0, Title: "培训提醒", MessageTemplate: "发展对象阶段培训提醒", Audience: "student", Enabled: false, Metadata: emptyJSON},
		{RuleCode: "party_probationary_transfer_365d", OrgType: "party", Status: "probationary_member", TriggerType: "status_started", OffsetDays: 365, RepeatIntervalDays: 0, Title: "转正提醒", MessageTemplate: "预备党员满一年请提交转正申请", Audience: "student", Enabled: true, Metadata: emptyJSON},
		{RuleCode: "party_probationary_report_every_90d", OrgType: "party", Status: "probationary_member", TriggerType: "status_started", OffsetDays: 90, RepeatIntervalDays: 90, Title: "预备期汇报提醒", MessageTemplate: "预备期按季度提交思想汇报", Audience: "student", Enabled: false, Metadata: emptyJSON},
	}
}
