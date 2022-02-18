package tenanthandler

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/utils"
)

func (h *Handler) AddTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	createForm := &forms.TenantUserCreateModifyForm{}
	if err := utils.BindData(req, createForm); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	tenant, err := h.getTenant(ctx, createForm.Tenant)
	if err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	user, err := h.getUser(ctx, createForm.User)
	if err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	rel := forms.TenantUserRelCommon{
		Tenant:   tenant.Data(),
		TenantID: tenant.Data().ID,
		User:     user.Data(),
		UserID:   user.Data().ID,
		Role:     createForm.Role,
	}
	if err := h.ModelClient.Create(ctx, rel.Object()); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(rel)
}

func (h *Handler) ModifyTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	createForm := &forms.TenantUserCreateModifyForm{}
	if err := utils.BindData(req, createForm); err != nil {
		utils.BadRequest(resp, err)
		return
	}
	tenant := forms.TenantCommon{}
	user := forms.UserCommon{}
	if err := h.ModelClient.Get(ctx, tenant.Object(), client.WhereEqual("tenant_name", createForm.Tenant)); err != nil {
		utils.BadRequest(resp, err)
		return
	}
	if err := h.ModelClient.Get(ctx, user.Object(), client.WhereEqual("username", createForm.User)); err != nil {
		utils.BadRequest(resp, err)
		return
	}
	rel := forms.TenantUserRelCommon{
		Tenant:   tenant.Data(),
		TenantID: tenant.Data().ID,
		User:     user.Data(),
		UserID:   user.Data().ID,
		Role:     createForm.Role,
	}
	if err := h.ModelClient.Update(ctx, rel.Object(), client.WhereEqual("tenant_id", tenant.Data().ID), client.WhereEqual("user_id", user.Data().ID)); err != nil {
		utils.BadRequest(resp, err)
		return
	}
	resp.WriteAsJson(rel)
}

func (h *Handler) DeleteTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	tenant := forms.TenantCommon{}
	user := forms.UserCommon{}
	if err := h.ModelClient.Get(ctx, tenant.Object(), client.WhereEqual("tenant_name", req.PathParameter("tenant"))); err != nil {
		utils.BadRequest(resp, err)
		return
	}
	if err := h.ModelClient.Get(ctx, user.Object(), client.WhereEqual("username", req.PathParameter("user"))); err != nil {
		utils.BadRequest(resp, err)
		return
	}

	rel := forms.TenantUserRelCommon{
		Tenant:   tenant.Data(),
		TenantID: tenant.Data().ID,
		User:     user.Data(),
		UserID:   user.Data().ID,
	}
	if err := h.ModelClient.Delete(ctx, rel.Object(), client.WhereEqual("tenant_id", tenant.Data().ID), client.WhereEqual("user_id", user.Data().ID)); err != nil {
		utils.BadRequest(resp, err)
		return
	}
	utils.NoContent(resp, nil)
}

func (h *Handler) ListTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	rel := &forms.UserCommonList{}
	tenant := &forms.TenantCommon{
		TenantName: req.PathParameter("tenant"),
	}
	opts := []client.Option{}
	if role := req.QueryParameter("role"); role != "" {
		opts = append(opts, client.ExistRelationWithKeyValue(tenant.Object(), "role", req.QueryParameter("role")))
	} else {
		opts = append(opts, client.ExistRelation(tenant.Object()))
	}
	if req.QueryParameter("isActive") != "" {
		opts = append(opts, client.Where("is_active", client.Eq, false))
	}
	if err := h.ModelClient.List(ctx, rel.Object(), opts...); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(rel.Data())
}
