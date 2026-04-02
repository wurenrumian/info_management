package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WechatClient provides an HTTP client for the WeChat subscribe message API.
type WechatClient struct {
	appID      string
	appSecret  string
	httpClient *http.Client
	baseURL    string
}

// NewWechatClient creates a WechatClient. If httpClient is nil, a default client is used.
func NewWechatClient(httpClient *http.Client, appID, appSecret string) *WechatClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &WechatClient{
		appID:      appID,
		appSecret:  appSecret,
		httpClient: httpClient,
		baseURL:    "https://api.weixin.qq.com",
	}
}

type subscribeMsgRequest struct {
	ToUser           string                 `json:"touser"`
	TemplateID       string                 `json:"template_id"`
	Page             string                 `json:"page"`
	Data             map[string]interface{} `json:"data"`
	MiniprogramState string                 `json:"miniprogram_state"`
}

type subscribeMsgResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// SendSubscribeMessage sends a subscribe message to the given openid.
func (c *WechatClient) SendSubscribeMessage(openid, templateID, page string, data map[string]interface{}) error {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	url := fmt.Sprintf("%s/cgi-bin/message/subscribe/send?access_token=%s", c.baseURL, accessToken)

	reqBody := subscribeMsgRequest{
		ToUser:           openid,
		TemplateID:       templateID,
		Page:             page,
		Data:             data,
		MiniprogramState: "formal",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var result subscribeMsgResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("wechat subscribe msg error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

type tokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

func (c *WechatClient) getAccessToken() (string, error) {
	url := fmt.Sprintf("%s/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		c.baseURL, c.appID, c.appSecret)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result tokenResp
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("get token error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return result.AccessToken, nil
}
