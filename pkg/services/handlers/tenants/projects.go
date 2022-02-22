package tenanthandler

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
)

func (h *Handler) CreatePorject(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	createForm := &forms.ProjectCreateForm{}
	if err := handlers.BindData(req, createForm); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	tenant, err := h.getTenantCommon(ctx, req.PathParameter("tenant"))
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	proj := forms.ProjectDetail{
		Tenant:        tenant.Data(),
		TenantID:      tenant.Data().ID,
		ProjectName:   createForm.Name,
		ResourceQuota: []byte(createForm.ResourceQuota),
		Remark:        createForm.Remark,
	}
	if err := h.ModelClient.Create(ctx, proj.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.Created(resp, err)
}

func (h *Handler) DeleteProject(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	proj, err := h.getTenantProject(ctx, req.PathParameter("tenant"), req.PathParameter("project"), false)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	if err := h.ModelClient.Delete(ctx, proj.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) RetrieveTenantProject(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	proj, err := h.getTenantProject(ctx, req.PathParameter("tenant"), req.PathParameter("project"), req.QueryParameter("detail") != "")
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	handlers.OK(resp, proj.DataPtr())
}
