package auth

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"
	"kubegems.io/pkg/v2/models"
)

const (
	AccountLoginName = "account"
	DefaultLoginURL  = "/token"
	TokenTypeJWT     = "JWT"
	TokenTypeBasic   = "BASIC"
	TokenTypePrivate = "PRIVATE-TOKEN"
)

type Credential struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code"`
	Source   string `json:"source"`
}

type UserInfo struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Source   string `json:"-"`
}

// AuthenticateIface 所有登录插件需要实现AuthenticateIface接口
type AuthenticateIface interface {
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

func (l *AuthenticateModule) GetAuthenticateModule(ctx context.Context, name string) AuthenticateIface {
	authSources := &[]models.AuthSource{}
	l.DB.WithContext(ctx).Find(authSources)
	for _, source := range *authSources {
		if source.Name == name {
			switch source.Kind {
			case "LDAP":
				ldapUt := &LdapLoginUtils{}
				json.Unmarshal(source.Config, ldapUt)
				return ldapUt
			case "OAUTH":
				opt := &OauthOption{}
				json.Unmarshal(source.Config, opt)
				return NewOauthUtils(opt)
			}
		}
	}
	return &AccountLoginUtil{
		DB:   l.DB,
		Name: AccountLoginName,
	}
}
