package oauth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type Options struct {
	Kind          string         `json:"kind,omitempty" description:"oauth kind"`
	UserInfoURL   string         `json:"userInfoURL,omitempty" description:"user info url"`
	AppID         string         `json:"appID,omitempty" description:"app id"`
	AppSecret     string         `json:"appSecret,omitempty" description:"app secret"`
	Scopes        []string       `json:"scopes,omitempty" description:"scopes"`
	TokenURL      string         `json:"tokenURL,omitempty" description:"token url"`
	AuthURL       string         `json:"authURL,omitempty" description:"auth url"`
	RedirectURL   string         `json:"redirectURL,omitempty" description:"redirect url"`
	BambooOptions *BambooOptions `json:"bambooOptions,omitempty" description:"bamboo options"`
}

func NewDefaultOauthOptions() *Options {
	return &Options{
		Kind:          "gitlab",
		UserInfoURL:   "https://git.kubegems.io/api/v4/user",
		AppID:         "",
		AppSecret:     "",
		Scopes:        []string{"api", "email"},
		TokenURL:      "https://git.kubegems.io/oauth/token",
		AuthURL:       "https://git.kubegems.io/oauth/authorize",
		RedirectURL:   "https://kubegems.io/oauth/callback",
		BambooOptions: NewDefaultBambooOptions(),
	}
}

type OauthTool struct {
	OauthConfig *oauth2.Config
	client      *http.Client
	ClientType  string
	UserInfoURL string
}

type GitlabUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func NewOauthTool(opts *Options) *OauthTool {
	ot := &OauthTool{
		ClientType:  opts.Kind,
		UserInfoURL: opts.UserInfoURL,
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
	return ot
}

func (ot *OauthTool) GetAuthAddr() string {
	state := uuid.NewString()
	url := ot.OauthConfig.AuthCodeURL(state)
	return url
}

func (ot *OauthTool) GetAccessToken(code string) (*oauth2.Token, error) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, ot.client)
	return ot.OauthConfig.Exchange(ctx, code)
}

func (ot *OauthTool) GetPersonInfo(token *oauth2.Token) (*GitlabUser, error) {
	ctx := context.Background()
	client := ot.OauthConfig.Client(ctx, token)
	restyClient := resty.NewWithClient(client)
	switch ot.ClientType {
	case "gitlab":
		rep := &GitlabUser{}
		if _, err := restyClient.SetHeader("Authorization", "Bearer "+token.AccessToken).R().SetResult(rep).Get(ot.UserInfoURL); err != nil {
			return nil, err
		} else {
			return rep, nil
		}
	case "bamboo":
		// TODO: 添加竹云的oauth
		return nil, fmt.Errorf("unsupport oauth kind %s", ot.ClientType)
	default:
		return nil, fmt.Errorf("unsupport oauth kind %s", ot.ClientType)
	}
}
