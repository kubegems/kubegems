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
	Name      string           `gorm:"unique"`
	Kind      string           `json:"kind"`
	Config    AuthSourceConfig `json:"config" validate:"json"`
	TokenType string           `json:"tokenType"`
	Enabled   bool             `json:"enabled"`
	CreatedAt *time.Time       `json:"createdAt"`
	UpdatedAt *time.Time       `json:"updatedAt"`
}

type AuthSourceConfig struct {
	// oauth
	AuthURL     string   `json:"url,omitempty"`
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
