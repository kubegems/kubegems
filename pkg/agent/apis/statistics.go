package apis

import (
	"context"

	"github.com/gin-gonic/gin"
	gemsv1beta1 "github.com/kubegems/gems/pkg/apis/gems/v1beta1"
	"github.com/kubegems/gems/pkg/controller/utils"
	"github.com/kubegems/gems/pkg/datas"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatisticsHandler struct {
	C client.Client
}

// @Tags Agent.V1
// @Summary 获取集群内各种workload的统计
// @Description  获取集群内各种workload的统计
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/custom/statistics.system/v1/workloads [get]
// @Security JWT
func (sth *StatisticsHandler) ClusterWorkloadStatistics(c *gin.Context) {
	ret := map[string]int{}

	deployments := &appsv1.DeploymentList{}
	_ = sth.C.List(c.Request.Context(), deployments)
	ret[utils.ResourceDeployments.String()] = len(deployments.Items)

	statefulsetCounter := &appsv1.StatefulSetList{}
	_ = sth.C.List(c.Request.Context(), statefulsetCounter)
	ret[utils.ResourceStatefulSets.String()] = len(statefulsetCounter.Items)

	daemonsetCounter := &appsv1.DaemonSetList{}
	_ = sth.C.List(c.Request.Context(), daemonsetCounter)
	ret[utils.ResourceDaemonsets.String()] = len(daemonsetCounter.Items)

	podCounter := &corev1.PodList{}
	_ = sth.C.List(c.Request.Context(), podCounter)
	ret[corev1.ResourcePods.String()] = len(podCounter.Items)

	configmapCounter := &corev1.ConfigMapList{}
	_ = sth.C.List(c.Request.Context(), configmapCounter)
	ret[utils.ResourceConfigMaps.String()] = len(configmapCounter.Items)

	secretCounter := &corev1.SecretList{}
	_ = sth.C.List(c.Request.Context(), secretCounter)
	ret[utils.ResourceSecrets.String()] = len(secretCounter.Items)

	pvcCounter := &corev1.PersistentVolumeList{}
	_ = sth.C.List(c.Request.Context(), pvcCounter)
	ret[utils.ResourcePersistentVolumeClaims.String()] = len(pvcCounter.Items)

	serviceCounter := &corev1.ServiceList{}
	_ = sth.C.List(c.Request.Context(), serviceCounter)
	ret[utils.ResourceServices.String()] = len(serviceCounter.Items)

	cronjobCounter := &batchv1beta1.CronJobList{}
	_ = sth.C.List(c.Request.Context(), cronjobCounter)
	ret[utils.ResourceCronJobs.String()] = len(cronjobCounter.Items)

	jobCounter := &batchv1.JobList{}
	_ = sth.C.List(c.Request.Context(), jobCounter)
	ret[utils.ResourceJobs.String()] = len(jobCounter.Items)

	namespaceCounter := &corev1.NamespaceList{}
	_ = sth.C.List(c.Request.Context(), namespaceCounter)
	ret["namespace"] = len(namespaceCounter.Items)

	nodeCounter := &corev1.NodeList{}
	_ = sth.C.List(c.Request.Context(), nodeCounter)
	ret["node"] = len(nodeCounter.Items)

	OK(c, ret)
}

// ClusterResourceStatistics  获取集群级别资源统计
// @Tags Agent.V1
// @Summary 获取集群级别资源统计
// @Description  获取集群级别资源统计
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/custom/statistics.system/v1/resources [get]
// @Security JWT
func (sth *StatisticsHandler) ClusterResourceStatistics(c *gin.Context) {
	var (
		usedcpu              int64
		usedmem              int64
		capacitycpu          int64
		capacitymem          int64
		capacitysto          int64
		leftEphemeralStorage int64
		allocatedlmtcpu      int64
		allocatedreqcpu      int64
		allocatedlmtmem      int64
		allocatedreqmem      int64
		tallocatedcpu        int64
		tallocatedmem        int64
		tallocatedsto        int64
	)
	// 集群物理资源总量
	capacity := corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("0"),
		corev1.ResourceMemory:           resource.MustParse("0"),
		corev1.ResourceEphemeralStorage: resource.MustParse("0"),
	}
	// 集群物理资源使用量
	used := corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("0"),
		corev1.ResourceMemory:           resource.MustParse("0"),
		corev1.ResourceEphemeralStorage: resource.MustParse("0"),
	}
	// 集群所有workload申请量
	allocated := corev1.ResourceList{
		corev1.ResourceLimitsCPU:      resource.MustParse("0"),
		corev1.ResourceLimitsMemory:   resource.MustParse("0"),
		corev1.ResourceRequestsCPU:    resource.MustParse("0"),
		corev1.ResourceRequestsMemory: resource.MustParse("0"),
	}
	// 集群所有租户的申请量
	tenantAllocated := corev1.ResourceList{
		corev1.ResourceLimitsCPU:       resource.MustParse("0"),
		corev1.ResourceLimitsMemory:    resource.MustParse("0"),
		corev1.ResourceRequestsStorage: resource.MustParse("0"),
	}

	nodeList := &corev1.NodeList{}
	_ = sth.C.List(c.Request.Context(), nodeList)
	for _, node := range nodeList.Items {
		capacitycpu += node.Status.Capacity.Cpu().MilliValue()
		capacitymem += node.Status.Capacity.Memory().MilliValue()
		capacitysto += node.Status.Capacity.StorageEphemeral().MilliValue()

		leftEphemeralStorage += node.Status.Allocatable.StorageEphemeral().MilliValue()
	}

	capacity[corev1.ResourceCPU] = *resource.NewMilliQuantity(capacitycpu, resource.BinarySI)
	capacity[corev1.ResourceMemory] = *resource.NewMilliQuantity(capacitymem, resource.BinarySI)
	capacity[corev1.ResourceEphemeralStorage] = *resource.NewMilliQuantity(capacitysto, resource.BinarySI)

	podList := &corev1.PodList{}
	_ = sth.C.List(c.Request.Context(), podList)

	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		for _, container := range pod.Spec.Containers {
			allocatedlmtcpu += container.Resources.Limits.Cpu().MilliValue()
			allocatedreqcpu += container.Resources.Requests.Cpu().MilliValue()
			allocatedlmtmem += (*container.Resources.Limits.Memory().ToDec()).MilliValue()
			allocatedreqmem += (*container.Resources.Requests.Memory().ToDec()).MilliValue()
		}
	}
	allocated[corev1.ResourceLimitsCPU] = *resource.NewMilliQuantity(allocatedlmtcpu, resource.BinarySI)
	allocated[corev1.ResourceLimitsMemory] = *resource.NewMilliQuantity(allocatedlmtmem, resource.BinarySI)
	allocated[corev1.ResourceRequestsCPU] = *resource.NewMilliQuantity(allocatedreqcpu, resource.BinarySI)
	allocated[corev1.ResourceRequestsMemory] = *resource.NewMilliQuantity(allocatedreqmem, resource.BinarySI)

	tenantResourceQuotaList := &gemsv1beta1.TenantResourceQuotaList{}
	_ = sth.C.List(c, tenantResourceQuotaList)
	for _, tquota := range tenantResourceQuotaList.Items {
		tcpu := tquota.Spec.Hard[corev1.ResourceLimitsCPU]
		tallocatedcpu += tcpu.MilliValue()

		tmem := tquota.Spec.Hard[corev1.ResourceLimitsMemory]
		tallocatedmem += tmem.MilliValue()

		tsto := tquota.Spec.Hard[corev1.ResourceRequestsStorage]
		tallocatedsto += tsto.MilliValue()
	}
	tenantAllocated[corev1.ResourceLimitsCPU] = *resource.NewMilliQuantity(tallocatedcpu, resource.BinarySI)
	tenantAllocated[corev1.ResourceLimitsMemory] = *resource.NewMilliQuantity(tallocatedmem, resource.BinarySI)
	tenantAllocated[corev1.ResourceRequestsStorage] = *resource.NewMilliQuantity(tallocatedsto, resource.BinarySI)

	used[corev1.ResourceEphemeralStorage] = *resource.NewMilliQuantity(capacity.StorageEphemeral().MilliValue()-leftEphemeralStorage, resource.BinarySI)
	ctx := context.Background()

	nodeMetricsList := &v1beta1.NodeMetricsList{}
	_ = sth.C.List(ctx, nodeMetricsList)
	for _, nodemc := range nodeMetricsList.Items {
		usedcpu += nodemc.Usage.Cpu().MilliValue()
		usedmem += nodemc.Usage.Memory().MilliValue()
	}
	used[corev1.ResourceCPU] = *resource.NewMilliQuantity(usedcpu, resource.BinarySI)
	used[corev1.ResourceMemory] = *resource.NewMilliQuantity(usedmem, resource.BinarySI)

	ret := datas.ClusterResourceStatistics{
		Capacity:        capacity,
		Used:            used,
		Allocated:       allocated,
		TenantAllocated: tenantAllocated,
	}
	OK(c, ret)
}
