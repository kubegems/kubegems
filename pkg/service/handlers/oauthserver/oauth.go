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

package oauthserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/generates"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/golang-jwt/jwt"
	"kubegems.io/kubegems/pkg/service/aaa/auth"
	"kubegems.io/kubegems/pkg/service/handlers"
	kmodels "kubegems.io/kubegems/pkg/service/models"

	"github.com/gin-gonic/gin"
	kjwt "kubegems.io/kubegems/pkg/utils/jwt"
)

var (
	manager     *manage.Manager
	srv         *server.Server
	clientStore *store.ClientStore
	m           sync.Mutex
)

func Init(opts *kjwt.Options) {
	manager = manage.NewDefaultManager()
	manager.SetAuthorizeCodeTokenCfg(manage.DefaultAuthorizeCodeTokenCfg)

	// token store
	manager.MustTokenStorage(store.NewMemoryTokenStore())

	jwtkey, err := ioutil.ReadFile(opts.Key)
	if err != nil {
		panic(err)
	}
	// generate jwt access token
	manager.MapAccessGenerate(generates.NewJWTAccessGenerate("kubegems", jwtkey, jwt.SigningMethodRS256))
	// manager.MapAccessGenerate(generates.NewAccessGenerate())

	clientStore = store.NewClientStore()
	manager.MapClientStorage(clientStore)

	srv = server.NewServer(server.NewConfig(), manager)
	srv.SetClientInfoHandler(func(r *http.Request) (clientID string, clientSecret string, err error) {
		loader := auth.BearerTokenUserLoader{JWT: opts.ToJWT()}
		user, exist := loader.GetUser(r)
		if !exist {
			err = fmt.Errorf("user not exist")
			return
		}
		return user.GetUsername(), "", nil
	})
	srv.SetClientScopeHandler(func(tgr *oauth2.TokenGenerateRequest) (allowed bool, err error) {
		if tgr.Scope != "validate" {
			return false, fmt.Errorf("scope now only support 'validate' to validate token")
		}
		return true, nil
	})
}

func Validate(c *gin.Context) {
	token, err := srv.ValidationBearerToken(c.Request)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	data := map[string]interface{}{
		"expires_in": int64(token.GetAccessCreateAt().Add(token.GetAccessExpiresIn()).Sub(time.Now()).Seconds()),
		"client_id":  token.GetClientID(),
		"user_id":    token.GetUserID(),
	}
	handlers.OK(c, data)
}

func Token(c *gin.Context) {
	u, _ := c.Get("current_user")
	user := u.(*kmodels.User)
	clientStore.Set(user.Username, &models.Client{
		ID:     user.Username,
		Secret: "",
	})

	expireSeconds, _ := strconv.Atoi(c.Query("expire"))
	// default 2 hours
	if expireSeconds != 0 {
		m.Lock()
		defer m.Unlock()
		manager.SetClientTokenCfg(&manage.Config{
			AccessTokenExp: time.Duration(expireSeconds) * time.Second,
		})
	}

	if err := srv.HandleTokenRequest(c.Writer, c.Request); err != nil {
		handlers.NotOK(c, err)
		return
	}
	return
}
