package tenanthandler

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/v2/models"
	"kubegems.io/pkg/v2/services/handlers"
)

var (
	tenantTags        = []string{"tenants"}
	tenantUserTags    = []string{"tenants", "users"}
	tenantProjectTags = []string{"tenants", "projects"}
)

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/v2/tenants")

	ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(handlers.ListCommonQuery(ws.GET("/").
		To(h.ListTenant).
		Doc("list tenants").
		Metadata(restfulspec.KeyOpenAPITags, tenantTags).
		Returns(http.StatusOK, handlers.MessageOK, TenantListResp{})))

	ws.Route(ws.POST("/").
		To(h.CreateTenant).
		Doc("create tenant").
		Metadata(restfulspec.KeyOpenAPITags, tenantTags).
		Reads(models.TenantSimple{}).
		Returns(http.StatusOK, handlers.MessageOK, TenantCreateResp{}))

	ws.Route(ws.GET("/{tenant}").
		To(h.RetrieveTenant).
		Doc("retrieve tenant").
		Notes("retrieve tenant").
		Param(restful.PathParameter("tenant", "tenant")).
		Metadata(restfulspec.KeyOpenAPITags, tenantTags).
		Returns(http.StatusOK, handlers.MessageOK, TenantCommonResp{}))

	ws.Route(ws.DELETE("/{tenant}").
		To(h.DeleteTenant).
		Doc("delete tenant").
		Param(restful.PathParameter("tenant", "tenant name")).
		Metadata(restfulspec.KeyOpenAPITags, tenantTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.PUT("/{tenant}").
		To(h.ModifyTenant).
		Doc("modify tenant").
		Param(restful.PathParameter("tenant", "tenant")).
		Metadata(restfulspec.KeyOpenAPITags, tenantTags).
		Reads(models.TenantCommon{}).
		Returns(http.StatusOK, handlers.MessageOK, TenantCommonResp{}))

	h.registUsers(ws)
	h.registProjects(ws)
	h.registProjectEnvironments(ws)
	container.Add(ws)
}

func (h *Handler) registUsers(ws *restful.WebService) {
	ws.Route(ws.POST("/{tenant}/users").
		To(h.AddTenantMember).
		Doc("add user to tenant members").
		Notes(`add user to tenant members`).
		Param(restful.PathParameter("tenant", "tenant name")).
		Reads(TenantUserCreateForm{}).
		Metadata(restfulspec.KeyOpenAPITags, tenantUserTags).
		Returns(http.StatusOK, handlers.MessageOK, TenantUserCreateResp{}))

	ws.Route(ws.PUT("/{tenant}/users").
		To(h.ModifyTenantMember).
		Doc("modify user role tenant members").
		Param(restful.PathParameter("tenant", "tenant name")).
		Reads(TenantUserCreateForm{}).
		Metadata(restfulspec.KeyOpenAPITags, tenantUserTags).
		Returns(http.StatusOK, handlers.MessageOK, TenantUserCreateResp{}))

	ws.Route(ws.GET("/{tenant}/users").
		To(h.ListTenantMember).
		Doc("list tenant members").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.QueryParameter("isActive", "isActive")).
		Param(restful.QueryParameter("role", "role")).
		Metadata(restfulspec.KeyOpenAPITags, tenantUserTags).
		Returns(http.StatusOK, handlers.MessageOK, UserSimpleListResp{}))

	ws.Route(ws.DELETE("/{tenant}/users/{user}").
		To(h.DeleteTenantMember).
		Doc("delete user from tenant members").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("user", "user name")).
		Metadata(restfulspec.KeyOpenAPITags, tenantUserTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))
}

func (h *Handler) registProjects(ws *restful.WebService) {
	ws.Route(ws.GET("/{tenant}/projects").
		To(h.ListTenantProject).
		Doc("list tenant's projects").
		Param(restful.PathParameter("tenant", "tenant name")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, ProjectListResp{}))

	ws.Route(ws.POST("/{tenant}/projects").
		To(h.CreatePorject).
		Doc("create a project").
		Param(restful.PathParameter("tenant", "tenant name")).
		Reads(ProjectCreateForm{}).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, ProjectResp{}))

	ws.Route(ws.DELETE("/{tenant}/projects/{project}").
		To(h.DeleteProject).
		Doc("delete a project").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{tenant}/projects/{project}").
		To(h.RetrieveTenantProject).
		Doc("retrieve project ").
		Param(restful.QueryParameter("detail", "show detail")).
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, ProjectResp{}))
}

func (h *Handler) registProjectEnvironments(ws *restful.WebService) {
	ws.Route(ws.POST("/{tenant}/projects/{project}/environments").
		To(h.CreateProjectEnvironment).
		Doc("create a environment in tenant/project").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Reads(EnvironmentCreateForm{}).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusCreated, handlers.MessageOK, EnvironmentCreateForm{}))

	ws.Route(ws.DELETE("/{tenant}/projects/{project}/environments/{environment}").
		To(h.DeleteProjectEnvironment).
		Doc("delete a environment belong to the tenant/project ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{tenant}/projects/{project}/environments").
		To(h.ListProjectEnvironment).
		Doc("list environment belong to the tenant/project ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, EnvironmentListResp{}))

	ws.Route(ws.GET("/{tenant}/projects/{project}/environments/{environment}").
		To(h.RetrieveProjectEnvironment).
		Doc("get environment belong to the tenant/project ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name")).
		Param(restful.QueryParameter("detail", "show detail").PossibleValues([]string{"true", "false"})).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, EnvironmentResp{}))

	ws.Route(ws.PUT("/{tenant}/projects/{project}/environments/{environment}").
		To(h.ModifyProjectEnvironment).
		Doc("list environment belong to the tenant/project ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name")).
		Reads(EnvironmentCreateForm{}).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, EnvironmentResp{}))

	ws.Route(ws.GET("/{tenant}/projects/{project}/environments/{environment}/users").
		To(h.ListEnvironmentMembers).
		Doc("list environment member").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Param(restful.QueryParameter("role", "filter role")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, UserSimpleListResp{}))

	ws.Route(ws.POST("/{tenant}/projects/{project}/environments/{environment}/users/{user}").
		To(h.AddOrModifyEnvironmentMembers).
		Doc("add or modify environment member ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Param(restful.PathParameter("user", "user to add")).
		Param(restful.QueryParameter("role", "filter role").Required(true)).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, EnvironmentUserRelsResp{}))

	ws.Route(ws.DELETE("/{tenant}/projects/{project}/environments/{environment}/users/{user}").
		To(h.DeleteEnvironmentMember).
		Doc("delete environment member ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Param(restful.PathParameter("user", "user to add")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{tenant}/projects/{project}/environments/{environment}/resource-aggregate").
		To(h.GetEnvironmentResourceAggregate).
		Doc("get environment resource history stastics").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Param(restful.QueryParameter("date", "date to speficy")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, EnvironmentResourceResp{}))

	ws.Route(ws.POST("/{tenant}/projects/{project}/environments/{environment}/network-isolate").
		To(h.SwitchEnvironmentNetworkIsolate).
		Doc("enable environment network isolate").
		Operation("EnableEnvironmentNetworkIsolate").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))

	ws.Route(ws.DELETE("/{tenant}/projects/{project}/environments/{environment}/network-isolate").
		To(h.SwitchEnvironmentNetworkIsolate).
		Doc("disable environment network isolate").
		Operation("DisableEnvironmentNetworkIsolate").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))
}
