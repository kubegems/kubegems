package orm

import (
	"time"

	"gorm.io/datatypes"
)

// AuthSource 认证插件
// +gen type:object pkcolume:id pkfield:ID
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
