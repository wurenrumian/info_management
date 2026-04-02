//go:build integration

package notification

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"manage/internal/model"
)

func TestNotificationRepoWithKingbase(t *testing.T) {
	dsn := os.Getenv("KINGBASE_DSN")
	if dsn == "" {
		t.Skip("KINGBASE_DSN is empty; skip integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetConnMaxLifetime(2 * time.Minute)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetMaxOpenConns(5)

	require.NoError(t, db.AutoMigrate(&model.NotificationTemplate{}, &model.NotificationLog{}))

	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "kingbase_test_remind",
		WechatTemplateID: "tmpl_kingbase_001",
		Name:             "金仓测试提醒",
		Fields:           `{"thing1":"事项","time2":"时间"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))
	require.NotZero(t, tmpl.ID)

	fetched, err := repo.GetTemplateByCode("kingbase_test_remind")
	require.NoError(t, err)
	require.Equal(t, "kingbase_test_remind", fetched.Code)
	require.Equal(t, "tmpl_kingbase_001", fetched.WechatTemplateID)

	log := &model.NotificationLog{
		UserID:       1,
		TemplateCode: "kingbase_test_remind",
		TemplateData: `{"thing1":{"value":"金仓测试通知"},"time2":{"value":"2026年4月10日"}}`,
		Status:       "pending",
	}
	require.NoError(t, repo.CreateLog(log))
	require.NotZero(t, log.ID)

	now := time.Now()
	log.SentAt = &now
	log.Status = "sent"
	require.NoError(t, db.Save(log).Error)

	logs, total, err := repo.ListLogs(LogFilter{
		UserID:       kbPtrUint(1),
		TemplateCode: kbPtrString("kingbase_test_remind"),
		Limit:        10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	require.Equal(t, "sent", logs[0].Status)
	require.NotNil(t, logs[0].SentAt)

	status := "failed"
	emptyLogs, emptyTotal, err := repo.ListLogs(LogFilter{
		UserID: kbPtrUint(1),
		Status: &status,
		Limit:  10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), emptyTotal)
	require.Empty(t, emptyLogs)

	_, err = repo.GetTemplateByCode("nonexistent_code")
	require.Error(t, err)
}

func kbPtrUint(v uint) *uint {
	return &v
}

func kbPtrString(v string) *string {
	return &v
}
