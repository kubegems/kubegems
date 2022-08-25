package models

import (
	"time"
)

type Announcement struct {
	ID      uint       `gorm:"primarykey" json:"id"`
	Type    string     `gorm:"type:varchar(50);" json:"type"`
	Message string     `json:"message"`
	StartAt *time.Time `json:"startAt"` // 开始时间，默认现在
	EndAt   *time.Time `json:"endAt"`   // 结束时间，默认一天后

	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}
