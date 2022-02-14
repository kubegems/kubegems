package clusterhandler

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/agent/apis/types"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/kubeclient"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/msgbus"
)

var (
	ModelName      = "Cluster"
	PrimaryKeyName = "cluster_id"
	SearchFields   = []string{"ClusterName"}
	FilterFields   = []string{"ClusterName"}
	PreloadFields  = []string{"Environments", "TenantResourceQuotas"}
)

// ListCluster 列表 Cluster
// @Tags Cluster
// @Summary Cluster列表
// @Description Cluster列表
// @Accept json
// @Produce json
// @Param ClusterName query string false "ClusterName"
// @Param preload query string false "choices Environments,TenantResourceQuotas"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (ClusterName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Cluster}} "Cluster"
// @Router /v1/cluster [get]
// @Security JWT
func (h *ClusterHandler) ListCluster(c *gin.Context) {
	var list []models.Cluster
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         ModelName,
		SearchFields:  SearchFields,
		PreloadFields: []string{"Environments", "TenantResourceQuotas"},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// ListClusterStatus 列出集群状态
// @Tags Cluster
// @Summary 列出集群状态
// @Description 列出集群状态
// @Accept json
// @Produce json
// @Success 200 {object} handlers.ResponseStruct{Data=map[string]bool} "集群状态"
// @Router /v1/cluster/_/status [get]
// @Security JWT
func (h *ClusterHandler) ListClusterStatus(c *gin.Context) {
	var clusters []*models.Cluster
	if err := h.GetDB().Find(&clusters).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	var wg sync.WaitGroup
	ret := map[string]bool{}
	m := sync.Mutex{}
	for _, cluster := range clusters {
		wg.Add(1)
		go func(clus *models.Cluster) { // 必须把i传进去
			m.Lock()
			defer m.Unlock()
			ret[clus.ClusterName] = kubeclient.GetClient().IsClusterHealth(clus.ClusterName)
			wg.Done()
		}(cluster)
	}
	wg.Wait()
	handlers.OK(c, ret)
}

// RetrieveCluster Cluster详情
// @Tags Cluster
// @Summary Cluster详情
// @Description get Cluster详情
// @Accept json
// @Produce json
// @Param cluster_id path uint true "cluster_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Cluster} "Cluster"
// @Router /v1/cluster/{cluster_id} [get]
// @Security JWT
func (h *ClusterHandler) RetrieveCluster(c *gin.Context) {
	var obj models.Cluster
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// PutCluster 修改Cluster
// @Tags Cluster
// @Summary 修改Cluster
// @Description 修改Cluster
// @Accept json
// @Produce json
// @Param cluster_id path uint true "cluster_id"
// @Param param body models.Cluster true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Cluster} "Cluster"
// @Router /v1/cluster/{cluster_id} [put]
// @Security JWT
func (h *ClusterHandler) PutCluster(c *gin.Context) {
	var obj models.Cluster
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "更新", "集群", obj.ClusterName)
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if c.Param(PrimaryKeyName) != strconv.Itoa(int(obj.ID)) {
		handlers.NotOK(c, fmt.Errorf("ID不匹配"))
		return
	}
	if err := h.GetDB().Save(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, obj)
}

// DeleteCluster 删除 Cluster
// @Tags Cluster
// @Summary 删除 Cluster
// @Description 删除 Cluster
// @Accept json
// @Produce json
// @Param cluster_id path uint true "cluster_id"
// @Success 204 {object} handlers.ResponseStruct "resp"
// @Router /v1/cluster/{cluster_id} [delete]
// @Security JWT
func (h *ClusterHandler) DeleteCluster(c *gin.Context) {
	var obj models.Cluster
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	h.SetAuditData(c, "删除", "集群", obj.ClusterName)

	if obj.Primary {
		handlers.NotOK(c, fmt.Errorf("不允许删除控制集群"))
		return
	}

	restconfig, client, err := kube.GetKubeClient(obj.KubeConfig)
	if err != nil {
		handlers.NotOK(c, fmt.Errorf("通过kubeconfig 获取resetclient失败, %v", err))
		return
	}

	trqs := []models.TenantResourceQuota{}
	h.GetDB().Where("cluster_id = ?", obj.ID).Find(&trqs)
	if len(trqs) != 0 {
		handlers.NotOK(c, fmt.Errorf("集群%s中还有关联的租户资源，删除失败", obj.ClusterName))
		return
	}

	envs := []models.Environment{}
	h.GetDB().Where("cluster_id = ?", obj.ID).Find(&envs)
	if len(envs) != 0 {
		handlers.NotOK(c, fmt.Errorf("集群%s中还有关联的环境，删除失败", obj.ClusterName))
		return
	}

	// 删除installer
	_ = (kube.NewClusterInstaller(restconfig, client, &obj, h.GetOptions().Installer)).Uninstall()

	if err := h.GetDB().Delete(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Delete).
		ResourceType(msgbus.Cluster).
		ResourceID(obj.ID).
		Content(fmt.Sprintf("删除了集群%s", obj.ClusterName)).
		SetUsersToSend(
			h.GetDataBase().SystemAdmins(),
		).
		Send()
	handlers.NoContent(c, nil)
}

// ListClusterEnvironment 获取属于Cluster的 Environment 列表
// @Tags Cluster
// @Summary 获取属于 Cluster 的 Environment 列表
// @Description 获取属于 Cluster 的 Environment 列表
// @Accept json
// @Produce json
// @Param cluster_id path uint true "cluster_id"
// @Param preload query string false "choices Creator,Cluster,Project,Applications,Users"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (EnvironmentName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Environment}} "models.Environment"
// @Router /v1/cluster/{cluster_id}/environment [get]
// @Security JWT
func (h *ClusterHandler) ListClusterEnvironment(c *gin.Context) {
	var list []models.Environment
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "Environment",
		SearchFields:  []string{"EnvironmentName"},
		PreloadFields: []string{"Project", "Cluster", "Creator", "Applications", "Users"},
		Where:         []*handlers.QArgs{handlers.Args("cluster_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// ListClusterLogQueryHistory 获取属于Cluster的 LogQueryHistory 列表
// @Tags Cluster
// @Summary 获取属于 Cluster 的 LogQueryHistory 列表
// @Description 获取属于 Cluster 的 LogQueryHistory 列表
// @Accept json
// @Produce json
// @Param cluster_id path uint true "cluster_id"
// @Param preload query string false "choices Cluster,Creator"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (LogQL)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.LogQueryHistory}} "models.LogQueryHistory"
// @Router /v1/cluster/{cluster_id}/logqueryhistory [get]
// @Security JWT
func (h *ClusterHandler) ListClusterLogQueryHistory(c *gin.Context) {
	var list []models.LogQueryHistory
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "LogQueryHistory",
		SearchFields:  []string{"LogQL"},
		PreloadFields: []string{"Cluster", "Creator"},
		Where:         []*handlers.QArgs{handlers.Args("cluster_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// ListLogQueryHistory 聚合查询日志查询历史[按照当前用户的查询历史聚合]
// @Tags Cluster
// @Summary 聚合查询日志查询历史, unique logql desc 按照当前用户的查询历史聚合
// @Description 聚合查询日志查询历史 unique logql desc 按照当前用户的查询历史聚合
// @Accept json
// @Produce json
// @Success 200 {object} handlers.ResponseStruct{Data=[]models.LogQueryHistoryWithCount} "LogQueryHistory"
// @Router /v1/cluster/{cluster_id}/logqueryhistoryv2 [get]
// @Security JWT
func (h *ClusterHandler) ListClusterLogQueryHistoryv2(c *gin.Context) {
	var list []models.LogQueryHistoryWithCount
	user, _ := h.GetContextUser(c)
	before15d := time.Now().Add(-15 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	rawsql := `select log_ql,
	max(id) as id,
	GROUP_CONCAT(id SEPARATOR ',') as ids,
	any_value(cluster_id) as cluster_id,
	max(create_at) as create_at,
	any_value(filter_json) as filter_json,
	any_value(label_json) as label_json,
	count(*) as total from log_query_histories where creator_id = ? and cluster_id = ? and create_at > ? group by log_ql order by total desc;`
	if err := h.GetDB().Raw(rawsql, user.ID, c.Param("cluster_id"), before15d).Scan(&list).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, list)
}

// ListClusterLogQuerySnapshot 获取属于Cluster的 LogQuerySnapshot 列表
// @Tags Cluster
// @Summary 获取属于 Cluster 的 LogQuerySnapshot 列表
// @Description 获取属于 Cluster 的 LogQuerySnapshot 列表
// @Accept json
// @Produce json
// @Param cluster_id path uint true "cluster_id"
// @Param preload query string false "choices Cluster,Creator"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (SnapshotName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.LogQuerySnapshot}} "models.LogQuerySnapshot"
// @Router /v1/cluster/{cluster_id}/logquerysnapshot [get]
// @Security JWT
func (h *ClusterHandler) ListClusterLogQuerySnapshot(c *gin.Context) {
	var list []models.LogQuerySnapshot
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "LogQuerySnapshot",
		SearchFields:  []string{"SnapshotName"},
		PreloadFields: []string{"Cluster", "Creator"},
		Where:         []*handlers.QArgs{handlers.Args("cluster_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// PostCluster 创建Cluster
// @Tags Cluster
// @Summary 创建Cluster
// @Description 创建Cluster
// @Accept json
// @Produce json
// @Param param body models.Cluster true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Cluster} "Cluster"
// @Router /v1/cluster [post]
// @Security JWT
func (h *ClusterHandler) PostCluster(c *gin.Context) {
	var obj models.Cluster
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "创建", "集群", obj.ClusterName)
	if obj.Runtime == "" {
		obj.Runtime = "containerd" // 默认containerd
	}
	apiserver, _, _, _, err := kube.GetKubeconfigInfos(obj.KubeConfig)
	if err != nil {
		handlers.NotOK(c, fmt.Errorf("kubeconfig 格式错误, %v", err))
		return
	}
	restconfig, client, err := kube.GetKubeClient(obj.KubeConfig)
	if err != nil {
		handlers.NotOK(c, fmt.Errorf("通过kubeconfig 获取resetclient失败, %v", err))
		return
	}
	info, err := client.ServerVersion()
	if err != nil {
		handlers.NotOK(c, fmt.Errorf("获取serverInfo失败, %v", err))
		return
	}
	obj.Version = info.String()
	obj.APIServer = apiserver
	if obj.Mode != models.ClusterModeService {
		obj.Mode = models.ClusterModeProxy
	}

	clusters := []models.Cluster{}
	h.GetDB().Where("api_server = ?", apiserver).Find(&clusters)
	if len(clusters) > 0 {
		handlers.NotOK(c, fmt.Errorf("已有apiserver为%s的集群%s", apiserver, clusters[0].ClusterName))
		return
	}
	// 非控制集群才安装插件
	if obj.Primary {
		primarys := []models.Cluster{}
		h.GetDB().Where("primary = ?", true).Find(&primarys)
		if len(clusters) > 0 {
			handlers.NotOK(c, fmt.Errorf("控制集群只能有一个"))
			return
		}
	} else {
		installer := kube.NewClusterInstaller(restconfig, client, &obj, h.GetOptions().Installer)
		if e := installer.Install(); e != nil {
			handlers.NotOK(c, fmt.Errorf("初始化集群错误, %v", err))
			return
		}
	}

	if err := h.GetDB().Save(&obj).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("保存集群配置错误, %v", err))
		return
	}

	h.GetMessageBusClient().
		GinContext(c).
		MessageType(msgbus.Message).
		ActionType(msgbus.Add).
		ResourceType(msgbus.Cluster).
		ResourceID(obj.ID).
		Content(fmt.Sprintf("添加了集群%s", obj.ClusterName)).
		SetUsersToSend(
			h.GetDataBase().SystemAdmins(),
		).
		Send()
	handlers.Created(c, obj)
}

// ClusterStatistics 集群资源状态
// @Tags Cluster
// @Summary 集群资源状态
// @Description 集群资源状态
// @Accept json
// @Produce json
// @Param cluster_id path int true "cluster_id"
// @Success 200 {object} handlers.ResponseStruct{Data=ClusterQuota} "statistics"
// @Router /v1/cluster/{cluster_id}/quota [get]
// @Security JWT
func (h *ClusterHandler) ListClusterQuota(c *gin.Context) {
	var cluster models.Cluster
	if err := h.GetDB().First(&cluster, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	resources := &types.ClusterResourceStatistics{}
	if err := kubeclient.GetClient().ClusterResourceStatistics(cluster.ClusterName, resources); err != nil {
		handlers.NotOK(c, err)
		return
	}

	workloads := types.ClusterWorkloadStatistics{}
	if err := kubeclient.GetClient().ClusterWorkloadStatistics(cluster.ClusterName, &workloads); err != nil {
		handlers.NotOK(c, err)
		return
	}
	data := ClusterQuota{
		Version:        cluster.Version,
		Resoruces:      *resources,
		OversoldConfig: cluster.OversoldConfig,
		Workloads:      workloads,
	}
	handlers.OK(c, data)
}

// @Tags Agent.Plugin
// @Summary 获取Plugin列表数据
// @Description 获取Plugin列表数据
// @Accept json
// @Produce json
// @Param cluster_id path int true "cluster_id"
// @Success 200 {object} handlers.ResponseStruct{Data=map[string]interface{}} "Plugins"
// @Router /v1/cluster/{cluster_id}/plugins [get]
// @Security JWT
func (h *ClusterHandler) ListPligins(c *gin.Context) {
	var cluster models.Cluster
	if err := h.GetDB().First(&cluster, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	ret, err := kubeclient.GetClient().ListPlugins(cluster.ClusterName)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// @Tags Agent.Plugin
// @Summary 启用插件
// @Description 启用插件
// @Accept json
// @Produce json
// @Param cluster_id path int true "cluster_id"
// @Param name path string true "name"
// @Param type query string true "type"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "Plugins"
// @Router /v1/cluster/{cluster_id}/plugins/{name}/actions/enable [post]
// @Security JWT
func (h *ClusterHandler) EnablePlugin(c *gin.Context) {
	var cluster models.Cluster
	if err := h.GetDB().First(&cluster, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	plugintype := c.Query("type")
	pluginname := c.Param("name")
	h.SetAuditData(c, "启用", "集群插件", fmt.Sprintf("集群[%v]/插件[%v]", cluster.ClusterName, pluginname))
	err := kubeclient.GetClient().EnablePlugin(cluster.ClusterName, plugintype, pluginname)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	if plugintype == "core" {
		h.GetMessageBusClient().
			GinContext(c).
			MessageType(msgbus.Message).
			ActionType(msgbus.Update).
			ResourceType(msgbus.Cluster).
			ResourceID(cluster.ID).
			Content(fmt.Sprintf("启用了集群%s中的插件%s", cluster.ClusterName, pluginname)).
			SetUsersToSend(
				h.GetDataBase().SystemAdmins(),
			).
			Send()
	}
	handlers.OK(c, "")
}

// @Tags Agent.Plugin
// @Summary 禁用插件
// @Description 禁用插件
// @Accept json
// @Produce json
// @Param cluster_id path int true "cluster_id"
// @Param name path string true "name"
// @Param type query string true "type"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "Plugins"
// @Router /v1/cluster/{cluster_id}/plugins/{name}/actions/disable [post]
// @Security JWT
func (h *ClusterHandler) DisablePlugin(c *gin.Context) {
	var cluster models.Cluster
	if err := h.GetDB().First(&cluster, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	plugintype := c.Query("type")
	pluginname := c.Param("name")
	h.SetAuditData(c, "禁用", "集群插件", fmt.Sprintf("集群[%v]/插件[%v]", cluster.ClusterName, pluginname))
	err := kubeclient.GetClient().DisablePlugin(cluster.ClusterName, plugintype, pluginname)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	if plugintype == "core" {
		h.GetMessageBusClient().
			GinContext(c).
			MessageType(msgbus.Message).
			ActionType(msgbus.Update).
			ResourceType(msgbus.Cluster).
			ResourceID(cluster.ID).
			Content(fmt.Sprintf("卸载了集群%s中的插件%s", cluster.ClusterName, pluginname)).
			SetUsersToSend(
				h.GetDataBase().SystemAdmins(),
			).
			Send()
	}
	handlers.OK(c, "")
}
