package apis

import (
	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"kubegems.io/pkg/utils/resourcequota"
	"kubegems.io/pkg/utils/statistics"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatisticsHandler struct {
	C client.Client
}

// @Tags         Agent.V1
// @Summary      获取集群内各种workload的统计
// @Description  获取集群内各种workload的统计
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                true  "cluster"
// @Success      200      {object}  handlers.ResponseStruct{Data=object}  "counter"
// @Router       /v1/proxy/cluster/{cluster}/custom/statistics.system/v1/workloads [get]
// @Security     JWT
func (sth *StatisticsHandler) ClusterWorkloadStatistics(c *gin.Context) {
	ret := map[string]int{}

	deployments := &appsv1.DeploymentList{}
	_ = sth.C.List(c.Request.Context(), deployments)
	ret[resourcequota.ResourceDeployments.String()] = len(deployments.Items)

	statefulsetCounter := &appsv1.StatefulSetList{}
	_ = sth.C.List(c.Request.Context(), statefulsetCounter)
	ret[resourcequota.ResourceStatefulSets.String()] = len(statefulsetCounter.Items)

	daemonsetCounter := &appsv1.DaemonSetList{}
	_ = sth.C.List(c.Request.Context(), daemonsetCounter)
	ret[resourcequota.ResourceDaemonsets.String()] = len(daemonsetCounter.Items)

	podCounter := &corev1.PodList{}
	_ = sth.C.List(c.Request.Context(), podCounter)
	ret[corev1.ResourcePods.String()] = len(podCounter.Items)

	configmapCounter := &corev1.ConfigMapList{}
	_ = sth.C.List(c.Request.Context(), configmapCounter)
	ret[resourcequota.ResourceConfigMaps.String()] = len(configmapCounter.Items)

	secretCounter := &corev1.SecretList{}
	_ = sth.C.List(c.Request.Context(), secretCounter)
	ret[resourcequota.ResourceSecrets.String()] = len(secretCounter.Items)

	pvcCounter := &corev1.PersistentVolumeList{}
	_ = sth.C.List(c.Request.Context(), pvcCounter)
	ret[resourcequota.ResourcePersistentVolumeClaims.String()] = len(pvcCounter.Items)

	serviceCounter := &corev1.ServiceList{}
	_ = sth.C.List(c.Request.Context(), serviceCounter)
	ret[resourcequota.ResourceServices.String()] = len(serviceCounter.Items)

	cronjobCounter := &batchv1beta1.CronJobList{}
	_ = sth.C.List(c.Request.Context(), cronjobCounter)
	ret[resourcequota.ResourceCronJobs.String()] = len(cronjobCounter.Items)

	jobCounter := &batchv1.JobList{}
	_ = sth.C.List(c.Request.Context(), jobCounter)
	ret[resourcequota.ResourceJobs.String()] = len(jobCounter.Items)

	namespaceCounter := &corev1.NamespaceList{}
	_ = sth.C.List(c.Request.Context(), namespaceCounter)
	ret["namespace"] = len(namespaceCounter.Items)

	nodeCounter := &corev1.NodeList{}
	_ = sth.C.List(c.Request.Context(), nodeCounter)
	ret["node"] = len(nodeCounter.Items)

	OK(c, ret)
}

// ClusterResourceStatistics  获取集群级别资源统计
// @Tags         Agent.V1
// @Summary      获取集群级别资源统计
// @Description  获取集群级别资源统计
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                true  "cluster"
// @Success      200      {object}  handlers.ResponseStruct{Data=object}  "counter"
// @Router       /v1/proxy/cluster/{cluster}/custom/statistics.system/v1/resources [get]
// @Security     JWT
func (sth *StatisticsHandler) ClusterResourceStatistics(c *gin.Context) {
	clusterResourceStatistics := statistics.GetClusterResourceStatistics(c, sth.C)
	OK(c, clusterResourceStatistics)
}
