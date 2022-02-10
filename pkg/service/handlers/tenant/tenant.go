package tenanthandler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	ext_v1beta1 "k8s.io/api/extensions/v1beta1"
	"kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/kubeclient"
	gemlabels "kubegems.io/pkg/labels"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/msgbus"
)

var (
	SearchFields   = []string{"tenant_name"}
	FilterFields   = []string{"tenant_name"}
	PreloadFields  = []string{"ResourceQuotas", "Users", "Projects"}
	OrderFields    = []string{"tenant_name", "ID"}
	ModelName      = "Tenant"
	PrimaryKeyName = "tenant_id"
)

// ListTenant 列表 Tenant
// @Tags Tenant
// @Summary Tenant列表
// @Description Tenant列表
// @Accept json
// @Produce json
// @Param TenantName query string false "TenantName"
// @Param preload query string false "choices ResourceQuotas,Users,Projects"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (TenantName,Remark)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Tenant}} "Tenant"
// @Router /v1/tenant [get]
// @Security JWT
func (h *TenantHandler) ListTenant(c *gin.Context) {
	var list []models.Tenant
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         ModelName,
		SearchFields:  SearchFields,
		SortFields:    OrderFields,
		PreloadFields: PreloadFields,
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveTenant Tenant详情
// @Tags Tenant
// @Summary Tenant详情
// @Description get Tenant详情
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Tenant} "Tenant"
// @Router /v1/tenant/{tenant_id} [get]
// @Security JWT
func (h *TenantHandler) RetrieveTenant(c *gin.Context) {
	var obj models.Tenant
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// PostTenant 创建Tenant
// @Tags Tenant
// @Summary 创建Tenant
// @Description 创建Tenant
// @Accept json
// @Produce json
// @Param param body models.Tenant true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Tenant} "Tenant"
// @Router /v1/tenant [post]
// @Security JWT
func (h *TenantHandler) PostTenant(c *gin.Context) {
	var obj models.Tenant
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	// 默认租户是启用的
	obj.IsActive = true
	if err := h.GetDB().Create(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.GetCacheLayer().GetGlobalResourceTree().UpsertTenant(obj.ID, obj.TenantName)

	h.SetAuditData(c, "创建", "租户", obj.TenantName)
	h.SetExtraAuditData(c, models.ResTenant, obj.ID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ResourceType(msgbus.Tenant).
		ActionType(msgbus.Add).
		ResourceID(obj.ID).
		Content(fmt.Sprintf("创建了租户%s", obj.TenantName)).
		SetUsersToSend(
			h.GetDataBase().SystemAdmins(),
		).
		Send()
	handlers.Created(c, obj)
}

// PutTenant 修改Tenant
// @Tags Tenant
// @Summary 修改Tenant
// @Description 修改Tenant
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param param body models.Tenant true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Tenant} "Tenant"
// @Router /v1/tenant/{tenant_id} [put]
// @Security JWT
func (h *TenantHandler) PutTenant(c *gin.Context) {
	var obj models.Tenant
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "租户", obj.TenantName)
	h.SetExtraAuditData(c, models.ResTenant, obj.ID)

	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if strconv.Itoa(int(obj.ID)) != c.Param(PrimaryKeyName) {
		handlers.NotOK(c, fmt.Errorf("数据ID错误"))
		return
	}
	if err := h.GetDB().Save(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.GetCacheLayer().GetGlobalResourceTree().UpsertTenant(obj.ID, obj.TenantName)
	handlers.OK(c, obj)
}

// DeleteTenant 删除 Tenant
// @Tags Tenant
// @Summary 删除 Tenant
// @Description 删除 Tenant
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Success 204 {object} handlers.ResponseStruct "resp"
// @Router /v1/tenant/{tenant_id} [delete]
// @Security JWT
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	var obj models.Tenant
	// 这儿是删除数据，不存在preload 敏感数据的情况
	if err := h.GetDB().Preload("ResourceQuotas.Cluster").First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, nil)
		return
	}

	if err := h.GetDB().Delete(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "租户", obj.TenantName)
	h.SetExtraAuditData(c, models.ResTenant, obj.ID)

	h.GetCacheLayer().GetGlobalResourceTree().DelTenant(obj.ID)

	handlers.NoContent(c, nil)
}

// ListTenantUser 获取属于Tenant的 User 列表
// @Tags Tenant
// @Summary 获取属于 Tenant 的 User 列表
// @Description 获取属于 Tenant 的 User 列表
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param preload query string false "choices Tenants,SystemRole"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (Username,Email)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}} "models.User"
// @Router /v1/tenant/{tenant_id}/user [get]
// @Security JWT
func (h *TenantHandler) ListTenantUser(c *gin.Context) {
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
		Select:        handlers.Args("users.*, tenant_user_rels.role"),
		Join:          handlers.Args("join tenant_user_rels on tenant_user_rels.user_id = users.id"),
		Where:         []*handlers.QArgs{handlers.Args("tenant_user_rels.tenant_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveTenantUser 获取Tenant 的一个 User详情
// @Tags Tenant
// @Summary 获取Tenant 的一个 User详情
// @Description 获取Tenant 的一个 User详情
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "models.User"
// @Router /v1/tenant/{tenant_id}/user/{user_id} [get]
// @Security JWT
func (h *TenantHandler) RetrieveTenantUser(c *gin.Context) {
	var obj models.User
	if err := h.GetDB().Table(
		"users",
	).Joins(
		"join  tenant_user_rels on tenant_user_rels.user_id = users.id",
	).Where(
		"tenant_user_rels.tenant_id = ?",
		c.Param(PrimaryKeyName),
	).First(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// PostTenantUser 在User和Tenant间添加关联关系
// @Tags Tenant
// @Summary 在User和Tenant间添加关联关系
// @Description 在User和Tenant间添加关联关系
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param param body models.TenantUserRels  true "表单"`
// @Success 200 {object} handlers.ResponseStruct{Data=models.TenantUserRels} "models.User"
// @Router /v1/tenant/{tenant_id}/user [post]
// @Security JWT
func (h *TenantHandler) PostTenantUser(c *gin.Context) {
	var (
		tenant models.Tenant
		rel    models.TenantUserRels
		user   models.User
	)
	if err := h.GetDB().Preload("ResourceQuotas").First(&tenant, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, nil)
		return
	}
	if err := c.BindJSON(&rel); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if c.Param(PrimaryKeyName) != strconv.Itoa(int(rel.TenantID)) {
		handlers.NotOK(c, fmt.Errorf("数据ID不匹配"))
		return
	}
	if err := h.GetDB().Preload("SystemRole").First(&user, rel.UserID).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("用户错误"))
		return
	}
	if err := h.GetDB().Save(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.GetCacheLayer().FlushUserAuthority(&user)
	h.SetAuditData(c, "添加", "租户成员", fmt.Sprintf("租户[%v]/用户[%v]", tenant.TenantName, user.Username))
	h.SetExtraAuditData(c, models.ResTenant, tenant.ID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Add).
		ResourceType(msgbus.Tenant).
		ResourceID(rel.TenantID).
		Content(fmt.Sprintf("向租户%s中添加了用户%s", tenant.TenantName, user.Username)).
		SetUsersToSend(
			[]uint{rel.UserID}, // 自己
		).
		AffectedUsers([]uint{rel.UserID}).
		Send()
	handlers.OK(c, rel)
}

// PutTenantUser 修改 User 和 Tenant 的关联关系
// @Tags Tenant
// @Summary  修改 User 和 Tenant 的关联关系
// @Description  修改 User 和 Tenant 的关联关系
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param user_id path uint true "user_id"
// @Param param body models.TenantUserRels  true "表单"`
// @Success 200 {object} handlers.ResponseStruct{Data=models.TenantUserRels} "models.User"
// @Router /v1/tenant/{tenant_id}/user/{user_id} [put]
// @Security JWT
func (h *TenantHandler) PutTenantUser(c *gin.Context) {
	var (
		rel  models.TenantUserRels
		user models.User
	)
	if err := h.GetDB().Preload("Tenant").First(&rel, "user_id = ? and tenant_id = ?", c.Param("user_id"), c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("不可以修改不存在得\"用户-租户\"关系"))
		return
	}
	if err := c.BindJSON(&rel); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if strconv.Itoa(int(rel.UserID)) != c.Param("user_id") || strconv.Itoa(int(rel.TenantID)) != c.Param(PrimaryKeyName) {
		handlers.NotOK(c, fmt.Errorf("请求体参数和url参数不匹配"))
		return
	}
	if err := h.GetDB().Save(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.GetDB().Preload("SystemRole").First(&user, rel.UserID)
	h.GetCacheLayer().FlushUserAuthority(&user)

	h.GetDB().Preload("Tenant").First(&rel, rel.ID)

	h.SetAuditData(c, "修改", "租户成员", fmt.Sprintf("租户[%v]/用户[%v]", rel.Tenant.TenantName, user.Username))
	h.SetExtraAuditData(c, models.ResTenant, rel.TenantID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Update).
		ResourceType(msgbus.Tenant).
		ResourceID(rel.TenantID).
		Content(fmt.Sprintf("将租户%s中的用户%s设置为了%s", rel.Tenant.TenantName, user.Username, rel.Role)).
		SetUsersToSend(
			[]uint{rel.UserID}, // 自己
		).
		AffectedUsers([]uint{rel.UserID}).
		Send()
	handlers.OK(c, rel)
}

// DeleteTenantUser 删除 User 和 Tenant 的关系
// @Tags Tenant
// @Summary 删除 User 和 Tenant 的关系
// @Description 删除 User 和 Tenant 的关系
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "models.User"
// @Router /v1/tenant/{tenant_id}/user/{user_id} [delete]
// @Security JWT
func (h *TenantHandler) DeleteTenantUser(c *gin.Context) {
	var (
		obj     models.Tenant
		subobj  models.User
		rel     models.TenantUserRels
		projrel models.ProjectUserRels
		envrel  models.EnvironmentUserRels
		user    models.User
		projids []uint
		envids  []uint
	)
	tenantid := c.Param(PrimaryKeyName)
	userid := c.Param("user_id")
	if err := h.GetDB().Preload("Projects.Environments").First(&obj, tenantid).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().First(&subobj, userid).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.GetDB().First(&rel, "tenant_id = ? and user_id = ?", tenantid, userid)
	h.GetDB().Delete(&rel)
	for _, proj := range obj.Projects {
		projids = append(projids, proj.ID)
		for _, env := range proj.Environments {
			envids = append(envids, env.ID)
		}
	}
	if err := h.GetDB().Delete(&projrel, "project_id in (?) and user_id = ?", projids, userid).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Delete(&envrel, "environment_id in (?) and user_id = ?", envids, userid).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.GetDB().Preload("SystemRole").First(&user, rel.UserID)
	h.GetCacheLayer().FlushUserAuthority(&user)

	h.SetAuditData(c, "删除", "租户成员", fmt.Sprintf("租户[%v]/用户[%v]", obj.TenantName, user.Username))
	h.SetExtraAuditData(c, models.ResTenant, rel.TenantID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Delete).
		ResourceType(msgbus.Tenant).
		ResourceID(rel.TenantID).
		Content(fmt.Sprintf("删除了租户%s中的用户%s", obj.TenantName, user.Username)).
		SetUsersToSend(
			[]uint{rel.UserID}, // 自己
		).
		AffectedUsers([]uint{rel.UserID}).
		Send()
	handlers.NoContent(c, nil)
}

// ListTenantProject 获取属于Tenant的 Project 列表
// @Tags Tenant
// @Summary 获取属于 Tenant 的 Project 列表
// @Description 获取属于 Tenant 的 Project 列表
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param preload query string false "choices Applications,Environments,Registries,Users,Tenant"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (ProjectName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Project}} "models.Project"
// @Router /v1/tenant/{tenant_id}/project [get]
// @Security JWT
func (h *TenantHandler) ListTenantProject(c *gin.Context) {
	var list []models.Project
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	tid := c.Param(PrimaryKeyName)
	user, _ := h.GetContextUser(c)
	userAuthority := h.GetCacheLayer().GetUserAuthority(user)

	cond := &handlers.PageQueryCond{
		Model:         "Project",
		SearchFields:  []string{"project_name"},
		PreloadFields: []string{"Applications", "Environments", "Registries", "Tenant", "Users"},
		Where:         []*handlers.QArgs{handlers.Args("tenant_id = ?", tid)},
	}

	if !userAuthority.IsTenantAdmin(utils.ToUint(tid)) && !userAuthority.IsSystemAdmin {
		cond.Join = handlers.Args("left join project_user_rels on project_user_rels.project_id = projects.id")
		cond.Where = append(cond.Where, handlers.Args("project_user_rels.user_id = ?", user.ID))
	}

	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveTenantProject 获取Tenant 的一个 Project详情
// @Tags Tenant
// @Summary 获取Tenant 的一个 Project详情
// @Description 获取Tenant 的一个 Project详情
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param project_id path uint true "project_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Project} "models.Project"
// @Router /v1/tenant/{tenant_id}/project/{project_id} [get]
// @Security JWT
func (h *TenantHandler) RetrieveTenantProject(c *gin.Context) {
	var project models.Project
	if err := h.GetDB().Where("tenant_id = ? and id = ?", c.Param(project.ProjectAlias), c.Param("project_id")).First(&project).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, project)
}

// PostTenantProject 创建一个属于 Tenant 的Project
// @Tags Tenant
// @Summary 创建一个属于 Tenant 的Project
// @Description 创建一个属于 Tenant 的Project
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param param body models.Project true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Project} "models.Project"
// @Router /v1/tenant/{tenant_id}/project [post]
// @Security JWT
func (h *TenantHandler) PostTenantProject(c *gin.Context) {
	var (
		tenant  models.Tenant
		project models.Project
	)
	if err := h.GetDB().First(&tenant, c.Param("tenant_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := c.BindJSON(&project); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if project.TenantID != tenant.ID {
		project.TenantID = tenant.ID
	}
	if err := h.GetDB().Create(&project).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	t := h.GetCacheLayer().GetGlobalResourceTree()

	_ = t.UpsertProject(tenant.ID, project.ID, project.ProjectName)

	h.SetAuditData(c, "创建", "项目", project.ProjectName)
	h.SetExtraAuditData(c, models.ResProject, project.ID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Add).
		ResourceType(msgbus.Project).
		ResourceID(project.ID).
		Content(fmt.Sprintf("在租户%s中创建了项目%s", tenant.TenantName, project.ProjectName)).
		SetUsersToSend(
			h.GetDataBase().TenantAdmins(tenant.ID),
		).
		Send()
	handlers.OK(c, project)
}

// ListTenantTenantResourceQuota 获取属于Tenant的 TenantResourceQuota 列表
// @Tags Tenant
// @Summary 获取属于 Tenant 的 TenantResourceQuota 列表
// @Description 获取属于 Tenant 的 TenantResourceQuota 列表
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param preload query string false "choices Tenant,Cluster"
// @Param page query int false "page"
// @Param size query int false "page"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.TenantResourceQuota}} "models.TenantResourceQuota"
// @Router /v1/tenant/{tenant_id}/tenantresourcequota [get]
// @Security JWT
func (h *TenantHandler) ListTenantTenantResourceQuota(c *gin.Context) {
	var list []models.TenantResourceQuota
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:                  "TenantResourceQuota",
		PreloadFields:          []string{"Cluster", "Tenant", "TenantResourceQuotaApply"},
		PreloadSensitiveFields: map[string]string{"Cluster": "id, cluster_name"},
		Where:                  []*handlers.QArgs{handlers.Args("tenant_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveTenantTenantResourceQuota 获取Tenant 的一个 TenantResourceQuota详情
// @Tags Tenant
// @Summary 获取Tenant 的一个 TenantResourceQuota详情
// @Description 获取Tenant 的一个 TenantResourceQuota详情
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param tenantresourcequota_id path uint true "tenantresourcequota_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.TenantResourceQuota} "models.TenantResourceQuota"
// @Router /v1/tenant/{tenant_id}/tenantresourcequota/{tenantresourcequota_id} [get]
// @Security JWT
func (h *TenantHandler) RetrieveTenantTenantResourceQuota(c *gin.Context) {
	var trq models.TenantResourceQuota
	if err := h.GetDB().First(&trq, "tenant_id = ? and id = ?", c.Param(PrimaryKeyName), c.Param("tenantresourcequota_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, trq)
}

// EnableTenant 激活租户
// @Tags Tenant
// @Summary 激活租户,当租户为未激活状态的时候才可用
// @Description 激活租户,当租户为未激活状态的时候才可用
// @Accept json
// @Produce json
// @Param tenantid path int true "tenantid"
// @Success 200 {object} handlers.ResponseStruct{Data=[]models.Tenant} "Tenant"
// @Router /v1/tenant/{tenant_id}/action/enable [post]
// @Security JWT
func (h *TenantHandler) EnableTenant(c *gin.Context) {
	var obj models.Tenant
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	obj.IsActive = true
	if err := h.GetDB().Save(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "启用", "租户", obj.TenantName)
	h.SetExtraAuditData(c, models.ResTenant, obj.ID)

	// 所有租户成员
	tenantUsers := h.GetDataBase().TenantUsers(obj.ID)
	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Update).
		ResourceType(msgbus.Tenant).
		ResourceID(obj.ID).
		Content(fmt.Sprintf("激活了租户%s", obj.TenantName)).
		SetUsersToSend(
			h.GetDataBase().SystemAdmins(),
			tenantUsers,
		).
		AffectedUsers(
			tenantUsers,
		).
		Send()
	handlers.OK(c, obj)
}

// DisableTenant 取消激活租户
// @Tags Tenant
// @Summary 取消激活租户,当租户为激活状态的时候才可用
// @Description 取消激活租户,当租户为激活状态的时候才可用
// @Accept json
// @Produce json
// @Param tenantid path int true "tenant_id"
// @Success 200 {object} handlers.ResponseStruct{Data=[]models.Tenant} "Tenant"
// @Router /v1/tenant/{tenant_id}/action/disable [post]
// @Security JWT
func (h *TenantHandler) DisableTenant(c *gin.Context) {
	var obj models.Tenant
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	obj.IsActive = false
	if err := h.GetDB().Save(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "禁用", "租户", obj.TenantName)
	h.SetExtraAuditData(c, models.ResTenant, obj.ID)

	// 所有租户成员
	tenantUsers := h.GetDataBase().TenantUsers(obj.ID)
	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Update).
		ResourceType(msgbus.Tenant).
		ResourceID(obj.ID).
		Content(fmt.Sprintf("禁用了租户%s", obj.TenantName)).
		SetUsersToSend(
			h.GetDataBase().SystemAdmins(),
			tenantUsers,
		).
		AffectedUsers(
			tenantUsers,
		).
		Send()
	handlers.OK(c, obj)
}

// PostTenantTenantResourceQuota 创建一个属于 Tenant 的TenantResourceQuota
// @Tags Tenant
// @Summary 创建一个属于 Tenant 的TenantResourceQuota
// @Description 创建一个属于 Tenant 的TenantResourceQuota
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param param body models.TenantResourceQuota true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.TenantResourceQuota} "models.TenantResourceQuota"
// @Router /v1/tenant/{tenant_id}/tenantresourcequota [post]
// @Security JWT
func (h *TenantHandler) PostTenantTenantResourceQuota(c *gin.Context) {
	var obj models.TenantResourceQuota
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if c.Param(PrimaryKeyName) != strconv.Itoa(int(obj.TenantID)) {
		handlers.NotOK(c, fmt.Errorf("请求体的租户ID和路由中的租户ID不匹配"))
		return
	}
	if err := h.GetDB().Create(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.GetDB().Preload("Tenant").Preload("Cluster", func(tx *gorm.DB) *gorm.DB { return tx.Select("id, cluster_name") }).First(&obj, obj.ID)

	h.SetAuditData(c, "创建", "租户集群资源限制", fmt.Sprintf("租户[%v]/集群[%v]", obj.Tenant.TenantName, obj.Cluster.ClusterName))
	h.SetExtraAuditData(c, models.ResTenant, obj.TenantID)

	handlers.Created(c, obj)
}

// PatchTenantTenantResourceQuota 修改一个属于 Tenant 的TenantResourceQuota
// @Tags Tenant
// @Summary 修改一个属于 Tenant 的TenantResourceQuota
// @Description 修改一个属于 Tenant 的TenantResourceQuota
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param cluster_id path uint true "cluster_id"
// @Param param body models.TenantResourceQuota true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.TenantResourceQuota} "models.TenantResourceQuota"
// @Router /v1/tenant/{tenant_id}/tenantresourcequota/{:cluster_id} [put]
// @Security JWT
func (h *TenantHandler) PutTenantTenantResourceQuota(c *gin.Context) {
	var trq models.TenantResourceQuota
	if err := h.GetDB().Preload(
		"Cluster",
		func(tx *gorm.DB) *gorm.DB { return tx.Select("id, cluster_name, oversold_config") },
	).Preload("Tenant").First(&trq, "tenant_id = ? and cluster_id = ?", c.Param(PrimaryKeyName), c.Param("cluster_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	cluster := trq.Cluster
	tenant := trq.Tenant
	need := v1.ResourceList{}
	origin := v1.ResourceList{}
	json.Unmarshal(trq.Content, &origin)
	oversold := trq.Cluster.OversoldConfig
	clustername := trq.Cluster.ClusterName
	trq.Cluster = nil
	trq.Tenant = nil

	if err := c.ShouldBind(&trq); err != nil {
		handlers.NotOK(c, err)
		return
	}
	json.Unmarshal(trq.Content, &need)
	if err := models.ValidateTenantResourceQuota(oversold, clustername, origin, need); err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "集群租户资源限制", fmt.Sprintf("集群[%v]/租户[%v]", cluster.ClusterName, tenant.TenantName))
	h.SetExtraAuditData(c, models.ResTenant, trq.TenantID)

	if err := h.GetDB().Save(&trq).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, trq)
}

// PatchTenantTenantResourceQuota 删除租户在一个集群下的资源
// @Tags Tenant
// @Summary 删除租户在一个集群下的资源
// @Description 删除租户在一个集群下的资源
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param cluster_id path uint true "cluster_id"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "obj"
// @Router /v1/tenant/{tenant_id}/tenantresourcequota/{:cluster_id} [put]
// @Security JWT
func (h *TenantHandler) DeleteTenantResourceQuota(c *gin.Context) {
	var trq models.TenantResourceQuota
	if err := h.GetDB().Preload("Tenant").Preload(
		"Cluster",
		func(tx *gorm.DB) *gorm.DB { return tx.Select("id, cluster_name") },
	).First(&trq, "tenant_id = ? and cluster_id = ?", c.Param(PrimaryKeyName), c.Param("cluster_id")).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	if err := h.GetDB().Delete(&trq).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "删除", "集群租户资源限制", fmt.Sprintf("集群[%v]/租户[%v]", trq.Cluster.ClusterName, trq.Tenant.TenantName))
	h.SetExtraAuditData(c, models.ResTenant, trq.TenantID)

	handlers.NoContent(c, nil)
}

// TenantEnvironments 获取租户下所有的环境
// @Tags Tenant
// @Summary 获取租户下所有的环境以及资源状态
// @Description 获取租户下所有的环境以及资源状态
// @Accept json
// @Produce json
// @Param tenant_id path int true "tenant_id"
// @Param search query string false "search in (EnvironmentName)"
// @Success 200 {object} handlers.ResponseStruct{Data=[]object} "object"
// @Router /v1/tenant/{tenant_id}/environment_with_quotas [get]
// @Security JWT
func (h *TenantHandler) TenantEnvironments(c *gin.Context) {
	var (
		tenant models.Tenant
		envs   []models.Environment
	)
	tenantid := c.Param(PrimaryKeyName)
	if err := h.GetDB().Preload("Projects").First(&tenant, "id = ?", tenantid).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	projectids := []uint{}
	for _, proj := range tenant.Projects {
		projectids = append(projectids, proj.ID)
	}

	search := c.Query("search")
	q := h.GetDB().Preload("Project").Preload(
		"Creator",
		func(tx *gorm.DB) *gorm.DB { return tx.Select("id, username") },
	).Preload(
		"Cluster",
		func(tx *gorm.DB) *gorm.DB { return tx.Select("id, cluster_name") },
	)
	if search != "" {
		q.Where("environment_name = ?", search)
	}
	if err := q.Find(&envs, "project_id in (?)", projectids).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	labels := map[string]string{
		gemlabels.LabelTenant: tenant.TenantName,
	}
	clusterMap := map[string]bool{}
	for _, env := range envs {
		clusterMap[env.Cluster.ClusterName] = true
	}

	quotaMap := map[string]interface{}{}
	for cluster := range clusterMap {
		quotas, err := kubeclient.GetClient().GetResourceQuotaList(cluster, "", labels)
		if err != nil {
			continue
		}
		for _, quota := range *quotas {
			envname, exist := quota.Labels[gemlabels.LabelEnvironment]
			if !exist {
				continue
			}
			key := fmt.Sprintf("%s__%s__%s", cluster, quota.Namespace, envname)
			quotaMap[key] = quota
		}
	}
	ret := []interface{}{}
	for _, env := range envs {
		key := fmt.Sprintf("%s__%s__%s", env.Cluster.ClusterName, env.Namespace, env.EnvironmentName)
		quota, exist := quotaMap[key]
		if exist {
			ret = append(ret, map[string]interface{}{"environment": env, "quota": quota})
		} else {
			ret = append(ret, map[string]interface{}{"environment": env, "quota": nil})
		}
	}
	handlers.OK(c, ret)
}

// ListEnvironment 获取租户下的所有环境列表
// @Tags Tenant
// @Summary 获取租户下的所有Environment列表
// @Description 获取租户下的所有Environment列表
// @Accept json
// @Produce json
// @Param tenant_id path int true "tenant_id"
// @Param EnvironmentName query string false "EnvironmentName"
// @Param preload query string false "choices Creator,Cluster,Project,ResourceQuota,Applications,Users"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (EnvironmentName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Environment}} "Environment"
// @Router /v1/tenant/{tenant_id}/environment [get]
// @Security JWT
func (h *TenantHandler) ListEnvironment(c *gin.Context) {
	var list []models.Environment
	projectids := []uint{}
	if err := h.GetDB().Model(&models.Project{}).Where("tenant_id = ?", c.Param(PrimaryKeyName)).Pluck("id", &projectids).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:                  "Environment",
		SearchFields:           []string{"EnvironmentName"},
		PreloadFields:          []string{"Creator", "Cluster", "Project", "ResourceQuota", "Applications", "Users"},
		PreloadSensitiveFields: map[string]string{"Cluster": "id, cluster_name"},
		Where:                  []*handlers.QArgs{handlers.Args("project_id in (?)", projectids)},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

type tenantStatisticsData struct {
	Clusters     int   `json:"count.clusters"`
	Projects     int   `json:"count.projects"`
	Environments int64 `json:"count.environments"`
	Users        int64 `json:"count.users"`
	Applications int64 `json:"count.applications"`
	Deployments  int64 `json:"count.deployments"`
	StatefulSets int64 `json:"count.statefulsets"`
	DaemonSets   int64 `json:"count.daemonsets"`
	Pods         int64 `json:"count.pods"`
	Workloads    int64 `json:"count.workloads"`
}

// TenantStatistics 租户非资源类型统计
// @Tags Tenant
// @Summary 租户非资源类型统计
// @Description 租户非资源类型统计
// @Accept json
// @Produce json
// @Param tenant_id path int true "tenant_id"
// @Success 200 {object} handlers.ResponseStruct{Data=tenantStatisticsData} "statistics"
// @Router /v1/tenant/{tenant_id}/statistics [get]
// @Security JWT
func (h *TenantHandler) TenantStatistics(c *gin.Context) {
	// 集群数，项目数，环境数，用户数，应用数
	var (
		tenant            models.Tenant
		usercount         int64
		envcount          int64
		appcount          int64
		deploymentsCount  int64
		daemonsetsCount   int64
		statefulsetsCount int64
		podCount          int64
	)
	if err := h.GetDB().Preload("ResourceQuotas.Cluster").Preload("Projects").First(&tenant, "id = ?", c.Param("tenant_id")).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("租户id %v 对应的租户不存在", c.Param("tenant_id")))
		return
	}
	h.GetDB().Model(&models.TenantUserRels{}).Where("tenant_id = ?", c.Param("tenant_id")).Count(&usercount)
	pids := []uint{}
	for _, p := range tenant.Projects {
		pids = append(pids, p.ID)
	}
	h.GetDB().Model(&models.Application{}).Where("project_id in (?)", pids).Count(&appcount)
	h.GetDB().Model(&models.Environment{}).Where("project_id in (?)", pids).Count(&envcount)

	if len(tenant.ResourceQuotas) != 0 {
		wg := sync.WaitGroup{}
		allQuotas := make(chan *[]v1.ResourceQuota, len(tenant.ResourceQuotas))
		for _, trq := range tenant.ResourceQuotas {
			wg.Add(1)
			go func(clustername string) {
				quotas, err := kubeclient.GetClient().GetResourceQuotaList(clustername, "", map[string]string{
					gemlabels.LabelTenant: tenant.TenantName,
				})
				if err != nil {
					log.Error(err, "get resource quotas failed")
				}
				allQuotas <- quotas
				wg.Done()
			}(trq.Cluster.ClusterName)
		}
		wg.Wait()
		// 1.在遍历时，如果 channel 没有关闭，则回出现 deadlock 的错误。
		// 2.在遍历时，如果 channel 已经关闭，则会正常遍历数据，遍历完后，就会退出遍历。
		close(allQuotas)
		for quotas := range allQuotas {
			for _, quota := range *quotas {
				for k, v := range quota.Status.Used {
					if strings.HasPrefix(k.String(), "count/deployments") {
						c, _ := v.AsInt64()
						deploymentsCount += c
						continue
					}
					if strings.HasPrefix(k.String(), "count/statefulsets") {
						c, _ := v.AsInt64()
						statefulsetsCount += c
						continue
					}
					if strings.HasPrefix(k.String(), "count/daemonsets") {
						c, _ := v.AsInt64()
						daemonsetsCount += c
						continue
					}
					// count/pods: 所有pod数
					// pods: 非终止状态的pod数
					if strings.EqualFold(k.String(), "count/pods") {
						c, _ := v.AsInt64()
						podCount += c
						continue
					}
				}
			}
		}
	}

	ret := tenantStatisticsData{
		Clusters:     len(tenant.ResourceQuotas),
		Projects:     len(tenant.Projects),
		Environments: envcount,
		Users:        usercount,
		Applications: appcount,
		Deployments:  deploymentsCount,
		StatefulSets: statefulsetsCount,
		DaemonSets:   daemonsetsCount,
		Pods:         podCount,
		Workloads:    deploymentsCount + statefulsetsCount + daemonsetsCount,
	}
	handlers.OK(c, ret)
}

// @Tags NetworkIsolated
// @Summary 租户网络隔离开关
// @Description 租户网络隔离开关
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param param body handlers.ClusterIsolatedSwitch true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.IsolatedSwitch} "object"
// @Router /v1/tenant/{tenant_id}/action/networkisolate [post]
// @Security JWT
func (h *TenantHandler) TenantSwitch(c *gin.Context) {
	form := &handlers.ClusterIsolatedSwitch{}
	if err := c.BindJSON(form); err != nil {
		handlers.NotOK(c, err)
		return
	}
	var (
		tenant  models.Tenant
		cluster models.Cluster
	)
	if e := h.GetDB().First(&tenant, "id = ?", c.Param(PrimaryKeyName)).Error; e != nil {
		handlers.NotOK(c, e)
		return
	}
	if e := h.GetDB().First(&cluster, "id = ?", form.ClusterID).Error; e != nil {
		handlers.NotOK(c, e)
		return
	}

	h.SetAuditData(c, "更新", "租户网络隔离", tenant.TenantName)
	h.SetExtraAuditData(c, models.ResTenant, tenant.ID)

	tnetpol, err := kubeclient.GetClient().GetTenantNetworkPolicy(cluster.ClusterName, tenant.TenantName, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	tnetpol.Spec.TenantIsolated = form.Isolate
	ret, err := kubeclient.GetClient().PatchTenantNetworkPolicy(cluster.ClusterName, tenant.TenantName, tnetpol)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// CreateTenantTenantResourceQuotaApply  创建or修改租户集群资源变更申请
// @Tags Tenant
// @Summary 创建or修改租户集群资源变更申请
// @Description 创建or修改租户集群资源变更申请
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param cluster_id path uint true "cluster_id"
// @Param param body models.TenantResourceQuotaApply true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.TenantResourceQuotaApply} "models.TenantResourceQuotaApply"
// @Router /v1/tenant/{tenant_id}/cluster/{cluster_id}/resourceApply [post]
// @Security JWT
func (h *TenantHandler) CreateTenantResourceQuotaApply(c *gin.Context) {
	var (
		quota models.TenantResourceQuota
		req   models.TenantResourceQuotaApply
	)

	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}

	tenantID, _ := strconv.Atoi(c.Param(PrimaryKeyName))
	if err := h.GetDB().Preload("Tenant").Preload(
		"Cluster",
		func(tx *gorm.DB) *gorm.DB { return tx.Select("id, cluster_name, oversold_config") },
	).Preload("TenantResourceQuotaApply").First(&quota, "tenant_id = ? and cluster_id = ?", tenantID, c.Param("cluster_id")).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("租户在当前集群不存在可以使用资源"))
		return
	}

	u, _ := h.GetContextUser(c)
	if u == nil {
		u = &models.User{}
	}

	// 没有就新建，有就更新
	if quota.TenantResourceQuotaApplyID == nil {
		quota.TenantResourceQuotaApply = &models.TenantResourceQuotaApply{}
	}
	quota.TenantResourceQuotaApply.Status = models.QuotaStatusPending
	quota.TenantResourceQuotaApply.Content = req.Content
	quota.TenantResourceQuotaApply.Username = u.Username

	need := v1.ResourceList{}
	origin := v1.ResourceList{}
	json.Unmarshal(req.Content, &need)
	json.Unmarshal(quota.Content, &origin)
	if err := models.ValidateTenantResourceQuota(quota.Cluster.OversoldConfig, quota.Cluster.ClusterName, origin, need); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := h.GetDB().Save(quota.TenantResourceQuotaApply).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Save(&quota).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "创建", "集群租户资源限制", fmt.Sprintf("集群[%v]/租户[%v]", quota.Cluster.ClusterName, quota.Tenant.TenantName))
	h.SetExtraAuditData(c, models.ResTenant, quota.TenantID)

	// 申请消息给系统管理员和当前用户
	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Approve).
		ActionType(msgbus.Update).
		ResourceType(msgbus.TenantResourceQuota).
		ResourceID(quota.ID).
		Content(fmt.Sprintf("申请调整租户%s在集群%s的资源", quota.Tenant.TenantName, quota.Cluster.ClusterName)).
		SetUsersToSend(
			h.GetDataBase().SystemAdmins(),
			[]uint{u.ID},
		).
		Send()

	handlers.OK(c, req)
}

// GetTenantTenantResourceQuotaApply  获取租户集群资源变更申请详情
// @Tags Tenant
// @Summary 获取租户集群资源变更申请详情
// @Description 获取租户集群资源变更申请详情
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Param tenantresourcequotaapply_id path uint true "tenantresourcequotaapply_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.TenantResourceQuotaApply} "models.TenantResourceQuotaApply"
// @Router /v1/tenant/{tenant_id}/tenantresourcequotaapply/{tenantresourcequotaapply_id} [get]
// @Security JWT
func (h *TenantHandler) GetTenantTenantResourceQuotaApply(c *gin.Context) {
	var obj models.TenantResourceQuotaApply
	if err := h.GetDB().First(&obj, "id = ?", c.Param("tenantresourcequotaapply_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

type TenantGatewayForm struct {
	Cluster string `json:"cluster" binding:"required"`
	Tenant  string `json:"tenant" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Name    string `json:"name" binding:"required"`

	Replicas      int32                   `json:"replicas"`
	Resources     v1.ResourceRequirements `json:"resources,omitempty"`
	ConfigmapData map[string]string       `json:"configmap_data"`
	IngressClass  string                  `json:"ingress_class"`

	HttpPort  int32 `json:"http_port"`
	HttpsPort int32 `json:"https_port"`
	IsHealthy bool  `json:"is_healthy"`
}

type TenantGatewayDetail struct {
	TenantGatewayForm `json:"tenant_gateway"`
	Ingresses         []ext_v1beta1.Ingress `json:"ingresses"`
	Pods              []v1.Pod              `json:"pods"`
	Addresses         []string              `json:"addresses"`
}

const (
	defaultGatewayName   = "default-gateway"
	defaultGatewayTenant = "notenant"
)

// @Tags Tenant
// @Summary 获取TenantGateway 列表
// @Description 获取TenantGateway 列表
// @Accept json
// @Produce json
// @Param cluster_id path string true "cluster_id"
// @Param tenant_id path string true "tenant_id"
// @Success 200 {object} handlers.ResponseStruct{Data=[]v1beta1.TenantGateway} "object"
// @Router /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways [get]
// @Security JWT
func (h *TenantHandler) ListTenantGateway(c *gin.Context) {
	clusterid := c.Param("cluster_id")
	cluster := models.Cluster{}
	if err := h.GetDB().First(&cluster, clusterid).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("集群%s不存在", clusterid))
		return
	}

	// _all 不筛选租户
	tenantidStr := c.Param("tenant_id")
	selector := map[string]string{}
	if tenantidStr != "_all" && tenantidStr != "0" {
		tenantid, _ := strconv.Atoi(tenantidStr)
		tenant := models.Tenant{ID: uint(tenantid)}
		if err := h.GetDB().First(&tenant).Error; err != nil {
			handlers.NotOK(c, fmt.Errorf("租户%v不存在", tenantid))
			return
		}
		selector[gemlabels.LabelTenant+"__in"] = tenant.TenantName + "," + defaultGatewayTenant
	}

	tgList, err := kubeclient.GetClient().GetTenantGatewayList(cluster.ClusterName, selector)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, tgList)
}

// @Tags Tenant
// @Summary 获取TenantGateway
// @Description 获取TenantGateway
// @Accept json
// @Produce json
// @Param cluster_id path string true "cluster_id"
// @Param tenant_id path string true "tenant_id"
// @Param ingressClass query string true "ingressClass"
// @Success 200 {object} handlers.ResponseStruct{Data=v1beta1.TenantGateway} "object"
// @Router /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways/{name} [get]
// @Security JWT
func (h *TenantHandler) GetTenantGateway(c *gin.Context) {
	ingressClass := c.Query("ingressClass")
	clusterid, _ := strconv.Atoi(c.Param("cluster_id"))
	cluster := models.Cluster{ID: uint(clusterid)}
	if err := h.GetDB().First(&cluster).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("集群%v不存在", clusterid))
		return
	}

	if ingressClass != "" {
		tglist, err := kubeclient.GetClient().GetTenantGatewayList(cluster.ClusterName,
			map[string]string{"gems.cloudminds.com/ingressClass": ingressClass},
		)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		tmp := *tglist
		if len(tmp) == 0 {
			handlers.NotOK(c, fmt.Errorf("can't find gateway by ingressClass %s", ingressClass))
			return
		}

		handlers.OK(c, tmp[0])
	} else {
		tg, err := kubeclient.GetClient().GetTenantGateway(cluster.ClusterName, c.Param("name"), nil)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}

		handlers.OK(c, tg)
	}
}

// @Tags Tenant
// @Summary 创建TenantGateway
// @Description 创建TenantGateway
// @Accept json
// @Produce json
// @Param cluster_id path string true "cluster_id"
// @Param tenant_id path string true "tenant_id"
// @Param param body v1beta1.TenantGateway true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=v1beta1.TenantGateway} "object"
// @Router /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways [post]
// @Security JWT
func (h *TenantHandler) CreateTenantGateway(c *gin.Context) {
	tenantid, _ := strconv.Atoi(c.Param("tenant_id"))
	clusterid, _ := strconv.Atoi(c.Param("cluster_id"))
	tenant := models.Tenant{ID: uint(tenantid)}
	cluster := models.Cluster{ID: uint(clusterid)}
	if err := h.GetDB().First(&tenant).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("租户%v不存在", tenantid))
		return
	}
	if err := h.GetDB().First(&cluster).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("集群%v不存在", clusterid))
		return
	}

	tg := v1beta1.TenantGateway{}
	if err := c.BindJSON(&tg); err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "创建", "集群租户网关", fmt.Sprintf("集群[%v]/租户[%v]", cluster.ClusterName, tenant.TenantName))
	h.SetExtraAuditData(c, models.ResTenant, tenant.ID)

	ret, err := kubeclient.GetClient().CreateTenantGateway(cluster.ClusterName, tg.Name, &tg)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// @Tags Tenant
// @Summary 更新TenantGateway
// @Description 更新TenantGateway
// @Accept json
// @Produce json
// @Param cluster_id path string true "cluster_id"
// @Param tenant_id path string true "tenant_id"
// @Param param body v1beta1.TenantGateway true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=v1beta1.TenantGateway} "object"
// @Router /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways/{name} [put]
// @Security JWT
func (h *TenantHandler) UpdateTenantGateway(c *gin.Context) {
	clusterid, _ := strconv.Atoi(c.Param("cluster_id"))
	cluster := models.Cluster{ID: uint(clusterid)}
	if err := h.GetDB().First(&cluster).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("集群%v不存在", clusterid))
		return
	}
	tg := v1beta1.TenantGateway{}
	if err := c.BindJSON(&tg); err != nil {
		handlers.NotOK(c, err)
		return
	}

	u, _ := h.GetContextUser(c)
	auth := h.GetCacheLayer().GetUserAuthority(u)
	// 非管理员不能编辑默认网关
	if tg.Name == defaultGatewayName && !auth.IsSystemAdmin {
		handlers.NotOK(c, fmt.Errorf("只有系统管理员能修改默认网关"))
		return
	}

	h.SetAuditData(c, "更新", "集群租户网关", fmt.Sprintf("集群[%v]/租户[%v]", cluster.ClusterName, tg.Spec.Tenant))
	if tg.Name != defaultGatewayName {
		tenantid, _ := strconv.Atoi(c.Param("tenant_id"))
		tenant := models.Tenant{ID: uint(tenantid)}
		if err := h.GetDB().First(&tenant).Error; err != nil {
			handlers.NotOK(c, fmt.Errorf("租户%v不存在", tenantid))
			return
		}
		h.SetExtraAuditData(c, models.ResTenant, tenant.ID)
	}

	ret, err := kubeclient.GetClient().UpdateTenantGateway(cluster.ClusterName, tg.Name, &tg)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// @Tags Tenant
// @Summary 删除TenantGateway
// @Description 删除TenantGateway
// @Accept json
// @Produce json
// @Param cluster_id path string true "cluster_id"
// @Param tenant_id path string true "tenant_id"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "object"
// @Router /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways/{name} [delete]
// @Security JWT
func (h *TenantHandler) DeleteTenantGateway(c *gin.Context) {
	tenantid, _ := strconv.Atoi(c.Param("tenant_id"))
	clusterid, _ := strconv.Atoi(c.Param("cluster_id"))
	name := c.Param("name")
	tenant := models.Tenant{ID: uint(tenantid)}
	cluster := models.Cluster{ID: uint(clusterid)}
	if err := h.GetDB().First(&tenant).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("租户%v不存在", tenantid))
		return
	}
	if err := h.GetDB().First(&cluster).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("集群%v不存在", clusterid))
		return
	}

	if name == defaultGatewayName {
		handlers.NotOK(c, fmt.Errorf("不允许删除默认网关"))
		return
	}

	h.SetAuditData(c, "删除", "集群租户网关", fmt.Sprintf("集群[%v]/租户[%v]", cluster.ClusterName, tenant.TenantName))
	h.SetExtraAuditData(c, models.ResTenant, tenant.ID)

	err := kubeclient.GetClient().DeleteTenantGateway(cluster.ClusterName, name)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

type GatewayAddr struct {
	Addr   string
	Ready  bool
	Status string
}

// @Tags Tenant
// @Summary 获取TenantGateway adddresses
// @Description 获取TenantGateway adddresses
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=[]string} "object"
// @Router /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways/{name}/addresses [get]
// @Security JWT
func (h *TenantHandler) GetObjectTenantGatewayAddr(c *gin.Context) {
	tenantid, _ := strconv.Atoi(c.Param("tenant_id"))
	clusterid, _ := strconv.Atoi(c.Param("cluster_id"))
	tenant := models.Tenant{ID: uint(tenantid)}
	cluster := models.Cluster{ID: uint(clusterid)}
	if err := h.GetDB().First(&tenant).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("租户%v不存在", tenantid))
		return
	}
	if err := h.GetDB().First(&cluster).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("集群%v不存在", clusterid))
		return
	}
	tg, err := kubeclient.GetClient().GetTenantGateway(cluster.ClusterName, c.Param("name"), nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	svc, err := kubeclient.GetClient().GetService(cluster.ClusterName, gemlabels.NamespaceGateway, tg.Name, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	ret := []GatewayAddr{}
	var httpPort, httpsPort int32
	for _, port := range tg.Status.Ports {
		if port.Name == "http" {
			httpPort = port.NodePort
		}
		if port.Name == "https" {
			httpsPort = port.NodePort
		}
	}
	switch svc.Spec.Type {
	case v1.ServiceTypeNodePort:
		nodes, _ := kubeclient.GetClient().GetNodeList(cluster.ClusterName, nil)
		for _, node := range *nodes {
			status := "notready"
			ready := false
			for _, v := range node.Status.Conditions {
				if v.Type == "Ready" {
					if v.Status == "True" {
						status = "ready"
						ready = true
						break
					}
				}
			}
			for _, addr := range node.Status.Addresses {
				if addr.Type == v1.NodeInternalIP {
					ret = append(ret, GatewayAddr{
						Addr:   fmt.Sprintf("http://%s:%d", addr.Address, httpPort),
						Ready:  ready,
						Status: status,
					})
					ret = append(ret, GatewayAddr{
						Addr:   fmt.Sprintf("https://%s:%d", addr.Address, httpsPort),
						Ready:  ready,
						Status: status,
					})
				}
			}
		}
	case v1.ServiceTypeLoadBalancer:
		for _, v := range svc.Status.LoadBalancer.Ingress {
			if v.IP != "" {
				ret = append(ret, GatewayAddr{
					Addr:   v.IP,
					Ready:  true,
					Status: "ready",
				})
			}
			if v.Hostname != "" {
				ret = append(ret, GatewayAddr{
					Addr:   v.Hostname,
					Ready:  true,
					Status: "ready",
				})
			}
		}
	}
	handlers.OK(c, ret)
}
