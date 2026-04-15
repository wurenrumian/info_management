package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"manage/internal/http/router"
	"manage/internal/model"
	"manage/internal/testutil"
)

func TestAnnouncementCreateAndPublish(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.Announcement{})

	r := router.New(db)

	token := testutil.GenerateTestToken(999, 4, 1, "2023")

	body := map[string]string{
		"title":   "test",
		"content": "content",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/admin/announcements", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}