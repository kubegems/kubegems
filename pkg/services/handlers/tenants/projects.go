package tenanthandler

import (
	"github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/handlers"
)

func (h *Handler) ListTenantProject(req *restful.Request, resp *restful.Response) {
	ol := &[]models.Project{}
	scopes := []func(*gorm.DB) *gorm.DB{
		handlers.ScopeTable(ol),
		handlers.ScopeOrder(req, []string{"create_at"}),
		handlers.ScopeBelongViaField(models.Tenant{}, ol, handlers.WhereNameEqual(req.PathParameter("tenant")), "tenant_id"),
		handlers.ScopeSearch(req, &models.Tenant{}, []string{"name"}),
	}
	var total int64
	if err := h.DBWithContext(req).Scopes(scopes...).Count(&total).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	scopes = append(scopes, handlers.ScopePageSize(req))
	db := h.DBWithContext(req).Scopes(scopes...).Find(ol)
	if err := db.Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(db, total, ol))
}

func (h *Handler) CreatePorject(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	createData := &ProjectCreateForm{}
	if err := handlers.BindData(req, createData); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	tenant, err := h.getTenant(ctx, req.PathParameter("tenant"))
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	proj := &models.Project{
		TenantID:      tenant.ID,
		Name:          createData.Name,
		ResourceQuota: []byte(createData.ResourceQuota),
		Remark:        createData.Remark,
	}
	if err := h.DB().WithContext(ctx).Create(proj).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.Created(resp, proj)
}

func (h *Handler) DeleteProject(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	proj, err := h.getProject(ctx, req.PathParameter("tenant"), req.PathParameter("project"))
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	if err := h.DB().WithContext(ctx).Delete(proj).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) RetrieveTenantProject(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	proj, err := h.getProject(ctx, req.PathParameter("tenant"), req.PathParameter("project"))
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	handlers.OK(resp, proj)
}
