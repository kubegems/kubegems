package loginhandler

import (
	"context"
	"time"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/auth"
	"kubegems.io/pkg/services/handlers"
)

var tags = []string{"login"}

type Handler struct {
	ModelClient client.ModelClientIface
}

func (h *Handler) Login(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	cred := &auth.Credential{}
	if err := handlers.BindData(req, cred); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	authModule := auth.NewAuthenticateModule(h.ModelClient)
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
	uinternel := h.getOrCreateUser(req.Request.Context(), uinfo)
	now := time.Now()
	uinternel.LastLoginAt = &now
	h.ModelClient.Update(req.Request.Context(), uinternel.Object())
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

func (h *Handler) getOrCreateUser(ctx context.Context, uinfo *auth.UserInfo) *forms.UserInternal {
	u := forms.UserInternal{}
	if err := h.ModelClient.Get(ctx, u.Object(), client.Where("username", client.Eq, uinfo.Username)); err != nil {
		return u.Data()
	}
	newUser := &forms.UserInternal{
		Name:   uinfo.Username,
		Email:  uinfo.Email,
		Source: uinfo.Source,
	}
	h.ModelClient.Create(ctx, newUser.Object())
	return newUser.Data()
}
