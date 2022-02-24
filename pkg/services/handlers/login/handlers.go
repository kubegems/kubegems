package loginhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/auth"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/handlers/base"
)

var tags = []string{"login"}

type Handler struct {
	base.BaseHandler
}

func (h *Handler) Login(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	cred := &auth.Credential{}
	if err := handlers.BindData(req, cred); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	authModule := auth.NewAuthenticateModule(h.Model())
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
	h.Model().Update(req.Request.Context(), uinternel.Object())
	user := &forms.UserCommon{
		Name:  uinternel.Name,
		Email: uinternel.Email,
		ID:    uinternel.ID,
		Role:  uinternel.Role,
	}
	jwt := &auth.JWT{}
	token, _, err := jwt.GenerateToken(user, time.Duration(time.Hour*24))
	if err != nil {
		handlers.Unauthorized(resp, err)
	}
	handlers.OK(resp, token)
}

func (h *Handler) getOrCreateUser(ctx context.Context, uinfo *auth.UserInfo) (*forms.UserInternal, error) {
	u := forms.UserInternal{}
	err := h.Model().Get(ctx, u.Object(), client.WhereNameEqual(uinfo.Username))
	if err != nil {
		return u.Data(), nil
	}
	if !handlers.IsNotFound(err) {
		return nil, err
	}
	newUser := &forms.UserInternal{
		Name:   uinfo.Username,
		Email:  uinfo.Email,
		Source: uinfo.Source,
	}
	err = h.Model().Create(ctx, newUser.Object())
	return newUser.Data(), err
}
