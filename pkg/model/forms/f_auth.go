package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:AuthSource
type AuthSourceCommon struct {
	BaseForm
	ID        uint           `json:"id"`
	Name      string         `json:"name"` // auth plugin name
	Kind      string         `json:"kind"` // ldap, oauth
	TokenType string         `json:"tokenType"`
	Config    datatypes.JSON `json:"config"`
	Enabled   bool           `json:"enabled"`
	CreatedAt *time.Time     `json:"createAt"`
	UpdatedAt *time.Time     `json:"updateAt"`
}
