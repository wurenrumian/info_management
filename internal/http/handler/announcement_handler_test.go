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

func setupAnnouncementHandlerRouter(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Class{},
		&model.User{},
		&model.AdminLog{},
		&model.Announcement{},
		&model.NotificationTemplate{},
		&model.NotificationLog{},
		&model.UserSubscribe{},
	))

	require.NoError(t, db.Create(&model.Class{ID: 1, ClassName: "信管1班", Grade: "2023", Major: "信息管理"}).Error)
	require.NoError(t, db.Create(&model.Class{ID: 2, ClassName: "计科2班", Grade: "2022", Major: "计算机"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 100, StudentID: "S100", Name: "张三", Role: model.RoleStudent, ClassID: 1, Grade: "2023", Major: "信息管理"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 101, StudentID: "S101", Name: "李四", Role: model.RoleStudent, ClassID: 2, Grade: "2022", Major: "计算机"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 999, StudentID: "A999", Name: "管理员", Role: model.RoleSuperAdmin, ClassID: 1, Grade: "2023"}).Error)

	return db, router.New(db)
}

func TestCreateAnnouncementSuccess(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)

	body := []byte(`{
		"title":"五一安全提醒",
		"content":"离校前请做好登记",
		"audience_type":"all",
		"tags":["假期","安全"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/announcements", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "五一安全提醒")

	var item model.Announcement
	require.NoError(t, db.First(&item).Error)
	require.Equal(t, "五一安全提醒", item.Title)
	require.Equal(t, "draft", item.Status)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "announcements.create").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, uint(999), logs[0].AdminID)
	require.Equal(t, "announcement", logs[0].TargetType)
	require.Equal(t, item.ID, logs[0].TargetID)
}

func TestCreateAnnouncementValidation(t *testing.T) {
	_, r := setupAnnouncementHandlerRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/announcements", bytes.NewReader([]byte(`{"content":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid request")
}

func TestPatchAnnouncementSuccess(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)
	item := model.Announcement{
		Title:        "旧标题",
		Content:      "旧内容",
		Status:       "draft",
		AudienceType: "all",
		TargetScope:  datatypes.JSON(`{}`),
		AuthorID:     999,
	}
	require.NoError(t, db.Create(&item).Error)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/announcements/1", bytes.NewReader([]byte(`{"title":"新标题","content":"新内容"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "新标题")

	var updated model.Announcement
	require.NoError(t, db.First(&updated, item.ID).Error)
	require.Equal(t, "新标题", updated.Title)
	require.Equal(t, "新内容", updated.Content)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "announcements.patch").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, item.ID, logs[0].TargetID)
}

func TestPublishAnnouncementSuccess(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)
	item := model.Announcement{
		Title:        "待发布公告",
		Content:      "正文",
		Status:       "draft",
		AudienceType: "all",
		TargetScope:  datatypes.JSON(`{}`),
		AuthorID:     999,
	}
	require.NoError(t, db.Create(&item).Error)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/announcements/1/publish", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "published")

	var published model.Announcement
	require.NoError(t, db.First(&published, item.ID).Error)
	require.Equal(t, "published", published.Status)
	require.NotNil(t, published.PublishedAt)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "announcements.publish").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, item.ID, logs[0].TargetID)
}

func TestArchiveAnnouncementSuccess(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)
	item := model.Announcement{
		Title:        "待归档公告",
		Content:      "正文",
		Status:       "published",
		AudienceType: "all",
		TargetScope:  datatypes.JSON(`{}`),
		AuthorID:     999,
	}
	require.NoError(t, db.Create(&item).Error)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/announcements/1/archive", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "archived")

	var archived model.Announcement
	require.NoError(t, db.First(&archived, item.ID).Error)
	require.Equal(t, "archived", archived.Status)

	var logs []model.AdminLog
	require.NoError(t, db.Where("action = ?", "announcements.archive").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, item.ID, logs[0].TargetID)
}

func TestStudentGetByIDNotFound(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)
	item := model.Announcement{
		Title:        "定向给别的班级",
		Content:      "正文",
		Status:       "published",
		AudienceType: "targeted",
		TargetScope:  datatypes.JSON(`{"class_ids":[2]}`),
		AuthorID:     999,
	}
	require.NoError(t, db.Create(&item).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/announcements/1", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "announcement not found")
}

func TestAdminAnnouncementsListForbiddenForStudent(t *testing.T) {
	_, r := setupAnnouncementHandlerRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/announcements?status=draft&limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "forbidden")
}

type announcementListResp struct {
	Data  []model.Announcement `json:"data"`
	Total int64                `json:"total"`
}

func TestStudentListReturnsOnlyPublished(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)
	rows := []model.Announcement{
		{
			Title:        "学生可见已发布",
			Content:      "正文",
			Status:       "published",
			AudienceType: "all",
			TargetScope:  datatypes.JSON(`{}`),
			AuthorID:     999,
		},
		{
			Title:        "草稿不可见",
			Content:      "正文",
			Status:       "draft",
			AudienceType: "all",
			TargetScope:  datatypes.JSON(`{}`),
			AuthorID:     999,
		},
	}
	require.NoError(t, db.Create(&rows).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/announcements?limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "学生可见已发布")
	require.NotContains(t, w.Body.String(), "草稿不可见")

	var resp announcementListResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, int64(1), resp.Total)
	require.Len(t, resp.Data, 1)
	require.Equal(t, "学生可见已发布", resp.Data[0].Title)
}
