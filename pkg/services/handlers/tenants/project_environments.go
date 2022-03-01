package tenanthandler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *Handler) CreateProjectEnvironment(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	env := &EnvironmentCreateForm{}
	if err := handlers.BindData(req, env); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	tenantName := req.PathParameter("tenant")
	projectName := req.PathParameter("project")
	cluster, err := h.getCluster(ctx, env.Cluster)
	if err != nil {
		handlers.NotFound(resp, fmt.Errorf("cluster not exists"))
		return
	}
	project, err := h.getProject(ctx, tenantName, projectName)
	if err != nil {
		handlers.NotFound(resp, fmt.Errorf("project not exists"))
		return
	}
	_, err = h.getTenantResourceQuota(ctx, tenantName, env.Cluster)
	if err != nil {
		handlers.BadRequest(resp, fmt.Errorf("tenant has not quota on cluster %s", env.Cluster))
		return
	}
	obj := models.Environment{
		Name:          env.Name,
		Namespace:     env.Namespace,
		MetaType:      env.MetaType,
		DeletePolicy:  env.DeletePolicy,
		Cluster:       cluster,
		Project:       project,
		ResourceQuota: []byte(env.ResourceQuota),
		LimitRange:    []byte(env.LimitRange),
	}
	// TODO: before create check
	if err := h.DB().WithContext(ctx).Create(obj).Error; err != nil {
		handlers.BadRequest(resp, err)
	}
	// TODO: after create sync
	handlers.Created(resp, env)
}

func (h *Handler) ListProjectEnvironment(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	envs := &[]models.EnvironmentCommon{}
	if err := h.DB().WithContext(ctx).
		Joins("LEFT JOIN projects on projects.id = environments.project_id").
		Where("projects.name = ?", req.PathParameter("project")).
		Preload("Creator").
		Preload("Cluster").
		Find(envs).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, envs)
}

func (h *Handler) RetrieveProjectEnvironment(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	env, err := h.getProjectEnvironment(ctx, req.PathParameter("project"), req.PathParameter("environment"))
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, env)
}

func (h *Handler) DeleteProjectEnvironment(req *restful.Request, resp *restful.Response) {
	// TODO: BEFORE DELETE ,NEED TO DEL RES
	ctx := req.Request.Context()
	env, err := h.getProjectEnvironment(ctx, req.PathParameter("project"), req.PathParameter("environment"))
	if err != nil {
		if handlers.IsNotFound(err) {
			handlers.NoContent(resp, err)
		} else {
			handlers.BadRequest(resp, err)
		}
		return
	}
	if err := h.DB().WithContext(ctx).Delete(env).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) ModifyProjectEnvironment(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	env := &EnvironmentCreateForm{}
	if err := handlers.BindData(req, env); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	existEnv, err := h.getProjectEnvironment(ctx, req.PathParameter("project"), req.PathParameter("environment"))
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
	}
	if env.Name != req.PathParameter("environment") {
		handlers.BadRequest(resp, fmt.Errorf("name can't modify"))
		return
	}
	existEnv.MetaType = env.MetaType
	existEnv.DeletePolicy = env.DeletePolicy
	existEnv.LimitRange = []byte(env.LimitRange)
	existEnv.ResourceQuota = []byte(env.ResourceQuota)
	existEnv.Remark = env.Remark
	if err := h.DB().WithContext(ctx).Updates(existEnv).Error; err != nil {
		handlers.BadRequest(resp, err)
	}
	handlers.OK(resp, existEnv)
}

func (h *Handler) ListEnvironmentMembers(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ul := []models.UserSimple{}
	env, err := h.getProjectEnvironment(ctx, req.PathParameter("project"), req.PathParameter("environment"))
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	if err := h.DB().WithContext(ctx).
		Joins("LEFT JOIN environment_user_rels on environment_user_rels.user_id = users.id").
		Where("environment_user_rels.environment_id = ?", env.ID).
		Find(ul).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	handlers.OK(resp, ul)
}

type EnvironmentUserRole struct {
	Role        string `json:"role,omitempty" validate:"required"`
	User        string `json:"user,omitempty" validate:"required"`
	Environment string `json:"environment,omitempty" vlidate:"required"`
}

func (h *Handler) AddOrModifyEnvironmentMembers(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	project := req.PathParameter("project")
	environment := req.PathParameter("environment")
	userName := req.PathParameter("user")
	role := req.QueryParameter("role")
	env, err := h.getProjectEnvironment(ctx, project, environment)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	user, err := h.getUser(ctx, userName)
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	envUserRel := &models.EnvironmentUserRels{
		EnvironmentID: env.ID,
		UserID:        user.ID,
	}
	envUserRel.Role = role
	if err := h.DB().Save(envUserRel).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	} else {
		handlers.OK(resp, envUserRel)
		return
	}
}

func (h *Handler) DeleteEnvironmentMember(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	environment := req.PathParameter("environment")
	username := req.PathParameter("user")
	envUserRel := &models.EnvironmentUserRels{}
	// todo：验证gorm join 删除的时候是否带了join的表(这儿要求不带)
	err := h.DB().WithContext(ctx).
		Joins("LEFT JOIN environments on environments.id = environemnt_user_rels.environment_id").
		Joins("LEFT JOIN users on users.id = environment_user_rels.user_id").
		Where("users.username = ? and environments.name = ?", username, environment).
		Delete(envUserRel).Error
	if err != nil {
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

	res := &models.EnvironmentResource{}
	if err := h.DB().WithContext(ctx).
		Where("tenant = ?", req.PathParameter("tenant")).
		Where("project = ?", req.PathParameter("project")).
		Where("environment = ?", req.PathParameter("envorinment")).
		Where("created_at >= ?", dayTime.Format(time.RFC3339)).
		Where("created_at < ?", dayTime.Add(time.Hour*24).Format(time.RFC3339)).
		Order("created_at").
		First(&res).Error; err != nil {
		log.Error(err, "get environment resource")
	}
	handlers.OK(resp, res)
}

func (h *Handler) SwitchEnvironmentNetworkIsolate(req *restful.Request, resp *restful.Response) {
	var isolate bool
	if req.Request.Method == http.MethodPost {
		isolate = true
	}
	ctx := req.Request.Context()
	tenant := req.PathParameter("tenant")
	project := req.PathParameter("project")
	environment := req.PathParameter("environment")

	env, err := h.getProjectEnvironment(ctx, project, environment)
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	clusterI := &models.Cluster{
		ID: env.ClusterID,
	}
	if err := h.DB().WithContext(ctx).First(clusterI).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	tnetpol := &v1beta1.TenantNetworkPolicy{}
	err = h.Execute(ctx, clusterI.Name, func(ctx context.Context, cli agents.Client) error {
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
