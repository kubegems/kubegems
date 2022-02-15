package orm

import (
	"time"

	"gorm.io/datatypes"
)

// UserMessageStatus 用户消息已读状态表
// +gen type:object pkcolume:id pkfield:ID
type UserMessageStatus struct {
	ID        uint
	UserID    uint
	User      *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	MessageID uint
	Message   *Message `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	IsRead    bool
}

// Message 用户消息通知表
// +gen type:object pkcolume:id pkfield:ID
type Message struct {
	ID          uint   `gorm:"primarykey"`
	MessageType string `gorm:"type:varchar(50);"`
	Title       string `gorm:"type:varchar(255);"`
	Content     datatypes.JSON
	CreatedAt   *time.Time        `gorm:"index" sql:"DEFAULT:'current_timestamp'"`
	ToUsers     map[uint]struct{} `gorm:"-" json:"-"`
}
