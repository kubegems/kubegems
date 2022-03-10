package approvehandler

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/v2/models"
	"kubegems.io/pkg/v2/services/handlers"
	"kubegems.io/pkg/v2/services/handlers/base"
)

type Handler struct {
	base.BaseHandler
}

type ApplyKind string

const (
	ApproveStatusPending  = "pending"
	ApproveStatusApproved = "approved"
	ApproveStatusRejected = "rejected"

	ApplyKindQuotaApply = "clusterQuota"
)

// List List Approve which status is not approved
func (h *Handler) List(req *restful.Request, resp *restful.Response) {
	ol := &[]models.TenantResourceQuotaApply{}
	scopes := []func(*gorm.DB) *gorm.DB{
		handlers.ScopeTable(ol),
		handlers.ScopeOrder(req, []string{"create_at"}),
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
	quotaApply2Approve(*ol)
	handlers.OK(resp, handlers.Page(db, total, ol))
}

func (h *Handler) Action(req *restful.Request, resp *restful.Response) {
	kind := req.PathParameter("kind")
	id := utils.ToUint(req.PathParameter("id"))
	action := req.PathParameter("action")
	var status string
	switch action {
	case "pass":
		status = ApproveStatusApproved
	case "reject":
		status = ApproveStatusRejected
	default:
		handlers.BadRequest(resp, fmt.Errorf("not supported action %s", action))
	}
	switch kind {
	case ApplyKindQuotaApply:
		obj := models.TenantResourceQuotaApply{Status: status}
		h.DBWithContext(req).Where("id = ?", id).Updates(obj)
	default:
		handlers.NotFound(resp, fmt.Errorf("not supported kind %s", kind))
		return
	}
	handlers.OK(resp, "ok")
}

func quotaApply2Approve(ol []models.TenantResourceQuotaApply) []Approve {
	ret := []Approve{}
	for idx := range ol {
		ret = append(ret, Approve{
			Title:   formatTitle(ol[idx]),
			Kind:    ApplyKindQuotaApply,
			KindID:  ol[idx].ID,
			Content: formatContent(ol[idx]),
			Time:    ol[idx].CreateAt,
		})
	}
	return ret
}

func formatTitle(apply models.TenantResourceQuotaApply) string {
	return fmt.Sprintf("%s 发起 集群资源申请", apply.Creator.Username)
}

func formatContent(apply models.TenantResourceQuotaApply) interface{} {
	return apply.Content
}
