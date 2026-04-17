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
	require.NoError(t, db.Create(&model.User{ID: 200, StudentID: "T200", Name: "老师", Role: model.RoleTeacher, ClassID: 1, Grade: "2023", Major: "信息管理"}).Error)
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

func TestStudentListFiltersByTargetScopeGrade(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)
	require.NoError(t, db.Create(&model.Class{ID: 3, ClassName: "软工3班", Grade: "2024", Major: "软件工程"}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        102,
		StudentID: "S102",
		Name:      "王五",
		Role:      model.RoleStudent,
		ClassID:   3,
		Grade:     "2024",
		Major:     "软件工程",
	}).Error)

	rows := []model.Announcement{
		{
			Title:        "仅2024级可见",
			Content:      "正文",
			Status:       "published",
			AudienceType: "targeted",
			TargetScope:  datatypes.JSON(`{"grades":["2024"]}`),
			AuthorID:     999,
		},
		{
			Title:        "全员公告",
			Content:      "正文",
			Status:       "published",
			AudienceType: "all",
			TargetScope:  datatypes.JSON(`{}`),
			AuthorID:     999,
		},
	}
	require.NoError(t, db.Create(&rows).Error)

	req2022 := httptest.NewRequest(http.MethodGet, "/api/v1/announcements?limit=20&offset=0", nil)
	req2022.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(101, model.RoleStudent, 2, "2022"))
	w2022 := httptest.NewRecorder()
	r.ServeHTTP(w2022, req2022)
	require.Equal(t, http.StatusOK, w2022.Code)
	require.NotContains(t, w2022.Body.String(), "仅2024级可见")
	require.Contains(t, w2022.Body.String(), "全员公告")

	req2024 := httptest.NewRequest(http.MethodGet, "/api/v1/announcements?limit=20&offset=0", nil)
	req2024.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(102, model.RoleStudent, 3, "2024"))
	w2024 := httptest.NewRecorder()
	r.ServeHTTP(w2024, req2024)
	require.Equal(t, http.StatusOK, w2024.Code)
	require.Contains(t, w2024.Body.String(), "仅2024级可见")
	require.Contains(t, w2024.Body.String(), "全员公告")
}

func TestListAllPublishedAllowsTeacherAndIgnoresTargetScope(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)
	rows := []model.Announcement{
		{
			Title:        "仅2024级可见",
			Content:      "正文",
			Status:       "published",
			AudienceType: "targeted",
			TargetScope:  datatypes.JSON(`{"grades":["2024"]}`),
			AuthorID:     999,
		},
		{
			Title:        "全员已发布",
			Content:      "正文",
			Status:       "published",
			AudienceType: "all",
			TargetScope:  datatypes.JSON(`{}`),
			AuthorID:     999,
		},
		{
			Title:        "草稿不应返回",
			Content:      "正文",
			Status:       "draft",
			AudienceType: "all",
			TargetScope:  datatypes.JSON(`{}`),
			AuthorID:     999,
		},
	}
	require.NoError(t, db.Create(&rows).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/announcements/all?limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(200, model.RoleTeacher, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "仅2024级可见")
	require.Contains(t, w.Body.String(), "全员已发布")
	require.NotContains(t, w.Body.String(), "草稿不应返回")
}

func TestListAllPublishedForbiddenForStudent(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)
	require.NoError(t, db.Create(&model.Announcement{
		Title:        "全员已发布",
		Content:      "正文",
		Status:       "published",
		AudienceType: "all",
		TargetScope:  datatypes.JSON(`{}`),
		AuthorID:     999,
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/announcements/all?limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "forbidden")
}

func TestGetAllPublishedByIDAllowsTeacherAndReturnsContent(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)
	item := model.Announcement{
		Title:        "仅2024级可见",
		Content:      "这是定向公告完整正文",
		Status:       "published",
		AudienceType: "targeted",
		TargetScope:  datatypes.JSON(`{"grades":["2024"]}`),
		AuthorID:     999,
	}
	require.NoError(t, db.Create(&item).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/announcements/all/1", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(200, model.RoleTeacher, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "仅2024级可见")
	require.Contains(t, w.Body.String(), "这是定向公告完整正文")
}

func TestGetAllPublishedByIDForbiddenForStudent(t *testing.T) {
	db, r := setupAnnouncementHandlerRouter(t)
	require.NoError(t, db.Create(&model.Announcement{
		Title:        "全员已发布",
		Content:      "正文",
		Status:       "published",
		AudienceType: "all",
		TargetScope:  datatypes.JSON(`{}`),
		AuthorID:     999,
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/announcements/all/1", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(100, model.RoleStudent, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "forbidden")
}

func TestPublishAnnouncementReturnsNotificationSummary(t *testing.T) {
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

	require.NoError(t, db.Create(&model.NotificationTemplate{
		Code:             "announcement",
		WechatTemplateID: "tmpl_announcement",
		Name:             "公告通知",
	}).Error)
	require.NoError(t, db.Create(&model.UserSubscribe{
		UserID:           100,
		TemplateCode:     "announcement",
		WechatTemplateID: "tmpl_announcement",
		Status:           "subscribed",
		GrantedCount:     1,
		ConsumedCount:    0,
	}).Error)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/announcements/1/publish", bytes.NewReader([]byte(`{"send_notification":true}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"notification_summary"`)
	require.Contains(t, w.Body.String(), `"attempted"`)
	require.Contains(t, w.Body.String(), `"sent"`)
	require.Contains(t, w.Body.String(), `"failed"`)
}
