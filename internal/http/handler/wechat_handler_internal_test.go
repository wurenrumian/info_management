package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"manage/internal/model"
	authsvc "manage/internal/service/auth"
)

func TestBuildDevSubscribeTemplateDataFillsLoginNotificationDefaults(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 45, 0, 0, time.UTC)
	user := &model.User{
		StudentID: "2024201514",
		Name:      "wuren",
	}

	data := buildDevSubscribeTemplateData("loging_notification", map[string]interface{}{
		"thing1": map[string]interface{}{"value": "登录提醒"},
		"time2":  map[string]interface{}{"value": ""},
	}, user, "127.0.0.1", now)

	require.Equal(t, "登录提醒", nestedValue(data, "thing1"))
	require.Equal(t, "2026-04-03 12:45:00", nestedValue(data, "time2"))
	require.Equal(t, "127.0.0.1", nestedValue(data, "character_string3"))
	require.Equal(t, "wuren (2024201514)", nestedValue(data, "thing4"))
}

func TestBuildUserResponse(t *testing.T) {
	user := &model.User{
		ID:        42,
		StudentID: "S042",
		Name:      "Alice",
		Role:      model.RoleCadre,
		ClassID:   7,
		Grade:     "2024",
		Major:     "CS",
	}

	got := buildUserResponse(user, user.Grade)
	require.Equal(t, uint(42), got["id"])
	require.Equal(t, "S042", got["student_id"])
	require.Equal(t, "Alice", got["name"])
	require.Equal(t, model.RoleCadre, got["role"])
	require.Equal(t, uint(7), got["class_id"])
	require.Equal(t, "2024", got["grade"])
	require.Equal(t, "CS", got["major"])
}

func nestedValue(data map[string]interface{}, key string) string {
	field, _ := data[key].(map[string]interface{})
	value, _ := field["value"].(string)
	return value
}

func TestWriteDevLoginErrorHandlesWrappedMissingStudentID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h := &WechatHandler{}

	h.writeDevLoginError(c, errors.Join(authsvc.ErrMissingStudentID, errors.New("wrapped")), "fallback")

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "missing student_id")
}

func TestWriteDevLoginErrorHandlesWrappedInvalidRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h := &WechatHandler{}

	h.writeDevLoginError(c, errors.Join(authsvc.ErrInvalidRole, errors.New("wrapped")), "fallback")

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid role")
}
