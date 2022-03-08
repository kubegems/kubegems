package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type AuthSource struct {
	ID        uint             `json:"id"`
	Name      string           `gorm:"unique" json:"name"`
	Kind      string           `json:"kind" binding:"oneof=LDAP OAUTH"`
	Config    AuthSourceConfig `json:"config" binding:"required,json"`
	TokenType string           `json:"tokenType" binding:"required,oneof=Bearer"`
	Enabled   bool             `json:"enabled"`
	CreatedAt *time.Time       `json:"createdAt"`
	UpdatedAt *time.Time       `json:"updatedAt"`
}

type AuthSourceSimple struct {
	ID      uint   `json:"id"`
	Name    string `gorm:"unique" json:"name"`
	Kind    string `json:"kind"`
	Enabled bool   `json:"enabled"`
}

func (AuthSourceSimple) TableName() string {
	return "auth_sources"
}

type AuthSourceConfig struct {
	AuthURL     string   `json:"authURL,omitempty"`
	TokenURL    string   `json:"tokenURL,omitempty"`
	UserInfoURL string   `json:"userInfoURL,omitempty"`
	RedirectURL string   `json:"redirectURL,omitempty"`
	AppID       string   `json:"appID,omitempty"`
	AppSecret   string   `json:"appSecret,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`

	// ldap
	Name         string `json:"name,omitempty"`
	LdapAddr     string `json:"ldapaddr,omitempty"`
	BaseDN       string `json:"basedn,omitempty"`
	BindUsername string `json:"binduser,omitempty"`
	BindPassword string `json:"password,omitempty"`
}

func (cfg *AuthSourceConfig) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	result := AuthSourceConfig{}
	err := json.Unmarshal(bytes, &result)
	*cfg = result
	return err
}

func (cfg AuthSourceConfig) Value() (driver.Value, error) {
	return json.Marshal(cfg)
}
