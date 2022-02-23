package tenanthandler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *Handler) CreateProjectEnvironment(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	env := &forms.EnvironmentDetail{}
	if err := handlers.BindData(req, env); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	// TODO: before create check
	if err := h.Model().Create(ctx, env.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	// TODO: after create sync
	handlers.OK(resp, env)
}

func (h *Handler) ListProjectEnvironment(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	proj := forms.ProjectCommon{
		Name: req.PathParameter("project"),
	}
	ol := forms.EnvironmentCommonList{}
	if err := h.Model().List(ctx, ol.Object(), client.BelongTo(proj.Object())); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(ol.Object(), ol.DataPtr()))
}

func (h *Handler) RetrieveProjectEnvironment(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	env, err := h.getProjectEnvironment(ctx, req.PathParameter("project"), req.PathParameter("environment"), req.QueryParameter("detail") == "true")
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, env.DataPtr())
}

func (h *Handler) DeleteProjectEnvironment(req *restful.Request, resp *restful.Response) {
	// TODO: BEFORE DELETE ,NEED TO DEL RES
	ctx := req.Request.Context()
	env, err := h.getProjectEnvironmentCommon(ctx, req.PathParameter("project"), req.PathParameter("environment"))
	if err != nil {
		if handlers.IsNotFound(err) {
			handlers.NoContent(resp, err)
		} else {
			handlers.BadRequest(resp, err)
		}
		return
	}
	if err := h.Model().Delete(ctx, env.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) ModifyProjectEnvironment(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	env := &forms.EnvironmentDetail{}
	if err := handlers.BindData(req, env); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	old := forms.EnvironmentCommon{}
	if err := h.Model().Get(ctx, old.Object(), client.WhereNameEqual(req.PathParameter("environment"))); err != nil {
		if handlers.IsNotFound(err) {
			handlers.NotFound(resp, err)
		} else {
			handlers.BadRequest(resp, err)
		}
		return
	}
	if env.Name != req.PathParameter("environment") {
		handlers.BadRequest(resp, fmt.Errorf("name can't modify"))
		return
	}
	env.ID = 0
	// TODO: BEFORE UPDATE, CHECK
	if err := h.Model().Update(ctx, env.Object(), client.WhereNameEqual(req.PathParameter("environment"))); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	// TODO: AFTER UPDATE, SYNC
	handlers.OK(resp, env.Data())
}

func (h *Handler) ListEnvironmentMembers(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ul := forms.UserCommonList{}
	env, err := h.getProjectEnvironmentCommon(ctx, req.PathParameter("project"), req.PathParameter("environment"))
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	var opt client.Option
	if req.QueryParameter("role") != "" {
		opt = client.ExistRelationWithKeyValue(env.Object(), "role", req.QueryParameter("role") != "")
	} else {
		opt = client.ExistRelation(env.Object())
	}
	if err := h.Model().List(ctx, ul.Object(), opt); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(ul.Object(), ul.Data()))
}

type EnvironmentUserRole struct {
	Role        string `json:"role,omitempty" validate:"required"`
	User        string `json:"user,omitempty" validate:"required"`
	Environment string `json:"environment,omitempty" vlidate:"required"`
}

func (h *Handler) AddEnvironmentMembers(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	project := req.PathParameter("project")
	environment := req.PathParameter("environment")
	userName := req.PathParameter("user")
	role := req.PathParameter("role")
	env, err := h.getProjectEnvironmentCommon(ctx, project, environment)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	user, err := h.getUserCommon(ctx, userName)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	envUserRel := &forms.EnvironmentUserRelCommon{
		EnvironmentID: env.ID,
		UserID:        user.ID,
	}
	exist := false
	if err := h.Model().Get(ctx, envUserRel.Object(), client.BelongTo(env.Object()), client.BelongTo(user.Object())); err != nil {
		if !handlers.IsNotFound(err) {
			handlers.BadRequest(resp, err)
			return
		}
	} else {
		exist = true
	}

	newOne := &forms.EnvironmentUserRelCommon{
		EnvironmentID: env.ID,
		UserID:        user.ID,
		Role:          role,
	}
	if exist {
		if err := h.Model().Update(ctx, newOne.Object(), client.BelongTo(env.Object()), client.BelongTo(user.Object())); err != nil {
			handlers.BadRequest(resp, err)
			return
		} else {
			handlers.OK(resp, newOne.Data())
			return
		}
	} else {
		if err := h.Model().Create(ctx, newOne.Object()); err != nil {
			handlers.BadRequest(resp, err)
			return
		} else {
			handlers.OK(resp, newOne.Data())
			return
		}
	}
}

func (h *Handler) DeleteEnvironmentMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	project := req.PathParameter("project")
	environment := req.PathParameter("environment")
	userName := req.PathParameter("user")
	env, err := h.getProjectEnvironmentCommon(ctx, project, environment)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	user, err := h.getUserCommon(ctx, userName)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	envUserRel := &forms.EnvironmentUserRelCommon{
		EnvironmentID: env.ID,
		UserID:        user.ID,
	}
	if err := h.Model().Get(ctx, envUserRel.Object(), client.BelongTo(env.Object()), client.BelongTo(user.Object())); err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	if err := h.Model().Delete(ctx, envUserRel.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.NoContent(resp, err)
}

func (h *Handler) GetEnvironmentResourceAggregate(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	dateTime, err := time.Parse(time.RFC3339, req.QueryParameter("date"))
	if err != nil {
		dateTime = time.Now().Add(-24 * time.Hour)
	}
	dayTime := utils.NextDayStartTime(dateTime)

	res := forms.EnvironmentResourceCommon{}
	if err := h.Model().Get(
		ctx, res.Object(),
		client.WhereEqual("tenant", req.PathParameter("tenant")),
		client.WhereEqual("project", req.PathParameter("project")),
		client.WhereEqual("environment", req.PathParameter("environment")),
		client.Where("create_at", client.Gte, dayTime.Format(time.RFC3339)),
		client.Where("create_at", client.Lt, dayTime.Add(time.Hour*24).Format(time.RFC3339)),
	); err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
	}
	handlers.OK(resp, res)
}

func (h *Handler) SwitchEnvironmentNetworkIsolate(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	tenant := req.PathParameter("tenant")
	project := req.PathParameter("project")
	environment := req.PathParameter("environment")
	isolate := req.QueryParameter("isolate") == "true"

	proj := forms.ProjectCommon{Name: project}
	env := forms.EnvironmentDetail{}
	if err := h.Model().Get(ctx, env.Object(), client.WhereNameEqual(environment), client.BelongTo(proj.Object()), client.Preloads([]string{"Cluster"})); err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	cluster := env.Data().Cluster.Name

	tnetpol := &v1beta1.TenantNetworkPolicy{}
	err := h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		if err := cli.Get(ctx, kclient.ObjectKey{Name: tenant}, tnetpol); err != nil {
			return err
		}

		index := -1
		for idx, envpol := range tnetpol.Spec.EnvironmentNetworkPolicies {
			if envpol.Name == environment {
				index = idx
			}
		}
		if index == -1 && isolate {
			tnetpol.Spec.EnvironmentNetworkPolicies = append(tnetpol.Spec.EnvironmentNetworkPolicies, v1beta1.EnvironmentNetworkPolicy{
				Name:    environment,
				Project: project,
			})
		}
		if index != -1 && !isolate {
			tnetpol.Spec.EnvironmentNetworkPolicies = append(tnetpol.Spec.EnvironmentNetworkPolicies[:index], tnetpol.Spec.EnvironmentNetworkPolicies[index+1:]...)
		}

		return cli.Update(ctx, tnetpol)
	})
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, tnetpol)
}

func (h *Handler) getProjectEnvironmentCommon(ctx context.Context, project, environment string) (*forms.EnvironmentCommon, error) {
	proj := forms.ProjectCommon{Name: project}
	env := forms.EnvironmentCommon{}
	return env.Data(), h.Model().Get(ctx, env.Object(), client.WhereNameEqual(environment), client.BelongTo(proj.Object()))
}

func (h *Handler) getProjectEnvironmentDetail(ctx context.Context, project, environment string) (*forms.EnvironmentDetail, error) {
	proj := forms.ProjectCommon{Name: project}
	env := forms.EnvironmentDetail{}
	return env.Data(), h.Model().Get(ctx, env.Object(), client.WhereNameEqual(environment), client.BelongTo(proj.Object()))
}

func (h *Handler) getProjectEnvironment(ctx context.Context, project, environment string, detail bool) (forms.FormInterface, error) {
	if detail {
		return h.getProjectEnvironmentDetail(ctx, project, environment)
	} else {
		return h.getProjectEnvironmentCommon(ctx, project, environment)
	}
}

func (h *Handler) registProjectEnvironments(ws *restful.WebService) {
	ws.Route(ws.POST("/{tenant}/projects/{project}/environments").
		To(h.CreateProjectEnvironment).
		Doc("create a environment in tenant/project").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Reads(forms.EnvironmentCommon{}).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusCreated, handlers.MessageOK, forms.EnvironmentCommon{}))

	ws.Route(ws.DELETE("/{tenant}/projects/{project}/environments/{environment}").
		To(h.DeleteProjectEnvironment).
		Doc("delete a environment belong to the tenant/project ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{tenant}/projects/{project}/environments").
		To(h.ListProjectEnvironment).
		Doc("list environment belong to the tenant/project ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, []forms.EnvironmentCommon{}))

	ws.Route(ws.GET("/{tenant}/projects/{project}/environments/{environment}").
		To(h.RetrieveProjectEnvironment).
		Doc("get environment belong to the tenant/project ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.QueryParameter("detail", "show detail").PossibleValues([]string{"true", "false"})).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, forms.EnvironmentDetail{}))

	ws.Route(ws.PUT("/{tenant}/projects/{project}/environments/{environment}").
		To(h.ListProjectEnvironment).
		Doc("list environment belong to the tenant/project ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Reads(forms.EnvironmentDetail{}).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, forms.EnvironmentDetail{}))

	ws.Route(ws.GET("/{tenant}/projects/{project}/environments/{environment}/users").
		To(h.ListEnvironmentMembers).
		Doc("list environment member ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Param(restful.QueryParameter("role", "filter role")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, []forms.UserCommon{}))

	ws.Route(ws.POST("/{tenant}/projects/{project}/environments/{environment}/users/{user}").
		To(h.AddEnvironmentMembers).
		Doc("add or modify environment member ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Param(restful.PathParameter("user", "user to add")).
		Param(restful.QueryParameter("role", "filter role").Required(true)).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusOK, handlers.MessageOK, []forms.EnvironmentUserRelCommon{}))

	ws.Route(ws.DELETE("/{tenant}/projects/{project}/environments/{environment}/users/{user}").
		To(h.DeleteEnvironmentMember).
		Doc("delete environment member ").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Param(restful.PathParameter("user", "user to add")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))

	ws.Route(ws.DELETE("/{tenant}/projects/{project}/environments/{environment}/resource-aggregate").
		To(h.GetEnvironmentResourceAggregate).
		Doc("get environment resource history stastics").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Param(restful.QueryParameter("date", "date to speficy")).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))

	ws.Route(ws.POST("/{tenant}/projects/{project}/environments/{environment}/network-isolate").
		To(h.SwitchEnvironmentNetworkIsolate).
		Doc("switch environment network isolate").
		Param(restful.PathParameter("tenant", "tenant name")).
		Param(restful.PathParameter("project", "project name belong to the tenant")).
		Param(restful.PathParameter("environment", "environment name belong to the tenant/project")).
		Param(restful.QueryParameter("isolate", "is isolate").PossibleValues([]string{"true", "false"}).Required(true)).
		Metadata(restfulspec.KeyOpenAPITags, tenantProjectTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))
}
