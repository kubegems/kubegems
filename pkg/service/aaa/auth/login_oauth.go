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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/oauth2"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils"
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
	Vendor      string
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

func NewOauthUtils(name, vendor string, opts *OauthOption) *OauthLoginUtils {
	return &OauthLoginUtils{
		Name:   name,
		Vendor: vendor,
		opts:   opts,
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
		return nil, i18n.Error(ctx, "exchange oauth2 token failed")
	}
	restyClient := resty.NewWithClient(ot.OauthConfig.Client(context.Background(), token))
	ret := &OauthCommonUserInfo{}
	if _, err := restyClient.SetHeader("Authorization", "Bearer "+token.AccessToken).R().SetResult(ret).Get(ot.opts.UserInfoURL); err != nil {
		log.Debugf("oauth2 get userinfo  failed: %v", err, "url", ot.opts.UserInfoURL)
		return nil, i18n.Error(ctx, "failed to get userinfo from oauth provider")
	}

	if ret.Username == "" {
		if ret.Name == "" {
			return nil, i18n.Error(ctx, "failed to get username from oauth provider")
		} else {
			ret.Username = ret.Name
		}
	}
	return &UserInfo{
		Username: ret.Username,
		Email:    ret.Email,
		Source:   cred.Source,
		Vendor:   ot.Vendor,
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
