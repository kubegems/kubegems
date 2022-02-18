package tenanthandler

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/utils"
)

func (h *Handler) CreatePorject(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	createForm := &forms.ProjectCreateForm{}
	if err := utils.BindData(req, createForm); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	tenant, err := h.getTenant(ctx, req.PathParameter("tenant"))
	if err != nil {
		resp.WriteError(http.StatusBadRequest, err)
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
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(proj)
}

func (h *Handler) DeleteProject(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	proj, err := h.getTenantProject(ctx, req.PathParameter("tenant"), req.PathParameter("project"), false)
	if err != nil {
		utils.BadRequest(resp, err)
		return
	}
	if err := h.ModelClient.Delete(ctx, proj.Object()); err != nil {
		utils.BadRequest(resp, err)
		return
	}
	resp.WriteAsJson(nil)
}

func (h *Handler) RetrieveTenantProject(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	detail := req.QueryParameter("detail") != ""
	proj, err := h.getTenantProject(ctx, req.PathParameter("tenant"), req.PathParameter("project"), detail)
	if err != nil {
		utils.BadRequest(resp, err)
		return
	}
	resp.WriteAsJson(proj.DataPtr())
}
