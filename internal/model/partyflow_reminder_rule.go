package model

import (
	"time"

	"gorm.io/datatypes"
)

// PartyflowReminderRule defines one configurable reminder rule.
type PartyflowReminderRule struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	RuleCode           string         `gorm:"type:varchar(80);uniqueIndex;not null" json:"rule_code"`
	OrgType            string         `gorm:"type:varchar(20);index;not null" json:"org_type"`
	Status             string         `gorm:"type:varchar(40);index;not null" json:"status"`
	TriggerType        string         `gorm:"type:varchar(30);not null" json:"trigger_type"`
	TriggerEventCode   string         `gorm:"type:varchar(80)" json:"trigger_event_code"`
	OffsetDays         int            `json:"offset_days"`
	RepeatIntervalDays int            `json:"repeat_interval_days"`
	Title              string         `gorm:"type:varchar(100);not null" json:"title"`
	MessageTemplate    string         `gorm:"type:varchar(500);not null" json:"message_template"`
	Audience           string         `gorm:"type:varchar(20);not null" json:"audience"`
	Enabled            bool           `gorm:"index;not null;default:true" json:"enabled"`
	Metadata           datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}
