package tenanthandler

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
)

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/" + h.Path)

	ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(handlers.ListCommonQuery(ws.GET("/").
		To(h.List).
		Doc("list tenants").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, handlers.MessageOK, handlers.PageData{
			List: []forms.TenantCommon{},
		})))

	ws.Route(ws.POST("/").
		To(h.Create).
		Doc("create tenant").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(forms.TenantDetail{}).
		Returns(http.StatusOK, handlers.MessageOK, forms.TenantCommon{}))

	ws.Route(ws.DELETE("/{tenant}").
		To(h.Delete).
		Doc("delete tenant via id").
		Param(restful.PathParameter("tenant", "tenant identify key, name or id")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.POST("/{tenant}/users").
		To(h.AddTenantMember).
		Doc("add user to tenant").
		Param(restful.PathParameter("tenant", "tenant identify key, name or id")).
		Reads(forms.TenantUserRelCommon{}).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, handlers.MessageOK, forms.TenantUserRelCommon{}))

	ws.Route(ws.GET("/{tenant}/users").
		To(h.ListTenantMember).
		Doc("list tenant members").
		Param(restful.PathParameter("tenant", "tenant identify key, name or id")).
		Param(restful.QueryParameter("isActive", "isActive")).
		Param(restful.QueryParameter("role", "role")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, handlers.MessageOK, forms.UserCommon{}))

	container.Add(ws)
}
