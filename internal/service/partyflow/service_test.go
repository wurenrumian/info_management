package partyflow

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/model"
)

func setupPartyflowService(t *testing.T) (*gorm.DB, *Service) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}, &model.PartyflowStatus{}, &model.PartyflowEvent{}, &model.PartyflowReminderRule{}))
	return db, NewService(db)
}

func TestCreateStatusRespectsScope(t *testing.T) {
	db, svc := setupPartyflowService(t)

	require.NoError(t, db.Create(&model.Class{ID: 1, ClassName: "C1", Grade: "2023", Major: "IM"}).Error)
	require.NoError(t, db.Create(&model.Class{ID: 2, ClassName: "C2", Grade: "2022", Major: "CS"}).Error)

	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "same-scope",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
		Major:     "IM",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        101,
		StudentID: "S101",
		Name:      "cross-scope",
		Role:      model.RoleStudent,
		ClassID:   2,
		Grade:     "2022",
		Major:     "CS",
	}).Error)

	actor := auth.Actor{UserID: 200, Role: model.RoleCadre, ClassID: 1, Grade: "2023"}
	rawMeta, err := json.Marshal(map[string]string{"contact_person": "Li"})
	require.NoError(t, err)

	created, err := svc.CreateStatus(actor, CreateStatusRequest{
		UserID:          100,
		OrgType:         "party",
		Status:          "applicant",
		StatusStartedAt: time.Now(),
		Metadata:        rawMeta,
		Note:            "init",
	})
	require.NoError(t, err)
	require.Equal(t, uint(100), created.UserID)
	require.Equal(t, "party", created.OrgType)

	_, err = svc.CreateStatus(actor, CreateStatusRequest{
		UserID:          101,
		OrgType:         "party",
		Status:          "applicant",
		StatusStartedAt: time.Now(),
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestScanRemindersGeneratesEvent(t *testing.T) {
	db, svc := setupPartyflowService(t)
	require.NoError(t, db.Create(&model.Class{ID: 1, ClassName: "C1", Grade: "2023", Major: "IM"}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "same-scope",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
		Major:     "IM",
	}).Error)
	started := time.Now().Add(-91 * 24 * time.Hour)
	require.NoError(t, db.Create(&model.PartyflowStatus{
		UserID:          100,
		OrgType:         "party",
		Status:          "activist",
		StatusStartedAt: started,
		Metadata:        []byte(`{}`),
	}).Error)

	result, err := svc.ScanReminders(ScanRemindersRequest{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, result.GeneratedCount, 1)

	var cnt int64
	require.NoError(t, db.Model(&model.PartyflowEvent{}).Where("event_type = ? AND event_code = ?", "reminder_sent", "party_activist_report_every_90d").Count(&cnt).Error)
	require.GreaterOrEqual(t, cnt, int64(1))
}
