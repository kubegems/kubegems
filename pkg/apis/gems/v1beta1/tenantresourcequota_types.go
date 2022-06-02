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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TenantResourceQuotaSpec defines the desired state of TenantResourceQuota
type TenantResourceQuotaSpec struct {
	// Hard 租户在本集群的可以使用的总资源限制
	Hard corev1.ResourceList `json:"hard,omitempty"`
}

// TenantResourceQuotaStatus defines the observed state of TenantResourceQuota
type TenantResourceQuotaStatus struct {
	// Hard 租户在本集群的总资源限制
	Hard corev1.ResourceList `json:"hard,omitempty"`
	// Allocated 已经申请了的资源
	Allocated corev1.ResourceList `json:"allocated,omitempty"`
	// Used 实际使用了的资源
	Used corev1.ResourceList `json:"used,omitempty"`
	// Deprecated: duplicate with LastUpdateTime.
	// LastCountTime last count time
	LastCountTime metav1.Time `json:"lastCountTime,omitempty"`
	// LastUpdateTime last update time
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=tquota,path=tenantresourcequotas
//+kubebuilder:subresource:status
//+kubebuilder:rbac:groups=gems,resources=TenantResourceQuota,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems,resources=TenantResourceQuota/status,verbs=get;list;watch;create;update;patch;delete

// TenantResourceQuota is the Schema for the tenantresourcequota API
type TenantResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantResourceQuotaSpec   `json:"spec,omitempty"`
	Status TenantResourceQuotaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TenantResourceQuotaList contains a list of TenantResourceQuota
type TenantResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TenantResourceQuota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TenantResourceQuota{}, &TenantResourceQuotaList{})
}
