/*
Copyright 2021 kubegems.io.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeletePolicy string

const (
	// 删除关联的namespace
	DeletePolicyDelNamespace = "delNamespace"
	// 仅删除关联的namespace的label
	DeletePolicyDelLabels = "delLabels"
)

// EnvironmentSpec defines the desired state of Environment
type EnvironmentSpec struct {
	// Tenant 租户
	Tenant string `json:"tenant"`
	// Project 项目
	Project string `json:"project"`
	// Namespace 关联的ns
	Namespace string `json:"namespace"`
	// DeletePolicy  删除策略,选项为 delNamespace,delLabels
	DeletePolicy string `json:"deletePolicy"`
	// ResourceQuota  资源限制
	ResourceQuota corev1.ResourceList `json:"resourceQuota,omitempty"`
	// LimitRange  默认limitrange
	LimitRage []corev1.LimitRangeItem `json:"limitRange,omitempty"`
	// ResourceQuotaName
	ResourceQuotaName string `json:"resourceQuotaName,omitempty"`
	// LimitRageName
	LimitRageName string `json:"limitRangeName,omitempty"`
}

// EnvironmentStatus defines the observed state of Environment
type EnvironmentStatus struct {
	// 最后更新时间
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=tenv,singular=environment
//+kubebuilder:subresource:status
//+kubebuilder:rbac:groups=gems,resources=Environment,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems,resources=Environment/status,verbs=get;list;watch;create;update;patch;delete

// Environment is the Schema for the environments API
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec   `json:"spec,omitempty"`
	Status EnvironmentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EnvironmentList contains a list of Environment
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Environment{}, &EnvironmentList{})
}
