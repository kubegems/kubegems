package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type OauthOption struct {
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	RedirectURL string
	AppID       string
	AppSecret   string
	Scopes      []string
}

type OauthLoginUtils struct {
	Name        string
	Options     *OauthOption
	OauthConfig *oauth2.Config
	client      *http.Client
}

func NewOauthUtils(opts *OauthOption) *OauthLoginUtils {
	return &OauthLoginUtils{
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

func (ot *OauthLoginUtils) LoginAddr() string {
	state := uuid.NewString()
	url := ot.OauthConfig.AuthCodeURL(state)
	return url
}

func (ot *OauthLoginUtils) GetUserInfo(cred *Credential) (*UserInfo, error) {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, ot.client)
	token, err := ot.OauthConfig.Exchange(ctx, cred.Code)
	if err != nil {
		return nil, err
	}
	restyClient := resty.NewWithClient(ot.OauthConfig.Client(context.Background(), token))
	ret := &UserInfo{}
	if _, err := restyClient.SetHeader("Authorization", "Bearer "+token.AccessToken).R().SetResult(ret).Get(ot.Options.UserInfoURL); err != nil {
		return nil, err
	}

	if ret.Username == "" {
		return nil, fmt.Errorf("failed to get username")
	}
	ret.Source = cred.Source
	return ret, nil
}
