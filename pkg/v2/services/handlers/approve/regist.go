package approvehandler

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

// TODO: sync cluster quota

var approvalTags = []string{"approval"}

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/v2/approvals")
	ws.Consumes(restful.MIME_JSON)

	ws.Route(ws.GET("").
		To(h.List).
		Doc("list approvals").
		Metadata(restfulspec.KeyOpenAPITags, approvalTags).
		Returns(http.StatusBadRequest, "list error", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, ApproveListResp{}))

	ws.Route(ws.POST("/{kind}/{id}/{action}").
		To(h.Action).
		Doc("approve action").
		Metadata(restfulspec.KeyOpenAPITags, approvalTags).
		Param(restful.PathParameter("kind", "resource kind").PossibleValues([]string{ApplyKindQuotaApply})).
		Param(restful.PathParameter("id", "resource id")).
		Param(restful.PathParameter("action", "approval action").PossibleValues([]string{"pass", "reject"})).
		Returns(http.StatusBadRequest, "not supported action", handlers.Response{}).
		Returns(http.StatusNotFound, "approve not found", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, "ok"))

	container.Add(ws)
}
