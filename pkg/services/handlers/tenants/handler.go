package tenanthandler

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/utils"
)

var (
	tags = []string{"tenants"}
)

type Handler struct {
	Path        string
	ModelClient client.ModelClientIface
}

func (h *Handler) Create(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	obj := &forms.TenantDetail{}
	if err := utils.BindData(req, obj); err != nil {
		utils.BadRequest(resp, err)
		return
	}
	if err := h.ModelClient.Create(ctx, obj.AsObject()); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(obj)
}

func (h *Handler) List(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.TenantCommonList{}
	l := ol.AsListObject()
	if err := h.ModelClient.List(ctx, l, handlers.CommonOptions(req)...); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(handlers.PageList(l, ol.AsListData()))
}

func (h *Handler) Retrieve(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "retrieve"}
	resp.WriteAsJson(msg)
}

func (h *Handler) Delete(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "retrieve"}
	resp.WriteAsJson(msg)
}

func (h *Handler) Put(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "put"}
	resp.WriteAsJson(msg)
}

func (h *Handler) Patch(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "patch"}
	resp.WriteAsJson(msg)
}
