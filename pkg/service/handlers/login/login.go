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

package loginhandler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
	auth "kubegems.io/kubegems/pkg/service/aaa/auth"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/jwt"
)

type LoginForm struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}
type OAuthHandler struct {
	DB         *gorm.DB
	AuthModule auth.AuthenticateModule
	JWTOptions *jwt.Options
}

// FakeLogin 实际上这个没有用的，只是为了生成swagger文档
// @Summary      JWT登录
// @Tags         AAAAA
// @Description  登录JWT
// @Accept       json
// @Produce      json
// @Param        param  body      LoginForm  true  "表单"
// @Success      200    {string}  string     "登录成功"
// @Failure      401    {string}  string     "登录失败"
// @Router       /v1/login [post]
func (h *OAuthHandler) LoginHandler(c *gin.Context) {
	h.commonLogin(c)
}

// @Summary      获取OAUTH登录地址
// @Description  获取OAUTH登录地址
// @Tags         AAAAA
// @Accept       json
// @Produce      json
// @Success      200  {string}  string  "地址"
// @Router       /v1/oauth/addr [get]
func (h *OAuthHandler) GetOauthAddr(c *gin.Context) {
	source := c.Query("source")
	if source == "" {
		handlers.NotOK(c, fmt.Errorf("source not provide"))
		return
	}
	sourceUtil := h.AuthModule.GetAuthenticateModule(c.Request.Context(), source)
	if sourceUtil == nil {
		handlers.NotOK(c, fmt.Errorf("source not exist"))
		return
	}
	if sourceUtil.GetName() != source {
		log.Info("mismatch auth source name", "sourceut", sourceUtil.GetName(), "provided", source)
		handlers.NotOK(c, fmt.Errorf("source not match"))
		return
	}
	handlers.OK(c, sourceUtil.LoginAddr())
}

// @Summary      OAUTH登录callback
// @Description  OAUTH登录callback
// @Tags         AAAAA
// @Accept       json
// @Produce      json
// @Success      200     {string}  string  "地址"
// @Param        source  path      string  true  "loginsource"
// @Router       /v1/oauth/callback [get]
func (h *OAuthHandler) GetOauthToken(c *gin.Context) {
	h.commonLogin(c)
}

func (h *OAuthHandler) getOrCreateUser(ctx context.Context, uinfo *auth.UserInfo) (*models.User, error) {
	u := &models.User{}
	if err := h.DB.WithContext(ctx).First(u, "username = ?", uinfo.Username).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		active := true
		newUser := &models.User{
			Username:     uinfo.Username,
			Email:        uinfo.Email,
			IsActive:     &active,
			Source:       uinfo.Source,
			SourceVendor: uinfo.Vendor,
			// todo: get systemrole via code from db
			SystemRoleID: 2,
		}
		err := h.DB.WithContext(ctx).Create(newUser).Error
		return newUser, err
	} else {
		return u, nil
	}
}

func (h *OAuthHandler) commonLogin(c *gin.Context) {
	ctx := c.Request.Context()
	cred := &auth.Credential{}
	// POST for account login and ldap
	if c.Request.Method == http.MethodPost {
		if err := c.BindJSON(cred); err != nil {
			handlers.NotOK(c, err)
			return
		}
	} else {
		// GET for oauth
		cred.Code = c.Query("code")
		if cred.Code == "" {
			handlers.NotOK(c, fmt.Errorf("empty code"))
			return
		}
	}

	if cred.Source == "" {
		state := c.Query("state")
		if state == "" {
			handlers.Unauthorized(c, fmt.Errorf("state not provide"))
			return
		}
		source, err := h.AuthModule.GetNameFromState(state)
		if err != nil {
			handlers.Unauthorized(c, fmt.Errorf("failed to get auth source"))
			return
		}
		cred.Source = source
	}
	authenticator := h.AuthModule.GetAuthenticateModule(ctx, cred.Source)
	if authenticator == nil {
		handlers.Unauthorized(c, fmt.Errorf("auth source not exist"))
		return
	}
	if cred.Source != authenticator.GetName() {
		handlers.Unauthorized(c, "auth source not exists or not enabled")
		return
	}
	uinfo, err := authenticator.GetUserInfo(ctx, cred)
	if err != nil {
		log.Error(err, "get user info", "source", cred.Source, "username", cred.Username)
		handlers.Unauthorized(c, err.Error())
		return
	}
	uinternel, err := h.getOrCreateUser(ctx, uinfo)
	if err != nil {
		log.Error(err, "update user", "username", uinfo.Username)
		handlers.Unauthorized(c, "system error")
		return
	}
	now := time.Now()
	uinternel.LastLoginAt = &now
	h.DB.WithContext(ctx).Updates(uinternel)

	userpayload := &models.User{
		Username:     uinternel.Username,
		Email:        uinternel.Email,
		ID:           uinternel.ID,
		SystemRoleID: uinternel.SystemRoleID,
		Source:       uinternel.Source,
	}
	token, _, err := h.JWTOptions.ToJWT().GenerateToken(userpayload, userpayload.Username, h.JWTOptions.Expire)
	if err != nil {
		handlers.Unauthorized(c, err)
		return
	}
	data := map[string]string{"token": token}
	handlers.OK(c, data)
}
