package notification

import (
	"context"
	"errors"
	"testing"

	"manage/internal/model"
	"manage/internal/testutil"

	"github.com/stretchr/testify/require"
)

type mockWechatClient struct {
	sendErr error
}

func (m *mockWechatClient) SendSubscribeMessage(openid, templateID, page string, data map[string]interface{}) error {
	return m.sendErr
}

type mockUserRepo struct {
	openID string
	err    error
}

func (m *mockUserRepo) GetUserOpenID(userID uint) (string, error) {
	return m.openID, m.err
}

func TestSendSuccess(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "test_remind",
		WechatTemplateID: "tmpl_123",
		Name:             "测试提醒",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	mockWechat := &mockWechatClient{sendErr: nil}
	mockUsers := &mockUserRepo{openID: "openid_123", err: nil}

	svc := NewService(mockWechat, repo, mockUsers)

	err := svc.Send(context.Background(), SendRequest{
		UserID:       1,
		TemplateCode: "test_remind",
		Page:         "/pages/index",
		TemplateData: map[string]interface{}{
			"thing1": map[string]string{"value": "测试通知"},
		},
	})
	require.NoError(t, err)

	logs, total, err := repo.ListLogs(LogFilter{UserID: ptrUint(1), Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "sent", logs[0].Status)
}

func TestSendNoOpenID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "test_remind",
		WechatTemplateID: "tmpl_123",
		Name:             "测试提醒",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	mockWechat := &mockWechatClient{}
	mockUsers := &mockUserRepo{openID: "", err: nil}

	svc := NewService(mockWechat, repo, mockUsers)

	err := svc.Send(context.Background(), SendRequest{
		UserID:       1,
		TemplateCode: "test_remind",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no openid")
}

func TestSendTemplateNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	mockWechat := &mockWechatClient{}
	mockUsers := &mockUserRepo{openID: "openid_123"}

	svc := NewService(mockWechat, repo, mockUsers)

	err := svc.Send(context.Background(), SendRequest{
		UserID:       1,
		TemplateCode: "nonexistent",
	})
	require.Error(t, err)
}

func TestSendWechatError(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "test_remind",
		WechatTemplateID: "tmpl_123",
		Name:             "测试提醒",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	mockWechat := &mockWechatClient{sendErr: errors.New("wechat error 43004")}
	mockUsers := &mockUserRepo{openID: "openid_123"}

	svc := NewService(mockWechat, repo, mockUsers)

	err := svc.Send(context.Background(), SendRequest{
		UserID:       1,
		TemplateCode: "test_remind",
	})
	require.Error(t, err)

	logs, _, _ := repo.ListLogs(LogFilter{UserID: ptrUint(1), Limit: 10})
	require.Equal(t, "failed", logs[0].Status)
	require.Contains(t, logs[0].ErrorMsg, "43004")
}

func TestSendBatch(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRepo(db)

	tmpl := &model.NotificationTemplate{
		Code:             "test_remind",
		WechatTemplateID: "tmpl_123",
		Name:             "测试提醒",
		Fields:           `{"thing1":"事项"}`,
	}
	require.NoError(t, repo.CreateTemplate(tmpl))

	mockWechat := &mockWechatClient{}
	mockUsers := &mockUserRepo{openID: "openid_123"}

	svc := NewService(mockWechat, repo, mockUsers)

	results := svc.SendBatch(context.Background(), "test_remind", []uint{1, 2, 3}, func(uid uint) map[string]interface{} {
		return map[string]interface{}{
			"thing1": map[string]string{"value": "通知"},
		}
	})
	require.Len(t, results, 3)
	for _, r := range results {
		require.NoError(t, r.Err)
	}
}
