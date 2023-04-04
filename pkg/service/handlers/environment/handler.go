// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package environment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	loggingv1beta1 "github.com/banzaicloud/logging-operator/pkg/sdk/logging/api/v1beta1"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	msgclient "kubegems.io/kubegems/pkg/msgbus/client"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	"kubegems.io/kubegems/pkg/service/handlers/registry/synchronizer"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/loki"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/resourcequota"
	"kubegems.io/kubegems/pkg/utils/slice"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	SearchFields           = []string{"environment_name"}
	FilterFields           = []string{"environment_name"}
	PreloadFields          = []string{"Creator", "Cluster", "Project", "Project.Tenant", "Applications", "Users"}
	PreloadSensitiveFields = map[string]string{"Cluster": "id, cluster_name"}
	OrderFields            = []string{"environment_name"}
	ModelName              = "Environment"
	PrimaryKeyName         = "environment_id"
	clusterSensitiveFunc   = func(tx *gorm.DB) *gorm.DB { return tx.Select("id, cluster_name") }
)

// ListEnvironment 列表 Environment
// @Tags        Environment
// @Summary     Environment列表
// @Description Environment列表
// @Accept      json
// @Produce     json
// @Param       EnvironmentName query    string                                                                     false "EnvironmentName"
// @Param       preload         query    string                                                                     false "choices Creator,Cluster,Project,Applications,Users"
// @Param       page            query    int                                                                        false "page"
// @Param       size            query    int                                                                        false "page"
// @Param       search          query    string                                                                     false "search in (EnvironmentName)"
// @Success     200             {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Environment}} "Environment"
// @Router      /v1/environment [get]
// @Security    JWT
func (h *EnvironmentHandler) ListEnvironment(c *gin.Context) {
	var list []models.Environment
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:                  ModelName,
		SearchFields:           SearchFields,
		PreloadSensitiveFields: PreloadSensitiveFields,
		PreloadFields:          PreloadFields,
	}
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveEnvironment Environment详情
// @Tags        Environment
// @Summary     Environment详情
// @Description get Environment详情
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                                             true "environment_id"
// @Success     200            {object} handlers.ResponseStruct{Data=models.Environment} "Environment"
// @Router      /v1/environment/{environment_id} [get]
// @Security    JWT
func (h *EnvironmentHandler) RetrieveEnvironment(c *gin.Context) {
	var (
		users []*models.User
		obj   models.Environment
	)
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Select(
		"users.*, environment_user_rels.role",
	).Joins(
		"join environment_user_rels  on  environment_user_rels.user_id = users.id",
	).Find(&users, "`environment_user_rels`.`environment_id` = ?", c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().WithContext(ctx).Preload("Cluster", clusterSensitiveFunc).First(&obj, "id = ?", c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	obj.Users = users
	handlers.OK(c, obj)
}

// PutEnvironment 修改Environment
// @Tags        Environment
// @Summary     修改Environment
// @Description 修改Environment
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                                             true "environment_id"
// @Param       param          body     models.Environment                               true "表单"
// @Success     200            {object} handlers.ResponseStruct{Data=models.Environment} "Environment"
// @Router      /v1/environment/{environment_id} [put]
// @Security    JWT
func (h *EnvironmentHandler) PutEnvironment(c *gin.Context) {
	var obj models.Environment
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Preload("Cluster").First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	action := i18n.Sprintf(context.TODO(), "update")
	module := i18n.Sprintf(context.TODO(), "environment")
	h.SetAuditData(c, action, module, obj.EnvironmentName)
	h.SetExtraAuditData(c, models.ResEnvironment, obj.ID)
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	obj.LimitRange = models.FillDefaultLimigrange(&obj)
	if strconv.Itoa(int(obj.ID)) != c.Param(PrimaryKeyName) {
		handlers.NotOK(c, i18n.Errorf(c, "URL parameter mismatched with body"))
		return
	}
	cluster := obj.Cluster
	if err := ValidateEnvironmentNamespace(ctx, h.BaseHandler, h.GetDB().WithContext(ctx), obj.Namespace, obj.EnvironmentName, obj.Cluster.ClusterName); err != nil {
		handlers.NotOK(c, err)
		return
	}
	err := h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(&obj).Error; err != nil {
			return err
		}
		return AfterEnvironmentSave(ctx, h.BaseHandler, tx, &obj)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.ModelCache().UpsertEnvironment(obj.ProjectID, obj.ID, obj.EnvironmentName, cluster.ClusterName, obj.Namespace)
	handlers.OK(c, obj)
}

// ValidateEnvironmentNamespace 校验绑定的namespace是否合法.
func ValidateEnvironmentNamespace(ctx context.Context, h base.BaseHandler, tx *gorm.DB, namespace, envname, clustername string) error {
	forbiddenBindNamespaces := []string{
		gems.NamespaceGateway,
		gems.NamespaceLogging,
		gems.NamespaceSystem,
		gems.NamespaceMonitor,
		"kube-system",
		"istio-system",
		"kube-node-lease",
		"local-path-storage",
	}
	if slice.ContainStr(forbiddenBindNamespaces, namespace) {
		return i18n.Errorf(ctx, "namespace  %s is not allowed, it's a system retain namespace", namespace)
	}
	agent, err := h.GetAgents().ClientOf(ctx, clustername)
	if err != nil {
		return err
	}
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if err := agent.Get(ctx, client.ObjectKeyFromObject(&ns), &ns); err != nil {
		if errors.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}
	if bindedEnv, exist := ns.Labels[gems.LabelEnvironment]; exist {
		if bindedEnv != envname {
			return i18n.Errorf(ctx, "namespace %s was bonded with another environment", namespace)
		}
	}
	return nil
}

/*
环境的创建，修改，删除，都会触发hook，将状态同步到对应的集群下
*/
func AfterEnvironmentSave(ctx context.Context, h base.BaseHandler, tx *gorm.DB, env *models.Environment) error {
	var (
		project       models.Project
		cluster       models.Cluster
		spec          v1beta1.EnvironmentSpec
		tmpLimitRange map[string]corev1.LimitRangeItem
		limitRange    []corev1.LimitRangeItem
		resourceQuota corev1.ResourceList
	)
	if e := tx.Preload("Tenant").Preload("Registries").First(&project, "id = ?", env.ProjectID).Error; e != nil {
		return e
	}
	if e := tx.First(&cluster, "id = ?", env.ClusterID).Error; e != nil {
		return e
	}

	if env.LimitRange != nil {
		e := json.Unmarshal(env.LimitRange, &tmpLimitRange)
		if e != nil {
			return e
		}
	}
	if env.ResourceQuota != nil {
		e := json.Unmarshal(env.ResourceQuota, &resourceQuota)
		if e != nil {
			return e
		}
		resourcequota.SetSameRequestWithLimit(resourceQuota)
	}

	for key, v := range tmpLimitRange {
		v.Type = corev1.LimitType(key)
		limitRange = append(limitRange, v)
	}
	spec.Namespace = env.Namespace
	spec.Project = project.ProjectName
	spec.Tenant = project.Tenant.TenantName
	spec.LimitRageName = "default"
	spec.ResourceQuotaName = "default"
	spec.DeletePolicy = env.DeletePolicy
	spec.ResourceQuota = resourceQuota
	if len(limitRange) > 0 {
		spec.LimitRage = limitRange
	}

	if e := createOrUpdateEnvironment(ctx, h, cluster.ClusterName, env.EnvironmentName, spec); e != nil {
		return e
	}
	env.Cluster = &cluster
	syncer := synchronizer.SynchronizerFor(h)
	return syncer.SyncRegistries(ctx, []*models.Environment{env}, project.Registries, synchronizer.SyncKindUpsert)
}

func createOrUpdateEnvironment(ctx context.Context, h base.BaseHandler, clustername, environment string, spec v1beta1.EnvironmentSpec) error {
	return h.Execute(ctx, clustername, func(ctx context.Context, cli agents.Client) error {
		env := &v1beta1.Environment{
			ObjectMeta: metav1.ObjectMeta{Name: environment},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, cli, env, func() error {
			env.Spec = spec
			return nil
		})
		if err != nil {
			logr.FromContextOrDiscard(ctx).Error(err, "apply environment cr", "name", env.Name)
			return err
		}
		return nil
	})
}

// DeleteEnvironment 删除 Environment
// @Tags        Environment
// @Summary     删除 Environment
// @Description 删除 Environment
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                    true "environment_id"
// @Success     204            {object} handlers.ResponseStruct "resp"
// @Router      /v1/environment/{environment_id} [delete]
// @Security    JWT
func (h *EnvironmentHandler) DeleteEnvironment(c *gin.Context) {
	var obj models.Environment
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Preload(
		"Cluster",
		clusterSensitiveFunc,
	).Preload(
		"Project.Tenant",
	).First(&obj, c.Param("environment_id")).Error; err != nil {
		handlers.NoContent(c, nil)
	}
	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "environment")
	h.SetAuditData(c, action, module, obj.EnvironmentName)
	h.SetExtraAuditData(c, models.ResEnvironment, obj.ID)

	envUsers := h.GetDataBase().EnvUsers(obj.ID)
	projAdmins := h.GetDataBase().ProjectAdmins(obj.ProjectID)

	err := h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&obj).Error; err != nil {
			return err
		}
		return h.afterEnvironmentDelete(ctx, tx, &obj)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.ModelCache().DelEnvironment(obj.ProjectID, obj.ID, obj.Cluster.ClusterName, obj.Namespace)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Delete
		msg.ResourceType = msgbus.Environment
		msg.ResourceID = obj.ID
		msg.Detail = i18n.Sprintf(context.TODO(), "deleted the environment %s in the project %s", obj.EnvironmentName, obj.Project.ProjectName)
		msg.ToUsers.Append(projAdmins...).Append(envUsers...)
		msg.AffectedUsers.Append(envUsers...) // 环境所有用户刷新权限
	})

	handlers.NoContent(c, nil)
}

// 环境删除,同步删除CRD
func (h *EnvironmentHandler) afterEnvironmentDelete(ctx context.Context, tx *gorm.DB, env *models.Environment) error {
	return h.Execute(ctx, env.Cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
		envobj := &v1beta1.Environment{}
		err := cli.Get(ctx, client.ObjectKey{Name: env.EnvironmentName}, envobj)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			} else {
				return err
			}
		}
		return cli.Delete(ctx, envobj)
	})
}

// ListEnvironmentUser 获取属于Environment的 User 列表
// @Tags        Environment
// @Summary     获取属于 Environment 的 User 列表
// @Description 获取属于 Environment 的 User 列表
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                                                                true  "environment_id"
// @Param       preload        query    string                                                              false "choices Tenants,SystemRole"
// @Param       page           query    int                                                                 false "page"
// @Param       size           query    int                                                                 false "page"
// @Param       search         query    string                                                              false "search in (Username,Email)"
// @Success     200            {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}} "models.User"
// @Router      /v1/environment/{environment_id}/user [get]
// @Security    JWT
func (h *EnvironmentHandler) ListEnvironmentUser(c *gin.Context) {
	var list []models.User
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:                  "User",
		SearchFields:           []string{"Username", "Email"},
		PreloadFields:          []string{"Tenants", "SystemRole"},
		PreloadSensitiveFields: PreloadSensitiveFields,
		Select:                 handlers.Args("users.*, environment_user_rels.role"),
		Join:                   handlers.Args("join environment_user_rels on environment_user_rels.user_id = users.id"),
		Where:                  []*handlers.QArgs{handlers.Args("environment_user_rels.environment_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveEnvironmentUser 获取Environment 的一个 User详情
// @Tags        Environment
// @Summary     获取Environment 的一个 User详情
// @Description 获取Environment 的一个 User详情
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                                      true "environment_id"
// @Param       user_id        path     uint                                      true "user_id"
// @Success     200            {object} handlers.ResponseStruct{Data=models.User} "models.User"
// @Router      /v1/environment/{environment_id}/user/{user_id} [get]
// @Security    JWT
func (h *EnvironmentHandler) RetrieveEnvironmentUser(c *gin.Context) {
	var user models.User
	if err := h.GetDB().WithContext(c.Request.Context()).Joins(
		"join environment_user_rels on environment_user_rels.user_id = users.id",
	).First(
		&user,
		"environment_user_rels.environment_id = ? and id = ?",
		c.Param(PrimaryKeyName), c.Param("user_id"),
	).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, user)
}

// PostEnvironmentUser 在User和Environment间添加关联关系
// @Tags        Environment
// @Summary     在User和Environment间添加关联关系
// @Description 在User和Environment间添加关联关系
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                                                     true "environment_id"
// @Param       param          body     models.EnvironmentUserRels                               true "表单"`
// @Success     200            {object} handlers.ResponseStruct{Data=models.EnvironmentUserRels} "models.User"
// @Router      /v1/environment/{environment_id}/user [post]
// @Security    JWT
func (h *EnvironmentHandler) PostEnvironmentUser(c *gin.Context) {
	var rel models.EnvironmentUserRels
	if err := c.BindJSON(&rel); err != nil {
		handlers.NotOK(c, err)
		return
	}
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Create(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	user := models.User{}
	h.GetDB().WithContext(ctx).Preload("SystemRole").First(&user, rel.UserID)
	h.ModelCache().FlushUserAuthority(&user)
	h.GetDB().WithContext(ctx).Preload("Environment").First(&rel, rel.ID)

	action := i18n.Sprintf(context.TODO(), "add")
	module := i18n.Sprintf(context.TODO(), "environment member")
	h.SetAuditData(c, action, module, i18n.Sprintf(context.TODO(), "environment %s / user %s / role %s", rel.Environment.EnvironmentName, user.Username, rel.Role))
	h.SetExtraAuditData(c, models.ResEnvironment, rel.EnvironmentID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Add
		msg.ResourceType = msgbus.Environment
		msg.ResourceID = rel.EnvironmentID
		msg.Detail = i18n.Sprintf(context.TODO(), "add user %s to environment %s member as role %s",
			user.Username, rel.Environment.EnvironmentName, rel.Role)
		msg.ToUsers.Append(rel.UserID)
		msg.AffectedUsers.Append(rel.UserID)
	})

	handlers.OK(c, rel)
}

// PutEnvironmentUser 修改 User 和 Environment 的关联关系
// @Tags        Environment
// @Summary     修改 User 和 Environment 的关联关系
// @Description 修改 User 和 Environment 的关联关系
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                                                     true "environment_id"
// @Param       user_id        path     uint                                                     true "user_id"
// @Param       param          body     models.EnvironmentUserRels                               true "表单"`
// @Success     200            {object} handlers.ResponseStruct{Data=models.EnvironmentUserRels} "models.User"
// @Router      /v1/environment/{environment_id}/user/{user_id} [put]
// @Security    JWT
func (h *EnvironmentHandler) PutEnvironmentUser(c *gin.Context) {
	var rel models.EnvironmentUserRels
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).First(&rel, "environment_id =? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NotOK(c, i18n.Errorf(c, "the environment member role you are modifying is not exist"))
		return
	}
	if err := c.BindJSON(&rel); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().WithContext(ctx).Save(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	user := models.User{}
	h.GetDB().WithContext(ctx).Preload("SystemRole").First(&user, rel.UserID)
	h.ModelCache().FlushUserAuthority(&user)
	h.GetDB().WithContext(ctx).Preload("Environment").First(&rel, rel.ID)

	action := i18n.Sprintf(context.TODO(), "update")
	module := i18n.Sprintf(context.TODO(), "environment member")
	h.SetAuditData(c, action, module, i18n.Sprintf(context.TODO(), "environment %s / user %s / role %s", rel.Environment.EnvironmentName, user.Username, rel.Role))
	h.SetExtraAuditData(c, models.ResEnvironment, rel.EnvironmentID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Update
		msg.ResourceType = msgbus.Environment
		msg.ResourceID = rel.EnvironmentID
		msg.Detail = i18n.Sprintf(context.TODO(), "set user %s to environment %s member as role %s",
			user.Username, rel.Environment.EnvironmentName, rel.Role)
		msg.ToUsers.Append(rel.UserID)
		msg.AffectedUsers.Append(rel.UserID) // 自己
	})
	handlers.OK(c, rel)
}

// DeleteEnvironmentUser 删除 User 和 Environment 的关系
// @Tags        Environment
// @Summary     删除 User 和 Environment 的关系
// @Description 删除 User 和 Environment 的关系
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                                      true "environment_id"
// @Param       user_id        path     uint                                      true "user_id"
// @Success     200            {object} handlers.ResponseStruct{Data=models.User} "models.User"
// @Router      /v1/environment/{environment_id}/user/{user_id} [delete]
// @Security    JWT
func (h *EnvironmentHandler) DeleteEnvironmentUser(c *gin.Context) {
	var rel models.EnvironmentUserRels
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).First(&rel, "environment_id =? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	h.GetDB().WithContext(ctx).Preload("Environment.Project.Tenant").First(&rel, rel.ID)
	if err := h.GetDB().WithContext(ctx).Delete(&rel, "environment_id =? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	user := models.User{}
	h.GetDB().WithContext(ctx).Preload("SystemRole").First(&user, c.Param("user_id"))
	h.ModelCache().FlushUserAuthority(&user)

	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "environment member")
	h.SetAuditData(c, action, module, i18n.Sprintf(context.TODO(), "environment %s / user %s / role %s", rel.Environment.EnvironmentName, user.Username, rel.Role))
	h.SetExtraAuditData(c, models.ResEnvironment, rel.EnvironmentID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Delete
		msg.ResourceType = msgbus.Environment
		msg.ResourceID = rel.EnvironmentID
		msg.Detail = i18n.Sprintf(context.TODO(), "delete user %s from environment %s member",
			user.Username, rel.Environment.EnvironmentName)
		msg.ToUsers.Append(rel.UserID)
		msg.AffectedUsers.Append(rel.UserID) // 自己
	})

	handlers.NoContent(c, nil)
}

// GetEnvironmentResource 获取环境资源清单
// @Tags        ResourceList
// @Summary     获取环境资源清单
// @Description 获取环境资源清单
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                                                       true  "environment_id"
// @Param       date           query    string                                                     false "date"
// @Success     200            {object} handlers.ResponseStruct{Data=[]models.EnvironmentResource} "EnvironmentResource"
// @Router      /v1/environment/{environment_id}/resources [get]
// @Security    JWT
func (h *EnvironmentHandler) GetEnvironmentResource(c *gin.Context) {
	dateTime, err := time.Parse(time.RFC3339, c.Query("date"))
	if err != nil {
		// 默认取到昨天的时间
		dateTime = time.Now().Add(-24 * time.Hour)
	}
	// 第二天的0点
	dayTime := utils.NextDayStartTime(dateTime)

	env := models.Environment{}
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Preload("Project.Tenant").Where("id = ?", c.Param("environment_id")).First(&env).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	tenantName := env.Project.Tenant.TenantName

	var envREs models.EnvironmentResource
	// eg. 查看1号的。要取2号的第一条数据
	if err := h.GetDB().WithContext(ctx).
		Where("tenant_name = ? and project_name = ? and environment_name = ? and created_at >= ? and created_at < ?", tenantName, env.Project.ProjectName, env.EnvironmentName, dayTime.Format(time.RFC3339), dayTime.Add(24*time.Hour).Format(time.RFC3339)).
		Order("created_at").
		First(&envREs).Error; err != nil {
		log.Error(err, "get environment resource")
	}
	handlers.OK(c, envREs)
}

// @Tags        NetworkIsolated
// @Summary     环境网络隔离开关
// @Description 环境网络隔离开关
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                                                  true "environment_id"
// @Param       param          body     handlers.IsolatedSwitch                               true "表单"
// @Success     200            {object} handlers.ResponseStruct{Data=handlers.IsolatedSwitch} "object"
// @Router      /v1/environment/{environment_id}/action/networkisolate [post]
// @Security    JWT
func (h *EnvironmentHandler) EnvironmentSwitch(c *gin.Context) {
	form := handlers.IsolatedSwitch{}
	if err := c.BindJSON(&form); err != nil {
		handlers.NotOK(c, err)
		return
	}
	var env models.Environment
	ctx := c.Request.Context()
	if e := h.GetDB().WithContext(ctx).Preload("Cluster", clusterSensitiveFunc).Preload("Project.Tenant").First(&env, "id = ?", c.Param(PrimaryKeyName)).Error; e != nil {
		handlers.NotOK(c, e)
		return
	}

	action := i18n.Sprintf(context.TODO(), "enable")
	module := i18n.Sprintf(context.TODO(), "environment network isolation")
	h.SetAuditData(c, action, module, env.EnvironmentName)

	tnetpol := &v1beta1.TenantNetworkPolicy{}
	err := h.Execute(ctx, env.Cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
		if err := cli.Get(ctx, client.ObjectKey{Name: env.Project.Tenant.TenantName}, tnetpol); err != nil {
			return err
		}

		index := -1
		for idx, envpol := range tnetpol.Spec.EnvironmentNetworkPolicies {
			if envpol.Name == env.EnvironmentName {
				index = idx
			}
		}
		if index == -1 && form.Isolate {
			tnetpol.Spec.EnvironmentNetworkPolicies = append(tnetpol.Spec.EnvironmentNetworkPolicies, v1beta1.EnvironmentNetworkPolicy{
				Name:    env.EnvironmentName,
				Project: env.Project.ProjectName,
			})
		}
		if index != -1 && !form.Isolate {
			tnetpol.Spec.EnvironmentNetworkPolicies = append(tnetpol.Spec.EnvironmentNetworkPolicies[:index], tnetpol.Spec.EnvironmentNetworkPolicies[index+1:]...)
		}

		return cli.Update(ctx, tnetpol)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, tnetpol)
}

type EnvironmentObservabilityRet struct {
	ProjectID       uint   `json:"projectID"`
	ProjectName     string `json:"projectName"`
	EnvironmentID   uint   `json:"environmentID"`
	EnvironmentName string `json:"environmentName"`
	ClusterName     string `json:"clusterName"`
	Namespace       string `json:"namespace"`

	Labels map[string]string `json:"labels"`

	Monitoring  bool `json:"monitoring"`  // 是否启用监控
	Logging     bool `json:"logging"`     // 是否启日志
	ServiceMesh bool `json:"serviceMesh"` // 是否启用服务网格

	ContainerRestartTotal int64 `json:"containerRestartTotal"`

	CPU    string `json:"cpu"`
	Memory string `json:"memory"`

	MonitorCollectorCount int `json:"monitorCollectorCount"` // metrics采集器数量

	AlertRuleCount     int            `json:"alertRuleCount"`   // 告警规则列表
	AlertResourceMap   map[string]int `json:"alertResourceMap"` // 告警规则的资源map
	ErrorAlertCount    int            `json:"errorAlertCount"`
	CriticalAlertCount int            `json:"criticalAlertCount"`

	LoggingCollectorCount int    `json:"loggingCollectorCount"`
	ErrorLogCount         int    `json:"errorLogCount"`
	LogRate               string `json:"logRate"`

	EventCount int `json:"eventCount"` // 事件数量

	Warning string `json:"warning"`
}

// @Tags        EnvironmentObservabilityDetails
// @Summary     环境可观测性概览
// @Description 环境可观测性概览
// @Accept      json
// @Produce     json
// @Param       environment_id path     uint                                                      true  "environment_id"
// @Param       duration       query    string                                                    false "过去多长时间: 30s,5m,1h,1d,1w, 默认1h"
// @Success     200            {object} handlers.ResponseStruct{Data=EnvironmentObservabilityRet} "object"
// @Router      /v1/environment/{environment_id}/observability [get]
// @Security    JWT
func (h *EnvironmentHandler) EnvironmentObservabilityDetails(c *gin.Context) {
	env := models.Environment{}
	ctx := c.Request.Context()
	db := h.GetDB().WithContext(ctx)
	if err := db.Preload("Cluster").Preload("Project").Where("id = ?", c.Param("environment_id")).First(&env).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	ret := EnvironmentObservabilityRet{
		EnvironmentID:   env.ID,
		EnvironmentName: env.EnvironmentName,
		ProjectID:       env.ProjectID,
		ProjectName:     env.Project.ProjectName,
		ClusterName:     env.Cluster.ClusterName,
		Namespace:       env.Namespace,

		AlertResourceMap: make(map[string]int),
	}

	alertrules := []models.AlertRule{}
	if err := db.Find(&alertrules, "cluster = ? and namespace = ?", env.Cluster.ClusterName, env.Namespace).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	ret.AlertRuleCount = len(alertrules)

	for _, v := range alertrules {
		var key string
		if v.AlertType == prometheus.AlertTypeMonitor {
			if v.PromqlGenerator != nil {
				key = v.PromqlGenerator.Resource
			} else {
				key = "raw promql"
			}
		} else {
			key = "logging"
		}
		if count, ok := ret.AlertResourceMap[key]; ok {
			count++
			ret.AlertResourceMap[key] = count
		} else {
			ret.AlertResourceMap[key] = 1
		}
		if v.State == "firing" {
			for _, level := range v.AlertLevels {
				if level.Severity == prometheus.SeverityError {
					ret.ErrorAlertCount++
				}
				if level.Severity == prometheus.SeverityCritical {
					ret.CriticalAlertCount++
				}
			}
		}
	}

	dur := c.DefaultQuery("duration", "1h")
	h.Execute(ctx, env.Cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
		eg := errgroup.Group{}

		// log, monitor, mesh status
		ret.ServiceMesh = env.VirtualSpaceID != nil
		eg.Go(func() error {
			smList := monitoringv1.ServiceMonitorList{}
			if err := cli.List(ctx, &smList, client.InNamespace(env.Namespace)); err != nil {
				return err
			}
			ret.Monitoring = len(smList.Items) != 0
			return nil
		})

		eg.Go(func() error {
			flowList := loggingv1beta1.FlowList{}
			if err := cli.List(ctx, &flowList, client.InNamespace(env.Namespace)); err != nil {
				return err
			}
			ret.Logging = len(flowList.Items) != 0
			return nil
		})

		// labels
		eg.Go(func() error {
			ns := corev1.Namespace{}
			if err := cli.Get(ctx, types.NamespacedName{Name: env.Namespace}, &ns); err != nil {
				return err
			}
			ret.Labels = ns.Labels
			return nil
		})

		// contaienr restart
		eg.Go(func() error {
			query := fmt.Sprintf(`sum(increase(kube_pod_container_status_restarts_total{namespace="%s"}[%s]))`, env.Namespace, dur)
			containerRestart, err := cli.Extend().PrometheusVector(ctx, query)
			if err != nil {
				return fmt.Errorf("prometheus vector query [%s] : %w", query, err)
			}
			if containerRestart.Len() == 0 {
				ret.ContainerRestartTotal = 0
			} else {
				ret.ContainerRestartTotal = int64(containerRestart[0].Value)
			}
			return nil
		})

		// cpu
		eg.Go(func() error {
			query := fmt.Sprintf(`round(gems_namespace_cpu_usage_cores{namespace="%s"}, 0.01)`, env.Namespace)
			cpu, err := cli.Extend().PrometheusVector(ctx, query)
			if err != nil {
				return fmt.Errorf("prometheus vector query [%s] : %w", query, err)
			}
			if cpu.Len() == 0 {
				ret.CPU = ""
			} else {
				ret.CPU = fmt.Sprintf("%.2fCore", cpu[0].Value)
			}
			return nil
		})

		// memory
		eg.Go(func() error {
			query := fmt.Sprintf(`gems_namespace_memory_usage_bytes{namespace="%s"}`, env.Namespace)
			memory, err := cli.Extend().PrometheusVector(ctx, query)
			if err != nil {
				return fmt.Errorf("prometheus vector query [%s] : %w", query, err)
			}
			if memory.Len() == 0 {
				ret.Memory = ""
			} else {
				ret.Memory = utils.ConvertBytes(float64(memory[0].Value))
			}
			return nil
		})

		// metrics
		eg.Go(func() error {
			sms := monitoringv1.ServiceMonitorList{}
			if err := cli.List(ctx, &sms, client.InNamespace(env.Namespace)); err != nil {
				return err
			}
			ret.MonitorCollectorCount = len(sms.Items)
			return nil
		})

		eg.Go(func() error {
			flowList := loggingv1beta1.FlowList{}
			if err := cli.List(ctx, &flowList, client.InNamespace(env.Namespace)); err != nil {
				return err
			}
			ret.LoggingCollectorCount = len(flowList.Items)
			return nil
		})

		eg.Go(func() error {
			query := fmt.Sprintf(`sum(count_over_time({namespace="%s", container="event-exporter"}| json | line_format "{{.metadata_namespace}}" |= "%s" [%s]))`, gems.NamespaceEventer, env.Namespace, dur)
			resp, err := cli.Extend().LokiQuery(ctx, query)
			if err != nil {
				return fmt.Errorf("loki query [%s]: %w", query, err)
			}
			ret.EventCount = getLokiRespValue(resp)
			return nil
		})

		eg.Go(func() error {
			query := fmt.Sprintf(`sum(sum_over_time(gems_loki_error_logs_count_last_1m{namespace="%s"}[%s]))`, env.Namespace, dur)
			resp, err := cli.Extend().PrometheusVector(ctx, query)
			if err != nil {
				return fmt.Errorf("prometheus vector query [%s]: %w", query, err)
			}
			if resp.Len() > 0 {
				ret.ErrorLogCount = int(resp[0].Value)
			}
			return nil
		})

		eg.Go(func() error {
			query := fmt.Sprintf(`sum(gems_loki_logs_count_last_1m{namespace="%s"})`, env.Namespace)
			resp, err := cli.Extend().PrometheusVector(ctx, query)
			if err != nil {
				return fmt.Errorf("prometheus vector query [%s]: %w", query, err)
			}
			if resp.Len() > 0 {
				ret.LogRate = fmt.Sprintf("%d/min", int(resp[0].Value))
			} else {
				ret.LogRate = "0/min"
			}
			return nil
		})

		if err := eg.Wait(); err != nil && ret.Warning == "" {
			ret.Warning = err.Error()
		}
		return nil
	})
	handlers.OK(c, ret)
}

func getLokiRespValue(resp loki.QueryResponseData) int {
	if len(resp.Result) == 0 {
		return 0
	}
	if result, ok := resp.Result[0].(map[string]interface{}); ok {
		value := result["value"]
		if vals, ok := value.([]interface{}); ok {
			for _, v := range vals {
				if count, ok := v.(string); ok {
					ret, _ := strconv.Atoi(count)
					return ret
				}
			}
		}
	}
	return 0
}
