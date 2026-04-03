package handler

import (
	"encoding/xml"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/model"
)

// SubscribeHandler handles subscription reporting and WeChat event callbacks.
type SubscribeHandler struct {
	db *gorm.DB
}

// NewSubscribeHandler creates a SubscribeHandler.
func NewSubscribeHandler(db *gorm.DB) *SubscribeHandler {
	return &SubscribeHandler{db: db}
}

type subscribeReportReq struct {
	TemplateCode     string `json:"template_code" binding:"required"`
	WechatTemplateID string `json:"wechat_template_id" binding:"required"`
	Status           string `json:"status" binding:"required"`
}

// ReportSubscribe handles POST /api/v1/user/subscribe/report.
func (h *SubscribeHandler) ReportSubscribe(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}

	var req subscribeReportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid request")
		return
	}

	status, ok := mapSubscribeStatus(req.Status)
	if !ok {
		response.Error(c, 400, "status must be accept, reject, ban or filter")
		return
	}

	var existing model.UserSubscribe
	err := h.db.Where("user_id = ? AND template_code = ?", actor.UserID, req.TemplateCode).First(&existing).Error

	now := time.Now()
	sub := model.UserSubscribe{
		UserID:           actor.UserID,
		TemplateCode:     req.TemplateCode,
		WechatTemplateID: req.WechatTemplateID,
		Status:           status,
		UpdatedAt:        now,
	}

	if err == gorm.ErrRecordNotFound {
		sub.SubscribedAt = now
		if err := h.db.Create(&sub).Error; err != nil {
			response.Error(c, 500, "failed to record subscription")
			return
		}
	} else {
		if err := h.db.Model(&model.UserSubscribe{}).Where("id = ?", existing.ID).Updates(sub).Error; err != nil {
			response.Error(c, 500, "failed to update subscription")
			return
		}
	}

	response.OK(c, gin.H{"ok": true})
}

// wechatEventXML represents the XML event push from WeChat server.
type wechatEventXML struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   string   `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Event        string   `xml:"Event"`
	List         []struct {
		TemplateID            string `xml:"TemplateId"`
		SubscribeStatusString string `xml:"SubscribeStatusString"`
		PopupScene            string `xml:"PopupScene"`
	} `xml:"SubscribeMsgPopupEvent>List"`
	ChangeEventList []struct {
		TemplateID            string `xml:"TemplateId"`
		SubscribeStatusString string `xml:"SubscribeStatusString"`
	} `xml:"SubscribeMsgChangeEvent>List"`
	SentEventList []struct {
		TemplateID  string `xml:"TemplateId"`
		MsgID       string `xml:"MsgID"`
		ErrorCode   string `xml:"ErrorCode"`
		ErrorStatus string `xml:"ErrorStatus"`
	} `xml:"SubscribeMsgSentEvent>List"`
}

// WechatCallback handles POST /api/v1/wechat/callback (WeChat server push).
func (h *SubscribeHandler) WechatCallback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	defer c.Request.Body.Close()
	if err != nil {
		c.String(400, "fail")
		return
	}

	var event wechatEventXML
	if err := xml.Unmarshal(body, &event); err != nil {
		c.String(200, "success")
		return
	}

	switch event.Event {
	case "subscribe_msg_popup_event":
		for _, item := range event.List {
			h.updateSubscribeByOpenID(event.FromUserName, item.TemplateID, item.SubscribeStatusString)
		}
	case "subscribe_msg_change_event":
		for _, item := range event.ChangeEventList {
			h.updateSubscribeByOpenID(event.FromUserName, item.TemplateID, item.SubscribeStatusString)
		}
	case "subscribe_msg_sent_event":
		for _, item := range event.SentEventList {
			log.Printf("wechat sent callback: openid=%s template=%s msg_id=%s error_code=%s error_status=%s",
				event.FromUserName, item.TemplateID, item.MsgID, item.ErrorCode, item.ErrorStatus)
		}
	}

	c.String(200, "success")
}

func (h *SubscribeHandler) updateSubscribeByOpenID(openID, templateID, statusStr string) {
	var user model.User
	if err := h.db.Where("open_id = ?", openID).First(&user).Error; err != nil {
		log.Printf("updateSubscribeByOpenID: user lookup failed for openID %s: %v", openID, err)
		return
	}

	status, ok := mapSubscribeStatus(statusStr)
	if !ok {
		log.Printf("updateSubscribeByOpenID: unknown status %q for user %d template %s", statusStr, user.ID, templateID)
		return
	}

	if err := h.db.Model(&model.UserSubscribe{}).
		Where("user_id = ? AND wechat_template_id = ?", user.ID, templateID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error; err != nil {
		log.Printf("updateSubscribeByOpenID: update failed for user %d, template %s: %v", user.ID, templateID, err)
	}
}

func mapSubscribeStatus(raw string) (string, bool) {
	switch raw {
	case "accept", "acceptWithAudio", "acceptWithAlert":
		return "subscribed", true
	case "reject":
		return "unsubscribed", true
	case "ban":
		return "banned", true
	case "filter":
		return "filtered", true
	default:
		return "", false
	}
}
