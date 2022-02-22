package tenanthandler

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
)

var (
	tenantTags        = []string{"tenants"}
	tenantUserTags    = []string{"tenants", "users"}
	tenantProjectTags = []string{"tenants", "projects"}
)

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/tenants")

	ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(handlers.ListCommonQuery(ws.GET("/").
		To(h.List).
		Doc("list tenants").
		Metadata(restfulspec.KeyOpenAPITags, tenantTags).
		Returns(http.StatusOK, handlers.MessageOK, handlers.PageData{
			List: &[]forms.TenantCommon{},
		})))

	ws.Route(ws.POST("/").
		To(h.Create).
		Doc("create tenant").
		Metadata(restfulspec.KeyOpenAPITags, tenantTags).
		Reads(forms.TenantDetail{}).
		Returns(http.StatusOK, handlers.MessageOK, forms.TenantCommon{}))

	ws.Route(handlers.ListCommonQuery(ws.GET("/{tenant}").
		To(h.Retrieve).
		Doc("retrieve tenant").
		Notes("retrieve tenant").
		Param(restful.QueryParameter("detail", "is show detail info")).
		Param(restful.PathParameter("tenant", "tenant")).
		Metadata(restfulspec.KeyOpenAPITags, tenantTags).
		Returns(http.StatusOK, handlers.MessageOK, forms.TenantDetail{})))

	ws.Route(ws.DELETE("/{tenant}").
		To(h.Delete).
		Doc("delete tenant via id").
		Param(restful.PathParameter("tenant", "tenant name")).
		Metadata(restfulspec.KeyOpenAPITags, tenantTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	h.registUsers(ws)
	h.registProjects(ws)
	container.Add(ws)
}

func (h *Handler) registUsers(ws *restful.WebService) {
	ws.Route(ws.POST("/{tenant}/users").
		To(h.AddTenantMember).
		Doc("add user to tenant members").
		Notes(`add user to tenant members`).
		Param(restful.PathParameter("tenant", "tenant name")).
		Reads(forms.TenantUserCreateModifyForm{}).
		Metadata(restfulspec.KeyOpenAPITags, tenantUserTags).
		Returns(http.StatusOK, handlers.MessageOK, forms.TenantUserRelCommon{}))

	ws.Route(ws.PUT("/{tenant}/users").
		To(h.ModifyTenantMember).
		Doc("modify user role tenant members").
		Param(restful.PathParameter("tenant", "tenant name")).
		Reads(forms.TenantUserCreateModifyForm{}).
		Metadata(restfulspec.KeyOpenAPITags, tenantUserTags).
		Returns(http.StatusOK, handlers.MessageOK, forms.TenantUserRelCommon{}))

	ws.Route(ws.GET("/{tenant}/users").
		To(h.ListTenantMember).
		Doc("list tenant members").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.QueryParameter("isActive", "isActive")).
		Param(restful.QueryParameter("role", "role")).
		Metadata(restfulspec.KeyOpenAPITags, tenantUserTags).
		Returns(http.StatusOK, handlers.MessageOK, &handlers.PageData{
			List: []forms.UserCommon{},
		}))

	ws.Route(ws.DELETE("/{tenant}/users/{user}").
		To(h.DeleteTenantMember).
		Doc("delete user from tenant members").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("user", "user name")).
		Metadata(restfulspec.KeyOpenAPITags, tenantUserTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))
}

func (h *Handler) registProjects(ws *restful.WebService) {
	ws.Route(ws.POST("/{tenant}/projects").
		To(h.CreatePorject).
		Doc("create a project").
		Param(restful.PathParameter("tenant", "tenant name")).
		Reads(forms.ProjectCreateForm{}).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, forms.ProjectCommon{}))

	ws.Route(ws.DELETE("/{tenant}/projects/{project}").
		To(h.DeleteProject).
		Doc("delete a project").
		Param(restful.PathParameter("tenant", "tenant name")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, forms.TenantUserRelCommon{}))

	ws.Route(ws.GET("/{tenant}/projects/{project}").
		To(h.RetrieveTenantProject).
		Doc("retrieve project ").
		Param(restful.QueryParameter("detail", "show detail")).
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, forms.ProjectCommon{}))
}
