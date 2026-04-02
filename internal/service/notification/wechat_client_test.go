package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSendSubscribeMessageSuccess(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/cgi-bin/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fake_token",
			"expires_in":   7200,
		})
	})

	mux.HandleFunc("/cgi-bin/message/subscribe/send", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 0,
			"errmsg":  "ok",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewWechatClient(nil, "fake_appid", "fake_secret")
	client.baseURL = srv.URL

	err := client.SendSubscribeMessage("openid123", "tmpl_123", "/pages/index", map[string]interface{}{
		"thing1": map[string]string{"value": "测试通知"},
	})
	require.NoError(t, err)
}

func TestSendSubscribeMessageWechatError(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/cgi-bin/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fake_token",
			"expires_in":   7200,
		})
	})

	mux.HandleFunc("/cgi-bin/message/subscribe/send", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 43004,
			"errmsg":  "require subscribe",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewWechatClient(nil, "fake_appid", "fake_secret")
	client.baseURL = srv.URL

	err := client.SendSubscribeMessage("openid123", "tmpl_123", "/pages/index", map[string]interface{}{
		"thing1": map[string]string{"value": "测试通知"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "43004")
}

func TestSendSubscribeMessageTokenError(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/cgi-bin/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 40013,
			"errmsg":  "invalid appid",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewWechatClient(nil, "fake_appid", "fake_secret")
	client.baseURL = srv.URL

	err := client.SendSubscribeMessage("openid123", "tmpl_123", "/pages/index", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "get access token")
}
