package loginhandler

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/services/handlers"
)

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/login")
	ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/").
		To(h.Login).
		Doc("login, get token").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	container.Add(ws)
}
