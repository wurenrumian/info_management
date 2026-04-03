package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"manage/internal/model"
	"manage/internal/service/wechat"
)

func TestWechatLoginReturnsDataEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockWechat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openid":"openid-123","session_key":"k"}`))
	}))
	defer mockWechat.Close()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}))

	openID := "openid-123"
	require.NoError(t, db.Create(&model.User{
		ID:        1,
		StudentID: "S001",
		Name:      "Alice",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
		Major:     "CS",
		OpenID:    &openID,
	}).Error)

	h := NewWechatHandler(db, "", "", "test-secret", nil)
	h.wechatSvc = wechat.NewServiceWithBaseURL("", "", mockWechat.URL)

	r := gin.New()
	r.POST("/wechat/login", h.Login)

	body := bytes.NewBufferString(`{"code":"wx-code"}`)
	req := httptest.NewRequest(http.MethodPost, "/wechat/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.Contains(t, payload, "data")
	require.NotContains(t, payload, "token")
}
