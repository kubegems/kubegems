package tenanthandler

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/utils"
)

func (h *Handler) AddTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	rel := &forms.TenantUserRelCommon{}
	if err := utils.BindData(req, rel); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	if err := h.ModelClient.Create(ctx, rel.AsObject()); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(rel)
}

func (h *Handler) ListTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	tenantPK := handlers.ParsePrimaryKey(req.PathParameter("tenant"))
	rel := &forms.UserCommonList{}
	tenant := &forms.TenantCommon{}
	if tenantPK.IsInt() {
		tenant.ID = tenantPK.Uint()
	} else {
		tenant.TenantName = tenantPK.String()
	}
	opts := []client.Option{}
	if role := req.QueryParameter("role"); role != "" {
		opts = append(opts, client.ExistRelationWithKeyValue(tenant.AsObject(), "role", req.QueryParameter("role")))
	} else {
		opts = append(opts, client.ExistRelation(tenant.AsObject()))
	}
	if req.QueryParameter("isActive") != "" {
		opts = append(opts, client.Where("is_active", client.Eq, false))
	}
	if err := h.ModelClient.List(ctx, rel.AsListObject(), opts...); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(rel.AsListData())
}
