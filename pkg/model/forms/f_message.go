package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:UserMessageStatus
type UserMessageStatusCommon struct {
	BaseForm
	ID        uint
	UserID    uint
	User      *UserCommon
	MessageID uint
	Message   *MessageCommon
	IsRead    bool
}

// +genform object:Message
type MessageCommon struct {
	BaseForm
	ID          uint
	MessageType string
	Title       string
	Content     datatypes.JSON
	CreatedAt   *time.Time
}
