package userhandler

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
)

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/users")
	ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(handlers.ListCommonQuery(ws.GET("/").
		To(h.List).
		Doc("list users").
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Returns(http.StatusOK, handlers.MessageOK, handlers.PageData{
			List: []forms.UserDetail{},
		})))

	ws.Route(ws.POST("/").
		To(h.Create).
		Doc("create user").
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Reads(forms.UserDetail{}).
		Returns(http.StatusOK, handlers.MessageOK, forms.UserDetail{}))

	ws.Route(ws.DELETE("/{name}").
		To(h.Delete).
		Doc("delete user").
		Param(restful.PathParameter("name", "user name")).
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{name}").
		To(h.Retrieve).
		Doc("retrieve user").
		Param(restful.PathParameter("name", "user name")).
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Returns(http.StatusOK, handlers.MessageOK, forms.UserDetail{}))

	container.Add(ws)
}
