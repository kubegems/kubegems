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
	"kubegems.io/kubegems/pkg/service/handlers/base"
	kmodels "kubegems.io/kubegems/pkg/service/models"

	"github.com/gin-gonic/gin"
	kjwt "kubegems.io/kubegems/pkg/utils/jwt"
)

type OauthServer struct {
	base.BaseHandler
	manager     *manage.Manager
	srv         *server.Server
	clientStore *store.ClientStore
	m           sync.Mutex
}

func NewOauthServer(opts *kjwt.Options, base base.BaseHandler) *OauthServer {
	s := &OauthServer{
		BaseHandler: base,
		manager:     manage.NewDefaultManager(),
		clientStore: store.NewClientStore(),
	}
	s.manager.SetAuthorizeCodeTokenCfg(manage.DefaultAuthorizeCodeTokenCfg)

	// token store
	s.manager.MustTokenStorage(store.NewMemoryTokenStore())

	jwtkey, err := ioutil.ReadFile(opts.Key)
	if err != nil {
		panic(err)
	}
	// generate jwt access token
	s.manager.MapAccessGenerate(generates.NewJWTAccessGenerate("", jwtkey, jwt.SigningMethodRS256))
	// manager.MapAccessGenerate(generates.NewAccessGenerate())

	s.manager.MapClientStorage(s.clientStore)

	s.srv = server.NewServer(server.NewConfig(), s.manager)
	s.srv.SetClientInfoHandler(func(r *http.Request) (clientID string, clientSecret string, err error) {
		loader := auth.BearerTokenUserLoader{JWT: opts.ToJWT()}
		user, exist := loader.GetUser(r)
		if !exist {
			err = fmt.Errorf("user not exist")
			return
		}
		return user.GetUsername(), "", nil
	})
	s.srv.SetClientScopeHandler(func(tgr *oauth2.TokenGenerateRequest) (allowed bool, err error) {
		if tgr.Scope != "validate" {
			return false, fmt.Errorf("scope now only support 'validate' to validate token")
		}
		return true, nil
	})
	return s
}

// @Tags        Oauth
// @Summary     检验oauth jwt token
// @Description 检验oauth jwt token
// @Accept      json
// @Produce     json
// @Success     200 {object} object "resp"
// @Router      /v1/oauth/validate [get]
func (s *OauthServer) Validate(c *gin.Context) {
	token, err := s.srv.ValidationBearerToken(c.Request)
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

// @Tags        Oauth
// @Summary     用户token列表
// @Description 用户token列表
// @Accept      json
// @Produce     json
// @Param       page query    int                                                                       false "page"
// @Param       size query    int                                                                       false "size"
// @Success     200  {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]kmodels.UserToken}} "resp"
// @Router      /v1/oauth/token [get]
// @Security    JWT
func (s *OauthServer) ListToken(c *gin.Context) {
	u, _ := c.Get("current_user")
	user := u.(*kmodels.User)
	ret := []*kmodels.UserToken{}
	if err := s.GetDB().Find(&ret, "user_id = ?", user.ID).Order("created_at").Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	now := time.Now()
	for _, v := range ret {
		v.Expired = now.After(*v.ExpireAt)
	}
	handlers.OK(c, handlers.NewPageDataFromContext(c, ret, nil, nil))
}

// @Tags        Oauth
// @Summary     删除用户token
// @Description 删除用户token
// @Accept      json
// @Produce     json
// @Param       token_id path     int    true "token id"
// @Success     200      {object} string "resp"
// @Router      /v1/oauth/token/{token_id} [delete]
// @Security    JWT
func (s *OauthServer) DeleteToken(c *gin.Context) {
	u, _ := c.Get("current_user")
	user := u.(*kmodels.User)
	t := kmodels.UserToken{}
	if err := s.GetDB().Delete(&t, "user_id = ? and id = ?", user.ID, c.Param("token_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "OK")
}

// @Tags        Oauth
// @Summary     签发oauth jwt token
// @Description 签发oauth jwt token
// @Accept      json
// @Produce     json
// @Param       grant_type query    string                               true "授权方式，目前只支持client_credentials"
// @Param       scope      query    string                               true "授权范围，目前只支持validate"
// @Param       expire     query    int                                  true "授权时长，单位秒"
// @Success     200        {object} handlers.ResponseStruct{Data=object} "resp"
// @Router      /v1/oauth/token [post]
// @Security    JWT
func (s *OauthServer) Token(c *gin.Context) {
	u, _ := c.Get("current_user")
	user := u.(*kmodels.User)
	s.clientStore.Set(user.Username, &models.Client{
		ID:     user.Username,
		Secret: "",
	})

	expireSeconds, _ := strconv.Atoi(c.Query("expire"))
	// default 2 hours
	if expireSeconds != 0 {
		s.m.Lock()
		defer s.m.Unlock()
		s.manager.SetClientTokenCfg(&manage.Config{
			AccessTokenExp: time.Duration(expireSeconds) * time.Second,
		})
	}

	// if err := srv.HandleTokenRequest(c.Writer, c.Request); err != nil {
	// 	handlers.NotOK(c, err)
	// 	return
	// }
	gt, tgr, err := s.srv.ValidationTokenRequest(c.Request)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	ti, err := s.srv.GetAccessToken(c.Request.Context(), gt, tgr)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	createdAt := ti.GetAccessCreateAt()
	exp := createdAt.Add(ti.GetAccessExpiresIn())
	t := kmodels.UserToken{
		Token:     ti.GetAccess(),
		GrantType: gt.String(),
		Scope:     tgr.Scope,
		ExpireAt:  &exp,
		UserID:    &user.ID,
		CreatedAt: &createdAt,
	}
	if err := s.GetDB().Create(&t).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, s.srv.GetTokenData(ti))
}

func (s *OauthServer) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/oauth/token", s.ListToken)
	rg.POST("/oauth/token", s.Token)
	rg.DELETE("/oauth/token/:token_id", s.DeleteToken)
}
