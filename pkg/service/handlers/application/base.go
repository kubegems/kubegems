package application

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
	"github.com/opentracing/opentracing-go"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/handlers/base"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/redis"
)

type Cache struct {
	kvs sync.Map
}

func (c *Cache) Set(key string, value interface{}) {
	c.kvs.Store(key, value)
}

func (c *Cache) Get(key string) (interface{}, error) {
	v, ok := c.kvs.Load(key)
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return v, nil
}

type BaseHandler struct {
	base.BaseHandler
	Database *database.Database
	Redis    *redis.Client
	dbcahce  *Cache
}

func (h *BaseHandler) DirectNamedRefFunc(c *gin.Context, body interface{}, fun func(ctx context.Context, ref PathRef) (interface{}, error)) {
	completes := []RefCompleteFunc{
		h.DirectRefNameFunc,
	}
	h.processfunc(c, body, completes, fun)
}

func (h *BaseHandler) NamedRefFunc(c *gin.Context, body interface{}, fun func(ctx context.Context, ref PathRef) (interface{}, error)) {
	completes := []RefCompleteFunc{
		h.BindTenantProjectedRefFunc,
		h.BindNameRefFunc,
		h.MayBindEnvRefFunc,
	}
	h.processfunc(c, body, completes, fun)
}

func (h *BaseHandler) NoNameRefFunc(c *gin.Context, body interface{}, fun func(ctx context.Context, ref PathRef) (interface{}, error)) {
	completes := []RefCompleteFunc{
		h.BindTenantProjectedRefFunc,
		h.MayBindEnvRefFunc,
	}
	h.processfunc(c, body, completes, fun)
}

func (h *BaseHandler) BindTenantProjectedRefFunc(c *gin.Context, ref *PathRef) error {
	params := &struct {
		TenantID  uint `uri:"tenant_id" binding:"required"`
		ProjectID uint `uri:"project_id" binding:"required"`
	}{}
	if err := c.ShouldBindUri(params); err != nil {
		return err
	}

	project := &models.Project{ID: params.ProjectID, TenantID: params.TenantID}

	// try cache
	key := fmt.Sprintf("project|%d", params.ProjectID)
	if obj, err := h.dbcahce.Get(key); err != nil {
		// try db
		if err := h.Database.DB().Where(project).Preload("Tenant").Take(project).Error; err != nil {
			return err
		}
		// save cache
		h.dbcahce.Set(key, project)
	} else {
		project = obj.(*models.Project)
	}

	ref.Tenant = project.Tenant.TenantName
	ref.Project = project.ProjectName

	// 审计
	h.SetExtraAuditData(c, models.ResTenant, params.TenantID)
	h.SetExtraAuditData(c, models.ResProject, params.ProjectID)

	return nil
}

func (h *BaseHandler) BindNameRefFunc(c *gin.Context, ref *PathRef) error {
	params := &struct {
		Name string `uri:"name" binding:"required"`
	}{}
	if err := c.ShouldBindUri(params); err != nil {
		return err
	}
	ref.Name = params.Name
	return nil
}

func (h *BaseHandler) DirectRefNameFunc(c *gin.Context, ref *PathRef) error {
	params := &struct {
		Tenant      string `uri:"tenant" binding:"required"`
		Project     string `uri:"project" binding:"required"`
		Environment string `uri:"environment_id"`
		Name        string `uri:"name" binding:"required"`
	}{}

	if err := c.ShouldBindUri(params); err != nil {
		return err
	}
	ref.Tenant = params.Tenant
	ref.Project = params.Project
	ref.Env = params.Environment
	ref.Name = params.Name

	var (
		ten  models.Tenant
		proj models.Project
		env  models.Environment
	)
	if err := h.Database.DB().First(&ten, models.Tenant{TenantName: ref.Tenant}).Error; err != nil {
		return fmt.Errorf("tenant with name %s not exist", ref.Tenant)
	}
	if err := h.Database.DB().First(&proj, models.Project{ProjectName: ref.Project, TenantID: ten.ID}).Error; err != nil {
		return fmt.Errorf("no project named %s belong to tenant %s", ref.Project, ref.Tenant)
	}
	if err := h.Database.DB().First(&env, models.Environment{EnvironmentName: ref.Env, ProjectID: proj.ID}).Error; err != nil {
		return fmt.Errorf("no environment named %s belong to project %s", ref.Env, ref.Project)
	}

	// check permission
	c.Params = append(c.Params, gin.Param{Key: "environment_id", Value: strconv.Itoa(int(env.ID))})
	h.CheckByEnvironmentID(c)

	// 审计
	h.SetExtraAuditData(c, models.ResEnvironment, env.ID)

	return nil
}

const ginContextKeyClusterNamespace = "CLUSTER-NAMESPACE"

func (h *BaseHandler) MayBindEnvRefFunc(c *gin.Context, ref *PathRef) error {
	params := &struct {
		EnvironmentID *uint `uri:"environment_id"`
	}{}
	if err := c.ShouldBindUri(params); err != nil {
		return err
	}
	// 设置环境
	if params.EnvironmentID != nil {
		env := &models.Environment{ID: *params.EnvironmentID}
		// try cache
		key := fmt.Sprintf("environment|%d", *params.EnvironmentID)
		if obj, err := h.dbcahce.Get(key); err != nil {
			// try db
			if err := h.Database.DB().Preload("Cluster").Take(env).Error; err != nil {
				return err
			}
			// save cache
			h.dbcahce.Set(key, env)
		} else {
			env = obj.(*models.Environment)
		}

		ref.Env = env.EnvironmentName

		c.Set(ginContextKeyClusterNamespace, ClusterNamespace{Cluster: env.Cluster.ClusterName, Namespace: env.Namespace})

		// 审计
		h.SetExtraAuditData(c, models.ResEnvironment, *params.EnvironmentID)
	}

	return nil
}

type RefCompleteFunc func(*gin.Context, *PathRef) error

type ClusterNamespace struct {
	Cluster   string
	Namespace string
}

func (h *BaseHandler) processfunc(c *gin.Context, body interface{}, completes []RefCompleteFunc, processfunc func(ctx context.Context, ref PathRef) (interface{}, error)) {
	ctx := c.Request.Context()

	span, ctx := opentracing.StartSpanFromContext(ctx, "start process")
	defer span.Finish()

	process := func(ctx context.Context) (interface{}, error) {
		if body != nil {
			if err := c.ShouldBind(body); err != nil {
				return nil, err
			}
		}
		ref := &PathRef{}

		for _, fun := range completes {
			if err := fun(c, ref); err != nil {
				return nil, err
			}
			// complete 中会进行响应
			if c.Writer.Written() {
				return nil, nil
			}
		}

		// 注入 logger
		logger := log.FromContextOrDiscard(ctx)
		logger = logger.WithValues("ref", ref)
		ctx = logr.NewContext(ctx, logger)

		// 注入 user
		if user, ok := h.GetContextUser(c); ok {
			ctx = context.WithValue(ctx, contextAuthorKey{}, &object.Signature{Name: user.Username, Email: user.Email})
		} else {
			ctx = context.WithValue(ctx, contextAuthorKey{}, &object.Signature{Name: "unknow", Email: "unknown"})
		}

		// 注入 cluster namespace
		if val, ok := c.Get(ginContextKeyClusterNamespace); ok {
			ctx = context.WithValue(ctx, contextClusterNamespaceKey{}, val.(ClusterNamespace))
		}

		// 这里是实际处理流程
		return processfunc(ctx, *ref)
	}

	data, err := process(ctx)
	if err != nil {
		handlers.NotOK(c, err)
	}
	// 如果未曾writer则响应 data，有的处理流程中会使用 sse 则不需要再次响应
	if data != nil && !c.Writer.Written() {
		handlers.OK(c, data)
	}
}
