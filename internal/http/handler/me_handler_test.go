package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/http/router"
	"manage/internal/model"
	"manage/internal/testutil"
)

func setupMeTestRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Class{}, &model.KnowledgeItem{}, &model.NotificationLog{}))

	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "张三",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
		Major:     "计算机科学与技术",
		ProfileAttrs: datatypes.JSON([]byte(`{
			"nickname":"阿三",
			"bio":"保持热爱，奔赴山海",
			"avatar_url":"https://example.com/avatar-old.png"
		}`)),
	}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        101,
		StudentID: "S101",
		Name:      "李四",
		Role:      model.RoleStudent,
		ClassID:   2,
		Grade:     "2023",
	}).Error)

	r := router.New(db)
	return db, r
}

func TestGetMeReturnsSelf(t *testing.T) {
	_, r := setupMeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "S100")
	require.NotContains(t, w.Body.String(), "S101")
}

func TestGetMeReturnsProfileFields(t *testing.T) {
	_, r := setupMeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	token := testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "张三", resp.Data["real_name"])
	require.Equal(t, "阿三", resp.Data["nickname"])
	require.Equal(t, "计算机科学与技术", resp.Data["major"])
	require.Equal(t, "保持热爱，奔赴山海", resp.Data["bio"])
	require.Equal(t, "https://example.com/avatar-old.png", resp.Data["avatar_url"])
	require.Contains(t, resp.Data, "college")
	require.Contains(t, resp.Data, "enrollment_year")
}

func TestGetMeReturnsTeacherSelfInsteadOfFirstScopedUser(t *testing.T) {
	db, r := setupMeTestRouter(t)

	require.NoError(t, db.Create(&model.User{
		ID:        102,
		StudentID: "T102",
		Name:      "王老师",
		Role:      model.RoleTeacher,
		ClassID:   1,
		Grade:     "2023",
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	token := testutil.GenerateTestToken(102, model.RoleTeacher, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "T102")
	require.NotContains(t, w.Body.String(), "S100")
}

func TestGetProfileHomeReturnsTeacherSelfInsteadOfFirstScopedUser(t *testing.T) {
	db, r := setupMeTestRouter(t)

	require.NoError(t, db.Create(&model.User{
		ID:        102,
		StudentID: "T102",
		Name:      "王老师",
		Role:      model.RoleTeacher,
		ClassID:   1,
		Grade:     "2023",
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile/home", nil)
	token := testutil.GenerateTestToken(102, model.RoleTeacher, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "T102")
	require.NotContains(t, w.Body.String(), "S100")
}

func TestGetMeForbiddenForUnknownRole(t *testing.T) {
	_, r := setupMeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	token := testutil.GenerateTestToken(999, 0, 0, "")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetMeNotFound(t *testing.T) {
	_, r := setupMeTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	token := testutil.GenerateTestToken(999, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "user not found")
}

func TestGetProfileHomeReturnsAggregatedData(t *testing.T) {
	db, r := setupMeTestRouter(t)

	require.NoError(t, db.Create(&model.KnowledgeItem{
		Question:  "Q1",
		Answer:    "A1",
		CreatedBy: 100,
		UpdatedBy: 100,
	}).Error)
	require.NoError(t, db.Create(&model.KnowledgeItem{
		Question:  "Q2",
		Answer:    "A2",
		CreatedBy: 100,
		UpdatedBy: 100,
	}).Error)
	require.NoError(t, db.Create(&model.NotificationLog{
		UserID:       100,
		TemplateCode: "tpl_pending",
		Status:       "pending",
	}).Error)
	require.NoError(t, db.Create(&model.NotificationLog{
		UserID:       100,
		TemplateCode: "tpl_sent",
		Status:       "sent",
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile/home", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Basic struct {
				ID        uint   `json:"id"`
				StudentID string `json:"student_id"`
			} `json:"basic"`
			QuickEntry struct {
				AnnouncementsCount int64 `json:"announcements_count"`
				ApprovalsCount     int64 `json:"approvals_count"`
				KnowledgeCount     int64 `json:"knowledge_count"`
			} `json:"quick_entry"`
			Account struct {
				WechatBound bool `json:"wechat_bound"`
			} `json:"account"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, uint(100), resp.Data.Basic.ID)
	require.Equal(t, "S100", resp.Data.Basic.StudentID)
	require.Equal(t, int64(0), resp.Data.QuickEntry.AnnouncementsCount)
	require.Equal(t, int64(0), resp.Data.QuickEntry.ApprovalsCount)
	require.Equal(t, int64(2), resp.Data.QuickEntry.KnowledgeCount)
	require.False(t, resp.Data.Account.WechatBound)
}

func TestPatchMeUpdatesColumnsAndProfileAttrs(t *testing.T) {
	db, r := setupMeTestRouter(t)

	body := []byte(`{
		"nickname":"阿三同学",
		"major":"人工智能",
		"college":"信息学院",
		"enrollment_year":2023,
		"avatar_url":"https://example.com/avatar.png",
		"bio":"个人简介"
	}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, 1, 1, "2023"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var user model.User
	require.NoError(t, db.First(&user, 100).Error)
	require.Equal(t, "人工智能", user.Major)

	var attrs map[string]any
	require.NoError(t, json.Unmarshal([]byte(user.ProfileAttrs), &attrs))
	require.Equal(t, "阿三同学", attrs["nickname"])
	require.Equal(t, "https://example.com/avatar.png", attrs["avatar_url"])
	require.Equal(t, "个人简介", attrs["bio"])

	require.NoError(t, db.Model(&model.User{}).Where("id = ?", 100).Update("profile_attrs", datatypes.JSON([]byte(`{"avatar_url":"old","bio":"old","theme":"blue"}`))).Error)

	body = []byte(`{
		"nickname":"new-nickname",
		"avatar_url":"https://example.com/new-avatar.png",
		"bio":"new-bio"
	}`)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, 1, 1, "2023"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, db.First(&user, 100).Error)
	require.NoError(t, json.Unmarshal([]byte(user.ProfileAttrs), &attrs))
	require.Equal(t, "https://example.com/new-avatar.png", attrs["avatar_url"])
	require.Equal(t, "new-bio", attrs["bio"])
	require.Equal(t, "new-nickname", attrs["nickname"])
	require.Equal(t, "blue", attrs["theme"])
}

func TestPatchMeRejectsReadOnlyFields(t *testing.T) {
	_, r := setupMeTestRouter(t)

	body := []byte(`{"real_name":"新名字"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), `"code":40002`)
	require.Contains(t, w.Body.String(), "read-only")
}

func TestPatchMeRejectsInvalidEnrollmentYear(t *testing.T) {
	_, r := setupMeTestRouter(t)

	body := []byte(`{"enrollment_year":1900}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), `"code":40003`)
}
