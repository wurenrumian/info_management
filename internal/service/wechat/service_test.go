package wechat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodeToOpenIDSuccess(t *testing.T) {
	mockResp := map[string]any{
		"openid":      "mock_openid_123",
		"session_key": "sk_xxx",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	svc := NewServiceWithBaseURL("fake_appid", "fake_secret", srv.URL)
	openID, err := svc.CodeToOpenID("test_code")
	require.NoError(t, err)
	require.Equal(t, "mock_openid_123", openID)
}

func TestCodeToOpenIDError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 40029,
			"errmsg":  "invalid code",
		})
	}))
	defer srv.Close()

	svc := NewServiceWithBaseURL("fake_appid", "fake_secret", srv.URL)
	_, err := svc.CodeToOpenID("bad_code")
	require.Error(t, err)
}
