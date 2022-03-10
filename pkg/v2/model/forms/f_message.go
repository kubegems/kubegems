package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:UserMessageStatus
type UserMessageStatusCommon struct {
	BaseForm
	ID        uint           `json:"id,omitempty"`
	UserID    uint           `json:"userID,omitempty"`
	User      *UserCommon    `json:"user,omitempty"`
	MessageID uint           `json:"messageID,omitempty"`
	Message   *MessageCommon `json:"message,omitempty"`
	IsRead    bool           `json:"isRead,omitempty"`
}

// +genform object:Message
type MessageCommon struct {
	BaseForm
	ID          uint           `json:"id,omitempty"`
	MessageType string         `json:"messageType,omitempty"`
	Title       string         `json:"title,omitempty"`
	Content     datatypes.JSON `json:"content,omitempty"`
	CreatedAt   *time.Time     `json:"createdAt,omitempty"`
}
