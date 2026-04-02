package notification

import (
	"testing"
	"time"

	"manage/internal/model"
	"manage/internal/testutil"

	"github.com/stretchr/testify/require"
)

func TestGetTemplateByCode(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "test_remind",
		WechatTemplateID: "tmpl_123",
		Name:             "测试提醒",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	result, err := repo.GetTemplateByCode("test_remind")
	require.NoError(t, err)
	require.Equal(t, "test_remind", result.Code)
	require.Equal(t, "tmpl_123", result.WechatTemplateID)
}

func TestGetTemplateByCodeNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	_, err := repo.GetTemplateByCode("nonexistent")
	require.Error(t, err)
}

func TestCreateLog(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	log := &model.NotificationLog{
		UserID:       1,
		TemplateCode: "test_remind",
		TemplateData: `{"thing1":"测试"}`,
		Status:       "pending",
	}
	require.NoError(t, repo.CreateLog(log))
	require.NotZero(t, log.ID)
}

func TestListLogs(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	for i := 0; i < 5; i++ {
		repo.CreateLog(&model.NotificationLog{
			UserID:       1,
			TemplateCode: "test_remind",
			Status:       "sent",
			CreatedAt:    time.Now(),
		})
	}

	logs, total, err := repo.ListLogs(LogFilter{
		UserID: ptrUint(1),
		Limit:  10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(5), total)
	require.Len(t, logs, 5)
}

func TestListLogsFilterByStatus(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	repo.CreateLog(&model.NotificationLog{
		UserID:       1,
		TemplateCode: "test_remind",
		Status:       "sent",
		CreatedAt:    time.Now(),
	})
	repo.CreateLog(&model.NotificationLog{
		UserID:       1,
		TemplateCode: "test_remind",
		Status:       "failed",
		CreatedAt:    time.Now(),
	})

	status := "failed"
	logs, total, err := repo.ListLogs(LogFilter{
		UserID: ptrUint(1),
		Status: &status,
		Limit:  10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	require.Equal(t, "failed", logs[0].Status)
}

func ptrUint(v uint) *uint {
	return &v
}
