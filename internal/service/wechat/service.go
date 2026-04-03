package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Service provides WeChat code2session lookups.
type Service struct {
	appID     string
	appSecret string
	baseURL   string
}

// NewService creates a WeChat API service with the official code2session endpoint.
func NewService(appID, appSecret string) *Service {
	return &Service{
		appID:     appID,
		appSecret: appSecret,
		baseURL:   "https://api.weixin.qq.com/sns/jscode2session",
	}
}

// NewServiceWithBaseURL creates a WeChat service with a custom base URL (for tests).
func NewServiceWithBaseURL(appID, appSecret, baseURL string) *Service {
	return &Service{
		appID:     appID,
		appSecret: appSecret,
		baseURL:   baseURL,
	}
}

type code2SessionResp struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// CodeToOpenID exchanges a login code for OpenID.
func (s *Service) CodeToOpenID(code string) (string, error) {
	url := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		s.baseURL, s.appID, s.appSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("wechat api request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response failed: %w", err)
	}

	var result code2SessionResp
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat api error: %d %s", result.ErrCode, result.ErrMsg)
	}

	if result.OpenID == "" {
		return "", fmt.Errorf("empty openid in response")
	}

	return result.OpenID, nil
}
