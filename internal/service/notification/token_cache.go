package notification

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// TokenCache caches the WeChat access token and refreshes it before expiry.
type TokenCache struct {
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
	appID     string
	appSecret string
	client    *http.Client
	baseURL   string
}

// NewTokenCache creates a TokenCache.
func NewTokenCache(appID, appSecret string, client *http.Client) *TokenCache {
	if client == nil {
		client = &http.Client{}
	}
	return &TokenCache{
		appID:     appID,
		appSecret: appSecret,
		client:    client,
		baseURL:   "https://api.weixin.qq.com",
	}
}

// GetToken returns a valid access token, refreshing it if expired or about to expire.
func (c *TokenCache) GetToken() (string, error) {
	c.mu.RLock()
	if time.Now().Before(c.expiresAt) {
		token := c.token
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	return c.RefreshToken()
}

// RefreshToken fetches a new access token from WeChat API.
func (c *TokenCache) RefreshToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if time.Now().Before(c.expiresAt) {
		return c.token, nil
	}

	url := fmt.Sprintf("%s/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		c.baseURL, c.appID, c.appSecret)

	resp, err := c.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("get token error %d: %s", result.ErrCode, result.ErrMsg)
	}

	c.token = result.AccessToken
	// WeChat token expires in 7200s; refresh 5 minutes early.
	c.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	return c.token, nil
}
