package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"kubegems.io/pkg/log"
)

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
	state := uuid.NewString()
	url := ot.OauthConfig.AuthCodeURL(state)
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
	ret := &UserInfo{}
	if _, err := restyClient.SetHeader("Authorization", "Bearer "+token.AccessToken).R().SetResult(ret).Get(ot.opts.UserInfoURL); err != nil {
		log.Debugf("oauth2 get userinfo  failed: %v", err)
		return nil, err
	}

	if ret.Username == "" {
		if ret.User == "" {
			return nil, fmt.Errorf("failed to get username")
		} else {
			ret.Username = ret.User
		}
	}
	ret.Source = cred.Source
	return ret, nil
}
