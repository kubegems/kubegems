// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"

	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
)

const (
	AccountLoginName = "account"
	DefaultLoginURL  = "/v1/login"
	TokenTypeJWT     = "JWT"
	TokenTypeBasic   = "BASIC"
	TokenTypePrivate = "PRIVATE-TOKEN"
)

type Credential struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
	Code     string `json:"code" form:"code"`
	Source   string `json:"source" form:"source"`
}

type UserInfo struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Source   string `json:"-"`
	Vendor   string `json:"vendor"`
}

// AuthenticateIface 所有登录插件需要实现AuthenticateIface接口
type AuthenticateIface interface {
	GetName() string
	// LoginAddr 获取登录地址
	LoginAddr() string
	// 验证凭据, 获取用户信息
	GetUserInfo(ctx context.Context, cred *Credential) (*UserInfo, error)
}

type AuthenticateModuleIface interface {
	GetAuthenticateModule(name string) AuthenticateIface
}

func NewAuthenticateModule(db *gorm.DB) *AuthenticateModule {
	return &AuthenticateModule{
		DB: db,
	}
}

type AuthenticateModule struct {
	DB *gorm.DB
}

func (l *AuthenticateModule) GetNameFromState(state string) (string, error) {
	return getNameFromState(state)
}

func (l *AuthenticateModule) GetAuthenticateModule(ctx context.Context, sourceName string) AuthenticateIface {
	authSource := models.AuthSource{}
	defaultUtil := &AccountLoginUtil{
		DB:   l.DB,
		Name: AccountLoginName,
	}
	if sourceName == AccountLoginName {
		return defaultUtil
	}

	if err := l.DB.WithContext(ctx).Where("name = ? and enabled = ?", sourceName, true).First(&authSource).Error; err != nil {
		log.Error(err, "no enabled auth source found", "name", sourceName)
		return defaultUtil
	}
	switch authSource.Kind {
	case "LDAP":
		ldapUt := &LdapLoginUtils{
			Vendor:       authSource.Vendor,
			BaseDN:       authSource.Config.BaseDN,
			Name:         authSource.Name,
			BindUsername: authSource.Config.BindUsername,
			BindPassword: authSource.Config.BindPassword,
			LdapAddr:     authSource.Config.LdapAddr,
			EnableTLS:    authSource.Config.EnableTLS,
		}
		return ldapUt
	case "OAUTH":
		opt := &OauthOption{
			AuthURL:     authSource.Config.AuthURL,
			TokenURL:    authSource.Config.TokenURL,
			UserInfoURL: authSource.Config.UserInfoURL,
			RedirectURL: authSource.Config.RedirectURL,
			AppID:       authSource.Config.AppID,
			AppSecret:   authSource.Config.AppSecret,
			Scopes:      authSource.Config.Scopes,
		}
		return NewOauthUtils(authSource.Name, authSource.Vendor, opt)
	}
	return nil
}
