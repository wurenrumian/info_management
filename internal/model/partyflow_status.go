package model

import (
	"time"

	"gorm.io/datatypes"
)

// PartyflowStatus stores one user's current party/league process status.
type PartyflowStatus struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UserID          uint           `gorm:"index;not null" json:"user_id"`
	OrgType         string         `gorm:"type:varchar(20);index;not null" json:"org_type"`
	Status          string         `gorm:"type:varchar(40);index;not null" json:"status"`
	StatusStartedAt time.Time      `json:"status_started_at"`
	JoinedAt        *time.Time     `json:"joined_at"`
	NextActionHint  string         `gorm:"type:varchar(255)" json:"next_action_hint"`
	Metadata        datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`

	User    User             `gorm:"foreignKey:UserID" json:"-"`
	History []PartyflowEvent `gorm:"foreignKey:PartyflowStatusID" json:"history,omitempty"`
}
