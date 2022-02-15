package clusterhandler

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/services/handlers"
)

type Handler struct {
	Path        string
	ModelClient client.ModelClientIface
}

func (h *Handler) List(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "ok1"}
	resp.WriteAsJson(msg)
}

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/" + h.Path)
	ws.Consumes(restful.MIME_JSON)

	ws.Route(handlers.ListCommonQuery(ws.GET("").
		To(h.List).
		Doc("list clusters").
		Returns(http.StatusOK, handlers.MessageOK, nil)))

	container.Add(ws)
}
