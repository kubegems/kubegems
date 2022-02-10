package datas

import (
	corev1 "k8s.io/api/core/v1"
)

type ClusterResourceStatistics struct {
	// 集群资源的总容量，即物理资源总量
	Capacity corev1.ResourceList `json:"capacity"`
	// 集群资源的真实使用量
	Used corev1.ResourceList `json:"used"`
	// 集群下的资源分配总量
	Allocated corev1.ResourceList `json:"allocated"`
	// 集群下的租户资源分配总量
	TenantAllocated corev1.ResourceList `json:"tenantAllocated"`
}

type ClusterWorkloadStatistics map[string]int
