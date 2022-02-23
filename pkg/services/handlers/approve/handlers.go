package approvehandler

import (
	"fmt"
	"time"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/handlers/base"
	"kubegems.io/pkg/utils"
)

type Approve struct {
	Title   string      `json:"title,omitempty"`
	Kind    ApplyKind   `json:"kind,omitempty"`
	KindID  uint        `json:"recordID,omitempty"`
	Content interface{} `json:"content,omitempty"`
	Time    time.Time   `json:"time,omitempty"`
	Status  string      `json:"status,omitempty"`
}

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

// List List TenantResouceQuotaApply which status is not approved
func (h *Handler) List(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.TenantResourceQuotaApplyCommonList{}
	if err := h.Model().List(ctx, ol.Object(), client.Preloads([]string{"Creator"}), client.WhereEqual("status", ApproveStatusPending)); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	ret := quotaApply2Approve(ol.Data())
	handlers.OK(resp, ret)
}

func (h *Handler) Action(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
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
		obj := forms.TenantResourceQuotaApplyCommon{Status: status}
		h.Model().Update(ctx, obj.Object(), client.WhereEqual("id", id))
	default:
		handlers.NotFound(resp, fmt.Errorf("not supported kind %s", kind))
		return
	}
}

func quotaApply2Approve(ol []*forms.TenantResourceQuotaApplyCommon) []Approve {
	ret := []Approve{}
	for idx := range ol {
		ret = append(ret, Approve{
			Title:   formatTitle(ol[idx]),
			Kind:    ApplyKindQuotaApply,
			KindID:  ol[idx].ID,
			Content: nil,
			Time:    *ol[idx].CreateAt,
		})
	}
	return ret
}

func formatTitle(apply *forms.TenantResourceQuotaApplyCommon) string {
	return fmt.Sprintf("%s 发起 集群资源申请", apply.Creator.Name)
}

func formatContent(apply *forms.TenantResourceQuotaApplyCommon) interface{} {
	return apply.Content
}
