package tenanthandler

import (
	"github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/v2/models"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

func (h *Handler) AddTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()

	createData := &TenantUserCreateForm{}
	if err := handlers.BindData(req, createData); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	tenant, err := h.getTenant(ctx, req.PathParameter("tenant"))
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	user, err := h.getUser(ctx, createData.User)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	rel := models.TenantUserRels{
		TenantID: tenant.ID,
		UserID:   user.ID,
		Role:     createData.Role,
	}
	if err := h.DB().WithContext(ctx).Create(rel).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.Created(resp, createData)
}

func (h *Handler) ModifyTenantMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	updateData := &TenantUserCreateForm{}
	if err := handlers.BindData(req, updateData); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	tenant, err := h.getTenant(ctx, req.PathParameter("tenant"))
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	user, err := h.getUser(ctx, updateData.User)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	rel := &models.TenantUserRels{TenantID: tenant.ID, UserID: user.ID}

	if err := h.DB().WithContext(ctx).First(rel, rel).Error; err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	rel.Role = updateData.Role
	if err := h.DB().WithContext(ctx).Updates(rel).Error; err != nil {
		handlers.BadRequest(resp, err)
	}
	handlers.OK(resp, updateData)
}

func (h *Handler) DeleteTenantMember(req *restful.Request, resp *restful.Response) {
	rel := &models.TenantUserRels{}
	scopes := []func(*gorm.DB) *gorm.DB{
		handlers.ScopeTable(rel),
		handlers.ScopeBelongViaField(models.User{}, rel, handlers.WhereEqual("username", req.PathParameter("user")), "user_id"),
		handlers.ScopeBelongViaField(models.Tenant{}, rel, handlers.WhereNameEqual(req.PathParameter("tenant")), "tenant_id"),
	}
	if err := h.DBWithContext(req).Scopes(scopes...).First(rel).Error; err != nil {
		if handlers.IsNotFound(err) {
			handlers.NoContent(resp, nil)
		} else {
			handlers.BadRequest(resp, err)
		}
		return
	}
	if err := h.DBWithContext(req).Delete(rel).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.NoContent(resp, nil)

}

func (h *Handler) ListTenantMember(req *restful.Request, resp *restful.Response) {
	ol := &[]models.UserSimple{}
	scopes := []func(*gorm.DB) *gorm.DB{
		handlers.ScopeTable(ol),
		handlers.ScopeSearch(req, ol, []string{"username"}),
		handlers.ScopeBelongM2M(models.Tenant{}, ol, models.TenantUserRels{}, handlers.WhereNameEqual(req.PathParameter("tenant")), "user_id", "tenant_id"),
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
