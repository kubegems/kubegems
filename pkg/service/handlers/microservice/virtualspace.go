package microservice

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"istio.io/api/annotation"
	istioclinetworkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/server/define"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/msgbus"
	"kubegems.io/pkg/utils/pagination"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualSpaceHandler struct {
	define.ServerInterface
	Logger *log.Logger
}

var (
	SearchFields   = []string{"VirtualSpaceName"}
	FilterFields   = []string{"VirtualSpaceName", "VirtualSpaceID"}
	PreloadFields  = []string{"Environments", "Users"}
	OrderFields    = []string{"VirtualSpaceName", "ID"}
	ModelName      = "VirtualSpace"
	PrimaryKeyName = "virtualspace_id"
)

// ListVirtualSpace 列表 VirtualSpace
// @Tags VirtualSpace
// @Summary VirtualSpace列表
// @Description VirtualSpace列表
// @Accept json
// @Produce json
// @Param VirtualSpaceName query string false "VirtualSpaceName"
// @Param VirtualSpaceID query string false "VirtualSpaceID"
// @Param preload query string false "choices Environments,Users"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (VirtualSpaceName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.VirtualSpace}} "VirtualSpace"
// @Router /v1/virtualspace [get]
// @Security JWT
func (h *VirtualSpaceHandler) ListVirtualSpace(c *gin.Context) {
	var list []models.VirtualSpace
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "VirtualSpace",
		SearchFields:  SearchFields,
		PreloadFields: PreloadFields,
		PreloadSensitiveFields: map[string]string{
			"Users":        "id",
			"Environments": "id, project_id, cluster_id, creator_id, virtual_space_id",
		},
	}
	u, _ := h.GetContextUser(c)
	auth := h.GetCacheLayer().GetUserAuthority(u)
	if !auth.IsSystemAdmin {
		subQuery := h.GetDB().Table("virtual_space_user_rels").
			Select("virtual_space_id").
			Where("user_id = ?", u.ID)
		cond.Where = append(cond.Where, handlers.Args("id in (?)", subQuery))
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// GetVirtualSpace VirtualSpace详情
// @Tags VirtualSpace
// @Summary VirtualSpace详情
// @Description get VirtualSpace详情
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.VirtualSpace} "VirtualSpace"
// @Router /v1/virtualspace/{virtualspace_id} [get]
// @Security JWT
func (h *VirtualSpaceHandler) GetVirtualSpace(c *gin.Context) {
	// get vs
	vs := models.VirtualSpace{}
	if err := h.GetDB().
		Preload("Users").
		Preload("Environments.Cluster", func(tx *gorm.DB) *gorm.DB {
			return tx.Select("id, cluster_name")
		}).
		Preload("Environments.Project").
		First(&vs, c.Param("virtualspace_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, vs)
}

// PostVirtualSpace 创建VirtualSpace
// @Tags VirtualSpace
// @Summary 创建VirtualSpace
// @Description 创建VirtualSpace
// @Accept json
// @Produce json
// @Param param body models.VirtualSpace true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.VirtualSpace} "VirtualSpace"
// @Router /v1/virtualspace [post]
// @Security JWT
func (h *VirtualSpaceHandler) PostVirtualSpace(c *gin.Context) {
	var vs models.VirtualSpace
	if err := c.BindJSON(&vs); err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "创建", "虚拟空间", vs.VirtualSpaceName)
	h.SetExtraAuditData(c, models.ResVirtualSpace, vs.ID)

	u, _ := h.GetContextUser(c)
	vs.CreatedBy = u.Username
	vs.IsActive = true

	if err := h.GetDB().Save(&vs).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.GetCacheLayer().GetGlobalResourceTree().UpsertVirtualSpace(vs.ID, vs.VirtualSpaceName)
	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ResourceType(msgbus.VirtualSpace).
		ActionType(msgbus.Add).
		ResourceID(vs.ID).
		Content(fmt.Sprintf("创建了虚拟空间%s", vs.VirtualSpaceName)).
		SetUsersToSend(
			h.GetDataBase().SystemAdmins(),
		).
		Send()
	handlers.OK(c, vs)
}

// PutVirtualSpace 更新VirtualSpace
// @Tags VirtualSpace
// @Summary 更新VirtualSpace
// @Description 更新VirtualSpace
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param param body models.VirtualSpace true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.VirtualSpace} "VirtualSpace"
// @Router /v1/virtualspace/{virtualspace_id} [put]
// @Security JWT
func (h *VirtualSpaceHandler) PutVirtualSpace(c *gin.Context) {
	var obj models.VirtualSpace
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "虚拟空间", obj.VirtualSpaceName)
	h.SetExtraAuditData(c, models.ResVirtualSpace, obj.ID)

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
	h.GetCacheLayer().GetGlobalResourceTree().UpsertVirtualSpace(obj.ID, obj.VirtualSpaceName)
	handlers.OK(c, obj)
}

// EnableOrDisableVirtualSpace 激活/禁用VirtualSpace
// @Tags VirtualSpace
// @Summary 激活/禁用VirtualSpace
// @Description 激活/禁用VirtualSpace
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param enable query bool true "激活/禁用镜像仓库"
// @Success 200 {object} handlers.ResponseStruct{Data=models.VirtualSpace} "VirtualSpace"
// @Router /v1/virtualspace/{virtualspace_id} [patch]
// @Security JWT
func (h *VirtualSpaceHandler) EnableOrDisableVirtualSpace(c *gin.Context) {
	var obj models.VirtualSpace
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	enable, _ := strconv.ParseBool(c.Query("enable"))
	if enable {
		h.SetAuditData(c, "激活", "虚拟空间", obj.VirtualSpaceName)
	} else {
		h.SetAuditData(c, "禁用", "虚拟空间", obj.VirtualSpaceName)
	}
	h.SetExtraAuditData(c, models.ResVirtualSpace, obj.ID)

	obj.IsActive = enable
	if err := h.GetDB().Save(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// DeleteVirtualSpace 删除 VirtualSpace
// @Tags VirtualSpace
// @Summary 删除 VirtualSpace
// @Description 删除 VirtualSpace
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Success 200 {object} handlers.ResponseStruct "resp"
// @Router /v1/virtualspace/{virtualspace_id} [delete]
// @Security JWT
func (h *VirtualSpaceHandler) DeleteVirtualSpace(c *gin.Context) {
	// get vs
	vs := models.VirtualSpace{}
	if err := h.GetDB().Preload("Environments").First(&vs, c.Param("virtualspace_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "删除", "虚拟空间", vs.VirtualSpaceName)
	h.SetExtraAuditData(c, models.ResVirtualSpace, vs.ID)

	// 环境外键ONDELETE: SET NULL
	// 用户外键为级联删除，不需要校验
	if len(vs.Environments) > 0 {
		handlers.NotOK(c, fmt.Errorf("虚拟空间%s还有关联的环境", vs.VirtualSpaceName))
		return
	}
	if err := h.GetDB().Delete(&vs).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.GetCacheLayer().GetGlobalResourceTree().DelVirtualSpace(vs.ID)
	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ResourceType(msgbus.VirtualSpace).
		ActionType(msgbus.Delete).
		ResourceID(vs.ID).
		Content(fmt.Sprintf("了虚拟空间%s", vs.VirtualSpaceName)).
		SetUsersToSend(
			h.GetDataBase().SystemAdmins(),
		).
		Send()

	handlers.OK(c, "")
}

// ListEnvironment 获取虚拟空间下的环境
// @Tags VirtualSpace
// @Summary 获取虚拟空间下的环境
// @Description 获取虚拟空间下的环境
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param preload query string false "choices Project,Cluster"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Environment}} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment [get]
// @Security JWT
func (h *VirtualSpaceHandler) ListEnvironment(c *gin.Context) {
	var list []models.Environment
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "Environment",
		PreloadFields: []string{"Project", "Cluster"},
		PreloadSensitiveFields: map[string]string{
			"Cluster": "id, cluster_name",
		},
		Where: []*handlers.QArgs{
			handlers.Args("virtual_space_id = ?", c.Param("virtualspace_id")),
		},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// AddEnvironment 向虚拟空间增加环境
// @Tags VirtualSpace
// @Summary 向虚拟空间增加环境
// @Description 向虚拟空间增加环境
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param param body models.Environment true "环境"
// @Success 200 {object} handlers.ResponseStruct "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment [post]
// @Security JWT
func (h *VirtualSpaceHandler) AddEnvironment(c *gin.Context) {
	// get and check env
	env := models.Environment{}
	if err := c.BindJSON(&env); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Preload("Cluster").First(&env).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if env.VirtualSpaceID != nil {
		handlers.NotOK(c, fmt.Errorf("环境%s已加入其他虚拟空间", env.EnvironmentName))
		return
	}

	// get vs
	vs := models.VirtualSpace{}
	if err := h.GetDB().First(&vs, c.Param("virtualspace_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		env.VirtualSpaceID = &vs.ID
		if err := tx.Save(&env).Error; err != nil {
			return err
		}
		ctx := c.Request.Context()
		return h.ensureEnvirment(ctx, &env, &vs, true)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, vs)
}

// AddEnvironment 从虚拟空间删除环境
// @Tags VirtualSpace
// @Summary 从虚拟空间删除环境
// @Description 从虚拟空间删除环境
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Success 200 {object} handlers.ResponseStruct "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id [delete]
// @Security JWT
func (h *VirtualSpaceHandler) RemoveEnvironment(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		if env.VirtualSpace == nil {
			return nil, errors.New("no virtualspace found")
		}
		vs := *env.VirtualSpace
		// 只删除关联
		if err := h.GetDB().Transaction(func(db *gorm.DB) error {
			if err := db.Model(&vs).Association("Environments").Delete(&env); err != nil {
				return err
			}
			return h.ensureEnvirment(ctx, &env, &vs, false)
		}); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

// GetWorkload workload详情
// @Tags VirtualSpace
// @Summary workload详情
// @Description workload详情（带相关service）
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param kind query string true "workload类型, Deployment, StatefulSet, DaemonSet"
// @Success 200 {object} handlers.ResponseStruct{WorkloadDetails} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/{environment_id}/workload/{name} [get]
// @Security JWT
func (h *VirtualSpaceHandler) GetWorkload(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		// list workload
		cli, err := h.clientOf(ctx, env.Cluster.ClusterName)
		if err != nil {
			h.Logger.Errorf("failed get client of cluster %s: %s", env.Cluster.ClusterName, err.Error())
			return nil, err
		}

		listRelatedServiceMatchLabels := func(l map[string]string) []corev1.Service {
			listopts := []client.ListOption{client.InNamespace(env.Namespace)}
			svcs := &corev1.ServiceList{}
			_ = cli.List(ctx, svcs, listopts...)
			list := []corev1.Service{}
			for i, svc := range svcs.Items {
				// 使用service 的selector选择 workload 的template labels
				if labels.Set(svc.Spec.Selector).AsSelector().Matches(labels.Set(l)) {
					list = append(list, svcs.Items[i])
				}
			}

			// sort
			sort.Slice(list, func(i, j int) bool {
				return len(list[i].Name) < len(list[j].Name)
			})
			return list
		}

		objmeta := v1.ObjectMeta{Name: c.Param("name"), Namespace: env.Namespace}
		switch c.Query("kind") {
		case WorkloadKindDeployment, "":
			workload := &appsv1.Deployment{ObjectMeta: objmeta}
			if err := cli.Get(ctx, client.ObjectKeyFromObject(workload), workload); err != nil {
				return nil, err
			}
			svcs := listRelatedServiceMatchLabels(workload.Spec.Template.Labels)
			return WorkloadDetails{Object: workload, Services: svcs}, nil
		case WorkloadKindStatefulSet:
			workload := &appsv1.StatefulSet{ObjectMeta: objmeta}
			if err := cli.Get(ctx, client.ObjectKeyFromObject(workload), workload); err != nil {
				return nil, err
			}
			svcs := listRelatedServiceMatchLabels(workload.Spec.Template.Labels)
			return WorkloadDetails{Object: workload, Services: svcs}, nil
		case WorkloadKindDaemonSet:
			workload := &appsv1.DaemonSet{ObjectMeta: objmeta}
			if err := cli.Get(ctx, client.ObjectKeyFromObject(workload), workload); err != nil {
				return nil, err
			}
			svcs := listRelatedServiceMatchLabels(workload.Spec.Template.Labels)
			return WorkloadDetails{Object: workload, Services: svcs}, nil
		}

		return nil, nil
	})
}

// ListWorkload workload列表
// @Tags VirtualSpace
// @Summary workload列表
// @Description workload列表
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param kind query string true "workload类型, Deployment, StatefulSet, DaemonSet"
// @Param search query string true "workload名称"
// @Success 200 {object} handlers.ResponseStruct{Data=pagination.PageData{List=WorkloadDetails}} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/workload [get]
// @Security JWT
func (h *VirtualSpaceHandler) ListWorkload(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		list := []WorkloadDetails{}
		// list workload
		cli, err := h.clientOf(ctx, env.Cluster.ClusterName)
		if err != nil {
			h.Logger.Errorf("failed get client of cluster %s: %s", env.Cluster.ClusterName, err.Error())
			return nil, err
		}

		listopts := []client.ListOption{client.InNamespace(env.Namespace)}

		switch c.Query("kind") {
		case WorkloadKindDeployment, "":
			workloadlist := &appsv1.DeploymentList{}
			if err := cli.List(ctx, workloadlist, listopts...); err != nil {
				return nil, err
			}
			for i, workload := range workloadlist.Items {
				list = append(list, toWorkloadDetails(&workloadlist.Items[i], env, workload.Spec.Template))
			}
		case WorkloadKindStatefulSet:
			workloadlist := &appsv1.StatefulSetList{}
			if err := cli.List(ctx, workloadlist, listopts...); err != nil {
				return nil, err
			}
			for i, workload := range workloadlist.Items {
				list = append(list, toWorkloadDetails(&workloadlist.Items[i], env, workload.Spec.Template))
			}
		case WorkloadKindDaemonSet:
			workloadlist := &appsv1.DaemonSetList{}
			if err := cli.List(ctx, workloadlist, listopts...); err != nil {
				return nil, err
			}
			for i, workload := range workloadlist.Items {
				list = append(list, toWorkloadDetails(&workloadlist.Items[i], env, workload.Spec.Template))
			}
		}

		ret := pagination.NewPageDataFromContextReflect(c, list)
		tmp := ret.List.([]pagination.SortAndSearchAble)
		sort.Slice(tmp, func(i, j int) bool {
			return tmp[i].(WorkloadDetails).IstioInjected
		})
		ret.List = tmp
		return ret, nil
	})
}

// InjectIstioSidecar 注入sidecar
// @Tags VirtualSpace
// @Summary 注入istio控制
// @Description 注入istio控制
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param kind query string true "workload类型, Deployment, StatefulSet, DaemonSet"
// @Param inject query bool true "注入(true) or 取消注入(false)"
// @Success 200 {object} handlers.ResponseStruct "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/workload/{name}/istiosidecar [put]
// @Security JWT
func (h *VirtualSpaceHandler) InjectIstioSidecar(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		var obj client.Object

		// get workload
		cli, err := h.clientOf(ctx, env.Cluster.ClusterName)
		if err != nil {
			return nil, err
		}

		workloadName := c.Param("name")
		switch c.Query("kind") {
		case WorkloadKindDeployment, "":
			obj = &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Name: workloadName, Namespace: env.Namespace}}
		case WorkloadKindStatefulSet:
			obj = &appsv1.StatefulSet{ObjectMeta: v1.ObjectMeta{Name: workloadName, Namespace: env.Namespace}}
		case WorkloadKindDaemonSet:
			obj = &appsv1.DaemonSet{ObjectMeta: v1.ObjectMeta{Name: workloadName, Namespace: env.Namespace}}
		}

		var patch client.Patch
		inject, err := strconv.ParseBool(c.Query("inject"))
		if err != nil {
			return nil, err
		}
		if inject {
			patch = client.RawPatch(types.JSONPatchType, istioInjectTemplateAnnotationPatch)
		} else {
			patch = client.RawPatch(types.JSONPatchType, istioUninjectTemplateAnnotationPatch)
		}
		if err := cli.Patch(ctx, obj, patch); err != nil {
			return nil, err
		}
		return obj, nil
	})
}

// ListVirtualSpaceUser 获取属于VirtualSpace的 User 列表
// @Tags VirtualSpace
// @Summary 获取属于 VirtualSpace 的 User 列表
// @Description 获取属于 VirtualSpace 的 User 列表
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (Username,Email)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}} "models.User"
// @Router /v1/virtualspace/{virtualspace_id}/user [get]
// @Security JWT
func (h *VirtualSpaceHandler) ListVirtualSpaceUser(c *gin.Context) {
	var list []models.User
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:        "User",
		SearchFields: []string{"Username", "Email"},
		Select:       handlers.Args("users.*, virtual_space_user_rels.role"),
		Join:         handlers.Args("join virtual_space_user_rels on virtual_space_user_rels.user_id = users.id"),
		Where:        []*handlers.QArgs{handlers.Args("virtual_space_user_rels.virtual_space_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// PostVirtualSpaceUser 在User和VirtualSpace间添加关联关系
// @Tags VirtualSpace
// @Summary 在User和VirtualSpace间添加关联关系
// @Description 在User和VirtualSpace间添加关联关系
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param param body models.VirtualSpaceUserRels  true "表单"`
// @Success 200 {object} handlers.ResponseStruct{Data=models.VirtualSpaceUserRels} "models.User"
// @Router /v1/virtualspace/{virtualspace_id}/user [post]
// @Security JWT
func (h *VirtualSpaceHandler) PostVirtualSpaceUser(c *gin.Context) {
	var rel models.VirtualSpaceUserRels
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

	h.GetDB().Preload("VirtualSpace").First(&rel, rel.ID)
	h.SetAuditData(c, "添加", "虚拟空间成员", fmt.Sprintf("虚拟空间[%v]/成员[%v]", rel.VirtualSpace.VirtualSpaceName, user.Username))
	h.SetExtraAuditData(c, models.ResVirtualSpace, rel.VirtualSpaceID)
	handlers.OK(c, rel)
}

// DeleteVirtualSpaceUser 删除 User 和 VirtualSpace 的关系
// @Tags VirtualSpace
// @Summary 删除 User 和 VirtualSpace 的关系
// @Description 删除 User 和 VirtualSpace 的关系
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct "models.User"
// @Router /v1/virtualspace/{virtualspace_id}/user/{user_id} [delete]
// @Security JWT
func (h *VirtualSpaceHandler) DeleteVirtualSpaceUser(c *gin.Context) {
	var rel models.VirtualSpaceUserRels
	if err := h.GetDB().Preload("VirtualSpace").First(&rel, "virtual_space_id =? and user_id = ?", c.Param(PrimaryKeyName), c.Param("user_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Delete(&rel).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	user := models.User{}
	h.GetDB().Preload("SystemRole").First(&user, c.Param("user_id"))
	h.GetCacheLayer().FlushUserAuthority(&user)

	h.SetAuditData(c, "删除", "虚拟空间成员", fmt.Sprintf("虚拟空间[%v]/成员[%v]", rel.VirtualSpace.VirtualSpaceName, user.Username))
	h.SetExtraAuditData(c, models.ResVirtualSpace, rel.VirtualSpaceID)

	handlers.OK(c, nil)
}

type WorkloadDetails struct {
	Name          string           `json:"name,omitempty"`
	Kind          string           `json:"kind,omitempty"`
	Cluster       string           `json:"cluster,omitempty"`
	Namespace     string           `json:"namespace,omitempty"`
	Tenant        string           `json:"tenant,omitempty"`
	VirtualSpace  string           `json:"virtualSpace,omitempty"`
	Environment   string           `json:"environment,omitempty"`
	IstioInjected bool             `json:"istioInjected,omitempty"`
	VirtualDomain []string         `json:"virtualDomain,omitempty"`
	Images        []string         `json:"images,omitempty"`
	IstioVersion  string           `json:"istioVersion,omitempty"`
	Services      []corev1.Service `json:"services,omitempty"` // 关联的service
	client.Object                  // 原始数据
}

var _ pagination.SortAndSearchAble = WorkloadDetails{}

const (
	WorkloadKindDeployment  = "Deployment"
	WorkloadKindStatefulSet = "StatefulSet"
	WorkloadKindDaemonSet   = "DaemonSet"
)

// 通过 patch template 实现在pod上增加注入标记
var istioInjectTemplateAnnotationPatch = []byte(`[
    {
        "op": "add",
        "path": "/spec/template/metadata/annotations",
        "value": {
            "sidecar.istio.io/inject": "true"
        }
    }
]`)

// 转义 https://datatracker.ietf.org/doc/html/rfc6901#section-3
var istioUninjectTemplateAnnotationPatch = []byte(`[
    {
        "op": "remove",
        "path": "/spec/template/metadata/annotations/sidecar.istio.io~1inject"
    }
]`)

func toWorkloadDetails(obj client.Object, env models.Environment, podtemplate corev1.PodTemplateSpec) WorkloadDetails {
	return WorkloadDetails{
		Cluster:       env.Cluster.ClusterName,
		Name:          obj.GetName(),
		Namespace:     obj.GetNamespace(),
		Kind:          obj.GetObjectKind().GroupVersionKind().Kind,
		Tenant:        env.Project.Tenant.TenantName,
		VirtualSpace:  env.VirtualSpace.VirtualSpaceName,
		Environment:   env.EnvironmentName,
		IstioInjected: isIstioInjected(podtemplate),
		VirtualDomain: []string{},
		IstioVersion:  istioVersion(podtemplate),
		Images:        containerImages(podtemplate),
		Object:        obj,
	}
}

func containerImages(podtemplate corev1.PodTemplateSpec) []string {
	images := []string{}
	for _, container := range podtemplate.Spec.Containers {
		images = append(images, container.Image)
	}
	return images
}

func istioVersion(podtemplate corev1.PodTemplateSpec) string {
	return podtemplate.ObjectMeta.GetLabels()["version"]
}

func isIstioInjected(podtemplate corev1.PodTemplateSpec) bool {
	annos := podtemplate.Annotations
	// https://github.com/istio/istio/blob/release-1.11/pkg/kube/inject/inject.go#L194
	inject := false
	switch strings.ToLower(annos[annotation.SidecarInject.Name]) {
	// http://yaml.org/type/bool.html
	case "y", "yes", "true", "on":
		inject = true
	}
	return inject
}

const GemsAnnotationVirtualspace = "gems.cloudminds.com/virtualspace"

// 保证环境在虚拟空间中的存在/不存在
func (h *VirtualSpaceHandler) ensureEnvirment(ctx context.Context, env *models.Environment, vs *models.VirtualSpace, exist bool) error {
	remove := !exist
	// 将环境添加至虚拟空间时，需要将同个集群下相同虚拟空间的空间设置sidecar连通
	if env.Cluster == nil {
		return errors.New("invalid environment, no cluster found")
	}

	cluster := env.Cluster.ClusterName
	namespace := env.Namespace
	virtualspacename := vs.VirtualSpaceName

	cli, err := h.clientOf(ctx, cluster)
	if err != nil {
		return err
	}

	ns := &corev1.Namespace{}
	if err := cli.Get(ctx, client.ObjectKey{Name: namespace}, ns); err != nil {
		return err
	}
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}

	if remove {
		if val, ok := ns.Annotations[GemsAnnotationVirtualspace]; ok {
			vss := strings.Split(val, ",")
			found := false
			for i, v := range vss {
				if v == virtualspacename {
					found = true
					vss = append(vss[:i], vss[i+1:]...)
				}
			}
			if !found {
				return nil
			}

			// issue:controller#3 即使没有虚拟空间，也要保留key
			ns.Annotations[GemsAnnotationVirtualspace] = strings.Join(vss, ",")
		}
	} else {
		// 将在 namespace 上增加 annotation 标记该空间属于哪些虚拟空间(假设空间可以存在与多个虚拟空间)
		if val, ok := ns.Annotations[GemsAnnotationVirtualspace]; ok {
			vss := strings.Split(val, ",")

			for _, v := range vss {
				// 已经存在了
				if v == virtualspacename {
					return nil
				}
			}
			vss = append(vss, virtualspacename)
			ns.Annotations[GemsAnnotationVirtualspace] = strings.Join(vss, ",")
		} else {
			ns.Annotations[GemsAnnotationVirtualspace] = virtualspacename
		}
	}

	return cli.Update(ctx, ns)

	// 后续处理交由 controller 完成

	// controller 需要寻找到相同虚拟空间的空间并在每个空间中生成sidecar
	// 对于没有被标记的空间删除其空间中生成的sidecar
}

// 通过 patch template 实现在pod上增加注入标记
var virtualdomainInjectTemplateAnnotationPatchTemplate = string(`[
    {
        "op": "add",
        "path": "/metadata/annotations/gems.cloudminds.com~1virtualdomain",
        "value": "%s"
    }
]`)

var virtualdomainUninjectTemplateAnnotationPatch = []byte(`[
    {
        "op": "remove",
        "path": "/metadata/annotations/gems.cloudminds.com~1virtualdomain"
    }
]`)

// InjectIstioSidecar 注入虚拟域名
// @Tags VirtualSpace
// @Summary 设置虚拟域名
// @Description 设置虚拟域名
// @Accept json
// @Produce json
// @Param virtualspace_id 	path 	uint	true 	"virtualspace_id"
// @Param environment_id	path 	uint 	true 	"environment_id"
// @Param kind 				query 	string 	true 	"workload类型, Deployment, StatefulSet, DaemonSet"
// @Param virtualdomain 	query 	string 	true 	"需要被设置的虚拟域名，取消时设置为空"
// @Success 200 {object} 	handlers.ResponseStruct	"resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/workload/{name}/sidecar [put]
// @Security JWT
func (h *VirtualSpaceHandler) InjectVirtualDomain(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		// 为workload 设置虚拟空间会为其 *同名service* 增加annotation: gems.cloudminds.com/virtualdomain={domain}
		cli, err := h.clientOf(ctx, env.Cluster.ClusterName)
		if err != nil {
			return nil, err
		}
		svc := &corev1.Service{ObjectMeta: v1.ObjectMeta{Name: c.Param("name"), Namespace: env.Namespace}}

		virtualdomain := c.Query("virtualdomain")

		patch := client.RawPatch(
			types.JSONPatchType,
			[]byte(fmt.Sprintf(virtualdomainInjectTemplateAnnotationPatchTemplate, virtualdomain)),
		)
		if err := cli.Patch(ctx, svc, patch); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("负载没有设置service: %w", err)
			}
			return nil, err
		}
		return svc, nil
		// 后续的处理交由各集群controller处理
		// controller 会为其生成serviceentry 和 更改其virtualservice hosts
	})
}

type envprocessfunc func(ctx context.Context, env models.Environment) (interface{}, error)

func (h *VirtualSpaceHandler) environmentProcess(c *gin.Context, decodebody interface{}, process envprocessfunc) {
	if decodebody != nil {
		if err := c.BindJSON(decodebody); err != nil {
			handlers.NotOK(c, err)
			return
		}
	}

	env := models.Environment{}
	if err := h.GetDB().Preload("VirtualSpace").Preload("Cluster").Preload("Project.Tenant").First(&env, c.Param("environment_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if strconv.Itoa(int(*env.VirtualSpaceID)) != c.Param("virtualspace_id") {
		handlers.NotOK(c, fmt.Errorf("环境%s不是该虚拟空间成员", env.EnvironmentName))
		return
	}

	respdata, err := process(c.Request.Context(), env)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if respdata != nil {
		handlers.OK(c, respdata)
	}
}

type IstioResources struct {
	Gateways        []istioclinetworkingv1alpha3.Gateway        `json:"gateways"`
	VirtualServices []istioclinetworkingv1alpha3.VirtualService `json:"virtualServices"`
	Sidecars        []istioclinetworkingv1alpha3.Sidecar        `json:"sidecars"`
	ServiceEntries  []istioclinetworkingv1alpha3.ServiceEntry   `json:"serviceEntries"`
}

// ListIstioResources 列举 istio 资源
// @Tags VirtualSpace
// @Summary 列举 istio 资源
// @Description 列举 istio 所以类型资源
// @Accept json
// @Produce json
// @Param virtualspace_id 	path 	uint	true 	"virtualspace_id"
// @Param environment_id	path 	uint 	true 	"environment_id"
// @Success 200 {object} 	handlers.ResponseStruct{Data=IstioResources}	"resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/istioresources [get]
// @Security JWT
func (h *VirtualSpaceHandler) ListIstioResources(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		clustername := env.Cluster.ClusterName
		namespace := env.Namespace

		// get workload
		cli, err := h.clientOf(ctx, clustername)
		if err != nil {
			return nil, err
		}

		// istio virtualservice
		vss := &istioclinetworkingv1alpha3.VirtualServiceList{}
		if err := cli.List(ctx, vss, client.InNamespace(namespace)); err != nil {
			return nil, err
		}

		// istio gateways
		gatways := &istioclinetworkingv1alpha3.GatewayList{}
		if err := cli.List(ctx, gatways, client.InNamespace(namespace)); err != nil {
			return nil, err
		}

		// istio sidecar
		sidecars := &istioclinetworkingv1alpha3.SidecarList{}
		if err := cli.List(ctx, sidecars, client.InNamespace(namespace)); err != nil {
			return nil, err
		}

		// istio serviceentry
		ses := &istioclinetworkingv1alpha3.ServiceEntryList{}
		if err := cli.List(ctx, ses, client.InNamespace(namespace)); err != nil {
			return nil, err
		}

		ret := IstioResources{
			Gateways:        gatways.Items,
			VirtualServices: vss.Items,
			Sidecars:        sidecars.Items,
			ServiceEntries:  ses.Items,
		}
		return ret, nil
	})
}

func (h *VirtualSpaceHandler) clientOf(ctx context.Context, cluster string) (*agents.TypedClient, error) {
	cli, err := h.GetAgentsClientSet().ClientOf(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cli.TypedClient, nil
}

func (h *VirtualSpaceHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/virtualspace", h.ListVirtualSpace)                                          // 所有用户能获取，俺权限返回
	rg.GET("/virtualspace/:virtualspace_id", h.CheckByVirtualSpaceID, h.GetVirtualSpace) // 虚拟空间成员
	rg.POST("/virtualspace", h.CheckIsSysADMIN, h.PostVirtualSpace)                      // 系统管理员才能创建
	rg.PUT("/virtualspace/:virtualspace_id", h.CheckByVirtualSpaceID, h.PutVirtualSpace)
	rg.PATCH("/virtualspace/:virtualspace_id", h.CheckByVirtualSpaceID, h.EnableOrDisableVirtualSpace)
	rg.DELETE("/virtualspace/:virtualspace_id", h.CheckByVirtualSpaceID, h.DeleteVirtualSpace)

	rg.GET("/virtualspace/:virtualspace_id/user", h.CheckByVirtualSpaceID, h.ListVirtualSpaceUser)
	rg.POST("/virtualspace/:virtualspace_id/user", h.CheckByVirtualSpaceID, h.PostVirtualSpaceUser)
	rg.DELETE("/virtualspace/:virtualspace_id/user/:user_id", h.CheckByVirtualSpaceID, h.DeleteVirtualSpaceUser)

	rg.GET("/virtualspace/:virtualspace_id/environment", h.CheckByVirtualSpaceID, h.ListEnvironment)
	rg.POST("/virtualspace/:virtualspace_id/environment", h.CheckByVirtualSpaceID, h.AddEnvironment)
	rg.DELETE("/virtualspace/:virtualspace_id/environment/:environment_id", h.CheckByVirtualSpaceID, h.RemoveEnvironment)
	rg.GET("/virtualspace/:virtualspace_id/environment/:environment_id/workload", h.CheckByVirtualSpaceID, h.ListWorkload)
	rg.GET("/virtualspace/:virtualspace_id/environment/:environment_id/workload/:name", h.CheckByVirtualSpaceID, h.GetWorkload)
	rg.PUT("/virtualspace/:virtualspace_id/environment/:environment_id/workload/:name/istiosidecar", h.CheckByVirtualSpaceID, h.InjectIstioSidecar)
	rg.PUT("/virtualspace/:virtualspace_id/environment/:environment_id/workload/:name/virtualdomain", h.CheckByVirtualSpaceID, h.InjectVirtualDomain)
	rg.GET("/virtualspace/:virtualspace_id/environment/:environment_id/istioresources", h.CheckByVirtualSpaceID, h.ListIstioResources)

	// service
	rg.GET("/virtualspace/:virtualspace_id/environment/:environment_id/service", h.CheckByVirtualSpaceID, h.ListServices)
	rg.GET("/virtualspace/:virtualspace_id/environment/:environment_id/service/:service_name", h.CheckByVirtualSpaceID, h.GetService)
	rg.POST("/virtualspace/:virtualspace_id/environment/:environment_id/service/:service_name/request_routing", h.CheckByVirtualSpaceID, h.ServiceRequestRouting)
	rg.POST("/virtualspace/:virtualspace_id/environment/:environment_id/service/:service_name/fault_injection", h.CheckByVirtualSpaceID, h.ServiceFaultInjection)
	rg.POST("/virtualspace/:virtualspace_id/environment/:environment_id/service/:service_name/traffic_shifting", h.CheckByVirtualSpaceID, h.ServiceTrafficShifting)
	rg.POST("/virtualspace/:virtualspace_id/environment/:environment_id/service/:service_name/tcp_traffic_shifting", h.CheckByVirtualSpaceID, h.ServiceTCPTrafficShifting)
	rg.POST("/virtualspace/:virtualspace_id/environment/:environment_id/service/:service_name/request_timeouts", h.CheckByVirtualSpaceID, h.ServiceRequestTimeout)
	rg.POST("/virtualspace/:virtualspace_id/environment/:environment_id/service/:service_name/reset", h.CheckByVirtualSpaceID, h.ServicetReset)
	// kiali
	rg.Any("/virtualspace/:virtualspace_id/environment/:environment_id/kiali/*path", h.CheckByVirtualSpaceID, h.KialiAPI)
}
