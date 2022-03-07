package models

import (
	"time"

	"gorm.io/datatypes"
)

type AuthSource struct {
	ID        uint
	Name      string `gorm:"unique"`
	Kind      string
	Config    datatypes.JSON
	TokenType string
	Enabled   bool
	CreatedAt *time.Time
	UpdatedAt *time.Time // 创建时间
}
