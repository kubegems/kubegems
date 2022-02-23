package tenanthandler

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/handlers/base"
)

type Handler struct {
	base.BaseHandler
}

func (h *Handler) Create(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	obj := &forms.TenantDetail{}
	if err := handlers.BindData(req, obj); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	if err := h.Model().Create(ctx, obj.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.Created(resp, obj)
}

func (h *Handler) List(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.TenantCommonList{}
	if err := h.Model().List(ctx, ol.Object(), handlers.CommonOptions(req)...); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(ol.Object(), ol.Data()))
}

func (h *Handler) Retrieve(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	form, err := h.getTenant(ctx, req.PathParameter("tenant"), req.QueryParameter("detail") != "")
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	resp.WriteAsJson(form.DataPtr())
}

func (h *Handler) Delete(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	tenant, err := h.getTenantCommon(ctx, req.PathParameter("tenant"))
	if err != nil {
		handlers.NoContent(resp, nil)
		return
	}
	if err := h.Model().Delete(ctx, tenant.Object()); err != nil {
		handlers.BadRequest(resp, err)
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
