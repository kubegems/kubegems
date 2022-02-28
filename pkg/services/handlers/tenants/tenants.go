package tenanthandler

import (
	"github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/handlers/base"
)

type Handler struct {
	base.BaseHandler
}

func (h *Handler) CreateTenant(req *restful.Request, resp *restful.Response) {
	obj := &models.TenantCommon{}
	if err := handlers.BindData(req, obj); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	tx := h.DBWithContext(req)
	tx = handlers.ScopeOmitAssociations(tx)
	if err := tx.Create(obj).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.Created(resp, obj)
}

func (h *Handler) ListTenant(req *restful.Request, resp *restful.Response) {
	ol := &[]models.TenantCommon{}
	scopes := []func(*gorm.DB) *gorm.DB{
		handlers.ScopeTable(ol),
		handlers.ScopeOrder(req, []string{"create_at"}),
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

func (h *Handler) RetrieveTenant(req *restful.Request, resp *restful.Response) {
	tx := h.DBWithContext(req)
	tenant := &models.TenantCommon{}
	conds := []*handlers.Cond{handlers.WhereNameEqual(req.PathParameter("tenant"))}
	tx = tx.Scopes(
		handlers.ScopeCondition(conds, tenant),
	)
	if err := tx.First(tenant).Error; err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	handlers.OK(resp, tenant)
}

func (h *Handler) DeleteTenant(req *restful.Request, resp *restful.Response) {
	tenant := &models.TenantCommon{Name: req.PathParameter("tenant")}
	ctx := req.Request.Context()
	if err := h.DB().WithContext(ctx).First(tenant, tenant).Error; err != nil {
		if handlers.IsNotFound(err) {
			handlers.NoContent(resp, nil)
		} else {
			handlers.BadRequest(resp, err)
		}
		return
	}
	if err := h.DB().WithContext(ctx).Delete(tenant).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) ModifyTenant(req *restful.Request, resp *restful.Response) {
	tenant := &models.TenantCommon{}
	if err := handlers.BindData(req, tenant); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	if err := h.DB().WithContext(req.Request.Context()).Where("name = ?", req.PathParameter("tenant")).Updates(tenant).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, tenant)
}
