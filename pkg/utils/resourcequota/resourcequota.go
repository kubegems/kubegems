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

package resourcequota

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
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

const (
	DefaultResourceQuotaName = "default"
	DefaultLimitRangeName    = "default"
)

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

func IsLimitRangeInvalid(limitRangeItems []corev1.LimitRangeItem) ([]string, bool) {
	var (
		errmsg  []string
		invalid bool
	)
	for _, item := range limitRangeItems {
		for k, v := range item.DefaultRequest {
			if limitv, exist := item.Default[k]; exist {
				if v.Cmp(limitv) == 1 {
					l, _ := limitv.MarshalJSON()
					r, _ := v.MarshalJSON()
					msg := fmt.Sprintf("limitType %v error: %v limit value %v, requests value %v", item.Type, k, string(l), string(r))
					errmsg = append(errmsg, msg)
					invalid = true
				}
			}
		}
	}
	return errmsg, invalid
}

// ResourceEnough 资源 是否足够，不够给出不够的错误项
func ResourceEnough(total, used, need corev1.ResourceList) (bool, []string) {
	valid := true
	errmsgs := []string{}

	for k, needv := range need {
		totalv, totalExist := total[k]
		usedv, usedExist := used[k]
		if !totalExist || !usedExist {
			continue
		}
		totalv.Sub(usedv)
		if totalv.Cmp(needv) == -1 {
			valid = false
			left := totalv.DeepCopy()
			needv := needv.DeepCopy()
			msg := fmt.Sprintf("%v left %v but need %v", k.String(), left.String(), needv.String())
			errmsgs = append(errmsgs, msg)
		}
	}
	return valid, errmsgs
}

func ResourceIsEnough(total, used, need corev1.ResourceList, resources []corev1.ResourceName) (bool, []string) {
	ret := true
	msgs := []string{}
	for _, resource := range resources {
		totalv := total[resource]
		usedv := used[resource]
		needv := need[resource]
		if needv.IsZero() {
			continue
		}
		tmp := totalv.DeepCopy()
		tmp.Sub(usedv)
		if tmp.Cmp(needv) == -1 {
			l, _ := tmp.MarshalJSON()
			n, _ := needv.MarshalJSON()
			msg := fmt.Sprintf("%s not enough to apply, tenant left %s but need %s", resource, string(l), string(n))
			msgs = append(msgs, msg)
			ret = false
		}
	}
	return ret, msgs
}

// SubResource 用新的值去减去旧的，得到差
func SubResource(oldres, newres corev1.ResourceList) corev1.ResourceList {
	retres := corev1.ResourceList{}
	for k, v := range newres {
		ov, exist := oldres[k]
		if exist {
			v.Sub(ov)
			retres[k] = v
		} else {
			retres[k] = v
		}
	}
	return retres
}

// set request same as limit if not set
func SetSameRequestWithLimit(list corev1.ResourceList) {
	for k, v := range list {
		if index := strings.Index(string(k), "limits."); index != -1 {
			requestsResourceName := "requests." + string(k[index+7:])

			// if val, ok := list[v1.ResourceName(requestsResourceName)]; !ok || val.IsZero() {
			list[v1.ResourceName(requestsResourceName)] = v.DeepCopy()
			// }
		}
	}
}
