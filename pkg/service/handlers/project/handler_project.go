package projecthandler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/msgbus"
)

var (
	SearchFields           = []string{"project_name"}
	FilterFields           = []string{"project_name", "TenantID"}
	PreloadFields          = []string{"Applications", "Environments", "Registries", "Users", "Tenant"}
	OrderFields            = []string{"project_name", "ID"}
	PreloadSensitiveFields = map[string]string{"Cluster": "id, cluster_name"}
	ModelName              = "Project"
	PrimaryKeyName         = "project_id"
)

// ListProject 列表 Project
// @Tags Project
// @Summary Project列表
// @Description Project列表
// @Accept json
// @Produce json
// @Param ProjectName query string false "ProjectName"
// @Param TenantID query string false "TenantID"
// @Param preload query string false "choices Applications,Environments,Registries,Users,Tenant"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (ProjectName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Project}} "Project"
// @Router /v1/project [get]
// @Security JWT
func (h *ProjectHandler) ListProject(c *gin.Context) {
	var list []models.Project
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "Project",
		SearchFields:  SearchFields,
		PreloadFields: PreloadFields,
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// RetrieveProject Project详情
// @Tags Project
// @Summary Project详情
// @Description get Project详情
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Project} "Project"
// @Router /v1/project/{project_id} [get]
// @Security JWT
func (h *ProjectHandler) RetrieveProject(c *gin.Context) {
	var (
		obj   models.Project
		users []*models.User
	)
	if err := h.GetDB().Select(
		"users.*, project_user_rels.role",
	).Joins(
		"join project_user_rels  on  project_user_rels.user_id = users.id",
	).Find(&users, "`project_user_rels`.`project_id` = ?", c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Preload("Tenant").First(&obj, "id = ?", c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	obj.Users = users
	handlers.OK(c, obj)
}

// PutProject 修改Project
// @Tags Project
// @Summary 修改Project
// @Description 修改Project
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param param body models.Project true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Project} "Project"
// @Router /v1/project/{project_id} [put]
// @Security JWT
func (h *ProjectHandler) PutProject(c *gin.Context) {
	var obj models.Project
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "项目", obj.ProjectName)
	h.SetExtraAuditData(c, models.ResProject, obj.ID)

	if c.Param(PrimaryKeyName) != strconv.Itoa(int(obj.ID)) {
		handlers.NotOK(c, fmt.Errorf("请求体参数和URL参数ID不一致"))
		return
	}
	if err := h.GetDB().Save(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.GetCacheLayer().GetGlobalResourceTree().UpsertProject(obj.TenantID, obj.ID, obj.ProjectName)
	handlers.OK(c, obj)
}

// DeleteProject 删除 Project
// @Tags Project
// @Summary 删除 Project
// @Description 删除 Project
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Success 204 {object} handlers.ResponseStruct "resp"
// @Router /v1/project/{project_id} [delete]
// @Security JWT
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	var obj models.Project
	if err := h.GetDB().Preload("Environments.Cluster").Preload("Tenant").First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}

	projUsers := h.GetDataBase().ProjectUsers(obj.ID)
	tenantAdmins := h.GetDataBase().TenantAdmins(obj.TenantID)
	h.SetAuditData(c, "删除", "项目", obj.ProjectName)
	h.SetExtraAuditData(c, models.ResProject, obj.ID)

	ctx := c.Request.Context()

	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&obj).Error; err != nil {
			return err
		}
		return h.afterProjectDelete(ctx, tx, &obj)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.GetCacheLayer().GetGlobalResourceTree().DelProject(obj.TenantID, obj.ID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Delete).
		ResourceType(msgbus.Project).
		ResourceID(obj.ID).
		Content(fmt.Sprintf("删除了租户%s中的项目%s", obj.Tenant.TenantName, obj.ProjectName)).
		SetUsersToSend(
			tenantAdmins,
			projUsers,
		).
		AffectedUsers(
			projUsers, // 项目用户需刷新权限
		).
		Send()
	handlers.NoContent(c, nil)
}

/*
	删除项目后
	删除各个集群的环境(tenv),tenv本身删除是Controller自带垃圾回收的，其ns下所有资源将清空
*/
func (h *ProjectHandler) afterProjectDelete(ctx context.Context, tx *gorm.DB, p *models.Project) error {
	for _, env := range p.Environments {
		err := h.Execute(ctx, env.Cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
			environment := &v1beta1.Environment{
				ObjectMeta: metav1.ObjectMeta{Name: env.EnvironmentName},
			}
			return cli.Delete(ctx, environment)
		})
		if err != nil {
			return err
		}
	}
	// TODO: 删除 GIT 中的数据
	// TODO: 删除 ARGO 中的数据
	return nil
}

// ListProjectUser 获取属于Project的 User 列表
// @Tags Project
// @Summary 获取属于 Project 的 User 列表
// @Description 获取属于 Project 的 User 列表
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param preload query string false "choices Tenants,SystemRole"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (Username,Email)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}} "models.User"
// @Router /v1/project/{project_id}/user [get]
// @Security JWT
func (h *ProjectHandler) ListProjectUser(c *gin.Context) {
	var list []models.User
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "User",
		SearchFields:  []string{"Username", "Email"},
		PreloadFields: []string{"Tenants", "SystemRole"},
		Select:        handlers.Args("users.*, project_user_rels.role"),
		Join:          handlers.Args("join project_user_rels on project_user_rels.user_id = users.id"),
		Where:         []*handlers.QArgs{handlers.Args("project_user_rels.project_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveProjectUser 获取Project 的一个 User详情
// @Tags Project
// @Summary 获取Project 的一个 User详情
// @Description 获取Project 的一个 User详情
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "models.User"
// @Router /v1/project/{project_id}/user/{user_id} [get]
// @Security JWT
func (h *ProjectHandler) RetrieveProjectUser(c *gin.Context) {
	var user models.User
	if err := h.GetDB().Model(
		&models.User{},
	).Joins(
		"join project_user_rels on project_user_rels.user_id = users.id",
	).Where(
		"users.id = ? and project_user_rels.project_id = ?",
		c.Param("user_id"),
		c.Param("project_id"),
	).First(&user).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, user)
}

// PostProjectUser 在User和Project间添加关联关系
// @Tags Project
// @Summary 在User和Project间添加关联关系
// @Description 在User和Project间添加关联关系
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param param body models.ProjectUserRels  true "表单"`
// @Success 200 {object} handlers.ResponseStruct{Data=models.ProjectUserRels} "models.User"
// @Router /v1/project/{project_id}/user [post]
// @Security JWT
func (h *ProjectHandler) PostProjectUser(c *gin.Context) {
	var rel models.ProjectUserRels
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

	h.GetDB().Preload("Project.Tenant").First(&rel, rel.ID)

	h.SetAuditData(c, "添加", "项目成员", fmt.Sprintf("项目[%v]/成员[%v]", rel.Project.ProjectName, user.Username))
	h.SetExtraAuditData(c, models.ResProject, rel.ProjectID)
	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Add).
		ResourceType(msgbus.Project).
		ResourceID(rel.ProjectID).
		Content(fmt.Sprintf("向租户%s/项目%s中添加了用户%s", rel.Project.Tenant.TenantName, rel.Project.ProjectName, user.Username)).
		SetUsersToSend(
			[]uint{rel.UserID}, // 自己
			func() []uint {
				if rel.Role == models.ProjectRoleAdmin {
					return h.GetDataBase().ProjectAdmins(rel.ProjectID)
				}
				return nil
			}(), // 项目管理员
		).
		AffectedUsers([]uint{rel.UserID}).
		Send()
	handlers.OK(c, rel)
}

// PutProjectUser 修改 User 和 Project 的关联关系
// @Tags Project
// @Summary  修改 User 和 Project 的关联关系
// @Description  修改 User 和 Project 的关联关系
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param user_id path uint true "user_id"
// @Param param body models.ProjectUserRels  true "表单"`
// @Success 200 {object} handlers.ResponseStruct{Data=models.ProjectUserRels} "models.User"
// @Router /v1/project/{project_id}/user/{user_id} [put]
// @Security JWT
func (h *ProjectHandler) PutProjectUser(c *gin.Context) {
	var rel models.ProjectUserRels
	if err := c.BindJSON(&rel); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().First(&rel, "project_id = ? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("不可以修改不存在的 \"项目-用户\" 关系"))
		return
	}
	if err := h.GetDB().Save(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	user := models.User{}
	h.GetDB().Preload("SystemRole").First(&user, rel.UserID)
	h.GetCacheLayer().FlushUserAuthority(&user)

	h.GetDB().Preload("Project.Tenant").First(&rel, rel.ID)
	h.SetAuditData(c, "修改", "项目成员", fmt.Sprintf("项目[%v]/成员[%v]", rel.Project.ProjectName, user.Username))
	h.SetExtraAuditData(c, models.ResProject, rel.ProjectID)
	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Update).
		ResourceType(msgbus.Project).
		ResourceID(rel.ProjectID).
		Content(fmt.Sprintf("将租户%s/项目%s中的用户%s设置为了%s", rel.Project.Tenant.TenantName, rel.Project.ProjectName, user.Username, rel.Role)).
		SetUsersToSend(
			[]uint{rel.UserID}, // 自己
			func() []uint {
				if rel.Role == models.ProjectRoleAdmin {
					return h.GetDataBase().ProjectAdmins(rel.ProjectID)
				}
				return nil
			}(), // 项目管理员
		).
		AffectedUsers([]uint{rel.UserID}).
		Send()
	handlers.OK(c, rel)
}

// DeleteProjectUser 删除 User 和 Project 的关系
// @Tags Project
// @Summary 删除 User 和 Project 的关系
// @Description 删除 User 和 Project 的关系
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "models.User"
// @Router /v1/project/{project_id}/user/{user_id} [delete]
// @Security JWT
func (h *ProjectHandler) DeleteProjectUser(c *gin.Context) {
	var (
		rel    models.ProjectUserRels
		envrel models.EnvironmentUserRels
	)
	if err := h.GetDB().Model(&rel).First(&rel, "project_id =? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	h.GetDB().Preload("Project.Tenant").First(&rel, rel.ID)
	if err := h.GetDB().Delete(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	// 从项目中删除用户同时也要从项目下各个环境中删除
	envids := []uint{}
	if err := h.GetDB().Model(&models.Environment{}).Where("project_id = ?", c.Param(PrimaryKeyName)).Pluck("id", &envids).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Delete(&envrel, "environment_id in (?) and user_id = ?", envids, c.Param("user_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	user := models.User{}
	h.GetDB().Preload("SystemRole").First(&user, c.Param("user_id"))
	h.GetCacheLayer().FlushUserAuthority(&user)

	h.SetAuditData(c, "删除", "项目成员", fmt.Sprintf("项目[%v]/成员[%v]", rel.Project.ProjectName, user.Username))
	h.SetExtraAuditData(c, models.ResProject, rel.ProjectID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Delete).
		ResourceType(msgbus.Project).
		ResourceID(rel.ProjectID).
		Content(fmt.Sprintf("删除了租户%s/项目%s中的用户%s", rel.Project.Tenant.TenantName, rel.Project.ProjectName, user.Username)).
		SetUsersToSend(
			[]uint{rel.UserID}, // 自己
			func() []uint {
				if rel.Role == models.ProjectRoleAdmin {
					return h.GetDataBase().ProjectAdmins(rel.ProjectID)
				}
				return nil
			}(), // 项目管理员
		).
		AffectedUsers([]uint{rel.UserID}).
		Send()
	handlers.NoContent(c, nil)
}

// ListProjectEnvironment 获取属于Project的 Environment 列表
// @Tags Project
// @Summary 获取属于 Project 的 Environment 列表
// @Description 获取属于 Project 的 Environment 列表
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param preload query string false "choices Creator,Cluster,Project,Applications,Users"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (EnvironmentName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Environment}} "models.Environment"
// @Router /v1/project/{project_id}/environment [get]
// @Security JWT
func (h *ProjectHandler) ListProjectEnvironment(c *gin.Context) {
	var list []models.Environment
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:                  "Environment",
		SearchFields:           []string{"EnvironmentName"},
		PreloadFields:          []string{"Creator", "Cluster", "Project", "Applications", "Users"},
		PreloadSensitiveFields: PreloadSensitiveFields,
		Where:                  []*handlers.QArgs{handlers.Args("project_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// RetrieveProjectEnvironment 获取Project 的一个 Environment详情
// @Tags Project
// @Summary 获取Project 的一个 Environment详情
// @Description 获取Project 的一个 Environment详情
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param environment_id path uint true "environment_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Environment} "models.Environment"
// @Router /v1/project/{project_id}/environment/{environment_id} [get]
// @Security JWT
func (h *ProjectHandler) RetrieveProjectEnvironment(c *gin.Context) {
	var env models.Environment
	if err := h.GetDB().First(&env, "project_id = ? and id = ?", c.Param(PrimaryKeyName), c.Param("environment_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, env)
}

// ListProjectRegistry 获取属于Project的 Registry 列表
// @Tags Project
// @Summary 获取属于 Project 的 Registry 列表
// @Description 获取属于 Project 的 Registry 列表
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param preload query string false "choices Creator,Project"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (RegistryName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Registry}} "models.Registry"
// @Router /v1/project/{project_id}/registry [get]
// @Security JWT
func (h *ProjectHandler) ListProjectRegistry(c *gin.Context) {
	var list []models.Registry
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "Registry",
		SearchFields:  []string{"RegistryName"},
		PreloadFields: []string{"Creator", "Project"},
		Where:         []*handlers.QArgs{handlers.Args("project_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// PostProjectRegistry 创建一个属于 Project 的Registry
// @Tags Project
// @Summary 创建一个属于 Project 的Registry
// @Description 创建一个属于 Project 的Registry
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param param body models.Registry true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Registry} "models.Registry"
// @Router /v1/project/{project_id}/registry [post]
// @Security JWT
func (h *ProjectHandler) PostProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := c.BindJSON(&registry); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if strconv.Itoa(int(registry.ProjectID)) != c.Param("project_id") {
		handlers.NotOK(c, fmt.Errorf("项目id不一致"))
		return
	}

	// 检查默认仓库
	defaultRegistries := []models.Registry{}
	if err := h.GetDB().Where("project_id = ? and id != ? and is_default = ?", registry.ProjectID, registry.ID, true).
		Find(&defaultRegistries).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if len(defaultRegistries) > 0 && registry.IsDefault {
		handlers.NotOK(c, fmt.Errorf("默认仓库只能有一个"))
		return
	}

	if err := h.GetDB().Create(&registry).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "创建", "镜像仓库", registry.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.Created(c, registry)
}

// RetrieveProjectRegistry 获取Project 的一个 Registry详情
// @Tags Project
// @Summary 获取Project 的一个 Registry详情
// @Description 获取Project 的一个 Registry详情
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param registry_id path uint true "registry_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Registry} "models.Registry"
// @Router /v1/project/{project_id}/registry/{registry_id} [get]
// @Security JWT
func (h *ProjectHandler) RetrieveProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := h.GetDB().First(&registry, "project_id = ? and id = ?", c.Param(PrimaryKeyName), c.Param("registry_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, registry)
}

// PutProjectRegistry 修改 Project 的 Registry
// @Tags Project
// @Summary  修改Project 的 Registry
// @Description  修改 Project 的 Registry
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param registry_id path uint true "registry_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Registry} "models.Registry"
// @Router /v1/project/{project_id}/registry/{registry_id} [put]
// @Security JWT
func (h *ProjectHandler) PutProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := h.GetDB().First(&registry, "project_id = ? and id = ?", c.Param(PrimaryKeyName), c.Param("registry_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := c.Bind(&registry); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if strconv.Itoa(int(registry.ProjectID)) != c.Param(PrimaryKeyName) || strconv.Itoa(int(registry.ID)) != c.Param("registry_id") {
		handlers.NotOK(c, fmt.Errorf("请求体参数和URL参数不匹配"))
		return
	}

	// 检查默认仓库
	defaultRegistries := []models.Registry{}
	if err := h.GetDB().Where("project_id = ? and id != ? and is_default = ?", registry.ProjectID, registry.ID, true).
		Find(&defaultRegistries).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if len(defaultRegistries) > 0 && registry.IsDefault {
		handlers.NotOK(c, fmt.Errorf("默认仓库只能有一个"))
		return
	}
	if err := h.GetDB().Save(&registry).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "镜像仓库", registry.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.OK(c, registry)
}

// SetDefaultProjectRegistry 设置 Project 的 默认 Registry
// @Tags Project
// @Summary  设置 Project 的 默认 Registry
// @Description  设置 Project 的 默认 Registry
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param registry_id path uint true "registry_id"
// @Param is_default query bool true "是否默认镜像仓库"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Registry} "models.Registry"
// @Router /v1/project/{project_id}/registry/{registry_id} [patch]
// @Security JWT
func (h *ProjectHandler) SetDefaultProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := h.GetDB().First(&registry, "project_id = ? and id = ?", c.Param(PrimaryKeyName), c.Param("registry_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	isDefault, _ := strconv.ParseBool(c.Query("is_default"))
	registry.IsDefault = isDefault

	if isDefault {
		h.SetAuditData(c, "设置默认", "镜像仓库", registry.RegistryName)
	} else {
		h.SetAuditData(c, "取消默认", "镜像仓库", registry.RegistryName)
	}
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	// 检查默认仓库
	defaultRegistries := []models.Registry{}
	if err := h.GetDB().Where("project_id = ? and id != ? and is_default = ?", registry.ProjectID, registry.ID, true).
		Find(&defaultRegistries).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if len(defaultRegistries) > 0 && registry.IsDefault {
		handlers.NotOK(c, fmt.Errorf("默认仓库只能有一个"))
		return
	}
	if err := h.GetDB().Save(&registry).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.OK(c, registry)
}

// DeleteProjectRegistry Project 的 Registry
// @Tags Project
// @Summary 删除 Project 的 Registry
// @Description 删除 Project 的 Registry
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param registry_id path uint true "registry_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Registry} "models.Registry"
// @Router /v1/project/{project_id}/registry/{registry_id} [delete]
// @Security JWT
func (h *ProjectHandler) DeleteProjectRegistry(c *gin.Context) {
	var registry models.Registry
	h.GetDB().First(&registry, c.Param("registry_id"))
	if err := h.GetDB().Delete(&registry, "project_id = ? and id = ?", c.Param(PrimaryKeyName), c.Param("registry_id")).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("删除仓库错误 %v", err))
		return
	}

	h.SetAuditData(c, "删除", "镜像仓库", registry.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.NoContent(c, nil)
}

// GetProjectResource 获取项目资源清单
// @Tags ResourceList
// @Summary 获取项目资源清单
// @Description 获取项目资源清单
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param date query string false "date"
// @Success 200 {object} handlers.ResponseStruct{Data=[]models.EnvironmentResource} "EnvironmentResource"
// @Router /v1/resources/project/{project_id} [get]
// @Security JWT
func (h *ProjectHandler) GetProjectResource(c *gin.Context) {
	dateTime, err := time.Parse(time.RFC3339, c.Query("date"))
	if err != nil {
		// 默认取到昨天的时间
		dateTime = time.Now().Add(-24 * time.Hour)
	}
	// 第二天的0点
	dayTime := utils.NextDayStartTime(dateTime)

	proj := models.Project{}
	if err := h.GetDB().Preload("Tenant").Where("id = ?", c.Param("project_id")).First(&proj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	envs := []models.Environment{}
	if err := h.GetDB().Where("project_id = ?", proj.ID).Find(&envs).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	var list []models.EnvironmentResource
	for _, env := range envs {
		var envREs models.EnvironmentResource
		if err := h.GetDB().
			Where("tenant_name = ? and project_name = ? and environment_name = ? and created_at >= ? and created_at < ?", proj.Tenant.TenantName, proj.ProjectName, env.EnvironmentName, dayTime.Format(time.RFC3339), dayTime.Add(24*time.Hour).Format(time.RFC3339)).
			Order("created_at").
			First(&envREs).Error; err != nil {
		} else {
			list = append(list, envREs)
		}
	}

	ret := models.EnvironmentResource{
		TenantName:      proj.Tenant.TenantName,
		ProjectName:     proj.ProjectName,
		EnvironmentName: "all",
	}
	if len(list) == 0 {
		handlers.OK(c, ret)
		return
	}

	ret.CreatedAt = list[0].CreatedAt
	for _, v := range list {
		ret.MaxCPUUsageCore += v.MaxCPUUsageCore
		ret.MaxMemoryUsageByte += v.MaxMemoryUsageByte
		ret.MinCPUUsageCore += v.MinCPUUsageCore
		ret.MinMemoryUsageByte += v.MinMemoryUsageByte
		ret.AvgCPUUsageCore += v.AvgCPUUsageCore
		ret.AvgMemoryUsageByte += v.AvgMemoryUsageByte
		ret.NetworkReceiveByte += v.NetworkReceiveByte
		ret.NetworkSendByte += v.NetworkSendByte
		ret.MaxPVCUsageByte += v.MaxPVCUsageByte
		ret.MinPVCUsageByte += v.MinPVCUsageByte
		ret.AvgPVCUsageByte += v.AvgPVCUsageByte
	}
	handlers.OK(c, ret)
}
