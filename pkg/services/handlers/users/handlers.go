package userhandler

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
)

var (
	userTags = []string{"users"}
)

type Handler struct {
	ModelClient client.ModelClientIface
}

func (h *Handler) Create(req *restful.Request, resp *restful.Response) {
	user := &forms.UserDetail{}
	if err := handlers.BindData(req, user); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	if err := h.ModelClient.Create(req.Request.Context(), user.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.Created(resp, user.Data())
}

func (h *Handler) List(req *restful.Request, resp *restful.Response) {
	userList := forms.UserCommonList{}
	l := userList.Object()
	if err := h.ModelClient.List(req.Request.Context(), l, handlers.CommonOptions(req)...); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(l, userList.Data()))
}

func (h *Handler) Retrieve(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	var form forms.FormInterface
	if req.QueryParameter("detail") != "" {
		form = &forms.UserDetail{}
	} else {
		form = &forms.UserCommon{}
	}
	if err := h.ModelClient.Get(ctx, form.Object(), client.WhereNameEqual(req.PathParameter("name"))); err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
	}
	handlers.OK(resp, form.DataPtr())
}

func (h *Handler) Delete(req *restful.Request, resp *restful.Response) {
	user := forms.UserCommon{}
	if err := h.ModelClient.Delete(req.Request.Context(), user.Object(), client.WhereNameEqual(req.PathParameter("name"))); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) Put(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "put"}
	resp.WriteAsJson(msg)
}

func (h *Handler) Patch(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "patch"}
	resp.WriteAsJson(msg)
}
