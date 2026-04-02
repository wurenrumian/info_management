package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/service/notification"
	"manage/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupNotificationTest(t *testing.T) (*NotificationHandler, *notification.Repo) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := notification.NewRepo(db)
	mockWechat := &mockNotifWechatClient{}
	mockUsers := &mockNotifUserRepo{openID: "test_openid"}
	svc := notification.NewService(mockWechat, repo, mockUsers)
	return NewNotificationHandler(svc), repo
}

type mockNotifWechatClient struct{}

func (m *mockNotifWechatClient) SendSubscribeMessage(openid, templateID, page string, data map[string]interface{}) error {
	return nil
}

type mockNotifUserRepo struct {
	openID string
}

func (m *mockNotifUserRepo) GetUserOpenID(userID uint) (string, error) {
	return m.openID, nil
}

func setAdminActor(c *gin.Context) {
	auth.SetActor(c, auth.Actor{UserID: 200, Role: 4})
}

func TestCreateTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := setupNotificationTest(t)

	body := map[string]string{
		"code":               "test_tpl",
		"wechat_template_id": "tmpl_abc",
		"name":               "测试模板",
		"fields":             `{"thing1":"事项"}`,
	}
	data, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/notification/templates", bytes.NewReader(data))
	c.Request.Header.Set("Content-Type", "application/json")
	setAdminActor(c)

	handler.CreateTemplate(c)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestCreateTemplateMissingField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := setupNotificationTest(t)

	body := map[string]string{
		"code": "test_tpl",
	}
	data, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/notification/templates", bytes.NewReader(data))
	c.Request.Header.Set("Content-Type", "application/json")
	setAdminActor(c)

	handler.CreateTemplate(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, repo := setupNotificationTest(t)

	tmpl := &model.NotificationTemplate{
		Code:             "test_tpl",
		WechatTemplateID: "tmpl_abc",
		Name:             "测试模板",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notification/templates/test_tpl", nil)
	c.AddParam("code", "test_tpl")
	setAdminActor(c)

	handler.GetTemplate(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	require.Equal(t, "test_tpl", data["code"])
}

func TestGetTemplateNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := setupNotificationTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notification/templates/nonexistent", nil)
	c.AddParam("code", "nonexistent")
	setAdminActor(c)

	handler.GetTemplate(c)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, repo := setupNotificationTest(t)

	repo.CreateLog(&model.NotificationLog{
		UserID:       1,
		TemplateCode: "test_tpl",
		Status:       "sent",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notification/logs", nil)
	setAdminActor(c)

	handler.ListLogs(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	require.Equal(t, float64(1), data["total"])
}

func TestUnreadCountReturnsPendingCountForCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, repo := setupNotificationTest(t)

	require.NoError(t, repo.CreateLog(&model.NotificationLog{
		UserID:       100,
		TemplateCode: "tpl_pending_1",
		Status:       "pending",
	}))
	require.NoError(t, repo.CreateLog(&model.NotificationLog{
		UserID:       100,
		TemplateCode: "tpl_pending_2",
		Status:       "pending",
	}))
	require.NoError(t, repo.CreateLog(&model.NotificationLog{
		UserID:       100,
		TemplateCode: "tpl_sent",
		Status:       "sent",
	}))
	require.NoError(t, repo.CreateLog(&model.NotificationLog{
		UserID:       101,
		TemplateCode: "tpl_other_user",
		Status:       "pending",
	}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread/count", nil)
	auth.SetActor(c, auth.Actor{UserID: 100, Role: model.RoleStudent, ClassID: 1, Grade: "2023"})

	handler.UnreadCount(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data struct {
			Count int64 `json:"count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, int64(2), resp.Data.Count)
}
