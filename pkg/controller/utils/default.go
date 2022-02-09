package utils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

/*
租户级别限制:
cpu memory storage

环境级别限制
cpu memory storage
pods deployments statefulsets services configmaps secrets jobs cronjobs persistentvolumeclaims
*/

var (
	ResourceDeployments            corev1.ResourceName = "count/deployments.apps"
	ResourceStatefulSets           corev1.ResourceName = "count/statefulsets.apps"
	ResourceJobs                   corev1.ResourceName = "count/jobs.batch"
	ResourceCronJobs               corev1.ResourceName = "count/cronjobs.batch"
	ResourceSecrets                corev1.ResourceName = "count/secrets"
	ResourceConfigMaps             corev1.ResourceName = "count/configmaps"
	ResourceServices               corev1.ResourceName = "count/services"
	ResourcePersistentVolumeClaims corev1.ResourceName = "count/persistentvolumeclaims"
	ResourceDaemonsets             corev1.ResourceName = "count/daemonsets.apps"
	ResourceIngresses              corev1.ResourceName = "count/ingresses.extensions"
)

var (
	TenantLimitResources = []corev1.ResourceName{
		corev1.ResourceLimitsCPU,
		corev1.ResourceLimitsMemory,
		corev1.ResourceRequestsStorage,
	}
	EnvironmentLimitResources = []corev1.ResourceName{
		corev1.ResourceCPU, corev1.ResourceMemory, corev1.ResourceRequestsStorage,
		corev1.ResourcePods, ResourceDeployments, ResourceStatefulSets, ResourceServices, ResourceConfigMaps, ResourceSecrets, ResourceJobs, ResourceCronJobs, ResourcePersistentVolumeClaims,
	}
)

const (
	DefaultResourceQuotaName = "default"
	DefaultLimitRangeName    = "default"
)

func EmptyTenantResourceQuota() corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceLimitsCPU:       resource.MustParse("0"),
		corev1.ResourceLimitsMemory:    resource.MustParse("0Gi"),
		corev1.ResourceRequestsStorage: resource.MustParse("0Gi"),
	}
}

// GetDefaultTeantResourceQuota 获取默认的ResourceQuota
func GetDefaultTeantResourceQuota() corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceLimitsCPU:       resource.MustParse("0"),
		corev1.ResourceLimitsMemory:    resource.MustParse("0Gi"),
		corev1.ResourceRequestsStorage: resource.MustParse("0Gi"),
	}
}

// GetDefaultEnvironmentResourceQuota 环境的默认资源限制
func GetDefaultEnvironmentResourceQuota() corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceRequestsCPU:    resource.MustParse("0"),
		corev1.ResourceLimitsCPU:      resource.MustParse("0"),
		corev1.ResourceRequestsMemory: resource.MustParse("0Gi"),
		corev1.ResourceLimitsMemory:   resource.MustParse("0Gi"),

		corev1.ResourceRequestsStorage: resource.MustParse("0Gi"),
		corev1.ResourcePods:            resource.MustParse("5120"),

		ResourceDeployments:            resource.MustParse("512"),
		ResourceDaemonsets:             resource.MustParse("512"),
		ResourceIngresses:              resource.MustParse("512"),
		ResourceStatefulSets:           resource.MustParse("512"),
		ResourceJobs:                   resource.MustParse("512"),
		ResourceCronJobs:               resource.MustParse("512"),
		ResourceSecrets:                resource.MustParse("512"),
		ResourceConfigMaps:             resource.MustParse("512"),
		ResourceServices:               resource.MustParse("512"),
		ResourcePersistentVolumeClaims: resource.MustParse("512"),
	}
}

// GetDefaultEnvironmentLimitRange 环境默认的limitranger
func GetDefaultEnvironmentLimitRange() []corev1.LimitRangeItem {
	return []corev1.LimitRangeItem{
		// 单个Container的资源限制 不限制最小值
		{
			//  默认限制
			Default: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			//  默认请求
			DefaultRequest: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("10Mi"),
			},
			// 最大值
			Max: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("24"),
				corev1.ResourceMemory: resource.MustParse("48Gi"),
			},
			// 不限制最小
			Min: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("0m"),
				corev1.ResourceMemory: resource.MustParse("0Mi"),
			},
			Type: corev1.LimitTypeContainer,
		},
		// 单个POD的资源限制 不限制最小值
		{
			// 单个pod下的所有容器资源总量最大值
			Max: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("48"),
				corev1.ResourceMemory: resource.MustParse("64Gi"),
			},
			// 不限制最小
			Min: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("0m"),
				corev1.ResourceMemory: resource.MustParse("0Mi"),
			},
			Type: corev1.LimitTypePod,
		},
		// 单个PVC的SIZE限制
		{
			// 最大1T
			Max: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Ti"),
			},
			// 最小不限制
			Min: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("0Mi"),
			},
			Type: corev1.LimitTypePersistentVolumeClaim,
		},
	}
}
