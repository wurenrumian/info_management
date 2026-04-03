package audit

import (
	"github.com/gin-gonic/gin"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/repo"
)

// Logger provides a tiny adapter for best-effort admin audit logs.
type Logger struct {
	logRepo *repo.AdminLogRepo
}

// NewLogger creates a Logger.
func NewLogger(logRepo *repo.AdminLogRepo) *Logger {
	return &Logger{logRepo: logRepo}
}

// Log writes one admin log entry. Failures are intentionally ignored by callers.
func (l *Logger) Log(c *gin.Context, actor auth.Actor, action, targetType string, targetID uint) {
	if l == nil || l.logRepo == nil {
		return
	}

	ip := ""
	if c != nil {
		ip = c.ClientIP()
	}

	_ = l.logRepo.Create(&model.AdminLog{
		AdminID:    actor.UserID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IPAddress:  ip,
	})
}
