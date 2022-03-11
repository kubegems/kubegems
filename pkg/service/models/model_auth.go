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
	AuthURL     string   `json:"authURL,omitempty" binding:"url,require_with=TokenURL UserInfoURL RedirectURL AppID AppSecret"`
	TokenURL    string   `json:"tokenURL,omitempty" binding:"url,require_with=AuthURL UserInfoURL RedirectURL AppID AppSecret"`
	UserInfoURL string   `json:"userInfoURL,omitempty" binding:"url,require_with=AuthURL TokenURL RedirectURL AppID AppSecret"`
	RedirectURL string   `json:"redirectURL,omitempty" binding:"url,require_with=AuthURL TokenURL UserInfoURL AppID AppSecret"`
	AppID       string   `json:"appID,omitempty" binding:"require_with=AuthURL TokenURL UserInfoURL RedirectURL AppSecret"`
	AppSecret   string   `json:"appSecret,omitempty" binding:"require_with=AuthURL TokenURL UserInfoURL RedirectURL AppID"`
	Scopes      []string `json:"scopes,omitempty"`

	// ldap
	Name         string `json:"name,omitempty"`
	LdapAddr     string `json:"ldapaddr,omitempty" binding:"require_with=BaseDN EnableTLS BindUsername BindPassword"`
	BaseDN       string `json:"basedn,omitempty" binding:"require_with=LdapAddr EnableTLS BindUsername BindPassword"`
	EnableTLS    bool   `json:"enableTLS,omitempty" binding:"require_with=LdapAddr BaseDN BindUsername BindPassword"`
	BindUsername string `json:"binduser,omitempty" binding:"require_with=LdapAddr BaseDN EnableTLS BindPassword"`
	BindPassword string `json:"password,omitempty" binding:"require_with=LdapAddr BaseDN EnableTLS BindUsername"`
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
