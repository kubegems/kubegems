package loginhandler

import (
	"time"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	auth "kubegems.io/pkg/service/aaa/authentication"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/service/oauth"
	"kubegems.io/pkg/utils"
)

type LoginForm struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

// FakeLogin 实际上这个没有用的，只是为了生成swagger文档
// @Summary JWT登录
// @Tags AAAAA
// @Description 登录JWT
// @Accept  json
// @Produce  json
// @Param param body LoginForm true "表单"
// @Success 200 {string} string	"登录成功"
// @Failure 401 {string} string "登录失败"
// @Router /v1/login [post]
func (h *OAuthHandler) LoginHandler(c *gin.Context) {
	h.Midware.LoginHandler(c)
}

type OAuthHandler struct {
	Midware *auth.Middleware
}

// @Summary 获取OAUTH登录地址
// @Description 获取OAUTH登录地址
// @Tags AAAAA
// @Accept  json
// @Produce  json
// @Success 200 {string} string	"地址"
// @Router /v1/oauth/addr [get]
func (h *OAuthHandler) GetOauthAddr(c *gin.Context) {
	t := oauth.GetOauthTool()
	handlers.OK(c, t.GetAuthAddr())
}

// @Summary 获取OAUTH登录地址
// @Description 获取OAUTH登录地址
// @Tags AAAAA
// @Accept  json
// @Produce  json
// @Success 200 {string} string	"地址"
// @Router /v1/oauth/callback [get]
func (h *OAuthHandler) GetOauthToken(c *gin.Context) {
	t := oauth.GetOauthTool()
	code := c.Query("code")
	tok, err := t.GetAccessToken(code)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	ret, err := t.GetPersonInfo(tok)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	var user models.User
	now := time.Now()
	if err := h.Midware.Database.DB().First(&user, "username = ?", ret.Username).Error; err != nil {
		if models.IsNotFound(err) {
			active := true
			user = models.User{
				Username:     ret.Username,
				Email:        ret.Email,
				Password:     utils.GeneratePassword(),
				IsActive:     &active,
				LastLoginAt:  &now,
				SystemRoleID: 2,
			}
			h.Midware.Database.DB().Create(&user)
		}
	} else {
		user.LastLoginAt = &now
		h.Midware.Database.DB().Save(&user)
	}
	mw := h.Midware
	token := jwtgo.New(jwtgo.GetSigningMethod(mw.SigningAlgorithm))
	claims := token.Claims.(jwtgo.MapClaims)

	if mw.PayloadFunc != nil {
		for key, value := range mw.PayloadFunc(&user) {
			claims[key] = value
		}
	}
	expire := mw.TimeFunc().Add(mw.Timeout)
	claims["exp"] = expire.Unix()
	tokenString, err := token.SignedString(h.Midware.PrivateKey)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	mw.LoginResponse(c, 200, tokenString, expire)
}
