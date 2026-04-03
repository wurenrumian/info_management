package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"manage/internal/model"
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

func nestedValue(data map[string]interface{}, key string) string {
	field, _ := data[key].(map[string]interface{})
	value, _ := field["value"].(string)
	return value
}
