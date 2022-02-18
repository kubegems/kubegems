package tenanthandler

import (
	"context"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/utils"
)

type Handler struct {
	Path        string
	ModelClient client.ModelClientIface
}

func (h *Handler) getTenant(ctx context.Context, name string) (forms.TenantCommon, error) {
	tenant := forms.TenantCommon{}
	err := h.ModelClient.Get(ctx, tenant.Object(), client.Where("tenant_name", client.Eq, name))
	return tenant, err
}

func (h *Handler) getUser(ctx context.Context, name string) (forms.UserCommon, error) {
	obj := forms.UserCommon{}
	err := h.ModelClient.Get(ctx, obj.Object(), client.Where("username", client.Eq, name))
	return obj, err
}

func (h *Handler) getTenantProject(ctx context.Context, tenant, name string, detail bool) (forms.FormInterface, error) {
	var form forms.FormInterface
	if detail {
		form = &forms.ProjectDetail{}
	} else {
		form = &forms.ProjectCommon{}
	}
	tenantobj := forms.TenantCommon{
		TenantName: tenant,
	}
	err := h.ModelClient.Get(ctx, form.Object(), client.WhereEqual("project_name", name), client.BelongTo(tenantobj.Object()))
	return form, err
}

func (h *Handler) Create(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	obj := &forms.TenantDetail{}
	if err := utils.BindData(req, obj); err != nil {
		utils.BadRequest(resp, err)
		return
	}
	if err := h.ModelClient.Create(ctx, obj.Object()); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(obj)
}

func (h *Handler) List(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.TenantCommonList{}
	if err := h.ModelClient.List(ctx, ol.Object(), handlers.CommonOptions(req)...); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(handlers.PageList(ol.Object(), ol.Data()))
}

func (h *Handler) Retrieve(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	var form forms.FormInterface
	opt := client.Where("tenant_name", client.Eq, req.PathParameter("tenant"))
	if req.QueryParameter("detail") != "" {
		form = &forms.TenantDetail{}
	} else {
		form = &forms.TenantCommon{}
	}
	if err := h.ModelClient.Get(ctx, form.Object(), opt); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(form.DataPtr())
}

func (h *Handler) Delete(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	form := &forms.TenantCommon{}
	opt := client.Where("tenant_name", client.Eq, req.PathParameter("tenant"))
	if err := h.ModelClient.Get(ctx, form.Object(), opt); err != nil {
		utils.NoContent(resp, nil)
		return
	}
	if err := h.ModelClient.Delete(ctx, form.Object(), opt); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	utils.NoContent(resp, nil)
}

func (h *Handler) Put(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "put"}
	resp.WriteAsJson(msg)
}

func (h *Handler) Patch(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "patch"}
	resp.WriteAsJson(msg)
}
