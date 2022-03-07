package loginhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	auth "kubegems.io/pkg/service/aaa/auth"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/jwt"
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
	h.commonLogin(c)
}

// @Summary 获取OAUTH登录地址
// @Description 获取OAUTH登录地址
// @Tags AAAAA
// @Accept  json
// @Produce  json
// @Success 200 {string} string	"地址"
// @Router /v1/oauth/addr [get]
func (h *OAuthHandler) GetOauthAddr(c *gin.Context) {
	source := c.Query("source")
	if source == "" {
		handlers.NotOK(c, fmt.Errorf("source not provide"))
		return
	}
	sourceUtil := h.AuthModule.GetAuthenticateModule(c.Request.Context(), source)
	handlers.OK(c, sourceUtil.LoginAddr())
}

// @Summary OAUTH登录callback
// @Description OAUTH登录callback
// @Tags AAAAA
// @Accept  json
// @Produce  json
// @Success 200 {string} string	"地址"
// @Param source path string true "loginsource"
// @Router /v1/oauth/callback/{source} [get]
func (h *OAuthHandler) GetOauthToken(c *gin.Context) {
	h.commonLogin(c)
}

func (h *OAuthHandler) getOrCreateUser(ctx context.Context, uinfo *auth.UserInfo) (*models.User, error) {
	u := &models.User{}
	if err := h.DB.WithContext(ctx).First(u, "username = ?", uinfo.Username).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		newUser := &models.User{
			Username: uinfo.Username,
			Email:    uinfo.Email,
			Source:   uinfo.Source,
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
	cred.Source = c.Param("source")
	if err := c.BindJSON(cred); err != nil {
		handlers.NotOK(c, err)
		return
	}
	authenticator := h.AuthModule.GetAuthenticateModule(ctx, cred.Source)
	uinfo, err := authenticator.GetUserInfo(ctx, cred)
	if err != nil {
		handlers.Unauthorized(c, err)
		return
	}
	uinternel, err := h.getOrCreateUser(ctx, uinfo)
	if err != nil {
		log.Error(err, "handle login error", "username", uinfo.Username)
		handlers.Unauthorized(c, fmt.Errorf("system error"))
		return
	}
	now := time.Now()
	uinternel.LastLoginAt = &now
	h.DB.WithContext(ctx).Updates(uinternel)
	user := &models.User{
		Username: uinternel.Username,
		Email:    uinternel.Email,
		ID:       uinternel.ID,
	}

	jwtInstance := h.JWTOptions.ToJWT()
	token, _, err := jwtInstance.GenerateToken(user, h.JWTOptions.Expire)
	if err != nil {
		handlers.Unauthorized(c, err)
	}
	handlers.OK(c, token)
}
