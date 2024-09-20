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

package projecthandler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/i18n"
	msgclient "kubegems.io/kubegems/pkg/msgbus/client"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/msgbus"
)

var (
	SearchFields           = []string{"ProjectName"}
	FilterFields           = []string{"project_name", "TenantID"}
	PreloadFields          = []string{"Applications", "Environments", "Registries", "Users", "Tenant"}
	OrderFields            = []string{"project_name", "ID"}
	PreloadSensitiveFields = map[string]string{"Cluster": "id, cluster_name"}
	ModelName              = "Project"
	PrimaryKeyName         = "project_id"
)

// ListProject 列表 Project
//	@Tags			Project
//	@Summary		Project列表
//	@Description	Project列表
//	@Accept			json
//	@Produce		json
//	@Param			ProjectName	query		string																	false	"ProjectName"
//	@Param			TenantID	query		string																	false	"TenantID"
//	@Param			preload		query		string																	false	"choices Applications,Environments,Registries,Users,Tenant"
//	@Param			page		query		int																		false	"page"
//	@Param			size		query		int																		false	"page"
//	@Param			search		query		string																	false	"search in (ProjectName)"
//	@Success		200			{object}	handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Project}}	"Project"
//	@Router			/v1/project [get]
//	@Security		JWT
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
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// RetrieveProject Project详情
//	@Tags			Project
//	@Summary		Project详情
//	@Description	get Project详情
//	@Accept			json
//	@Produce		json
//	@Param			project_id	path		uint											true	"project_id"
//	@Success		200			{object}	handlers.ResponseStruct{Data=models.Project}	"Project"
//	@Router			/v1/project/{project_id} [get]
//	@Security		JWT
func (h *ProjectHandler) RetrieveProject(c *gin.Context) {
	var (
		obj   models.Project
		users []*models.User
	)
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Select(
		"users.*, project_user_rels.role",
	).Joins(
		"join project_user_rels  on  project_user_rels.user_id = users.id",
	).Find(&users, "`project_user_rels`.`project_id` = ?", c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().WithContext(ctx).Preload("Tenant").First(&obj, "id = ?", c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	obj.Users = users
	handlers.OK(c, obj)
}

// PutProject 修改Project
//	@Tags			Project
//	@Summary		修改Project
//	@Description	修改Project
//	@Accept			json
//	@Produce		json
//	@Param			project_id	path		uint											true	"project_id"
//	@Param			param		body		models.Project									true	"表单"
//	@Success		200			{object}	handlers.ResponseStruct{Data=models.Project}	"Project"
//	@Router			/v1/project/{project_id} [put]
//	@Security		JWT
func (h *ProjectHandler) PutProject(c *gin.Context) {
	var obj models.Project
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}

	action := i18n.Sprintf(context.TODO(), "update")
	module := i18n.Sprintf(context.TODO(), "project")
	h.SetAuditData(c, action, module, obj.ProjectName)
	h.SetExtraAuditData(c, models.ResProject, obj.ID)

	if c.Param(PrimaryKeyName) != strconv.Itoa(int(obj.ID)) {
		handlers.NotOK(c, i18n.Errorf(c, "URL parameter mismatched with body"))
		return
	}
	if err := h.GetDB().WithContext(ctx).Save(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.ModelCache().UpsertProject(obj.TenantID, obj.ID, obj.ProjectName)
	handlers.OK(c, obj)
}

// DeleteProject 删除 Project
//	@Tags			Project
//	@Summary		删除 Project
//	@Description	删除 Project
//	@Accept			json
//	@Produce		json
//	@Param			project_id	path		uint					true	"project_id"
//	@Success		204			{object}	handlers.ResponseStruct	"resp"
//	@Router			/v1/project/{project_id} [delete]
//	@Security		JWT
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	var obj models.Project
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Preload("Environments.Cluster").Preload("Tenant").First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}

	projUsers := h.GetDataBase().ProjectUsers(obj.ID)
	tenantAdmins := h.GetDataBase().TenantAdmins(obj.TenantID)
	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "project")
	h.SetAuditData(c, action, module, obj.ProjectName)
	h.SetExtraAuditData(c, models.ResProject, obj.ID)

	err := h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&obj).Error; err != nil {
			return err
		}
		return h.afterProjectDelete(ctx, tx, &obj)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.ModelCache().DelProject(obj.TenantID, obj.ID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Delete
		msg.ResourceType = msgbus.Project
		msg.ResourceID = obj.ID
		msg.Detail = i18n.Sprintf(context.TODO(), "delete project %s belong to tenant %s", obj.ProjectName, obj.Tenant.TenantName)
		msg.ToUsers.Append(tenantAdmins...).Append(projUsers...)
		msg.AffectedUsers.Append(projUsers...) // 项目用户需刷新权限
	})
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
//	@Tags			Project
//	@Summary		获取属于 Project 的 User 列表
//	@Description	获取属于 Project 的 User 列表
//	@Accept			json
//	@Produce		json
//	@Param			project_id	path		uint																true	"project_id"
//	@Param			preload		query		string																false	"choices Tenants,SystemRole"
//	@Param			page		query		int																	false	"page"
//	@Param			size		query		int																	false	"page"
//	@Param			search		query		string																false	"search in (Username,Email)"
//	@Success		200			{object}	handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}}	"models.User"
//	@Router			/v1/project/{project_id}/user [get]
//	@Security		JWT
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
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveProjectUser 获取Project 的一个 User详情
//	@Tags			Project
//	@Summary		获取Project 的一个 User详情
//	@Description	获取Project 的一个 User详情
//	@Accept			json
//	@Produce		json
//	@Param			project_id	path		uint										true	"project_id"
//	@Param			user_id		path		uint										true	"user_id"
//	@Success		200			{object}	handlers.ResponseStruct{Data=models.User}	"models.User"
//	@Router			/v1/project/{project_id}/user/{user_id} [get]
//	@Security		JWT
func (h *ProjectHandler) RetrieveProjectUser(c *gin.Context) {
	var user models.User
	if err := h.GetDB().WithContext(c.Request.Context()).Model(
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
//	@Tags			Project
//	@Summary		在User和Project间添加关联关系
//	@Description	在User和Project间添加关联关系
//	@Accept			json
//	@Produce		json
//	@Param			project_id	path		uint													true	"project_id"
//	@Param			param		body		models.ProjectUserRels									true	"表单"`
//	@Success		200			{object}	handlers.ResponseStruct{Data=models.ProjectUserRels}	"models.User"
//	@Router			/v1/project/{project_id}/user [post]
//	@Security		JWT
func (h *ProjectHandler) PostProjectUser(c *gin.Context) {
	var rel models.ProjectUserRels
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

	h.GetDB().WithContext(ctx).Preload("Project.Tenant").First(&rel, rel.ID)

	action := i18n.Sprintf(context.TODO(), "add")
	module := i18n.Sprintf(context.TODO(), "project member")
	h.SetAuditData(c, action, module, i18n.Sprintf(context.TODO(), "project %s / user %s / role %s", rel.Project.ProjectName, user.Username, rel.Role))
	h.SetExtraAuditData(c, models.ResProject, rel.ProjectID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Add
		msg.ResourceType = msgbus.Project
		msg.ResourceID = rel.ProjectID
		msg.Detail = i18n.Sprintf(context.TODO(), "add user %s to project %s members as role %s", user.Username, rel.Project.ProjectName, rel.Role)
		msg.ToUsers.
			Append(rel.UserID). // 自己
			Append(func() []uint {
				if rel.Role == models.ProjectRoleAdmin {
					return h.GetDataBase().ProjectAdmins(rel.ProjectID)
				}
				return nil
			}()...) // 项目管理员
		msg.AffectedUsers.Append(rel.UserID)
	})
	handlers.OK(c, rel)
}

// PutProjectUser 修改 User 和 Project 的关联关系
//	@Tags			Project
//	@Summary		修改 User 和 Project 的关联关系
//	@Description	修改 User 和 Project 的关联关系
//	@Accept			json
//	@Produce		json
//	@Param			project_id	path		uint													true	"project_id"
//	@Param			user_id		path		uint													true	"user_id"
//	@Param			param		body		models.ProjectUserRels									true	"表单"`
//	@Success		200			{object}	handlers.ResponseStruct{Data=models.ProjectUserRels}	"models.User"
//	@Router			/v1/project/{project_id}/user/{user_id} [put]
//	@Security		JWT
func (h *ProjectHandler) PutProjectUser(c *gin.Context) {
	var (
		tmp, rel models.ProjectUserRels
	)
	if err := c.BindJSON(&tmp); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if tmp.ProjectID != utils.ToUint(c.Param("project_id")) || tmp.UserID != utils.ToUint(c.Param("user_id")) {
		handlers.NotOK(c, i18n.Errorf(c, "parameters missmatched"))
		return
	}
	ctx := c.Request.Context()

	if err := h.GetDB().WithContext(ctx).Preload("Project.Tenant").First(&rel, "project_id = ? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NotOK(c, i18n.Errorf(c, "can't modify project member role, the user is not a member of the project"))
		return
	}
	rel.Role = tmp.Role
	if err := h.GetDB().WithContext(ctx).Save(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	user := models.User{}
	h.GetDB().WithContext(ctx).Preload("SystemRole").First(&user, rel.UserID)
	h.ModelCache().FlushUserAuthority(&user)

	action := i18n.Sprintf(context.TODO(), "update")
	module := i18n.Sprintf(context.TODO(), "project member")
	h.SetAuditData(c, action, module, i18n.Sprintf(context.TODO(), "project %s / user %s / role %s", rel.Project.ProjectName, user.Username, rel.Role))
	h.SetExtraAuditData(c, models.ResProject, rel.ProjectID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Update
		msg.ResourceType = msgbus.Project
		msg.ResourceID = rel.ProjectID
		msg.Detail = i18n.Sprintf(context.TODO(), "update user %s to project %s members as role %s", user.Username, rel.Project.ProjectName, rel.Role)
		msg.ToUsers.
			Append(rel.UserID). // 自己
			Append(func() []uint {
				if rel.Role == models.ProjectRoleAdmin {
					return h.GetDataBase().ProjectAdmins(rel.ProjectID)
				}
				return nil
			}()...) // 项目管理员
		msg.AffectedUsers.Append(rel.UserID)
	})
	handlers.OK(c, rel)
}

// DeleteProjectUser 删除 User 和 Project 的关系
//	@Tags			Project
//	@Summary		删除 User 和 Project 的关系
//	@Description	删除 User 和 Project 的关系
//	@Accept			json
//	@Produce		json
//	@Param			project_id	path		uint										true	"project_id"
//	@Param			user_id		path		uint										true	"user_id"
//	@Success		200			{object}	handlers.ResponseStruct{Data=models.User}	"models.User"
//	@Router			/v1/project/{project_id}/user/{user_id} [delete]
//	@Security		JWT
func (h *ProjectHandler) DeleteProjectUser(c *gin.Context) {
	var (
		rel    models.ProjectUserRels
		envrel models.EnvironmentUserRels
	)
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Model(&rel).First(&rel, "project_id =? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	h.GetDB().WithContext(ctx).Preload("Project.Tenant").First(&rel, rel.ID)
	if err := h.GetDB().WithContext(ctx).Delete(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	// 从项目中删除用户同时也要从项目下各个环境中删除
	envids := []uint{}
	if err := h.GetDB().WithContext(ctx).Model(&models.Environment{}).Where("project_id = ?", c.Param(PrimaryKeyName)).Pluck("id", &envids).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().WithContext(ctx).Delete(&envrel, "environment_id in (?) and user_id = ?", envids, c.Param("user_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	user := models.User{}
	h.GetDB().WithContext(ctx).Preload("SystemRole").First(&user, c.Param("user_id"))
	h.ModelCache().FlushUserAuthority(&user)

	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "project member")
	h.SetAuditData(c, action, module, i18n.Sprintf(context.TODO(), "project %s / user %s / role %s", rel.Project.ProjectName, user.Username, rel.Role))
	h.SetExtraAuditData(c, models.ResProject, rel.ProjectID)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Delete
		msg.ResourceType = msgbus.Project
		msg.ResourceID = rel.ProjectID
		msg.Detail = i18n.Sprintf(context.TODO(), "delete user %s from project %s members", user.Username, rel.Project.ProjectName)
		msg.ToUsers.
			Append(rel.UserID). // 自己
			Append(func() []uint {
				if rel.Role == models.ProjectRoleAdmin {
					return h.GetDataBase().ProjectAdmins(rel.ProjectID)
				}
				return nil
			}()...) // 项目管理员
		msg.AffectedUsers.Append(rel.UserID)
	})

	handlers.NoContent(c, nil)
}

// ListProjectEnvironment 获取属于Project的 Environment 列表
//	@Tags			Project
//	@Summary		获取属于 Project 的 Environment 列表
//	@Description	获取属于 Project 的 Environment 列表
//	@Accept			json
//	@Produce		json
//	@Param			project_id		path		uint																		true	"project_id"
//	@Param			preload			query		string																		false	"choices Creator,Cluster,Project,Applications,Users"
//	@Param			page			query		int																			false	"page"
//	@Param			size			query		int																			false	"page"
//	@Param			search			query		string																		false	"search in (EnvironmentName)"
//	@Param			containNSLabels	query		bool																		false	"是否包含命名空间标签"
//	@Success		200				{object}	handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Environment}}	"models.Environment"
//	@Router			/v1/project/{project_id}/environment [get]
//	@Security		JWT
func (h *ProjectHandler) ListProjectEnvironment(c *gin.Context) {
	var list []models.Environment
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	// 避免获取集群名空指针
	if !strings.Contains(query.Preload, "Cluster") {
		query.MustPreload([]string{"Cluster"})
	}
	cond := &handlers.PageQueryCond{
		Model:                  "Environment",
		SearchFields:           []string{"EnvironmentName"},
		PreloadFields:          []string{"Creator", "Cluster", "Project", "Applications", "Users"},
		PreloadSensitiveFields: PreloadSensitiveFields,
		Where:                  []*handlers.QArgs{handlers.Args("project_id = ?", c.Param(PrimaryKeyName))},
	}
	ctx := c.Request.Context()
	total, page, size, err := query.PageList(h.GetDB().WithContext(ctx), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	if containNSLabels, _ := strconv.ParseBool(c.Query("containNSLabels")); containNSLabels {
		eg := errgroup.Group{}
		for i := range list {
			index := i
			eg.Go(func() error {
				return h.Execute(ctx, list[index].Cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
					ns := corev1.Namespace{}
					if err := cli.Get(ctx, types.NamespacedName{Name: list[index].Namespace}, &ns); err != nil {
						return err
					}
					list[index].NSLabels = ns.Labels
					return nil
				})
			})
		}
		if err := eg.Wait(); err != nil {
			handlers.NotOK(c, err)
			return
		}
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// RetrieveProjectEnvironment 获取Project 的一个 Environment详情
//	@Tags			Project
//	@Summary		获取Project 的一个 Environment详情
//	@Description	获取Project 的一个 Environment详情
//	@Accept			json
//	@Produce		json
//	@Param			project_id		path		uint												true	"project_id"
//	@Param			environment_id	path		uint												true	"environment_id"
//	@Success		200				{object}	handlers.ResponseStruct{Data=models.Environment}	"models.Environment"
//	@Router			/v1/project/{project_id}/environment/{environment_id} [get]
//	@Security		JWT
func (h *ProjectHandler) RetrieveProjectEnvironment(c *gin.Context) {
	var env models.Environment
	if err := h.GetDB().WithContext(c.Request.Context()).First(&env, "project_id = ? and id = ?", c.Param(PrimaryKeyName), c.Param("environment_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, env)
}

// GetProjectResource 获取项目资源清单
//	@Tags			ResourceList
//	@Summary		获取项目资源清单
//	@Description	获取项目资源清单
//	@Accept			json
//	@Produce		json
//	@Param			project_id	path		uint														true	"project_id"
//	@Param			date		query		string														false	"date"
//	@Success		200			{object}	handlers.ResponseStruct{Data=[]models.EnvironmentResource}	"EnvironmentResource"
//	@Router			/v1/resources/project/{project_id} [get]
//	@Security		JWT
func (h *ProjectHandler) GetProjectResource(c *gin.Context) {
	dateTime, err := time.Parse(time.RFC3339, c.Query("date"))
	if err != nil {
		// 默认取到昨天的时间
		dateTime = time.Now().Add(-24 * time.Hour)
	}
	// 第二天的0点
	dayTime := utils.NextDayStartTime(dateTime)

	proj := models.Project{}
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Preload("Tenant").Where("id = ?", c.Param("project_id")).First(&proj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	envs := []models.Environment{}
	if err := h.GetDB().WithContext(ctx).Where("project_id = ?", proj.ID).Find(&envs).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	var list []models.EnvironmentResource
	for _, env := range envs {
		var envREs models.EnvironmentResource
		if err := h.GetDB().WithContext(ctx).
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
