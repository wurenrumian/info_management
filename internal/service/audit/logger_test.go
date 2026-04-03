package audit

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/repo"
)

func TestLoggerLogWritesAdminLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AdminLog{}))

	l := NewLogger(repo.NewAdminLogRepo(db))
	c, _ := gin.CreateTestContext(nil)
	c.Request = httptest.NewRequest("GET", "/", nil)

	l.Log(c, auth.Actor{UserID: 99}, "users.patch", "user", 1)

	var logs []model.AdminLog
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, uint(99), logs[0].AdminID)
	require.Equal(t, "users.patch", logs[0].Action)
	require.Equal(t, "user", logs[0].TargetType)
	require.Equal(t, uint(1), logs[0].TargetID)
}
