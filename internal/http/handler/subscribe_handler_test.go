package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupSubscribeTest(t *testing.T) (*SubscribeHandler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	h := NewSubscribeHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("actor", auth.Actor{UserID: 1, Role: 1})
		c.Next()
	})
	r.POST("/user/subscribe/report", h.ReportSubscribe)
	r.POST("/wechat/callback", h.WechatCallback)
	return h, r
}

func TestReportSubscribeAccept(t *testing.T) {
	_, r := setupSubscribeTest(t)

	body, _ := json.Marshal(map[string]string{
		"template_code":      "deadline_remind",
		"wechat_template_id": "tmpl_abc",
		"status":             "accept",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/user/subscribe/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestReportSubscribeReject(t *testing.T) {
	_, r := setupSubscribeTest(t)

	body, _ := json.Marshal(map[string]string{
		"template_code":      "deadline_remind",
		"wechat_template_id": "tmpl_abc",
		"status":             "reject",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/user/subscribe/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestReportSubscribeInvalidStatus(t *testing.T) {
	_, r := setupSubscribeTest(t)

	body, _ := json.Marshal(map[string]string{
		"template_code":      "deadline_remind",
		"wechat_template_id": "tmpl_abc",
		"status":             "unknown",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/user/subscribe/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReportSubscribeIdempotent(t *testing.T) {
	db := testutil.NewTestDB(t)
	h := NewSubscribeHandler(db)

	actor := auth.Actor{UserID: 1, Role: 1}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("actor", actor)

	ctx.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(
		mustJSON(map[string]string{
			"template_code":      "test_tpl",
			"wechat_template_id": "tmpl_1",
			"status":             "accept",
		}),
	))
	ctx.Request.Header.Set("Content-Type", "application/json")
	h.ReportSubscribe(ctx)

	ctx.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(
		mustJSON(map[string]string{
			"template_code":      "test_tpl",
			"wechat_template_id": "tmpl_1",
			"status":             "reject",
		}),
	))
	ctx.Request.Header.Set("Content-Type", "application/json")
	h.ReportSubscribe(ctx)

	var count int64
	db.Model(&model.UserSubscribe{}).Where("user_id = ? AND template_code = ?", 1, "test_tpl").Count(&count)
	require.Equal(t, int64(1), count)
}

func TestWechatCallbackPopupEvent(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Create(&model.User{ID: 1, OpenID: ptrStr("test_openid")})

	h := NewSubscribeHandler(db)
	r := gin.New()
	r.POST("/wechat/callback", h.WechatCallback)

	xmlBody := `<xml>
		<ToUserName><![CDATA[gh_123]]></ToUserName>
		<FromUserName><![CDATA[test_openid]]></FromUserName>
		<CreateTime>1610969440</CreateTime>
		<MsgType><![CDATA[event]]></MsgType>
		<Event><![CDATA[subscribe_msg_popup_event]]></Event>
		<SubscribeMsgPopupEvent>
			<List>
				<TemplateId><![CDATA[tmpl_abc]]></TemplateId>
				<SubscribeStatusString><![CDATA[accept]]></SubscribeStatusString>
				<PopupScene>0</PopupScene>
			</List>
		</SubscribeMsgPopupEvent>
	</xml>`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/wechat/callback", bytes.NewReader([]byte(xmlBody)))
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "success", w.Body.String())
}

func TestWechatCallbackChangeEvent(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Create(&model.User{ID: 1, OpenID: ptrStr("test_openid")})
	db.Create(&model.UserSubscribe{UserID: 1, WechatTemplateID: "tmpl_abc", TemplateCode: "test", Status: "subscribed"})

	h := NewSubscribeHandler(db)
	r := gin.New()
	r.POST("/wechat/callback", h.WechatCallback)

	xmlBody := `<xml>
		<ToUserName><![CDATA[gh_123]]></ToUserName>
		<FromUserName><![CDATA[test_openid]]></FromUserName>
		<CreateTime>1610969440</CreateTime>
		<MsgType><![CDATA[event]]></MsgType>
		<Event><![CDATA[subscribe_msg_change_event]]></Event>
		<SubscribeMsgChangeEvent>
			<List>
				<TemplateId><![CDATA[tmpl_abc]]></TemplateId>
				<SubscribeStatusString><![CDATA[reject]]></SubscribeStatusString>
			</List>
		</SubscribeMsgChangeEvent>
	</xml>`

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/wechat/callback", bytes.NewReader([]byte(xmlBody)))
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var sub model.UserSubscribe
	db.Where("wechat_template_id = ?", "tmpl_abc").First(&sub)
	require.Equal(t, "unsubscribed", sub.Status)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func ptrStr(s string) *string {
	return &s
}
