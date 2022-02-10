package oauth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/spf13/pflag"
	"golang.org/x/oauth2"
	"kubegems.io/pkg/utils"
)

var _ot *OauthTool

type OauthOptions struct {
	Kind          string   `yaml:"kind"`
	UserInfoURL   string   `yaml:"userinfourl"`
	Appid         string   `yaml:"appid"`
	Appsecret     string   `yaml:"appsecret"`
	Scopes        []string `yaml:"scopes"`
	TokenURL      string   `yaml:"tokenurl"`
	AuthURL       string   `yaml:"authurl"`
	RedirectURL   string   `yaml:"redirecturl"`
	BambooOptions *BambooOptions
}

func (o *OauthOptions) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&o.Kind, utils.JoinFlagName(prefix, "kind"), o.Kind, "oauth kind")
	fs.StringVar(&o.Appid, utils.JoinFlagName(prefix, "appid"), o.Appid, "oauth client id")
	fs.StringVar(&o.Appsecret, utils.JoinFlagName(prefix, "appsecret"), o.Appsecret, "oauth client secret")
	fs.StringVar(&o.AuthURL, utils.JoinFlagName(prefix, "authurl"), o.AuthURL, "oauth authurl")
	fs.StringVar(&o.TokenURL, utils.JoinFlagName(prefix, "tokenurl"), o.TokenURL, "oauth token url")
	fs.StringVar(&o.RedirectURL, utils.JoinFlagName(prefix, "redirecturl"), o.RedirectURL, "oauth redirect url")
	fs.StringVar(&o.UserInfoURL, utils.JoinFlagName(prefix, "userinfourl"), o.UserInfoURL, "oauth userinfo url")
}

func NewDefaultOauthOptions() *OauthOptions {
	return &OauthOptions{
		Kind:          "gitlab",
		UserInfoURL:   "https://src.cloudminds.com/api/v4/user",
		Appid:         "",
		Appsecret:     "",
		Scopes:        []string{"api", "email"},
		TokenURL:      "https://src.cloudminds.com/oauth/token",
		AuthURL:       "https://src.cloudminds.com/oauth/authorize",
		RedirectURL:   "https://gemdev.cloudminds.com/oauth/callback",
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

func InitOauth(opts *OauthOptions) error {
	_ot = &OauthTool{
		ClientType:  opts.Kind,
		UserInfoURL: opts.UserInfoURL,
		OauthConfig: &oauth2.Config{
			ClientID:     opts.Appid,
			ClientSecret: opts.Appsecret,
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
	InitBamBooSyncTool(opts.BambooOptions)
	return nil
}

func GetOauthTool() *OauthTool {
	return _ot
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
