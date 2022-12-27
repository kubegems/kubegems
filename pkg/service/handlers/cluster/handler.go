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

package clusterhandler

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	msgclient "kubegems.io/kubegems/pkg/msgbus/client"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/gemsplugin"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"kubegems.io/kubegems/pkg/utils/statistics"
	"kubegems.io/kubegems/pkg/version"
)

var (
	ModelName      = "Cluster"
	PrimaryKeyName = "cluster_id"
	SearchFields   = []string{"ClusterName"}
	FilterFields   = []string{"ClusterName"}
	PreloadFields  = []string{"Environments", "TenantResourceQuotas"}
)

// ListCluster 列表 Cluster
// @Tags        Cluster
// @Summary     Cluster列表
// @Description Cluster列表
// @Accept      json
// @Produce     json
// @Param       ClusterName query    string                                                                 false "ClusterName"
// @Param       preload     query    string                                                                 false "choices Environments,TenantResourceQuotas"
// @Param       page        query    int                                                                    false "page"
// @Param       size        query    int                                                                    false "page"
// @Param       search      query    string                                                                 false "search in (ClusterName)"
// @Success     200         {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Cluster}} "Cluster"
// @Router      /v1/cluster [get]
// @Security    JWT
func (h *ClusterHandler) ListCluster(c *gin.Context) {
	var list []*models.Cluster
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
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// ListClusterStatus 列出集群状态
// @Tags        Cluster
// @Summary     列出集群状态
// @Description 列出集群状态
// @Accept      json
// @Produce     json
// @Success     200 {object} handlers.ResponseStruct{Data=map[string]bool} "集群状态"
// @Router      /v1/cluster/_/status [get]
// @Security    JWT
func (h *ClusterHandler) ListClusterStatus(c *gin.Context) {
	var clusters []*models.Cluster
	if err := h.GetDB().WithContext(c.Request.Context()).Find(&clusters).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	ret := map[string]bool{}
	for _, cluster := range clusters {
		ret[cluster.ClusterName] = false
	}
	mu := sync.Mutex{}
	innerCtx, cancel := context.WithCancel(c)
	defer cancel()
	h.batchWithTimeout(c, clusters, time.Duration(time.Second*3), func(idx int, name string, cli agents.Client) {
		if err := cli.Extend().Healthy(innerCtx); err != nil {
			log.Error(err, "cluster unhealthy", "cluster", name)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		ret[name] = true
	})

	handlers.OK(c, ret)
}

// RetrieveCluster Cluster详情
// @Tags        Cluster
// @Summary     Cluster详情
// @Description get Cluster详情
// @Accept      json
// @Produce     json
// @Param       cluster_id path     uint                                         true "cluster_id"
// @Success     200        {object} handlers.ResponseStruct{Data=models.Cluster} "Cluster"
// @Router      /v1/cluster/{cluster_id} [get]
// @Security    JWT
func (h *ClusterHandler) RetrieveCluster(c *gin.Context) {
	var obj models.Cluster
	if err := h.GetDB().WithContext(c.Request.Context()).First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	if obj.Version == "" {
		cli, err := h.GetAgents().ClientOf(c.Request.Context(), obj.ClusterName)
		if err != nil {
			log.Error(err, "unable get agents client", "cluster", obj.ClusterName)
		} else {
			obj.Version = cli.APIServerVersion()
		}
	}

	handlers.OK(c, obj)
}

// PutCluster 修改Cluster
// @Tags        Cluster
// @Summary     修改Cluster
// @Description 修改Cluster
// @Accept      json
// @Produce     json
// @Param       cluster_id path     uint                                         true "cluster_id"
// @Param       param      body     models.Cluster                               true "表单"
// @Success     200        {object} handlers.ResponseStruct{Data=models.Cluster} "Cluster"
// @Router      /v1/cluster/{cluster_id} [put]
// @Security    JWT
func (h *ClusterHandler) PutCluster(c *gin.Context) {
	var obj models.Cluster
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	action := i18n.Sprintf(context.TODO(), "update")
	module := i18n.Sprintf(context.TODO(), "cluster")
	h.SetAuditData(c, action, module, obj.ClusterName)
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if c.Param(PrimaryKeyName) != strconv.Itoa(int(obj.ID)) {
		handlers.NotOK(c, i18n.Errorf(c, "URL parameter mismatched with body"))
		return
	}
	if err := h.GetDB().WithContext(ctx).Save(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	// invalidate agent config
	h.GetAgents().Invalidate(c, obj.ClusterName)
	handlers.OK(c, obj)
}

// DeleteCluster 删除 Cluster
// @Tags        Cluster
// @Summary     删除 Cluster
// @Description 删除 Cluster
// @Accept      json
// @Produce     json
// @Param       record_only query    string                  false "only delete record in database"
// @Param       cluster_id  path     uint                    true  "cluster_id"
// @Success     204         {object} handlers.ResponseStruct "resp"
// @Router      /v1/cluster/{cluster_id} [delete]
// @Security    JWT
func (h *ClusterHandler) DeleteCluster(c *gin.Context) {
	cluster := &models.Cluster{}
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).First(cluster, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	action := i18n.Sprintf(context.TODO(), "update")
	module := i18n.Sprintf(context.TODO(), "cluster")
	h.SetAuditData(c, action, module, cluster.ClusterName)

	if cluster.Primary {
		handlers.NotOK(c, i18n.Errorf(c, "can't delete this cluster, it's the primary cluster which the api server run"))
		return
	}

	trqs := []models.TenantResourceQuota{}
	h.GetDB().WithContext(ctx).Where("cluster_id = ?", cluster.ID).Find(&trqs)
	if len(trqs) != 0 {
		handlers.NotOK(c, i18n.Errorf(c, "can't delete the cluster %s, some tenants has resources on it", cluster.ClusterName))
		return
	}

	envs := []models.Environment{}
	h.GetDB().WithContext(ctx).Where("cluster_id = ?", cluster.ID).Find(&envs)
	if len(envs) != 0 {
		handlers.NotOK(c, i18n.Errorf(c, "can't delete the cluster %s, some environments has resources on it", cluster.ClusterName))
		return
	}
	recordOnly := c.DefaultQuery("record_only", "true") == "true"
	if recordOnly {
		if err := h.GetDB().WithContext(ctx).Delete(cluster).Error; err != nil {
			handlers.NotOK(c, err)
			return
		}
	} else {
		if err := withClusterAndK8sClient(c, cluster, func(ctx context.Context, clientSet *kubernetes.Clientset, config *rest.Config) error {
			return h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				if err := tx.Delete(cluster).Error; err != nil {
					return err
				}
				return gemsplugin.Bootstrap{Config: config}.Remove(ctx)
			})
		}); err != nil {
			handlers.NotOK(c, err)
			return
		}
	}

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Delete
		msg.ResourceType = msgbus.Cluster
		msg.ResourceID = cluster.ID
		msg.Detail = i18n.Sprintf(context.TODO(), "deleted the cluster %s", cluster.ClusterName)
		msg.ToUsers.Append(h.GetDataBase().SystemAdmins()...)
	})
	handlers.NoContent(c, nil)
}

// ListClusterEnvironment 获取属于Cluster的 Environment 列表
// @Tags        Cluster
// @Summary     获取属于 Cluster 的 Environment 列表
// @Description 获取属于 Cluster 的 Environment 列表
// @Accept      json
// @Produce     json
// @Param       cluster_id path     uint                                                                       true  "cluster_id"
// @Param       preload    query    string                                                                     false "choices Creator,Cluster,Project,Applications,Users"
// @Param       page       query    int                                                                        false "page"
// @Param       size       query    int                                                                        false "page"
// @Param       search     query    string                                                                     false "search in (EnvironmentName)"
// @Success     200        {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Environment}} "models.Environment"
// @Router      /v1/cluster/{cluster_id}/environment [get]
// @Security    JWT
func (h *ClusterHandler) ListClusterEnvironment(c *gin.Context) {
	var list []models.Environment
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	clusterid := utils.ToUint(c.Param(PrimaryKeyName))
	cond := &handlers.PageQueryCond{
		Model:         "Environment",
		SearchFields:  []string{"EnvironmentName"},
		PreloadFields: []string{"Project", "Cluster", "Creator", "Applications", "Users"},
		Where: []*handlers.QArgs{
			handlers.Args("cluster_id = ?", clusterid),
		},
	}
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// ListClusterLogQueryHistory 获取属于Cluster的 LogQueryHistory 列表
// @Tags        Cluster
// @Summary     获取属于 Cluster 的 LogQueryHistory 列表
// @Description 获取属于 Cluster 的 LogQueryHistory 列表
// @Accept      json
// @Produce     json
// @Param       cluster_id path     uint                                                                           true  "cluster_id"
// @Param       preload    query    string                                                                         false "choices Cluster,Creator"
// @Param       page       query    int                                                                            false "page"
// @Param       size       query    int                                                                            false "page"
// @Param       search     query    string                                                                         false "search in (LogQL)"
// @Success     200        {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.LogQueryHistory}} "models.LogQueryHistory"
// @Router      /v1/cluster/{cluster_id}/logqueryhistory [get]
// @Security    JWT
func (h *ClusterHandler) ListClusterLogQueryHistory(c *gin.Context) {
	var (
		list    []models.LogQueryHistory
		cluster models.Cluster
	)
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	clusterid := utils.ToUint(c.Param(PrimaryKeyName))
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).First(&cluster, clusterid).Error; err != nil {
		handlers.NotOK(c, i18n.Errorf(c, "the cluster you are querying doesn't exist"))
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "LogQueryHistory",
		SearchFields:  []string{"LogQL"},
		PreloadFields: []string{"Cluster", "Creator"},
		Where: []*handlers.QArgs{
			handlers.Args("cluster_id = ?", clusterid),
		},
	}
	total, page, size, err := query.PageList(h.GetDB().WithContext(ctx), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// ListLogQueryHistory 聚合查询日志查询历史[按照当前用户的查询历史聚合]
// @Tags        Cluster
// @Summary     聚合查询日志查询历史, unique logql desc 按照当前用户的查询历史聚合
// @Description 聚合查询日志查询历史 unique logql desc 按照当前用户的查询历史聚合
// @Accept      json
// @Produce     json
// @Success     200 {object} handlers.ResponseStruct{Data=[]models.LogQueryHistoryWithCount} "LogQueryHistory"
// @Router      /v1/cluster/{cluster_id}/logqueryhistoryv2 [get]
// @Security    JWT
func (h *ClusterHandler) ListClusterLogQueryHistoryv2(c *gin.Context) {
	var list []models.LogQueryHistoryWithCount
	user, _ := h.GetContextUser(c)
	clusterid := utils.ToUint(c.Param(PrimaryKeyName))
	before15d := time.Now().Add(-15 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	rawsql := `select log_ql,
		max(id) as id,
		GROUP_CONCAT(id SEPARATOR ',') as ids,
		GROUP_CONCAT(DISTINCT(time_range) SEPARATOR ',') as time_ranges,
		any_value(cluster_id) as cluster_id,
		max(create_at) as create_at,
		any_value(filter_json) as filter_json,
		any_value(label_json) as label_json,
		count(*) as total
	from log_query_histories
	where
		creator_id = ? and cluster_id = ? and create_at > ?
	group by
		log_ql
	order by total desc;`
	if err := h.GetDB().WithContext(c.Request.Context()).Raw(
		rawsql,
		user.GetID(),
		clusterid,
		before15d,
		false,
	).Scan(&list).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, list)
}

// ListClusterLogQuerySnapshot 获取属于Cluster的 LogQuerySnapshot 列表
// @Tags        Cluster
// @Summary     获取属于 Cluster 的 LogQuerySnapshot 列表
// @Description 获取属于 Cluster 的 LogQuerySnapshot 列表
// @Accept      json
// @Produce     json
// @Param       cluster_id path     uint                                                                            true  "cluster_id"
// @Param       preload    query    string                                                                          false "choices Cluster,Creator"
// @Param       page       query    int                                                                             false "page"
// @Param       size       query    int                                                                             false "page"
// @Param       search     query    string                                                                          false "search in (SnapshotName)"
// @Success     200        {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.LogQuerySnapshot}} "models.LogQuerySnapshot"
// @Router      /v1/cluster/{cluster_id}/logquerysnapshot [get]
// @Security    JWT
func (h *ClusterHandler) ListClusterLogQuerySnapshot(c *gin.Context) {
	var (
		list    []models.LogQuerySnapshot
		cluster models.Cluster
	)
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	clusterid := utils.ToUint(c.Param(PrimaryKeyName))
	if err := h.GetDB().WithContext(c.Request.Context()).First(&cluster, clusterid).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "LogQuerySnapshot",
		SearchFields:  []string{"SnapshotName"},
		PreloadFields: []string{"Cluster", "Creator"},
		Where: []*handlers.QArgs{
			handlers.Args("cluster_id = ?", clusterid),
		},
	}
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// PostCluster 创建Cluster
// @Tags        Cluster
// @Summary     创建Cluster
// @Description 创建Cluster
// @Accept      json
// @Produce     json
// @Param       param body     models.Cluster                               true "表单"
// @Success     200   {object} handlers.ResponseStruct{Data=models.Cluster} "Cluster"
// @Router      /v1/cluster [post]
// @Security    JWT
func (h *ClusterHandler) PostCluster(c *gin.Context) {
	cluster := &models.Cluster{}
	if err := c.BindJSON(cluster); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if cluster.ClusterName == "" {
		handlers.NotOK(c, errors.New("empty cluster name"))
		return
	}

	// nolint: dogsled
	apiserver, _, _, _, err := kube.GetKubeconfigInfos(cluster.KubeConfig)
	if err != nil {
		log.Error(err, "failed to validate kubeconfg, may format error")
		handlers.NotOK(c, i18n.Errorf(c, "failed to validate kubeconfg, may format error"))
		return
	}
	var existCount int64
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Model(
		&models.Cluster{},
	).Where(
		"cluster_name = ? or api_server = ?", cluster.ClusterName, apiserver,
	).Count(&existCount).Error; err != nil {
		log.Error(err, "failed to detect the cluster is existed %v", err)
		handlers.NotOK(c, i18n.Errorf(c, "failed to detect the cluster is existed"))
		return
	}
	if existCount > 0 {
		handlers.NotOK(c, i18n.Errorf(c, "the cluster with name %s existed, can't add the same one"))
		return
	}
	action := i18n.Sprintf(context.TODO(), "create")
	module := i18n.Sprintf(context.TODO(), "cluster")
	h.SetAuditData(c, action, module, cluster.ClusterName)

	if err := withClusterAndK8sClient(c, cluster, func(ctx context.Context, clientSet *kubernetes.Clientset, config *rest.Config) error {
		txClause := clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{
				"kube_config",
				"version",
				"runtime",
				"primary",
				"vendor",
				"image_repo",
				"default_storage_class",
				"deleted_at",
			}),
		}

		// 如果为第一个添加的集群，则设置为主集群
		count := int64(0)
		if err := h.GetDB().WithContext(ctx).Model(&models.Cluster{}).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			cluster.Primary = true
		}

		// 控制集群检验
		if cluster.Primary {
			var primarysCount int64
			if err := h.GetDB().WithContext(ctx).Model(&models.Cluster{}).Where(`'primary' = ?`, true).Count(&primarysCount).Error; err != nil {
				return err
			}
			if primarysCount > 0 {
				return i18n.Errorf(c, "the primary cluster existed already, more than one primary cluster is not allowed")
			}
		}
		if err := h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(txClause).Create(cluster).Error; err != nil {
				return err
			}
			splits := strings.Split(cluster.ImageRepo, "/")
			if len(splits) == 1 {
				splits = append(splits, "")
			}
			registry, repository := splits[0], splits[1]
			return gemsplugin.Bootstrap{Config: config}.Install(ctx, gemsplugin.GlobalValues{
				ImageRegistry:   registry,
				ImageRepository: repository,
				ClusterName:     cluster.ClusterName,
				StorageClass:    cluster.DefaultStorageClass,
				KubegemsVersion: version.Get().GitVersion,
				Runtime:         cluster.Runtime,
			})
		}); err != nil {
			log.Error(err, "create cluster failed")
			return err
		}

		h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
			msg.EventKind = msgbus.Add
			msg.ResourceType = msgbus.Cluster
			msg.ResourceID = cluster.ID
			msg.Detail = i18n.Sprintf(context.TODO(), "add a new cluster %s into kubegems", cluster.ClusterName)
			msg.ToUsers.Append(h.GetDataBase().SystemAdmins()...)
		})

		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.Created(c, cluster)
}

type ClusterQuota struct {
	statistics.ClusterStatistics
	OversoldConfig datatypes.JSON `json:"oversoldConfig"`
}

// ClusterStatistics 集群资源状态
// @Tags        Cluster
// @Summary     集群资源状态
// @Description 集群资源状态
// @Accept      json
// @Produce     json
// @Param       cluster_id path     int                                        true "cluster_id"
// @Success     200        {object} handlers.ResponseStruct{Data=ClusterQuota} "statistics"
// @Router      /v1/cluster/{cluster_id}/quota [get]
// @Security    JWT
func (h *ClusterHandler) ListClusterQuota(c *gin.Context) {
	h.cluster(c, func(ctx context.Context, cluster models.Cluster, cli agents.Client) (interface{}, error) {
		statistics := statistics.ClusterStatistics{}
		if err := cli.Extend().ClusterStatistics(ctx, &statistics); err != nil {
			return nil, err
		}
		return ClusterQuota{
			ClusterStatistics: statistics,
			OversoldConfig:    cluster.OversoldConfig,
		}, nil
	})
}

func (h *ClusterHandler) cluster(c *gin.Context, fun func(ctx context.Context, cluster models.Cluster, cli agents.Client) (interface{}, error)) {
	var cluster models.Cluster
	if err := h.GetDB().WithContext(c.Request.Context()).First(&cluster, c.Param(PrimaryKeyName)).Error; err != nil {
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
	restconfig, clientSet, err := kube.GetKubeClient(cluster.KubeConfig)
	if err != nil {
		return i18n.Errorf(c, "failed to build client via kubeconfig: %w", err)
	}
	serverSersion, err := clientSet.ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get the cluster APIServerInfo: %w", err)
	}
	ctx := c.Request.Context()
	nodes, err := clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list cluster's nodes: %w", err)
	}
	cluster.APIServer = restconfig.Host
	cluster.Version = serverSersion.String()

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

func (h *ClusterHandler) batchWithTimeout(ctx *gin.Context, clusters []*models.Cluster, timeout time.Duration, f func(idx int, name string, cli agents.Client)) {
	wg := sync.WaitGroup{}
	for idx, cluster := range clusters {
		wg.Add(1)
		go func(idx int, name string) error {
			cli, err := h.GetAgents().ClientOf(ctx, name)
			if err != nil {
				log.Error(err, "unable get agents client", "cluster", name)
				wg.Done()
				return nil
			}
			f(idx, name, cli)
			wg.Done()
			return nil
		}(idx, cluster.ClusterName)
	}
	utils.WaitGroupWithTimeout(&wg, timeout)
}
