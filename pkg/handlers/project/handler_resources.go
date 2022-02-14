package projecthandler

/*
	各个维度的资源统计模块
	1. 集群
		集群实际物理资源总量
		集群请求总量
		集群限制总量
		集群实际使用量
	2. 租户 (租户下所有项目的总量)
		租户-集群 请求总量
		租户-集群 限制总量
		租户-集群 实际使用量
	2. 项目 (项目下所有环境的总量)
		请求总量
		限制总量
		实际使用量
	3. 环境
		请求总量
		限制总量
		实际使用量
*/

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	gemsv1beta1 "github.com/kubegems/gems/pkg/apis/gems/v1beta1"
	"github.com/kubegems/gems/pkg/handlers"
	"github.com/kubegems/gems/pkg/kubeclient"
	gemlabels "github.com/kubegems/gems/pkg/labels"
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/utils/msgbus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type EnvironmentAggregateData struct {
	Env           models.Environment
	ResourceQuota interface{}
}

type ClusterAggregateData struct {
	ClusterName   string
	ClusterID     uint
	NetworkPolicy interface{}
	Environments  []EnvironmentAggregateData
}

// ProjectNoneResourceStatistics 项目非资源类型数据统计
// @Tags Project
// @Summary 项目非资源类型数据统计
// @Description 项目非资源类型数据统计
// @Accept json
// @Produce json
// @Param project_id path int true "project_id"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "statistics"
// @Router /v1/project/{project_id}/none_resource_statistics [get]
// @Security JWT
func (h *ProjectHandler) ProjectNoneResourceStatistics(c *gin.Context) {
	var (
		proj      models.Project
		appCount  int64
		envCount  int64
		userCount int64
	)
	if err := h.GetDB().First(&proj, c.Param("project_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	appCount = h.GetDB().Model(&proj).Association("Applications").Count()
	envCount = h.GetDB().Model(&proj).Association("Environments").Count()
	userCount = h.GetDB().Model(&proj).Association("Users").Count()
	handlers.OK(c, gin.H{
		"ApplicationCount": appCount,
		"EnvironmentCount": envCount,
		"UserCount":        userCount,
	})
}

// ProjectStatistics 项目资源统计
// @Tags Project
// @Summary 项目资源统计
// @Description 项目资源统计
// @Accept json
// @Produce json
// @Param project_id path int true "project_id"
// @Param aggregate query string false "是否聚合(yes,no;default no)"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "statistics"
// @Router /v1/project/{project_id}/statistics [get]
// @Security JWT
func (h *ProjectHandler) ProjectStatistics(c *gin.Context) {
	var proj models.Project
	if err := h.GetDB().Preload("Environments").Preload("Environments.Cluster").First(&proj, c.Param("project_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	envTotal := len(proj.Environments)
	if envTotal == 0 {
		handlers.OK(c, GetResourceCount())
		return
	}
	ret := make([]*v1.ResourceQuota, len(proj.Environments))
	wg := sync.WaitGroup{}
	wg.Add(envTotal)
	for idx, env := range proj.Environments {
		go func(env *models.Environment, idx int) {
			rq, err := kubeclient.GetClient().GetResourceQuota(env.Cluster.ClusterName, env.Namespace, "default", nil)
			if err != nil {
				ret[idx] = nil
			}
			if rq == nil {
				ret[idx] = nil
			}
			ret[idx] = rq
			wg.Done()
		}(env, idx)
	}
	wg.Wait()
	if c.Query("aggregate") == "yes" {
		nsret := []map[string]interface{}{}
		for idx, i := range ret {
			nsret = append(nsret, map[string]interface{}{
				"Resource":        GetResourceCount(i),
				"EnvironmentName": proj.Environments[idx].EnvironmentName,
				"Namespace":       proj.Environments[idx].Namespace,
			})
		}
		handlers.OK(c, nsret)
	} else {
		handlers.OK(c, GetResourceCount(ret...))
	}
}

// EnvironmentStatistics 项目环境资源统计
// @Tags Project
// @Summary 项目环境资源统计
// @Description 项目环境资源统计
// @Accept json
// @Produce json
// @Param project_id path int true "project_id"
// @Param environment_id path int true "environment_id"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "statistics"
// @Router /v1/project/{project_id}/environment/{environment_id}/statistics [get]
// @Security JWT
func (h *ProjectHandler) EnvironmentStatistics(c *gin.Context) {
	var env models.Environment
	if err := h.GetDB().Preload("Cluster").First(&env, "project_id = ? and id = ?", c.Param("project_id"), c.Param("environment_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	rq, err := kubeclient.GetClient().GetResourceQuota(env.Cluster.ClusterName, env.Namespace, "default", nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, GetResourceCount(rq))
}

// EnvironmentStatistics 项目环境资源top N
// @Tags Project
// @Summary 项目环境资源top N
// @Description 项目环境资源top N
// @Accept json
// @Produce json
// @Param project_id path int true "project_id"
// @Param environment_id path int true "environment_id"
// @Param n query int false "top n"
// @Param by query string false "排序字段，默认cpu; choice:cpu,memory"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "top"
// @Router /v1/project/{project_id}/environment/{environment_id}/top [get]
// @Security JWT
func (h *ProjectHandler) EnvironmentStatisticsTop(c *gin.Context) {
	var (
		env models.Environment
		ret []interface{}
	)
	if err := h.GetDB().Preload("Cluster").First(&env, "project_id = ? and id = ?", c.Param("project_id"), c.Param("environment_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	podMetrics, err := kubeclient.GetClient().GetPodsMetrics(env.Cluster.ClusterName, env.Namespace)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	for _, mc := range *podMetrics {
		podUsage := containerTotal(mc.Containers)
		ret = append(ret, gin.H{
			"pod": mc.Name,
			"res": podUsage,
		})
	}
	handlers.OK(c, ret)
}

type ResourceCount struct {
	Total v1.ResourceList
	Used  v1.ResourceList
}

func GetResourceCount(rqs ...*v1.ResourceQuota) *ResourceCount {
	hardRes := NewEmptyResoureList()
	usedRes := NewEmptyResoureList()
	hardRes.Total("hard", rqs...)
	usedRes.Total("used", rqs...)
	return &ResourceCount{
		Total: hardRes.AsResourceList(),
		Used:  usedRes.AsResourceList(),
	}
}

type Res map[v1.ResourceName]int64

func NewEmptyResoureList() Res {
	return Res{
		v1.ResourceCPU:                      0,
		v1.ResourceMemory:                   0,
		v1.ResourceStorage:                  0,
		v1.ResourcePods:                     0,
		v1.ResourceServices:                 0,
		v1.ResourceReplicationControllers:   0,
		v1.ResourceQuotas:                   0,
		v1.ResourceSecrets:                  0,
		v1.ResourceConfigMaps:               0,
		v1.ResourcePersistentVolumeClaims:   0,
		v1.ResourceServicesNodePorts:        0,
		v1.ResourceServicesLoadBalancers:    0,
		v1.ResourceRequestsCPU:              0,
		v1.ResourceRequestsMemory:           0,
		v1.ResourceRequestsStorage:          0,
		v1.ResourceRequestsEphemeralStorage: 0,
		v1.ResourceLimitsCPU:                0,
		v1.ResourceLimitsMemory:             0,
		v1.ResourceLimitsEphemeralStorage:   0,
	}
}

func (r *Res) AsResourceList() v1.ResourceList {
	return v1.ResourceList{
		v1.ResourceCPU:                      *(resource.NewQuantity((*r)[v1.ResourceCPU], resource.DecimalSI)),
		v1.ResourceMemory:                   *(resource.NewQuantity((*r)[v1.ResourceMemory], resource.BinarySI)),
		v1.ResourceStorage:                  *(resource.NewQuantity((*r)[v1.ResourceStorage], resource.BinarySI)),
		v1.ResourcePods:                     *(resource.NewQuantity((*r)[v1.ResourcePods], resource.DecimalSI)),
		v1.ResourceServices:                 *(resource.NewQuantity((*r)[v1.ResourceServices], resource.DecimalSI)),
		v1.ResourceReplicationControllers:   *(resource.NewQuantity((*r)[v1.ResourceReplicationControllers], resource.DecimalSI)),
		v1.ResourceQuotas:                   *(resource.NewQuantity((*r)[v1.ResourceQuotas], resource.DecimalSI)),
		v1.ResourceSecrets:                  *(resource.NewQuantity((*r)[v1.ResourceSecrets], resource.DecimalSI)),
		v1.ResourceConfigMaps:               *(resource.NewQuantity((*r)[v1.ResourceConfigMaps], resource.DecimalSI)),
		v1.ResourcePersistentVolumeClaims:   *(resource.NewQuantity((*r)[v1.ResourcePersistentVolumeClaims], resource.DecimalSI)),
		v1.ResourceServicesNodePorts:        *(resource.NewQuantity((*r)[v1.ResourceServicesNodePorts], resource.DecimalSI)),
		v1.ResourceServicesLoadBalancers:    *(resource.NewQuantity((*r)[v1.ResourceServicesLoadBalancers], resource.DecimalSI)),
		v1.ResourceRequestsCPU:              *(resource.NewQuantity((*r)[v1.ResourceRequestsCPU], resource.DecimalSI)),
		v1.ResourceRequestsMemory:           *(resource.NewQuantity((*r)[v1.ResourceRequestsMemory], resource.BinarySI)),
		v1.ResourceRequestsStorage:          *(resource.NewQuantity((*r)[v1.ResourceRequestsStorage], resource.BinarySI)),
		v1.ResourceRequestsEphemeralStorage: *(resource.NewQuantity((*r)[v1.ResourceRequestsEphemeralStorage], resource.BinarySI)),
		v1.ResourceLimitsCPU:                *(resource.NewQuantity((*r)[v1.ResourceLimitsCPU], resource.DecimalSI)),
		v1.ResourceLimitsMemory:             *(resource.NewQuantity((*r)[v1.ResourceLimitsMemory], resource.BinarySI)),
		v1.ResourceLimitsEphemeralStorage:   *(resource.NewQuantity((*r)[v1.ResourceLimitsEphemeralStorage], resource.BinarySI)),
	}
}

func (r *Res) Total(rtype string, rqs ...*v1.ResourceQuota) {
	for _, rq := range rqs {
		if rq == nil {
			continue
		}
		var actionRes v1.ResourceList
		if rtype == "hard" {
			actionRes = rq.Status.Hard
		} else {
			actionRes = rq.Status.Used
		}
		if cpu := actionRes.Cpu(); cpu != nil {
			(*r)[v1.ResourceCPU] += cpu.AsDec().UnscaledBig().Int64()
		}
		if mem := actionRes.Memory(); mem != nil {
			(*r)[v1.ResourceMemory] += mem.AsDec().UnscaledBig().Int64()
		}
		if storage := actionRes.Storage(); storage != nil {
			(*r)[v1.ResourceStorage] += storage.AsDec().UnscaledBig().Int64()
		}
		if pods := actionRes.Pods(); pods != nil {
			(*r)[v1.ResourcePods] += pods.AsDec().UnscaledBig().Int64()
		}
		if services, exist := actionRes[v1.ResourceServices]; exist {
			(*r)[v1.ResourceServices] += services.AsDec().UnscaledBig().Int64()
		}
		if rc, exist := actionRes[v1.ResourceReplicationControllers]; exist {
			(*r)[v1.ResourceReplicationControllers] += rc.AsDec().UnscaledBig().Int64()
		}
		if quotas, exist := actionRes[v1.ResourceReplicationControllers]; exist {
			(*r)[v1.ResourceQuotas] += quotas.AsDec().UnscaledBig().Int64()
		}
		if secrets, exist := actionRes[v1.ResourceSecrets]; exist {
			(*r)[v1.ResourceSecrets] += secrets.AsDec().UnscaledBig().Int64()
		}
		if cm, exist := actionRes[v1.ResourceConfigMaps]; exist {
			(*r)[v1.ResourceConfigMaps] += cm.AsDec().UnscaledBig().Int64()
		}
		if pvc, exist := actionRes[v1.ResourcePersistentVolumeClaims]; exist {
			(*r)[v1.ResourcePersistentVolumeClaims] += pvc.AsDec().UnscaledBig().Int64()
		}
		if nodeport, exist := actionRes[v1.ResourceServicesNodePorts]; exist {
			(*r)[v1.ResourceServicesNodePorts] += nodeport.AsDec().UnscaledBig().Int64()
		}
		if lb, exist := actionRes[v1.ResourceServicesLoadBalancers]; exist {
			(*r)[v1.ResourceServicesLoadBalancers] += lb.AsDec().UnscaledBig().Int64()
		}
		if reqcpu, exist := actionRes[v1.ResourceRequestsCPU]; exist {
			(*r)[v1.ResourceRequestsCPU] += reqcpu.AsDec().UnscaledBig().Int64()
		}
		if reqmem, exist := actionRes[v1.ResourceRequestsMemory]; exist {
			(*r)[v1.ResourceRequestsMemory] += reqmem.AsDec().UnscaledBig().Int64()
		}
		if reqsto, exist := actionRes[v1.ResourceRequestsStorage]; exist {
			(*r)[v1.ResourceRequestsStorage] += reqsto.AsDec().UnscaledBig().Int64()
		}
		if reqesto, exist := actionRes[v1.ResourceRequestsEphemeralStorage]; exist {
			(*r)[v1.ResourceRequestsEphemeralStorage] += reqesto.AsDec().UnscaledBig().Int64()
		}
		if lmtcpu, exist := actionRes[v1.ResourceLimitsCPU]; exist {
			(*r)[v1.ResourceLimitsCPU] += lmtcpu.AsDec().UnscaledBig().Int64()
		}
		if lmtmem, exist := actionRes[v1.ResourceLimitsMemory]; exist {
			(*r)[v1.ResourceLimitsMemory] += lmtmem.AsDec().UnscaledBig().Int64()
		}
		if lmtesto, exist := actionRes[v1.ResourceLimitsEphemeralStorage]; exist {
			(*r)[v1.ResourceLimitsEphemeralStorage] += lmtesto.AsDec().UnscaledBig().Int64()
		}
	}
}

func GetNodeTotal(nodes *[]v1.Node, tquotalist *[]gemsv1beta1.TenantResourceQuota, nodeMetrics *[]v1beta1.NodeMetrics) map[string]interface{} {
	// 总容量
	capacity := v1.ResourceList{
		v1.ResourceCPU:              resource.MustParse("0"),
		v1.ResourceMemory:           resource.MustParse("0"),
		v1.ResourceEphemeralStorage: resource.MustParse("0"),
	}
	// 被租户分配的
	allocated := v1.ResourceList{
		v1.ResourceCPU:             resource.MustParse("0"),
		v1.ResourceMemory:          resource.MustParse("0"),
		v1.ResourceRequestsStorage: resource.MustParse("0"),
	}
	// 实际使用的
	used := v1.ResourceList{
		v1.ResourceCPU:             resource.MustParse("0"),
		v1.ResourceMemory:          resource.MustParse("0"),
		v1.ResourceRequestsStorage: resource.MustParse("0"),
	}

	for _, node := range *nodes {
		tcpu := capacity[v1.ResourceCPU]
		tcpu.Add(node.Status.Capacity.Cpu().DeepCopy())
		capacity[v1.ResourceCPU] = tcpu

		tmem := capacity[v1.ResourceMemory]
		tmem.Add(node.Status.Capacity.Memory().DeepCopy())
		capacity[v1.ResourceMemory] = tmem

		tstorage := capacity[v1.ResourceEphemeralStorage]
		tstorage.Add(node.Status.Capacity.StorageEphemeral().DeepCopy())
		capacity[v1.ResourceEphemeralStorage] = tstorage

	}

	for _, nodemetric := range *nodeMetrics {
		used[v1.ResourceCPU] = nodemetric.Usage.Cpu().DeepCopy()
		used[v1.ResourceMemory] = nodemetric.Usage.Memory().DeepCopy()
		used[v1.ResourceRequestsStorage] = nodemetric.Usage.StorageEphemeral().DeepCopy()
	}

	for _, trq := range *tquotalist {
		tcpu := allocated[v1.ResourceCPU]
		tcpu.Add(trq.Spec.Hard[v1.ResourceCPU])
		allocated[v1.ResourceCPU] = tcpu

		tmem := allocated[v1.ResourceMemory]
		tmem.Add(trq.Spec.Hard[v1.ResourceMemory])
		allocated[v1.ResourceMemory] = tmem

		tsto := allocated[v1.ResourceRequestsStorage]
		tsto.Add(trq.Spec.Hard[v1.ResourceRequestsStorage])
		allocated[v1.ResourceRequestsStorage] = tsto
	}

	return map[string]interface{}{
		"total":    capacity,
		"allocatd": allocated,
		"used":     used,
	}
}

func containerTotal(containers []v1beta1.ContainerMetrics) interface{} {
	ret := v1.ResourceList{
		v1.ResourceCPU:    *resource.NewQuantity(0, resource.BinarySI),
		v1.ResourceMemory: *resource.NewQuantity(0, resource.BinarySI),
	}
	for _, c := range containers {
		tcpu := ret[v1.ResourceCPU]
		tpcpu := &tcpu
		tpcpu.Add(c.Usage.Cpu().DeepCopy())
		ret[v1.ResourceCPU] = *tpcpu

		tmem := ret[v1.ResourceMemory]
		tpmem := &tmem
		tpmem.Add(c.Usage.Memory().DeepCopy())
		ret[v1.ResourceMemory] = *tpmem
	}
	res := Res{
		v1.ResourceCPU:    ret.Cpu().ScaledValue(resource.Milli),
		v1.ResourceMemory: ret.Memory().Value() / (1024 * 1024),
	}
	return res
}

// GetEnvironmentResourceQuota 单个环境下的资源统计[quota]
// @Tags Project
// @Summary 单个环境下的资源统计[quota]
// @Description 单个环境下的资源统计[quota]
// @Accept json
// @Produce json
// @Param project_id path int true "project_id"
// @Param environment_id path int true "environment_id"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "quota"
// @Router /v1/project/{project_id}/environment/{environment_id}/quota [get]
// @Security JWT
func (h *ProjectHandler) GetEnvironmentResourceQuota(c *gin.Context) {
	var (
		proj models.Project
		env  models.Environment
	)
	if err := h.GetDB().First(&proj, c.Param("project_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Preload("Cluster").First(&env, "id = ?", c.Param("environment_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	envAppCount := struct {
		AppCount      int
		EnvironmentID uint
	}{}
	sql := "select count(*) as app_count, environment_id from application_environment_rels where environment_id = ? group by environment_id"
	if err := h.GetDB().Table("application_environment_rels").Exec(sql, c.Param("environment_id")).Scan(&envAppCount).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	quota, err := kubeclient.GetClient().GetResourceQuota(env.Cluster.ClusterName, env.Namespace, "default", nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	ret := map[string]interface{}{
		"quota": quota,
		"statistics": map[string]interface{}{
			"applications": envAppCount.AppCount,
		},
	}

	handlers.OK(c, ret)
}

// GetEnvironmentResourceQuotas 获取项目下的环境资源统计列表[quota]
// @Tags Project
// @Summary 获取项目下的环境资源统计列表[quota]
// @Description 获取项目下的环境资源统计列表[quota]
// @Accept json
// @Produce json
// @Param project_id path int true "project_id"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "quota"
// @Router /v1/project/{project_id}/environment/_/quotas [get]
// @Security JWT
func (h *ProjectHandler) GetEnvironmentResourceQuotas(c *gin.Context) {
	projid, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	rets, err := h.getProjectNoAggretateQuota(projid)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, rets)
}

// GetProjectResourceQuota 获取单个项目资源统计[quota]
// @Tags Project
// @Summary 获取单个项目资源统计[quota]
// @Description 获取单个项目资源统计[quota]
// @Accept json
// @Produce json
// @Param project_id path int true "project_id"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "quota"
// @Router /v1/project/{project_id}/quota [get]
// @Security JWT
func (h *ProjectHandler) GetProjectResourceQuota(c *gin.Context) {
	projid, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	ret, err := h.getProjectAggretateQuota(projid)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// GetProjectListResourceQuotas 获取项目资源统计列表[quota]
// @Tags Project
// @Summary 获取项目资源统计列表[quota]
// @Description 获取项目资源统计列表[quota]
// @Accept json
// @Produce json
// @Param TenantID query string false "TenantID"
// @Param page query int false "page"
// @Param size query int false "page"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]object}} "quotas"
// @Router /v1/project/_/quotas [get]
// @Security JWT
func (h *ProjectHandler) GetProjectListResourceQuotas(c *gin.Context) {
	var (
		projects []models.Project
		ret      []interface{}
	)
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model: "Project",
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &projects)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	for _, proj := range projects {
		tmpret, err := h.getProjectAggretateQuota(int(proj.ID))
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		item := map[string]interface{}{
			"projectid":      proj.ID,
			"project":        proj.ProjectName,
			"resourceStatus": tmpret,
		}
		ret = append(ret, item)
	}
	handlers.OK(c, handlers.Page(total, ret, page, size))
}

type res struct {
	Quota    *v1.ResourceQuota      `json:"quota"`
	Resoruce map[string]interface{} `json:"resource"`
}

func (h *ProjectHandler) getProjectNoAggretateQuota(projectId int) (map[string]res, error) {
	ret := map[string]res{}
	var proj models.Project
	if err := h.GetDB().Preload("Tenant").Preload("Environments.Cluster").First(&proj, "id = ?", projectId).Error; err != nil {
		return nil, err
	}
	envids := []uint{}
	for _, env := range proj.Environments {
		envids = append(envids, env.ID)
	}
	envAppCount := []struct {
		AppCount      int
		EnvironmentID uint
	}{}
	sql := "select count(*) as app_count, environment_id from application_environment_rels where environment_id in (?) group by environment_id"
	if err := h.GetDB().Table("application_environment_rels").Exec(sql, envids).Scan(&envAppCount).Error; err != nil {
		return nil, err
	}
	countMap := map[uint]int{}
	for _, count := range envAppCount {
		countMap[count.EnvironmentID] = count.AppCount
	}

	for _, env := range proj.Environments {
		quota, err := kubeclient.GetClient().GetResourceQuota(env.Cluster.ClusterName, env.Namespace, "default", nil)
		if err != nil {
			ret[env.EnvironmentName] = res{
				Quota: nil,
				Resoruce: map[string]interface{}{
					"appliaction": countMap[env.ID],
				},
			}
		} else {
			ret[env.EnvironmentName] = res{
				Quota: quota,
				Resoruce: map[string]interface{}{
					"appliaction": countMap[env.ID],
				},
			}
		}
	}

	return ret, nil
}

type projectRes struct {
	ResourceQuotaStatus *v1.ResourceQuotaStatus `json:"quota"`
	Resource            map[string]interface{}  `json:"resource"`
}

// 获取项目在各个环境下的资源的聚合数据
func (h *ProjectHandler) getProjectAggretateQuota(projectId int) (*projectRes, error) {
	var proj models.Project
	if err := h.GetDB().Preload("Tenant").Preload("Environments.Cluster").First(&proj, "id = ?", projectId).Error; err != nil {
		return nil, err
	}
	var (
		appCount    int64
		envCount    int64
		personCount int64
	)
	h.GetDB().Table("applications").Where("project_id = ?", projectId).Count(&appCount)
	h.GetDB().Table("environments").Where("project_id = ?", projectId).Count(&envCount)
	h.GetDB().Table("project_user_rels").Where("project_id = ?", projectId).Count(&personCount)

	clusterMap := map[string][]string{}
	for _, env := range proj.Environments {
		if arr, exist := clusterMap[env.Cluster.ClusterName]; exist {
			arr = append(arr, env.Namespace)
			clusterMap[env.Cluster.ClusterName] = arr
		} else {
			clusterMap[env.Cluster.ClusterName] = []string{env.Namespace}
		}
	}
	ret := &projectRes{
		ResourceQuotaStatus: &v1.ResourceQuotaStatus{
			Hard: v1.ResourceList{},
			Used: v1.ResourceList{},
		},
		Resource: map[string]interface{}{
			"applications": appCount,
			"environments": envCount,
			"person":       personCount,
		},
	}
	labels := map[string]string{
		gemlabels.LabelTenant:  proj.Tenant.TenantName,
		gemlabels.LabelProject: proj.ProjectName,
	}
	for cluster := range clusterMap {
		quotas, err := kubeclient.GetClient().GetResourceQuotaList(cluster, "", labels)
		if err != nil {
			continue
		}
		for _, quota := range *quotas {
			for k, v := range quota.Status.Hard {
				if rv, rexist := ret.ResourceQuotaStatus.Hard[k]; rexist {
					rv.Add(v)
					ret.ResourceQuotaStatus.Hard[k] = rv
				} else {
					ret.ResourceQuotaStatus.Hard[k] = v.DeepCopy()
				}
			}
			for k, v := range quota.Status.Used {
				if rv, rexist := ret.ResourceQuotaStatus.Used[k]; rexist {
					rv.Add(v)
					ret.ResourceQuotaStatus.Used[k] = rv
				} else {
					ret.ResourceQuotaStatus.Used[k] = v.DeepCopy()
				}
			}
		}
	}
	return ret, nil
}

// PostProjectEnvironment 创建一个属于 Project 的Environment
// @Tags Project
// @Summary 创建一个属于 Project 的Environment
// @Description 创建一个属于 Project 的Environment
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param param body models.Environment true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Environment} "models.Environment"
// @Router /v1/project/{project_id}/environment [post]
// @Security JWT
func (h *ProjectHandler) PostProjectEnvironment(c *gin.Context) {
	var obj models.Project
	if err := h.GetDB().Preload("Tenant").First(&obj, c.Param("project_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	var env models.Environment
	if err := c.BindJSON(&env); err != nil {
		handlers.NotOK(c, err)
		return
	}
	var cluster models.Cluster
	if err := h.GetDB().First(&cluster, env.ClusterID).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	env.LimitRange = models.FillDefaultLimigrange(&env)
	if err := h.GetDB().Save(&env).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	t := h.GetCacheLayer().GetGlobalResourceTree()
	t.UpsertEnvironment(obj.ID, env.ID, env.EnvironmentName, cluster.ClusterName, env.Namespace)

	h.SetAuditData(c, "创建", "环境", env.EnvironmentName)
	h.SetExtraAuditData(c, models.ResEnvironment, env.ID)

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Add).
		ResourceType(msgbus.Environment).
		ResourceID(env.ID).
		Content(fmt.Sprintf("在租户%s/项目%s中创建了环境%s", obj.Tenant.TenantName, obj.ProjectName, env.EnvironmentName)).
		SetUsersToSend(
			h.GetDataBase().ProjectAdmins(obj.ID),
		).
		Send()
	handlers.OK(c, env)
}

// ProjectEnvironments 获取项目下环境列表,按照集群聚合,同时获取集群的下的租户网络策略
// @Tags Project
// @Summary 获取项目下环境列表,按照集群聚合,同时获取集群的下的租户网络策略
// @Description 获取项目下环境列表,按照集群聚合,同时获取集群的下的租户网络策略
// @Accept json
// @Produce json
// @Param project_id path int true "project_id"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "object"
// @Router /v1/project/{project_id}/agg_environment [get]
// @Security JWT
func (h *ProjectHandler) ProjectEnvironments(c *gin.Context) {
	projectid := c.Param("project_id")
	ret := map[string]ClusterAggregateData{}

	var environments []models.Environment
	var proj models.Project

	if e := h.GetDB().Preload("Tenant").First(&proj, "id = ?", projectid).Error; e != nil {
		handlers.NotOK(c, e)
		return
	}
	if e := h.GetDB().Preload("Cluster").Preload("Creator").Find(&environments, "project_id = ?", projectid).Error; e != nil {
		handlers.NotOK(c, e)
		return
	}

	for _, env := range environments {
		env.Cluster.KubeConfig = nil
		if tmp, exist := ret[env.Cluster.ClusterName]; exist {
			tmp.Environments = append(tmp.Environments, EnvironmentAggregateData{
				Env: env,
			})
			ret[env.Cluster.ClusterName] = tmp
		} else {
			ret[env.Cluster.ClusterName] = ClusterAggregateData{
				ClusterName: env.Cluster.ClusterName,
				ClusterID:   env.ClusterID,
				Environments: []EnvironmentAggregateData{
					{
						Env: env,
					},
				},
			}
		}
	}
	for key, cluster := range ret {
		netpol, _ := kubeclient.GetClient().GetTenantNetworkPolicy(cluster.ClusterName, proj.Tenant.TenantName, nil)
		tmp := ret[key]
		tmp.NetworkPolicy = netpol
		ret[key] = tmp
		for idx, env := range tmp.Environments {
			quota, _ := kubeclient.GetClient().GetResourceQuota(cluster.ClusterName, env.Env.Namespace, "default", nil)
			tmp.Environments[idx].ResourceQuota = quota
		}
	}
	handlers.OK(c, ret)
}

// @Tags NetworkIsolated
// @Summary 项目网络隔离开关
// @Description 项目网络隔离开关
// @Accept json
// @Produce json
// @Param project_id path uint true "project_id"
// @Param param body handlers.ClusterIsolatedSwitch true "表单 "
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.IsolatedSwitch} "object"
// @Router /v1/project/{project_id}/action/networkisolate [post]
// @Security JWT
func (h *ProjectHandler) ProjectSwitch(c *gin.Context) {
	form := &handlers.ClusterIsolatedSwitch{}
	if err := c.BindJSON(form); err != nil {
		handlers.NotOK(c, err)
		return
	}
	var (
		proj    models.Project
		cluster models.Cluster
	)
	if e := h.GetDB().Preload("Tenant").First(&proj, "id = ?", c.Param("project_id")).Error; e != nil {
		handlers.NotOK(c, e)
		return
	}
	if e := h.GetDB().First(&cluster, "id = ?", form.ClusterID).Error; e != nil {
		handlers.NotOK(c, e)
		return
	}

	h.SetAuditData(c, "更新", "项目网络隔离", proj.ProjectName)
	h.SetExtraAuditData(c, models.ResProject, proj.ID)
	tnetpol, err := kubeclient.GetClient().GetTenantNetworkPolicy(cluster.ClusterName, proj.Tenant.TenantName, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	index := -1
	for idx, projpol := range tnetpol.Spec.ProjectNetworkPolicies {
		if projpol.Name == proj.ProjectName {
			index = idx
		}
	}
	if index == -1 && form.Isolate {
		tnetpol.Spec.ProjectNetworkPolicies = append(tnetpol.Spec.ProjectNetworkPolicies, gemsv1beta1.ProjectNetworkPolicy{
			Name: proj.ProjectName,
		})
	}
	if index != -1 && !form.Isolate {
		tnetpol.Spec.ProjectNetworkPolicies = append(tnetpol.Spec.ProjectNetworkPolicies[:index], tnetpol.Spec.ProjectNetworkPolicies[index+1:]...)
	}
	ret, err := kubeclient.GetClient().PatchTenantNetworkPolicy(cluster.ClusterName, proj.Tenant.TenantName, tnetpol)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// TenantProjectListResourceQuotas 租户下所有项目的资源统计列表[quota]
// @Tags Tenant
// @Summary 租户下所有项目的资源统计列表[quota]
// @Description 租户下所有项目的资源统计列表[quota]
// @Accept json
// @Produce json
// @Param tenant_id path uint true "tenant_id"
// @Success 200 {object} handlers.ResponseStruct{Data=[]object} "quotas"
// @Router /v1/tenant/{tenant_id}/projectquotas [get]
// @Security JWT
func (h *ProjectHandler) TenantProjectListResourceQuotas(c *gin.Context) {
	var (
		projects []models.Project
		ret      []interface{}
	)
	if err := h.GetDB().Find(&projects, "tenant_id = ?", c.Param("tenant_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return

	}
	for _, proj := range projects {
		tmpret, err := h.getProjectAggretateQuota(int(proj.ID))
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		item := map[string]interface{}{
			"projectid":      proj.ID,
			"project":        proj.ProjectName,
			"resourceStatus": tmpret,
		}
		ret = append(ret, item)
	}
	handlers.OK(c, ret)
}
