package tenanthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	ext_v1beta1 "k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	gemlabels "kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/apis/networking"
	"kubegems.io/pkg/log"
	msgclient "kubegems.io/pkg/msgbus/client"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/handlers/base"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/msgbus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
// @Tags         Tenant
// @Summary      Tenant列表
// @Description  Tenant列表
// @Accept       json
// @Produce      json
// @Param        TenantName                     query     string                                                                 false  "TenantName"
// @Param        preload                        query     string                                                                 false  "choices ResourceQuotas,Users,Projects"
// @Param        page                           query     int                                                                    false  "page"
// @Param        size                           query     int                                                                    false  "page"
// @Param        search                         query     string                                                                 false  "search in (TenantName,Remark)"
// @Param        containAllocatedResourcequota  query     bool                                                                   false  "是否包含已分配的resourcequota"
// @Success      200                            {object}  handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Tenant}}  "Tenant"
// @Router       /v1/tenant [get]
// @Security     JWT
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

	pagedata := handlers.Page(total, list, int64(page), int64(size))
	if ok, _ := strconv.ParseBool(c.Query("containAllocatedResourcequota")); ok {
		tids := []uint{}
		tenants := pagedata.List.([]models.Tenant)
		for i := range tenants {
			tids = append(tids, tenants[i].ID)
		}

		type tenantallocated struct {
			EnvironmentID uint
			TenantID      uint
			ResourceQuota datatypes.JSON
		}
		allocated := []tenantallocated{}
		if err := h.GetDB().Raw(`select environments.id as environment_id, tenants.id as tenant_id, environments.resource_quota
		from environments left join projects on environments.project_id = projects.id
				left join tenants on projects.tenant_id = tenants.id where tenant_id in ?`, tids).Scan(&allocated).Error; err != nil {
			handlers.NotOK(c, err)
			return
		}

		tenantAllocatedMap := map[uint]v1.ResourceList{}
		// for
		for _, v := range allocated {
			envquota := v1.ResourceList{}
			if err := json.Unmarshal(v.ResourceQuota, &envquota); err != nil {
				log.Error(err, "unmarshal env quota")
			}
			if quota, ok := tenantAllocatedMap[v.TenantID]; ok {
				cpuReq := quota[v1.ResourceRequestsCPU]
				cpuReq.Add(envquota[v1.ResourceRequestsCPU])
				quota[v1.ResourceRequestsCPU] = cpuReq

				cpuLim := quota[v1.ResourceLimitsCPU]
				cpuLim.Add(envquota[v1.ResourceLimitsCPU])
				quota[v1.ResourceLimitsCPU] = cpuLim

				memReq := quota[v1.ResourceRequestsMemory]
				memReq.Add(envquota[v1.ResourceRequestsMemory])
				quota[v1.ResourceRequestsMemory] = memReq

				memLim := quota[v1.ResourceLimitsMemory]
				memLim.Add(envquota[v1.ResourceLimitsMemory])
				quota[v1.ResourceLimitsMemory] = memLim

				storageReq := quota[v1.ResourceRequestsStorage]
				storageReq.Add(envquota[v1.ResourceRequestsStorage])
				quota[v1.ResourceRequestsStorage] = storageReq

				pods := quota["count/pods"]
				pods.Add(envquota["count/pods"])
				quota["count/pods"] = pods

				tenantAllocatedMap[v.TenantID] = quota
			} else {
				tenantAllocatedMap[v.TenantID] = envquota
			}
		}

		for i := range tenants {
			tenants[i].AllocatedResourcequota = tenantAllocatedMap[tenants[i].ID]
		}

		pagedata.List = tenants
	}

	handlers.OK(c, pagedata)
}

// RetrieveTenant Tenant详情
// @Tags         Tenant
// @Summary      Tenant详情
// @Description  get Tenant详情
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                         true  "tenant_id"
// @Success      200        {object}  handlers.ResponseStruct{Data=models.Tenant}  "Tenant"
// @Router       /v1/tenant/{tenant_id} [get]
// @Security     JWT
func (h *TenantHandler) RetrieveTenant(c *gin.Context) {
	var obj models.Tenant
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// PostTenant 创建Tenant
// @Tags         Tenant
// @Summary      创建Tenant
// @Description  创建Tenant
// @Accept       json
// @Produce      json
// @Param        param  body      models.Tenant                                true  "表单"
// @Success      200    {object}  handlers.ResponseStruct{Data=models.Tenant}  "Tenant"
// @Router       /v1/tenant [post]
// @Security     JWT
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
	h.ModelCache().UpsertTenant(obj.ID, obj.TenantName)

	h.SetAuditData(c, "创建", "租户", obj.TenantName)
	h.SetExtraAuditData(c, models.ResTenant, obj.ID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Add
		msg.ResourceType = msgbus.Tenant
		msg.ResourceID = obj.ID
		msg.Detail = fmt.Sprintf("创建了租户%s", obj.TenantName)
		msg.ToUsers.Append(h.GetDataBase().SystemAdmins()...)
	})

	handlers.Created(c, obj)
}

// PutTenant 修改Tenant
// @Tags         Tenant
// @Summary      修改Tenant
// @Description  修改Tenant
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                         true  "tenant_id"
// @Param        param      body      models.Tenant                                true  "表单"
// @Success      200        {object}  handlers.ResponseStruct{Data=models.Tenant}  "Tenant"
// @Router       /v1/tenant/{tenant_id} [put]
// @Security     JWT
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
	h.ModelCache().UpsertTenant(obj.ID, obj.TenantName)
	handlers.OK(c, obj)
}

// DeleteTenant 删除 Tenant
// @Tags         Tenant
// @Summary      删除 Tenant
// @Description  删除 Tenant
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                     true  "tenant_id"
// @Success      204        {object}  handlers.ResponseStruct  "resp"
// @Router       /v1/tenant/{tenant_id} [delete]
// @Security     JWT
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	var obj models.Tenant
	// 这儿是删除数据，不存在preload 敏感数据的情况
	if err := h.GetDB().Preload("ResourceQuotas.Cluster").First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, nil)
		return
	}

	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&obj).Error; err != nil {
			return err
		}
		return h.afterTenantDelete(c.Request.Context(), tx, &obj)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "删除", "租户", obj.TenantName)
	h.SetExtraAuditData(c, models.ResTenant, obj.ID)

	h.ModelCache().DelTenant(obj.ID)

	handlers.NoContent(c, nil)
}

/*
	删除租户后，需要删除这个租户在各个集群下占用的资源
*/
func (h *TenantHandler) afterTenantDelete(ctx context.Context, tx *gorm.DB, t *models.Tenant) error {
	for _, quota := range t.ResourceQuotas {
		if err := h.afterTenantResourceQuotaDelete(ctx, tx, quota); err != nil {
			return err
		}
	}
	return nil
}

// ListTenantUser 获取属于Tenant的 User 列表
// @Tags         Tenant
// @Summary      获取属于 Tenant 的 User 列表
// @Description  获取属于 Tenant 的 User 列表
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                                                 true   "tenant_id"
// @Param        preload    query     string                                                               false  "choices Tenants,SystemRole"
// @Param        page       query     int                                                                  false  "page"
// @Param        size       query     int                                                                  false  "page"
// @Param        search     query     string                                                               false  "search in (Username,Email)"
// @Success      200        {object}  handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}}  "models.User"
// @Router       /v1/tenant/{tenant_id}/user [get]
// @Security     JWT
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
// @Tags         Tenant
// @Summary      获取Tenant 的一个 User详情
// @Description  获取Tenant 的一个 User详情
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                       true  "tenant_id"
// @Param        user_id    path      uint                                       true  "user_id"
// @Success      200        {object}  handlers.ResponseStruct{Data=models.User}  "models.User"
// @Router       /v1/tenant/{tenant_id}/user/{user_id} [get]
// @Security     JWT
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
// @Tags         Tenant
// @Summary      在User和Tenant间添加关联关系
// @Description  在User和Tenant间添加关联关系
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                                 true  "tenant_id"
// @Param        param      body      models.TenantUserRels                                true  "表单"`
// @Success      200        {object}  handlers.ResponseStruct{Data=models.TenantUserRels}  "models.User"
// @Router       /v1/tenant/{tenant_id}/user [post]
// @Security     JWT
func (h *TenantHandler) PostTenantUser(c *gin.Context) {
	var (
		tenant models.Tenant
		rel    models.TenantUserRels
		user   models.User
	)
	if err := h.GetDB().Preload("ResourceQuotas").First(&tenant, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("tenant not exists"))
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
	h.ModelCache().FlushUserAuthority(&user)
	h.SetAuditData(c, "添加", "租户成员", fmt.Sprintf("租户[%v]/用户[%v]", tenant.TenantName, user.Username))
	h.SetExtraAuditData(c, models.ResTenant, tenant.ID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Add
		msg.ResourceType = msgbus.Tenant
		msg.ResourceID = rel.TenantID
		msg.Detail = fmt.Sprintf("向租户%s中添加了用户%s", tenant.TenantName, user.Username)
		msg.ToUsers.Append(rel.UserID)
		msg.AffectedUsers.Append(rel.UserID)
	})

	handlers.OK(c, rel)
}

// PutTenantUser 修改 User 和 Tenant 的关联关系
// @Tags         Tenant
// @Summary      修改 User 和 Tenant 的关联关系
// @Description  修改 User 和 Tenant 的关联关系
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                                 true  "tenant_id"
// @Param        user_id    path      uint                                                 true  "user_id"
// @Param        param      body      models.TenantUserRels                                true  "表单"`
// @Success      200        {object}  handlers.ResponseStruct{Data=models.TenantUserRels}  "models.User"
// @Router       /v1/tenant/{tenant_id}/user/{user_id} [put]
// @Security     JWT
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
	h.ModelCache().FlushUserAuthority(&user)

	h.GetDB().Preload("Tenant").First(&rel, rel.ID)

	h.SetAuditData(c, "修改", "租户成员", fmt.Sprintf("租户[%v]/用户[%v]", rel.Tenant.TenantName, user.Username))
	h.SetExtraAuditData(c, models.ResTenant, rel.TenantID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Update
		msg.ResourceType = msgbus.Tenant
		msg.ResourceID = rel.TenantID
		msg.Detail = fmt.Sprintf("将租户%s中的用户%s设置为了%s", rel.Tenant.TenantName, user.Username, rel.Role)
		msg.ToUsers.Append(rel.UserID)
		msg.AffectedUsers.Append(rel.UserID)
	})
	handlers.OK(c, rel)
}

// DeleteTenantUser 删除 User 和 Tenant 的关系
// @Tags         Tenant
// @Summary      删除 User 和 Tenant 的关系
// @Description  删除 User 和 Tenant 的关系
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                       true  "tenant_id"
// @Param        user_id    path      uint                                       true  "user_id"
// @Success      200        {object}  handlers.ResponseStruct{Data=models.User}  "models.User"
// @Router       /v1/tenant/{tenant_id}/user/{user_id} [delete]
// @Security     JWT
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
	h.ModelCache().FlushUserAuthority(&user)

	h.SetAuditData(c, "删除", "租户成员", fmt.Sprintf("租户[%v]/用户[%v]", obj.TenantName, user.Username))
	h.SetExtraAuditData(c, models.ResTenant, rel.TenantID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Delete
		msg.ResourceType = msgbus.Tenant
		msg.ResourceID = rel.TenantID
		msg.Detail = fmt.Sprintf("删除了租户%s中的用户%s", obj.TenantName, user.Username)
		msg.ToUsers.Append(rel.UserID)
		msg.AffectedUsers.Append(rel.UserID)
	})
	handlers.NoContent(c, nil)
}

// ListTenantProject 获取属于Tenant的 Project 列表
// @Tags         Tenant
// @Summary      获取属于 Tenant 的 Project 列表
// @Description  获取属于 Tenant 的 Project 列表
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                                                    true   "tenant_id"
// @Param        preload    query     string                                                                  false  "choices Applications,Environments,Registries,Users,Tenant"
// @Param        page       query     int                                                                     false  "page"
// @Param        size       query     int                                                                     false  "page"
// @Param        search     query     string                                                                  false  "search in (ProjectName)"
// @Success      200        {object}  handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Project}}  "models.Project"
// @Router       /v1/tenant/{tenant_id}/project [get]
// @Security     JWT
func (h *TenantHandler) ListTenantProject(c *gin.Context) {
	var list []models.Project
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	tid := c.Param(PrimaryKeyName)
	user, _ := h.GetContextUser(c)
	userAuthority := h.ModelCache().GetUserAuthority(user)

	cond := &handlers.PageQueryCond{
		Model:         "Project",
		SearchFields:  []string{"project_name"},
		PreloadFields: []string{"Applications", "Environments", "Registries", "Tenant", "Users"},
		Where:         []*handlers.QArgs{handlers.Args("tenant_id = ?", tid)},
	}

	if !userAuthority.IsTenantAdmin(utils.ToUint(tid)) && !userAuthority.IsSystemAdmin() {
		cond.Join = handlers.Args("left join project_user_rels on project_user_rels.project_id = projects.id")
		cond.Where = append(cond.Where, handlers.Args("project_user_rels.user_id = ?", user.GetID()))
	}

	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveTenantProject 获取Tenant 的一个 Project详情
// @Tags         Tenant
// @Summary      获取Tenant 的一个 Project详情
// @Description  获取Tenant 的一个 Project详情
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      uint                                          true  "tenant_id"
// @Param        project_id  path      uint                                          true  "project_id"
// @Success      200         {object}  handlers.ResponseStruct{Data=models.Project}  "models.Project"
// @Router       /v1/tenant/{tenant_id}/project/{project_id} [get]
// @Security     JWT
func (h *TenantHandler) RetrieveTenantProject(c *gin.Context) {
	var project models.Project
	if err := h.GetDB().Where("tenant_id = ? and id = ?", c.Param(project.ProjectAlias), c.Param("project_id")).First(&project).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, project)
}

// PostTenantProject 创建一个属于 Tenant 的Project
// @Tags         Tenant
// @Summary      创建一个属于 Tenant 的Project
// @Description  创建一个属于 Tenant 的Project
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                          true  "tenant_id"
// @Param        param      body      models.Project                                true  "表单"
// @Success      200        {object}  handlers.ResponseStruct{Data=models.Project}  "models.Project"
// @Router       /v1/tenant/{tenant_id}/project [post]
// @Security     JWT
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

	_ = h.ModelCache().UpsertProject(tenant.ID, project.ID, project.ProjectName)

	h.SetAuditData(c, "创建", "项目", project.ProjectName)
	h.SetExtraAuditData(c, models.ResProject, project.ID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Add
		msg.ResourceType = msgbus.Project
		msg.ResourceID = project.ID
		msg.Detail = fmt.Sprintf("在租户%s中创建了项目%s", tenant.TenantName, project.ProjectName)
		msg.ToUsers.Append(h.GetDataBase().TenantAdmins(tenant.ID)...)
	})
	handlers.OK(c, project)
}

// ListTenantTenantResourceQuota 获取属于Tenant的 TenantResourceQuota 列表
// @Tags         Tenant
// @Summary      获取属于 Tenant 的 TenantResourceQuota 列表
// @Description  获取属于 Tenant 的 TenantResourceQuota 列表
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                                                                true   "tenant_id"
// @Param        preload    query     string                                                                              false  "choices Tenant,Cluster"
// @Param        page       query     int                                                                                 false  "page"
// @Param        size       query     int                                                                                 false  "page"
// @Success      200        {object}  handlers.ResponseStruct{Data=handlers.PageData{List=[]models.TenantResourceQuota}}  "models.TenantResourceQuota"
// @Router       /v1/tenant/{tenant_id}/tenantresourcequota [get]
// @Security     JWT
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
// @Tags         Tenant
// @Summary      获取Tenant 的一个 TenantResourceQuota详情
// @Description  获取Tenant 的一个 TenantResourceQuota详情
// @Accept       json
// @Produce      json
// @Param        tenant_id               path      uint                                                      true  "tenant_id"
// @Param        tenantresourcequota_id  path      uint                                                      true  "tenantresourcequota_id"
// @Success      200                     {object}  handlers.ResponseStruct{Data=models.TenantResourceQuota}  "models.TenantResourceQuota"
// @Router       /v1/tenant/{tenant_id}/tenantresourcequota/{tenantresourcequota_id} [get]
// @Security     JWT
func (h *TenantHandler) RetrieveTenantTenantResourceQuota(c *gin.Context) {
	var trq models.TenantResourceQuota
	if err := h.GetDB().First(&trq, "tenant_id = ? and id = ?", c.Param(PrimaryKeyName), c.Param("tenantresourcequota_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, trq)
}

// EnableTenant 激活租户
// @Tags         Tenant
// @Summary      激活租户,当租户为未激活状态的时候才可用
// @Description  激活租户,当租户为未激活状态的时候才可用
// @Accept       json
// @Produce      json
// @Param        tenantid  path      int                                            true  "tenantid"
// @Success      200       {object}  handlers.ResponseStruct{Data=[]models.Tenant}  "Tenant"
// @Router       /v1/tenant/{tenant_id}/action/enable [post]
// @Security     JWT
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
	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Update
		msg.ResourceType = msgbus.Tenant
		msg.ResourceID = obj.ID
		msg.Detail = fmt.Sprintf("激活了租户%s", obj.TenantName)
		msg.ToUsers.Append(h.GetDataBase().SystemAdmins()...).Append(tenantUsers...)
		msg.AffectedUsers.Append(tenantUsers...)
	})

	handlers.OK(c, obj)
}

// DisableTenant 取消激活租户
// @Tags         Tenant
// @Summary      取消激活租户,当租户为激活状态的时候才可用
// @Description  取消激活租户,当租户为激活状态的时候才可用
// @Accept       json
// @Produce      json
// @Param        tenantid  path      int                                            true  "tenant_id"
// @Success      200       {object}  handlers.ResponseStruct{Data=[]models.Tenant}  "Tenant"
// @Router       /v1/tenant/{tenant_id}/action/disable [post]
// @Security     JWT
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
	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Update
		msg.ResourceType = msgbus.Tenant
		msg.ResourceID = obj.ID
		msg.Detail = fmt.Sprintf("禁用了租户%s", obj.TenantName)
		msg.ToUsers.Append(h.GetDataBase().SystemAdmins()...).Append(tenantUsers...)
		msg.AffectedUsers.Append(tenantUsers...)
	})
	handlers.OK(c, obj)
}

// PostTenantTenantResourceQuota 创建一个属于 Tenant 的TenantResourceQuota
// @Tags         Tenant
// @Summary      创建一个属于 Tenant 的TenantResourceQuota
// @Description  创建一个属于 Tenant 的TenantResourceQuota
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                                      true  "tenant_id"
// @Param        param      body      models.TenantResourceQuota                                true  "表单"
// @Success      200        {object}  handlers.ResponseStruct{Data=models.TenantResourceQuota}  "models.TenantResourceQuota"
// @Router       /v1/tenant/{tenant_id}/tenantresourcequota [post]
// @Security     JWT
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
	ctx := c.Request.Context()

	h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&obj).Error; err != nil {
			return err
		}
		return AfterTenantResourceQuotaSave(ctx, h.BaseHandler, tx, &obj)
	})

	h.GetDB().Preload("Tenant").Preload("Cluster", func(tx *gorm.DB) *gorm.DB { return tx.Select("id, cluster_name") }).First(&obj, obj.ID)

	h.SetAuditData(c, "创建", "租户集群资源限制", fmt.Sprintf("租户[%v]/集群[%v]", obj.Tenant.TenantName, obj.Cluster.ClusterName))
	h.SetExtraAuditData(c, models.ResTenant, obj.TenantID)

	handlers.Created(c, obj)
}

func AfterTenantResourceQuotaSave(ctx context.Context, h base.BaseHandler, tx *gorm.DB, trq *models.TenantResourceQuota) error {
	var (
		tenant  models.Tenant
		cluster models.Cluster
		rels    []models.TenantUserRels
	)
	tx.First(&cluster, "id = ?", trq.ClusterID)
	tx.First(&tenant, "id = ?", trq.TenantID)
	tx.Preload("User").Find(&rels, "tenant_id = ?", trq.TenantID)

	admins := []string{}
	members := []string{}
	for _, rel := range rels {
		if rel.Role == models.TenantRoleAdmin {
			admins = append(admins, rel.User.Username)
		} else {
			members = append(members, rel.User.Username)
		}
	}
	// 创建or更新 租户
	if err := CreateOrUpdateTenant(ctx, h, cluster.ClusterName, tenant.TenantName, admins, members); err != nil {
		return err
	}
	// 这儿有个坑，controller还没有成功创建出来TenantResourceQuota，就去更新租户资源，会报错404；先睡会儿把
	<-time.NewTimer(time.Second * 2).C
	// 创建or更新 租户资源
	if err := CreateOrUpdateTenantResourceQuota(ctx, h, cluster.ClusterName, tenant.TenantName, trq.Content); err != nil {
		return err
	}
	return nil
}

func CreateOrUpdateTenant(ctx context.Context, h base.BaseHandler, clustername, tenantname string, admins, members []string) error {
	return h.Execute(ctx, clustername, func(ctx context.Context, cli agents.Client) error {
		crdTenant := &v1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{Name: tenantname},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, cli, crdTenant, func() error {
			crdTenant.Spec = v1beta1.TenantSpec{
				TenantName: tenantname,
				Admin:      admins,
				Members:    members,
			}
			return nil
		})
		return err
	})
}

func CreateOrUpdateTenantResourceQuota(ctx context.Context, h base.BaseHandler, clustername, tenantname string, data []byte) error {
	var hard v1.ResourceList
	if err := json.Unmarshal(data, &hard); err != nil {
		return err
	}

	return h.Execute(ctx, clustername, func(ctx context.Context, cli agents.Client) error {
		tquota := &v1beta1.TenantResourceQuota{
			ObjectMeta: metav1.ObjectMeta{Name: tenantname},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, cli, tquota, func() error {
			tquota.Spec = v1beta1.TenantResourceQuotaSpec{Hard: hard}
			return nil
		})
		return err
	})
}

// PatchTenantTenantResourceQuota 修改一个属于 Tenant 的TenantResourceQuota
// @Tags         Tenant
// @Summary      修改一个属于 Tenant 的TenantResourceQuota
// @Description  修改一个属于 Tenant 的TenantResourceQuota
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      uint                                                      true  "tenant_id"
// @Param        cluster_id  path      uint                                                      true  "cluster_id"
// @Param        param       body      models.TenantResourceQuota                                true  "表单"
// @Success      200         {object}  handlers.ResponseStruct{Data=models.TenantResourceQuota}  "models.TenantResourceQuota"
// @Router       /v1/tenant/{tenant_id}/tenantresourcequota/{:cluster_id} [put]
// @Security     JWT
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

	ctx := c.Request.Context()

	if err := c.ShouldBind(&trq); err != nil {
		handlers.NotOK(c, err)
		return
	}
	json.Unmarshal(trq.Content, &need)
	if err := h.ValidateTenantResourceQuota(ctx, oversold, clustername, origin, need); err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "集群租户资源限制", fmt.Sprintf("集群[%v]/租户[%v]", cluster.ClusterName, tenant.TenantName))
	h.SetExtraAuditData(c, models.ResTenant, trq.TenantID)

	if e := h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Save(&trq).Error; e != nil {
			return e
		}
		return AfterTenantResourceQuotaSave(ctx, h.BaseHandler, tx, &trq)
	}); e != nil {
		handlers.NotOK(c, e)
		return
	}
	handlers.OK(c, trq)
}

// PatchTenantTenantResourceQuota 删除租户在一个集群下的资源
// @Tags         Tenant
// @Summary      删除租户在一个集群下的资源
// @Description  删除租户在一个集群下的资源
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      uint                                  true  "tenant_id"
// @Param        cluster_id  path      uint                                  true  "cluster_id"
// @Success      200         {object}  handlers.ResponseStruct{Data=object}  "obj"
// @Router       /v1/tenant/{tenant_id}/tenantresourcequota/{:cluster_id} [put]
// @Security     JWT
func (h *TenantHandler) DeleteTenantResourceQuota(c *gin.Context) {
	var trq models.TenantResourceQuota
	if err := h.GetDB().Preload("Tenant").Preload(
		"Cluster",
		func(tx *gorm.DB) *gorm.DB { return tx.Select("id, cluster_name") },
	).First(&trq, "tenant_id = ? and cluster_id = ?", c.Param(PrimaryKeyName), c.Param("cluster_id")).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	ctx := c.Request.Context()
	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&trq).Error; err != nil {
			return err
		}
		return h.afterTenantResourceQuotaDelete(ctx, tx, &trq)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "删除", "集群租户资源限制", fmt.Sprintf("集群[%v]/租户[%v]", trq.Cluster.ClusterName, trq.Tenant.TenantName))
	h.SetExtraAuditData(c, models.ResTenant, trq.TenantID)

	handlers.NoContent(c, nil)
}

/*
	同步删除对应集群的资源
*/
func (h *TenantHandler) afterTenantResourceQuotaDelete(ctx context.Context, tx *gorm.DB, trq *models.TenantResourceQuota) error {
	return h.Execute(ctx, trq.Cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
		return cli.Delete(ctx, &v1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name: trq.Tenant.TenantName,
			},
		})
	})
}

// TenantEnvironments 获取租户下所有的环境
// @Tags         Tenant
// @Summary      获取租户下所有的环境以及资源状态
// @Description  获取租户下所有的环境以及资源状态
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      int                                     true   "tenant_id"
// @Param        search     query     string                                  false  "search in (EnvironmentName)"
// @Success      200        {object}  handlers.ResponseStruct{Data=[]object}  "object"
// @Router       /v1/tenant/{tenant_id}/environment_with_quotas [get]
// @Security     JWT
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

	ctx := c.Request.Context()

	search := c.Query("search")
	q := h.GetDB().Preload("Project").Preload(
		"Creator",
		func(tx *gorm.DB) *gorm.DB { return tx.Select("id, username") },
	).Preload(
		"Cluster",
		func(tx *gorm.DB) *gorm.DB { return tx.Select("id, cluster_name") },
	)
	if search != "" {
		q.Where("environment_name like ?", fmt.Sprintf("%%%s%%", search))
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

		quotas := &v1.ResourceQuotaList{}
		err := h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
			return cli.List(ctx, quotas, client.MatchingLabels(labels))
		})
		if err != nil {
			continue
		}
		for _, quota := range quotas.Items {
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
// @Tags         Tenant
// @Summary      获取租户下的所有Environment列表
// @Description  获取租户下的所有Environment列表
// @Accept       json
// @Produce      json
// @Param        tenant_id        path      int                                                                         true   "tenant_id"
// @Param        EnvironmentName  query     string                                                                      false  "EnvironmentName"
// @Param        preload          query     string                                                                      false  "choices Creator,Cluster,Project,ResourceQuota,Applications,Users"
// @Param        page             query     int                                                                         false  "page"
// @Param        size             query     int                                                                         false  "page"
// @Param        search           query     string                                                                      false  "search in (EnvironmentName)"
// @Success      200              {object}  handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Environment}}  "Environment"
// @Router       /v1/tenant/{tenant_id}/environment [get]
// @Security     JWT
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
// @Tags         Tenant
// @Summary      租户非资源类型统计
// @Description  租户非资源类型统计
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      int                                                 true  "tenant_id"
// @Success      200        {object}  handlers.ResponseStruct{Data=tenantStatisticsData}  "statistics"
// @Router       /v1/tenant/{tenant_id}/statistics [get]
// @Security     JWT
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

	ctx := c.Request.Context()

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
				quotas := &v1.ResourceQuotaList{}
				err := h.Execute(ctx, clustername, func(ctx context.Context, cli agents.Client) error {
					return cli.List(ctx, quotas, client.MatchingLabels{
						gemlabels.LabelTenant: tenant.TenantName,
					})
				})
				if err != nil {
					log.Error(err, "get resource quotas failed")
				}
				allQuotas <- &quotas.Items
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

// @Tags         NetworkIsolated
// @Summary      租户网络隔离开关
// @Description  租户网络隔离开关
// @Accept       json
// @Produce      json
// @Param        tenant_id  path      uint                                                   true  "tenant_id"
// @Param        param      body      handlers.ClusterIsolatedSwitch                         true  "表单"
// @Success      200        {object}  handlers.ResponseStruct{Data=handlers.IsolatedSwitch}  "object"
// @Router       /v1/tenant/{tenant_id}/action/networkisolate [post]
// @Security     JWT
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

	ctx := c.Request.Context()
	tnetpol := &v1beta1.TenantNetworkPolicy{}
	err := h.Execute(ctx, cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
		if err := cli.Get(ctx, client.ObjectKey{Name: tenant.TenantName}, tnetpol); err != nil {
			return err
		}
		tnetpol.Spec.TenantIsolated = form.Isolate
		return cli.Update(ctx, tnetpol)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, tnetpol)
}

// CreateTenantTenantResourceQuotaApply  创建or修改租户集群资源变更申请
// @Tags         Tenant
// @Summary      创建or修改租户集群资源变更申请
// @Description  创建or修改租户集群资源变更申请
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      uint                                                           true  "tenant_id"
// @Param        cluster_id  path      uint                                                           true  "cluster_id"
// @Param        param       body      models.TenantResourceQuotaApply                                true  "表单"
// @Success      200         {object}  handlers.ResponseStruct{Data=models.TenantResourceQuotaApply}  "models.TenantResourceQuotaApply"
// @Router       /v1/tenant/{tenant_id}/cluster/{cluster_id}/resourceApply [post]
// @Security     JWT
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

	ctx := c.Request.Context()

	// 没有就新建，有就更新
	if quota.TenantResourceQuotaApplyID == nil {
		quota.TenantResourceQuotaApply = &models.TenantResourceQuotaApply{}
	}
	quota.TenantResourceQuotaApply.Status = models.QuotaStatusPending
	quota.TenantResourceQuotaApply.Content = req.Content
	quota.TenantResourceQuotaApply.Username = u.GetUsername()

	need := v1.ResourceList{}
	origin := v1.ResourceList{}
	json.Unmarshal(req.Content, &need)
	json.Unmarshal(quota.Content, &origin)
	if err := h.ValidateTenantResourceQuota(ctx, quota.Cluster.OversoldConfig, quota.Cluster.ClusterName, origin, need); err != nil {
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
	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.MessageType = msgbus.Approve
		msg.EventKind = msgbus.Update
		msg.ResourceType = msgbus.TenantResourceQuota
		msg.ResourceID = quota.ID
		msg.Detail = fmt.Sprintf("申请调整租户%s在集群%s的资源", quota.Tenant.TenantName, quota.Cluster.ClusterName)
		msg.ToUsers.Append(h.GetDataBase().SystemAdmins()...).Append(u.GetID())
	})

	handlers.OK(c, req)
}

// GetTenantTenantResourceQuotaApply  获取租户集群资源变更申请详情
// @Tags         Tenant
// @Summary      获取租户集群资源变更申请详情
// @Description  获取租户集群资源变更申请详情
// @Accept       json
// @Produce      json
// @Param        tenant_id                    path      uint                                                           true  "tenant_id"
// @Param        tenantresourcequotaapply_id  path      uint                                                           true  "tenantresourcequotaapply_id"
// @Success      200                          {object}  handlers.ResponseStruct{Data=models.TenantResourceQuotaApply}  "models.TenantResourceQuotaApply"
// @Router       /v1/tenant/{tenant_id}/tenantresourcequotaapply/{tenantresourcequotaapply_id} [get]
// @Security     JWT
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

// @Tags         Tenant
// @Summary      获取TenantGateway 列表
// @Description  获取TenantGateway 列表
// @Accept       json
// @Produce      json
// @Param        cluster_id  path      string                                                 true  "cluster_id"
// @Param        tenant_id   path      string                                                 true  "tenant_id"
// @Success      200         {object}  handlers.ResponseStruct{Data=[]v1beta1.TenantGateway}  "object"
// @Router       /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways [get]
// @Security     JWT
func (h *TenantHandler) ListTenantGateway(c *gin.Context) {
	clusterid := c.Param("cluster_id")
	cluster := models.Cluster{}
	if err := h.GetDB().First(&cluster, clusterid).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("集群%s不存在", clusterid))
		return
	}
	ctx := c.Request.Context()
	// _all 不筛选租户
	tenantidStr := c.Param("tenant_id")
	var selector labels.Selector
	if tenantidStr != "_all" && tenantidStr != "0" {
		tenantid, _ := strconv.Atoi(tenantidStr)
		tenant := models.Tenant{ID: uint(tenantid)}
		if err := h.GetDB().First(&tenant).Error; err != nil {
			handlers.NotOK(c, fmt.Errorf("租户%v不存在", tenantid))
			return
		}
		r, _ := labels.NewRequirement(gemlabels.LabelTenant, selection.In, []string{tenant.TenantName, defaultGatewayTenant})
		selector = labels.NewSelector().Add(*r)
	}

	tgList, err := h.listGateways(ctx, cluster.ClusterName, selector)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, tgList)
}

func (h *TenantHandler) listGateways(ctx context.Context, cluster string, selector labels.Selector) ([]v1beta1.TenantGateway, error) {
	gatewaylist := &v1beta1.TenantGatewayList{}
	err := h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.List(ctx, gatewaylist, &client.ListOptions{
			LabelSelector: selector,
		})
	})
	return gatewaylist.Items, err
}

func (h *TenantHandler) getGateway(ctx context.Context, cluster string, name string) (*v1beta1.TenantGateway, error) {
	gateway := &v1beta1.TenantGateway{}
	err := h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Get(ctx, client.ObjectKey{Name: name}, gateway)
	})
	return gateway, err
}

func (h *TenantHandler) createGateway(ctx context.Context, cluster string, gateway *v1beta1.TenantGateway) error {
	return h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		dep := appsv1.Deployment{}
		err := cli.Get(ctx, types.NamespacedName{
			Namespace: gemlabels.NamespaceGateway,
			Name:      gateway.Name,
		}, &dep)
		// 避免与istio网关同名
		if err == nil {
			return fmt.Errorf("网关%s已存在", gateway.Name)
		}
		if !kerrors.IsNotFound(err) {
			return err
		}
		return cli.Create(ctx, gateway)
	})
}

func (h *TenantHandler) updateGateway(ctx context.Context, cluster string, gateway *v1beta1.TenantGateway) error {
	return h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Update(ctx, gateway)
	})
}

func (h *TenantHandler) deleteGateway(ctx context.Context, cluster string, name string) error {
	return h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Delete(ctx, &v1beta1.TenantGateway{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
	})
}

// @Tags         Tenant
// @Summary      获取TenantGateway
// @Description  获取TenantGateway
// @Accept       json
// @Produce      json
// @Param        cluster_id    path      string                                               true  "cluster_id"
// @Param        tenant_id     path      string                                               true  "tenant_id"
// @Param        ingressClass  query     string                                               true  "ingressClass"
// @Success      200           {object}  handlers.ResponseStruct{Data=v1beta1.TenantGateway}  "object"
// @Router       /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways/{name} [get]
// @Security     JWT
func (h *TenantHandler) GetTenantGateway(c *gin.Context) {
	ingressClass := c.Query("ingressClass")
	clusterid, _ := strconv.Atoi(c.Param("cluster_id"))
	cluster := models.Cluster{ID: uint(clusterid)}
	if err := h.GetDB().First(&cluster).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("集群%v不存在", clusterid))
		return
	}

	ctx := c.Request.Context()

	if ingressClass != "" {
		tglist, err := h.listGateways(ctx, cluster.ClusterName,
			labels.SelectorFromSet(map[string]string{networking.LabelIngressClass: ingressClass}),
		)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		tmp := tglist
		if len(tmp) == 0 {
			handlers.NotOK(c, fmt.Errorf("can't find gateway by ingressClass %s", ingressClass))
			return
		}

		handlers.OK(c, tmp[0])
	} else {
		tg, err := h.getGateway(ctx, cluster.ClusterName, c.Param("name"))
		if err != nil {
			handlers.NotOK(c, err)
			return
		}

		handlers.OK(c, tg)
	}
}

// @Tags         Tenant
// @Summary      创建TenantGateway
// @Description  创建TenantGateway
// @Accept       json
// @Produce      json
// @Param        cluster_id  path      string                                               true  "cluster_id"
// @Param        tenant_id   path      string                                               true  "tenant_id"
// @Param        param       body      v1beta1.TenantGateway                                true  "表单"
// @Success      200         {object}  handlers.ResponseStruct{Data=v1beta1.TenantGateway}  "object"
// @Router       /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways [post]
// @Security     JWT
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
	ctx := c.Request.Context()

	if err := h.createGateway(ctx, cluster.ClusterName, &tg); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, tg)
}

// @Tags         Tenant
// @Summary      更新TenantGateway
// @Description  更新TenantGateway
// @Accept       json
// @Produce      json
// @Param        cluster_id  path      string                                               true  "cluster_id"
// @Param        tenant_id   path      string                                               true  "tenant_id"
// @Param        param       body      v1beta1.TenantGateway                                true  "表单"
// @Success      200         {object}  handlers.ResponseStruct{Data=v1beta1.TenantGateway}  "object"
// @Router       /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways/{name} [put]
// @Security     JWT
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

	ctx := c.Request.Context()

	u, _ := h.GetContextUser(c)
	auth := h.ModelCache().GetUserAuthority(u)
	// 非管理员不能编辑默认网关
	if tg.Name == defaultGatewayName && !auth.IsSystemAdmin() {
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

	err := h.updateGateway(ctx, cluster.ClusterName, &tg)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, &tg)
}

// @Tags         Tenant
// @Summary      删除TenantGateway
// @Description  删除TenantGateway
// @Accept       json
// @Produce      json
// @Param        cluster_id  path      string                                true  "cluster_id"
// @Param        tenant_id   path      string                                true  "tenant_id"
// @Param        name        path      string                                true  "name"
// @Success      200         {object}  handlers.ResponseStruct{Data=string}  "object"
// @Router       /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways/{name} [delete]
// @Security     JWT
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

	err := h.deleteGateway(c.Request.Context(), cluster.ClusterName, name)
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

// @Tags         Tenant
// @Summary      获取TenantGateway adddresses
// @Description  获取TenantGateway adddresses
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                  true  "cluster"
// @Param        name     path      string                                  true  "name"
// @Success      200      {object}  handlers.ResponseStruct{Data=[]string}  "object"
// @Router       /v1/tenant/{tenant_id}/cluster/{cluster_id}/tenantgateways/{name}/addresses [get]
// @Security     JWT
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

	ctx := c.Request.Context()

	tg, err := h.getGateway(ctx, cluster.ClusterName, c.Param("name"))
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tg.Name,
			Namespace: gemlabels.NamespaceGateway,
		},
	}
	err = h.Execute(ctx, cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
		return cli.Get(ctx, client.ObjectKeyFromObject(svc), svc)
	})
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

		nodes := &v1.NodeList{}
		_ = h.Execute(ctx, cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
			return cli.List(ctx, nodes)
		})

		for _, node := range nodes.Items {
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
