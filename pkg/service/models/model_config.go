package models

import (
	"gorm.io/datatypes"
)

// OnlineConfig 系统配置
type OnlineConfig struct {
	Name    string         `gorm:"type:varchar(50);primaryKey" binding:"required" json:"name"` // 配置名
	Content datatypes.JSON `json:"content"`                                                    // 配置内容
}
