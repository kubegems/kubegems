package clusterhandler

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/services/handlers"
)

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/clusters")
	ws.Consumes(restful.MIME_JSON)

	ws.Route(handlers.ListCommonQuery(ws.GET("").
		To(h.List).
		Doc("list clusters").
		Returns(http.StatusOK, handlers.MessageOK, nil)))

	container.Add(ws)
}
