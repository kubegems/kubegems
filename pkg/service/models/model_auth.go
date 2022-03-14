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
	Vendor    string           `gorm:"type:varchar(30)" json:"vendor" binding:"omitempty,oneof=github gitlab oauth ldap"`
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
	Vendor  bool   `json:"vendor"`
}

func (AuthSourceSimple) TableName() string {
	return "auth_sources"
}

type AuthSourceConfig struct {
	AuthURL     string   `json:"authURL,omitempty" binding:"omitempty,url,required_with=TokenURL UserInfoURL RedirectURL AppID AppSecret"`
	TokenURL    string   `json:"tokenURL,omitempty" binding:"omitempty,url,required_with=AuthURL UserInfoURL RedirectURL AppID AppSecret"`
	UserInfoURL string   `json:"userInfoURL,omitempty" binding:"omitempty,url,required_with=AuthURL TokenURL RedirectURL AppID AppSecret"`
	RedirectURL string   `json:"redirectURL,omitempty" binding:"omitempty,url,required_with=AuthURL TokenURL UserInfoURL AppID AppSecret"`
	AppID       string   `json:"appID,omitempty" binding:"required_with=AuthURL TokenURL UserInfoURL RedirectURL AppSecret"`
	AppSecret   string   `json:"appSecret,omitempty" binding:"required_with=AuthURL TokenURL UserInfoURL RedirectURL AppID"`
	Scopes      []string `json:"scopes,omitempty"`

	// ldap
	Name         string `json:"name,omitempty"`
	LdapAddr     string `json:"ldapaddr,omitempty" binding:"omitempty,hostname_port,required_with=BaseDN BindUsername BindPassword"`
	BaseDN       string `json:"basedn,omitempty" binding:"required_with=LdapAddr BindUsername BindPassword"`
	EnableTLS    bool   `json:"enableTLS,omitempty"`
	Filter       string `json:"filter,omitempty"`
	BindUsername string `json:"binduser,omitempty" binding:"required_with=LdapAddr BaseDN BindPassword"`
	BindPassword string `json:"password,omitempty" binding:"required_with=LdapAddr BaseDN BindUsername"`
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
