package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTokenCacheReturnsCachedToken(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/cgi-bin/token", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "cached_token",
			"expires_in":   7200,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cache := NewTokenCache("appid", "secret", nil)
	cache.baseURL = srv.URL

	token1, err := cache.GetToken()
	require.NoError(t, err)
	require.Equal(t, "cached_token", token1)
	require.Equal(t, 1, callCount)

	token2, err := cache.GetToken()
	require.NoError(t, err)
	require.Equal(t, "cached_token", token2)
	require.Equal(t, 1, callCount)
}

func TestTokenCacheRefreshesOnExpiry(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/cgi-bin/token", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token_v2",
			"expires_in":   6,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cache := NewTokenCache("appid", "secret", nil)
	cache.baseURL = srv.URL

	token1, err := cache.GetToken()
	require.NoError(t, err)
	require.Equal(t, 1, callCount)
	_ = token1

	time.Sleep(4 * time.Second)

	token2, err := cache.GetToken()
	require.NoError(t, err)
	require.Equal(t, "token_v2", token2)
	require.Equal(t, 2, callCount)
}

func TestTokenCacheRefreshesOnError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cgi-bin/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 40013,
			"errmsg":  "invalid appid",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cache := NewTokenCache("bad_appid", "bad_secret", nil)
	cache.baseURL = srv.URL

	_, err := cache.GetToken()
	require.Error(t, err)
	require.Contains(t, err.Error(), "40013")
}
