package model

import (
	"time"

	"gorm.io/datatypes"
)

// PartyflowEvent records one status/milestone/reminder event in partyflow lifecycle.
type PartyflowEvent struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	PartyflowStatusID uint           `gorm:"index;not null" json:"partyflow_status_id"`
	EventType         string         `gorm:"type:varchar(30);index;not null" json:"event_type"`
	EventCode         string         `gorm:"type:varchar(80);index;not null" json:"event_code"`
	FromStatus        string         `gorm:"type:varchar(40)" json:"from_status,omitempty"`
	ToStatus          string         `gorm:"type:varchar(40)" json:"to_status,omitempty"`
	Note              string         `gorm:"type:text" json:"note,omitempty"`
	HappenedAt        time.Time      `json:"happened_at"`
	Metadata          datatypes.JSON `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}
