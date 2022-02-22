package tenanthandler

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
)

func (h *Handler) AddTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	createForm := &forms.TenantUserCreateModifyForm{}
	if err := handlers.BindData(req, createForm); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	tenant, err := h.getTenantCommon(ctx, createForm.Tenant)
	if err != nil {
		handlers.NotFound(resp, err)
		return
	}
	user, err := h.getUserCommon(ctx, createForm.User)
	if err != nil {
		handlers.NotFound(resp, err)
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
		handlers.BadRequest(resp, err)
		return
	}
	handlers.Created(resp, rel)
}

func (h *Handler) ModifyTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	createForm := &forms.TenantUserCreateModifyForm{}
	if err := handlers.BindData(req, createForm); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	tenant, err := h.getTenantCommon(ctx, createForm.Tenant)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	user, err := h.getUserCommon(ctx, createForm.User)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
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
		handlers.BadRequest(resp, err)
		return
	}
	resp.WriteAsJson(rel)
}

func (h *Handler) DeleteTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	tenant := forms.TenantCommon{}
	user := forms.UserCommon{}
	if err := h.ModelClient.Get(ctx, tenant.Object(), client.WhereEqual("tenant_name", req.PathParameter("tenant"))); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	if err := h.ModelClient.Get(ctx, user.Object(), client.WhereEqual("username", req.PathParameter("user"))); err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	rel := forms.TenantUserRelCommon{
		Tenant:   tenant.Data(),
		TenantID: tenant.Data().ID,
		User:     user.Data(),
		UserID:   user.Data().ID,
	}
	if err := h.ModelClient.Delete(ctx, rel.Object(), client.WhereEqual("tenant_id", tenant.Data().ID), client.WhereEqual("user_id", user.Data().ID)); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) ListTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	rel := &forms.UserCommonList{}
	tenant := &forms.TenantCommon{
		Name: req.PathParameter("tenant"),
	}
	opts := []client.Option{}
	if role := req.QueryParameter("role"); role != "" {
		opts = append(opts, client.ExistRelationWithKeyValue(tenant.Object(), "role", req.QueryParameter("role")))
	} else {
		opts = append(opts, client.ExistRelation(tenant.Object()))
	}
	if req.QueryParameter("isActive") != "" {
		opts = append(opts, client.WhereEqual("is_active", false))
	}
	if err := h.ModelClient.List(ctx, rel.Object(), opts...); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, rel.Data())
}
