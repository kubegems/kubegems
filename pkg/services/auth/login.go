package auth

import (
	"encoding/json"

	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
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
	GetUserInfo(cred *Credential) (*UserInfo, error)
}

type AuthenticateModuleIface interface {
	GetAuthenticateModule(name string) AuthenticateIface
}

func NewAuthenticateModule(client client.ModelClientIface) *AuthenticateModule {
	return &AuthenticateModule{
		ModelClient: client,
	}
}

type AuthenticateModule struct {
	ModelClient client.ModelClientIface
}

func (l *AuthenticateModule) GetAuthenticateModule(name string) AuthenticateIface {
	authSources := &forms.AuthSourceCommonList{}
	l.ModelClient.List(authSources.AsListObject())
	sources := authSources.AsListData()
	for _, source := range sources {
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
			default:
				return &AccountLoginUtil{
					ModelClient: l.ModelClient,
					Name:        AccountLoginName,
				}
			}
		}
	}
	return nil
}
