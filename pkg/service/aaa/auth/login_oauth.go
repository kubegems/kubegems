package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/oauth2"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils"
)

const desKey = "semgebuk"

var des = &utils.DesEncryptor{
	Key: []byte(desKey),
}

type OauthOption struct {
	AuthURL     string   `json:"url"`
	TokenURL    string   `json:"tokenURL"`
	UserInfoURL string   `json:"userInfoURL"`
	RedirectURL string   `json:"redirectURL"`
	AppID       string   `json:"appID"`
	AppSecret   string   `json:"appSecret"`
	Scopes      []string `json:"scopes"`
}

type OauthLoginUtils struct {
	Name        string
	OauthConfig *oauth2.Config
	opts        *OauthOption
	client      *http.Client
}

// OauthCommonUserInfo adaptor all source
type OauthCommonUserInfo struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

func NewOauthUtils(name string, opts *OauthOption) *OauthLoginUtils {
	return &OauthLoginUtils{
		Name: name,
		opts: opts,
		OauthConfig: &oauth2.Config{
			ClientID:     opts.AppID,
			ClientSecret: opts.AppSecret,
			Scopes:       opts.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL:  opts.TokenURL,
				AuthURL:   opts.AuthURL,
				AuthStyle: oauth2.AuthStyleInParams,
			},
			RedirectURL: opts.RedirectURL,
		},
		client: &http.Client{},
	}
}

func (ot *OauthLoginUtils) GetName() string {
	return ot.Name
}

func (ot *OauthLoginUtils) LoginAddr() string {
	url := ot.OauthConfig.AuthCodeURL(generateState(ot.Name))
	return url
}

func (ot *OauthLoginUtils) GetUserInfo(ctx context.Context, cred *Credential) (*UserInfo, error) {
	ctxinner := context.WithValue(ctx, oauth2.HTTPClient, ot.client)
	token, err := ot.OauthConfig.Exchange(ctxinner, cred.Code)
	if err != nil {
		log.Debugf("oauth2 exchange token failed: %v", err)
		return nil, err
	}
	restyClient := resty.NewWithClient(ot.OauthConfig.Client(context.Background(), token))
	ret := &OauthCommonUserInfo{}
	if _, err := restyClient.SetHeader("Authorization", "Bearer "+token.AccessToken).R().SetResult(ret).Get(ot.opts.UserInfoURL); err != nil {
		log.Debugf("oauth2 get userinfo  failed: %v", err)
		return nil, err
	}

	if ret.Username == "" {
		if ret.Name == "" {
			return nil, fmt.Errorf("failed to get username")
		} else {
			ret.Username = ret.Name
		}
	}
	return &UserInfo{
		Username: ret.Username,
		Email:    ret.Email,
		Source:   cred.Source,
	}, nil
}

func generateState(name string) string {
	s := fmt.Sprintf("%d/%s", time.Now().Unix(), name)
	state, _ := des.EncryptBase64(s)
	return state
}

func getNameFromState(state string) (string, error) {
	s, err := des.DecryptBase64(state)
	if err != nil {
		return "", err
	}
	seps := strings.Split(s, "/")
	if len(seps) != 2 {
		return "", fmt.Errorf("failed to get state")
	}
	return seps[1], nil
}
