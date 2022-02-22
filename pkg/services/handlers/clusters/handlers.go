package clusterhandler

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
)

type Handler struct {
	Path        string
	ModelClient client.ModelClientIface
}

func (h *Handler) List(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.ClusterCommonList{}
	if err := h.ModelClient.List(ctx, ol.Object(), handlers.CommonOptions(req)...); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(ol.Object(), ol.Data()))
}
