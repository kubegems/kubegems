package environment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/handlers/base"
	"kubegems.io/pkg/service/models"
	ut "kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/msgbus"
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
// @Tags Environment
// @Summary Environment列表
// @Description Environment列表
// @Accept json
// @Produce json
// @Param EnvironmentName query string false "EnvironmentName"
// @Param preload query string false "choices Creator,Cluster,Project,Applications,Users"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (EnvironmentName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Environment}} "Environment"
// @Router /v1/environment [get]
// @Security JWT
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
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveEnvironment Environment详情
// @Tags Environment
// @Summary Environment详情
// @Description get Environment详情
// @Accept json
// @Produce json
// @Param environment_id path uint true "environment_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Environment} "Environment"
// @Router /v1/environment/{environment_id} [get]
// @Security JWT
func (h *EnvironmentHandler) RetrieveEnvironment(c *gin.Context) {
	var (
		users []*models.User
		obj   models.Environment
	)
	if err := h.GetDB().Select(
		"users.*, environment_user_rels.role",
	).Joins(
		"join environment_user_rels  on  environment_user_rels.user_id = users.id",
	).Find(&users, "`environment_user_rels`.`environment_id` = ?", c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Preload("Cluster", clusterSensitiveFunc).First(&obj, "id = ?", c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	obj.Users = users
	handlers.OK(c, obj)
}

// PutEnvironment 修改Environment
// @Tags Environment
// @Summary 修改Environment
// @Description 修改Environment
// @Accept json
// @Produce json
// @Param environment_id path uint true "environment_id"
// @Param param body models.Environment true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Environment} "Environment"
// @Router /v1/environment/{environment_id} [put]
// @Security JWT
func (h *EnvironmentHandler) PutEnvironment(c *gin.Context) {
	var obj models.Environment
	if err := h.GetDB().Preload("Cluster").First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "更新", "环境", obj.EnvironmentName)
	h.SetExtraAuditData(c, models.ResEnvironment, obj.ID)
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	obj.LimitRange = models.FillDefaultLimigrange(&obj)
	if strconv.Itoa(int(obj.ID)) != c.Param(PrimaryKeyName) {
		handlers.NotOK(c, fmt.Errorf("请求体参数和URL参数中ID不同"))
		return
	}
	ctx := c.Request.Context()
	cluster := obj.Cluster
	if err := ValidateEnvironmentNamespace(ctx, h.BaseHandler, h.GetDB(), obj.Namespace, obj.EnvironmentName, obj.Cluster.ClusterName); err != nil {
		handlers.NotOK(c, err)
		return
	}
	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(&obj).Error; err != nil {
			return err
		}
		return AfterEnvironmentSave(ctx, h.BaseHandler, tx, &obj)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.GetCacheLayer().GetGlobalResourceTree().UpsertEnvironment(obj.ProjectID, obj.ID, obj.EnvironmentName, cluster.ClusterName, obj.Namespace)
	handlers.OK(c, obj)
}

// ValidateEnvironmentNamespace 校验绑定的namespace是否合法.
func ValidateEnvironmentNamespace(ctx context.Context, h base.BaseHandler, tx *gorm.DB, namespace, envname, clustername string) error {
	forbiddenBindNamespaces := []string{
		gems.NamespaceGateway,
		gems.NamespaceLogging,
		gems.NamespaceSystem,
		gems.NamespaceMonitor,
		gems.NamespaceWorkflow,
		"kube-system",
		"default",
		"istio-system",
	}
	if ut.ContainStr(forbiddenBindNamespaces, namespace) {
		return fmt.Errorf("namespace  %s is now allowed, it's a system retain namespace", namespace)
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
			return fmt.Errorf("namespace %s is binded with other environment", namespace)
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
	if e := tx.Preload("Tenant").First(&project, "id = ?", env.ProjectID).Error; e != nil {
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
	return nil
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
		return err
	})
}

// DeleteEnvironment 删除 Environment
// @Tags Environment
// @Summary 删除 Environment
// @Description 删除 Environment
// @Accept json
// @Produce json
// @Param environment_id path uint true "environment_id"
// @Success 204 {object} handlers.ResponseStruct "resp"
// @Router /v1/environment/{environment_id} [delete]
// @Security JWT
func (h *EnvironmentHandler) DeleteEnvironment(c *gin.Context) {
	var obj models.Environment
	if err := h.GetDB().Preload("Cluster", clusterSensitiveFunc).Preload("Project.Tenant").First(&obj, c.Param("environment_id")).Error; err != nil {
		handlers.NoContent(c, nil)
	}
	h.SetAuditData(c, "删除", "环境", obj.EnvironmentName)
	h.SetExtraAuditData(c, models.ResEnvironment, obj.ID)

	envUsers := h.GetDataBase().EnvUsers(obj.ID)
	projAdmins := h.GetDataBase().ProjectAdmins(obj.ProjectID)

	ctx := c.Request.Context()

	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&obj).Error; err != nil {
			return err
		}
		return h.afterEnvironmentDelete(ctx, tx, &obj)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.GetCacheLayer().GetGlobalResourceTree().DelEnvironment(obj.ProjectID, obj.ID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Delete).
		ResourceType(msgbus.Environment).
		ResourceID(obj.ID).
		Content(fmt.Sprintf("删除了租户%s/项目%s中的环境%s", obj.Project.Tenant.TenantName, obj.Project.ProjectName, obj.EnvironmentName)).
		SetUsersToSend(
			projAdmins,
			envUsers,
		).
		AffectedUsers(
			envUsers, // 环境所有用户刷新权限
		).
		Send()
	handlers.NoContent(c, nil)
}

// 环境删除,同步删除CRD
func (h *EnvironmentHandler) afterEnvironmentDelete(ctx context.Context, tx *gorm.DB, env *models.Environment) error {
	return h.Execute(ctx, env.Cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
		return cli.Delete(ctx, &v1beta1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name: env.EnvironmentName,
			},
		})
	})
}

// ListEnvironmentUser 获取属于Environment的 User 列表
// @Tags Environment
// @Summary 获取属于 Environment 的 User 列表
// @Description 获取属于 Environment 的 User 列表
// @Accept json
// @Produce json
// @Param environment_id path uint true "environment_id"
// @Param preload query string false "choices Tenants,SystemRole"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (Username,Email)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}} "models.User"
// @Router /v1/environment/{environment_id}/user [get]
// @Security JWT
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
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveEnvironmentUser 获取Environment 的一个 User详情
// @Tags Environment
// @Summary 获取Environment 的一个 User详情
// @Description 获取Environment 的一个 User详情
// @Accept json
// @Produce json
// @Param environment_id path uint true "environment_id"
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "models.User"
// @Router /v1/environment/{environment_id}/user/{user_id} [get]
// @Security JWT
func (h *EnvironmentHandler) RetrieveEnvironmentUser(c *gin.Context) {
	var user models.User
	if err := h.GetDB().Joins(
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
// @Tags Environment
// @Summary 在User和Environment间添加关联关系
// @Description 在User和Environment间添加关联关系
// @Accept json
// @Produce json
// @Param environment_id path uint true "environment_id"
// @Param param body models.EnvironmentUserRels  true "表单"`
// @Success 200 {object} handlers.ResponseStruct{Data=models.EnvironmentUserRels} "models.User"
// @Router /v1/environment/{environment_id}/user [post]
// @Security JWT
func (h *EnvironmentHandler) PostEnvironmentUser(c *gin.Context) {
	var rel models.EnvironmentUserRels
	if err := c.BindJSON(&rel); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Create(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	user := models.User{}
	h.GetDB().Preload("SystemRole").First(&user, rel.UserID)
	h.GetCacheLayer().FlushUserAuthority(&user)

	h.GetDB().Preload("Environment.Project.Tenant").First(&rel, rel.ID)

	h.SetAuditData(c, "添加", "环境成员", fmt.Sprintf("环境[%v]/用户[%v]", rel.Environment.EnvironmentName, user.Username))
	h.SetExtraAuditData(c, models.ResEnvironment, rel.EnvironmentID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Add).
		ResourceType(msgbus.Environment).
		ResourceID(rel.EnvironmentID).
		Content(fmt.Sprintf("向租户%s/项目%s/环境%s中添加了用户%s",
			rel.Environment.Project.Tenant.TenantName, rel.Environment.Project.ProjectName, rel.Environment.EnvironmentName, user.Username)).
		SetUsersToSend(
			[]uint{rel.UserID}, // 自己
		).
		AffectedUsers([]uint{rel.UserID}).
		Send()
	handlers.OK(c, rel)
}

// PutEnvironmentUser 修改 User 和 Environment 的关联关系
// @Tags Environment
// @Summary  修改 User 和 Environment 的关联关系
// @Description  修改 User 和 Environment 的关联关系
// @Accept json
// @Produce json
// @Param environment_id path uint true "environment_id"
// @Param user_id path uint true "user_id"
// @Param param body models.EnvironmentUserRels  true "表单"`
// @Success 200 {object} handlers.ResponseStruct{Data=models.EnvironmentUserRels} "models.User"
// @Router /v1/environment/{environment_id}/user/{user_id} [put]
// @Security JWT
func (h *EnvironmentHandler) PutEnvironmentUser(c *gin.Context) {
	var rel models.EnvironmentUserRels
	if err := h.GetDB().First(&rel, "environment_id =? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("不存在 \"环境-用户\" 关系"))
		return
	}
	if err := c.BindJSON(&rel); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Save(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	user := models.User{}
	h.GetDB().Preload("SystemRole").First(&user, rel.UserID)
	h.GetCacheLayer().FlushUserAuthority(&user)

	h.GetDB().Preload("Environment.Project.Tenant").First(&rel, rel.ID)
	h.SetAuditData(c, "更新", "环境成员", fmt.Sprintf("环境[%v]/用户[%v]", rel.Environment.EnvironmentName, user.Username))
	h.SetExtraAuditData(c, models.ResEnvironment, rel.EnvironmentID)
	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Update).
		ResourceType(msgbus.Environment).
		ResourceID(rel.EnvironmentID).
		Content(fmt.Sprintf("将租户%s/项目%s/环境%s中的用户%s设置为了%s",
			rel.Environment.Project.Tenant.TenantName, rel.Environment.Project.ProjectName, rel.Environment.EnvironmentName, user.Username, rel.Role)).
		SetUsersToSend(
			[]uint{rel.UserID}, // 自己
		).
		AffectedUsers([]uint{rel.UserID}).
		Send()
	handlers.OK(c, rel)
}

// DeleteEnvironmentUser 删除 User 和 Environment 的关系
// @Tags Environment
// @Summary 删除 User 和 Environment 的关系
// @Description 删除 User 和 Environment 的关系
// @Accept json
// @Produce json
// @Param environment_id path uint true "environment_id"
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "models.User"
// @Router /v1/environment/{environment_id}/user/{user_id} [delete]
// @Security JWT
func (h *EnvironmentHandler) DeleteEnvironmentUser(c *gin.Context) {
	var rel models.EnvironmentUserRels
	if err := h.GetDB().First(&rel, "environment_id =? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	h.GetDB().Preload("Environment.Project.Tenant").First(&rel, rel.ID)
	if err := h.GetDB().Delete(&rel, "environment_id =? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	user := models.User{}
	h.GetDB().Preload("SystemRole").First(&user, c.Param("user_id"))
	h.GetCacheLayer().FlushUserAuthority(&user)

	h.SetAuditData(c, "删除", "环境成员", fmt.Sprintf("环境[%v]/用户[%v]", rel.Environment.EnvironmentName, user.Username))
	h.SetExtraAuditData(c, models.ResEnvironment, rel.EnvironmentID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Delete).
		ResourceType(msgbus.Environment).
		ResourceID(rel.EnvironmentID).
		Content(fmt.Sprintf("删除了租户%s/项目%s/环境%s中的用户%s",
			rel.Environment.Project.Tenant.TenantName, rel.Environment.Project.ProjectName, rel.Environment.EnvironmentName, user.Username)).
		SetUsersToSend(
			[]uint{rel.UserID}, // 自己
		).
		AffectedUsers([]uint{rel.UserID}).
		Send()
	handlers.NoContent(c, nil)
}

// GetEnvironmentResource 获取环境资源清单
// @Tags ResourceList
// @Summary 获取环境资源清单
// @Description 获取环境资源清单
// @Accept json
// @Produce json
// @Param environment_id path uint true "environment_id"
// @Param date query string false "date"
// @Success 200 {object} handlers.ResponseStruct{Data=[]models.EnvironmentResource} "EnvironmentResource"
// @Router /v1/environment/{environment_id}/resources [get]
// @Security JWT
func (h *EnvironmentHandler) GetEnvironmentResource(c *gin.Context) {
	dateTime, err := time.Parse(time.RFC3339, c.Query("date"))
	if err != nil {
		// 默认取到昨天的时间
		dateTime = time.Now().Add(-24 * time.Hour)
	}
	// 第二天的0点
	dayTime := ut.NextDayStartTime(dateTime)

	env := models.Environment{}
	if err := h.GetDB().Preload("Project.Tenant").Where("id = ?", c.Param("environment_id")).First(&env).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	tenantName := env.Project.Tenant.TenantName

	var envREs models.EnvironmentResource
	// eg. 查看1号的。要取2号的第一条数据
	if err := h.GetDB().
		Where("tenant_name = ? and project_name = ? and environment_name = ? and created_at >= ? and created_at < ?", tenantName, env.Project.ProjectName, env.EnvironmentName, dayTime.Format(time.RFC3339), dayTime.Add(24*time.Hour).Format(time.RFC3339)).
		Order("created_at").
		First(&envREs).Error; err != nil {
		log.Error(err, "get environment resource")
	}
	handlers.OK(c, envREs)
}

// @Tags NetworkIsolated
// @Summary 环境网络隔离开关
// @Description 环境网络隔离开关
// @Accept json
// @Produce json
// @Param environment_id path uint true "environment_id"
// @Param param body handlers.IsolatedSwitch true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.IsolatedSwitch} "object"
// @Router /v1/environment/{environment_id}/action/networkisolate [post]
// @Security JWT
func (h *EnvironmentHandler) EnvironmentSwitch(c *gin.Context) {
	form := handlers.IsolatedSwitch{}
	if err := c.BindJSON(&form); err != nil {
		handlers.NotOK(c, err)
		return
	}
	var env models.Environment
	if e := h.GetDB().Preload("Cluster", clusterSensitiveFunc).Preload("Project.Tenant").First(&env, "id = ?", c.Param(PrimaryKeyName)).Error; e != nil {
		handlers.NotOK(c, e)
		return
	}
	h.SetAuditData(c, "开启", "环境网络隔离", env.EnvironmentName)

	ctx := c.Request.Context()

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
