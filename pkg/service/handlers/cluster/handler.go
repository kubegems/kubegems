package clusterhandler

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"kubegems.io/pkg/agent/apis/types"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/msgbus"
	"kubegems.io/pkg/version"
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
	total, page, size, err := query.PageList(h.GetDataBase().DB(), cond, &list)
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
	if err := h.GetDataBase().DB().Find(&clusters).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	ctx := c.Request.Context()

	ret := map[string]bool{}

	eg := &errgroup.Group{}
	mu := sync.Mutex{}
	for _, cluster := range clusters {
		name := cluster.ClusterName
		eg.Go(func() error {
			cli, err := h.GetAgents().ClientOf(ctx, name)
			if err != nil {
				log.Error(err, "unable get agents client", "cluster", name)
				return nil
			}
			if err := cli.Extend().Healthy(ctx); err != nil {
				log.Error(err, "cluster unhealthy", "cluster", name)
				return nil
			}

			mu.Lock()
			defer mu.Unlock()
			ret[name] = true
			return nil
		})
	}
	_ = eg.Wait()

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
	if err := h.GetDataBase().DB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
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
	if err := h.GetDataBase().DB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
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
	if err := h.GetDataBase().DB().Save(&obj).Error; err != nil {
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
	cluster := &models.Cluster{}
	if err := h.GetDataBase().DB().First(cluster, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	h.SetAuditData(c, "删除", "集群", cluster.ClusterName)

	if cluster.Primary {
		handlers.NotOK(c, fmt.Errorf("不允许删除控制集群"))
		return
	}

	trqs := []models.TenantResourceQuota{}
	h.GetDataBase().DB().Where("cluster_id = ?", cluster.ID).Find(&trqs)
	if len(trqs) != 0 {
		handlers.NotOK(c, fmt.Errorf("集群%s中还有关联的租户资源，删除失败", cluster.ClusterName))
		return
	}

	envs := []models.Environment{}
	h.GetDataBase().DB().Where("cluster_id = ?", cluster.ID).Find(&envs)
	if len(envs) != 0 {
		handlers.NotOK(c, fmt.Errorf("集群%s中还有关联的环境，删除失败", cluster.ClusterName))
		return
	}

	if err := withClusterAndK8sClient(c, cluster, func(ctx context.Context, clientSet *kubernetes.Clientset, config *rest.Config) error {
		if err := h.GetDataBase().DB().Transaction(func(tx *gorm.DB) error {
			if err := h.GetDataBase().DB().Delete(cluster).Error; err != nil {
				return err
			}

			installer := ClusterInstaller{
				Cluster:         cluster,
				Clientset:       clientSet,
				Config:          config,
				KubegemsVersion: version.Get(),
			}
			return installer.UnInstall(ctx)
		}); err != nil {
			log.Error(err, "delete cluster")
			return err
		}

		h.GetMessageBusClient().
			GinContext(c).
			MessageType(msgbus.Message).
			ActionType(msgbus.Delete).
			ResourceType(msgbus.Cluster).
			ResourceID(cluster.ID).
			Content(fmt.Sprintf("删除了集群%s", cluster.ClusterName)).
			SetUsersToSend(
				h.GetDataBase().SystemAdmins(),
			).
			Send()
		handlers.NoContent(c, nil)
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
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
	total, page, size, err := query.PageList(h.GetDataBase().DB(), cond, &list)
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
	total, page, size, err := query.PageList(h.GetDataBase().DB(), cond, &list)
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
	if err := h.GetDataBase().DB().Raw(rawsql, user.ID, c.Param("cluster_id"), before15d).Scan(&list).Error; err != nil {
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
	total, page, size, err := query.PageList(h.GetDataBase().DB(), cond, &list)
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
	cluster := &models.Cluster{}
	if err := c.BindJSON(cluster); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "创建", "集群", cluster.ClusterName)

	// 控制集群只检验
	if cluster.Primary {
		primarys := []models.Cluster{}
		if err := h.GetDataBase().DB().Where("primary = ?", true).Find(&primarys).Error; err != nil {
			handlers.NotOK(c, err)
			return
		}
		if len(primarys) > 0 {
			handlers.NotOK(c, fmt.Errorf("控制集群只能有一个"))
			return
		}
		handlers.Created(c, cluster)
		return
	}

	if err := withClusterAndK8sClient(c, cluster, func(ctx context.Context, clientSet *kubernetes.Clientset, config *rest.Config) error {
		if err := h.GetDataBase().DB().Transaction(func(tx *gorm.DB) error {
			if err := h.GetDataBase().DB().Save(cluster).Error; err != nil {
				return err
			}

			installer := ClusterInstaller{
				Cluster:         cluster,
				Clientset:       clientSet,
				Config:          config,
				KubegemsVersion: version.Get(),
			}
			return installer.Install(ctx)
		}); err != nil {
			log.Error(err, "create cluster")
			return err
		}

		h.GetMessageBusClient().
			GinContext(c).
			MessageType(msgbus.Message).
			ActionType(msgbus.Add).
			ResourceType(msgbus.Cluster).
			ResourceID(cluster.ID).
			Content(fmt.Sprintf("添加了集群%s", cluster.ClusterName)).
			SetUsersToSend(
				h.GetDataBase().SystemAdmins(),
			).
			Send()
		handlers.Created(c, cluster)
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
}

type ClusterQuota struct {
	Version        string                          `json:"version"`
	OversoldConfig datatypes.JSON                  `json:"oversoldConfig"`
	Resoruces      types.ClusterResourceStatistics `json:"resources"`
	Workloads      types.ClusterWorkloadStatistics `json:"workloads"`
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
	h.cluster(c, func(ctx context.Context, cluster models.Cluster, cli agents.Client) (interface{}, error) {
		resources := types.ClusterResourceStatistics{}
		if err := cli.Extend().ClusterResourceStatistics(ctx, &resources); err != nil {
			return nil, err
		}
		workloads := types.ClusterWorkloadStatistics{}
		if err := cli.Extend().ClusterWorkloadStatistics(ctx, &workloads); err != nil {
			return nil, err
		}

		return ClusterQuota{
			Version:        cluster.Version,
			Resoruces:      resources,
			OversoldConfig: cluster.OversoldConfig,
			Workloads:      workloads,
		}, nil
	})
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
	h.cluster(c, func(ctx context.Context, _ models.Cluster, cli agents.Client) (interface{}, error) {
		return cli.Extend().ListPlugins(ctx)
	})
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
	h.cluster(c, func(ctx context.Context, cluster models.Cluster, cli agents.Client) (interface{}, error) {
		plugintype := c.Query("type")
		pluginname := c.Param("name")

		h.SetAuditData(c, "启用", "集群插件", fmt.Sprintf("集群[%v]/插件[%v]", cluster.ClusterName, pluginname))

		if err := cli.Extend().EnablePlugin(ctx, plugintype, pluginname); err != nil {
			return nil, err
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

		return "", nil
	})
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
	h.cluster(c, func(ctx context.Context, cluster models.Cluster, cli agents.Client) (interface{}, error) {
		plugintype := c.Query("type")
		pluginname := c.Param("name")

		h.SetAuditData(c, "禁用", "集群插件", fmt.Sprintf("集群[%v]/插件[%v]", cluster.ClusterName, pluginname))

		if err := cli.Extend().DisablePlugin(ctx, plugintype, pluginname); err != nil {
			return nil, err
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

		return "", nil
	})
}

func (h *ClusterHandler) cluster(c *gin.Context, fun func(ctx context.Context, cluster models.Cluster, cli agents.Client) (interface{}, error)) {
	var cluster models.Cluster
	if err := h.GetDataBase().DB().First(&cluster, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.ClusterFunc(cluster.ClusterName, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		return fun(ctx, cluster, cli)
	})(c)
}

func withClusterAndK8sClient(
	c *gin.Context,
	cluster *models.Cluster,
	f func(ctx context.Context, clientSet *kubernetes.Clientset, config *rest.Config) error,
) error {
	// 获取clientSet
	apiserver, _, _, _, err := kube.GetKubeconfigInfos(cluster.KubeConfig)
	if err != nil {
		return fmt.Errorf("kubeconfig 格式错误, %w", err)
	}
	restconfig, clientSet, err := kube.GetKubeClient(cluster.KubeConfig)
	if err != nil {
		return fmt.Errorf("通过kubeconfig 获取restclient失败, %v", err)
	}
	serverSersion, err := clientSet.ServerVersion()
	if err != nil {
		return fmt.Errorf("获取serverInfo失败, %v", err)
	}
	ctx := c.Request.Context()
	nodes, err := clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list node failed: %w", err)
	}

	cluster.APIServer = apiserver
	cluster.Version = serverSersion.String()
	if cluster.Mode != models.ClusterModeService {
		cluster.Mode = models.ClusterModeProxy
	}
	// get container runtime
	reg := regexp.MustCompile("(.*)://(.*)")
	for _, n := range nodes.Items {
		matches := reg.FindStringSubmatch(n.Status.NodeInfo.ContainerRuntimeVersion)
		if len(matches) == 3 {
			cluster.Runtime = matches[1]
			break
		}
	}
	return f(ctx, clientSet, restconfig)
}

const (
	kubegemsInstallerNamespace = "kubegems-installer"
)

func (i *ClusterInstaller) CreateNamespaceIfNotExists(ctx context.Context) error {
	_, err := i.Clientset.CoreV1().Namespaces().Get(ctx, kubegemsInstallerNamespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if errors.IsNotFound(err) {
		if _, err = i.Clientset.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubegemsInstallerNamespace,
			},
		}, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

type ClusterInstaller struct {
	Cluster         *models.Cluster
	Clientset       *kubernetes.Clientset
	Config          *rest.Config
	KubegemsVersion version.Version
}

func (i *ClusterInstaller) getInstallerBts() ([]byte, error) {
	installerbuf := new(bytes.Buffer)
	if err := installerTpl.Execute(installerbuf, i); err != nil {
		log.Error(err, "installer template")
		return nil, err
	}
	return installerbuf.Bytes(), nil
}

func (i *ClusterInstaller) getPluginsBts() ([]byte, error) {
	pluginBuf := new(bytes.Buffer)
	if err := installerTpl.Execute(pluginBuf, i); err != nil {
		log.Error(err, "plugins template")
		return nil, err
	}
	return pluginBuf.Bytes(), nil
}

func (i *ClusterInstaller) Install(ctx context.Context) error {
	// install crd
	if err := i.CreateNamespaceIfNotExists(ctx); err != nil {
		return err
	}
	installerBts, err := i.getInstallerBts()
	if err != nil {
		return err
	}
	if err := kube.CreateByYamlOrJson(ctx, i.Config, installerBts); err != nil {
		log.Error(err, "create installer yaml")
		return err
	}

	// install plugin, 与crd分开部署，以刷新restmap
	pluginsBts, err := i.getPluginsBts()
	if err != nil {
		return err
	}
	return kube.CreateByYamlOrJson(ctx, i.Config, pluginsBts)
}

func (i *ClusterInstaller) UnInstall(ctx context.Context) error {
	installerBts, err := i.getInstallerBts()
	if err != nil {
		return err
	}
	if err := kube.DeleteByYamlOrJson(ctx, i.Config, installerBts); err != nil {
		log.Error(err, "create installer yaml")
		return err
	}

	pluginsBts, err := i.getPluginsBts()
	if err != nil {
		return err
	}
	return kube.DeleteByYamlOrJson(ctx, i.Config, pluginsBts)
}
