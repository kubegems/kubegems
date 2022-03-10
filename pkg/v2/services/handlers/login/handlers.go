package loginhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/jwt"
	"kubegems.io/pkg/v2/models"
	"kubegems.io/pkg/v2/services/auth"
	"kubegems.io/pkg/v2/services/handlers"
	"kubegems.io/pkg/v2/services/handlers/base"
)

var tags = []string{"login"}

type Handler struct {
	base.BaseHandler
	JWTOptions *jwt.Options
}

func (h *Handler) Login(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	cred := &auth.Credential{}
	if err := handlers.BindData(req, cred); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	authModule := auth.NewAuthenticateModule(h.DB())
	authenticator := authModule.GetAuthenticateModule(ctx, cred.Source)
	if authenticator == nil {
		handlers.Unauthorized(resp, nil)
		return
	}
	uinfo, err := authenticator.GetUserInfo(ctx, cred)
	if err != nil {
		handlers.Unauthorized(resp, err)
		return
	}
	uinternel, err := h.getOrCreateUser(req.Request.Context(), uinfo)
	if err != nil {
		log.Error(err, "handle login error", "username", uinfo.Username)
		handlers.Unauthorized(resp, fmt.Errorf("system error"))
		return
	}
	now := time.Now()
	uinternel.LastLoginAt = &now
	h.DBWithContext(req).Updates(uinternel)
	user := &models.UserCommon{
		Username: uinternel.Username,
		Email:    uinternel.Email,
		ID:       uinternel.ID,
	}

	jwtInstance := h.JWTOptions.ToJWT()
	token, _, err := jwtInstance.GenerateToken(user, h.JWTOptions.Expire)
	if err != nil {
		handlers.Unauthorized(resp, err)
	}
	handlers.OK(resp, token)
}

func (h *Handler) getOrCreateUser(ctx context.Context, uinfo *auth.UserInfo) (*models.User, error) {
	u := &models.User{}
	if err := h.DB().WithContext(ctx).First(u, "username = ?", uinfo.Username).Error; err != nil {
		if !handlers.IsNotFound(err) {
			return nil, err
		}
	} else {
		return u, nil
	}
	newUser := &models.User{
		Username: uinfo.Username,
		Email:    uinfo.Email,
		Source:   uinfo.Source,
	}
	err := h.DB().WithContext(ctx).Create(newUser).Error
	return newUser, err
}
