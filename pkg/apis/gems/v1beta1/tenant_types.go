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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
	// TenantName 租户名字
	TenantName string `json:"tenantName,omitempty"`
	// Admin 租户管理员列表
	Admin []string `json:"admin"`
	// Members 租户成员列表
	Members []string `json:"members,omitempty"`
}

// TenantStatus defines the observed state of Tenant
type TenantStatus struct {
	// Environments 租户在本集群管控的环境
	Environments []string `json:"environments,omitempty"`
	// Namespaces 租户在本集群管控的namespace
	Namespaces []string `json:"namespaces,omitempty"`
	// LastUpdateTime 最后更新时间
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=ten,singular=tenant
//+kubebuilder:subresource:status
//+kubebuilder:rbac:groups=gems,resources=Tenant,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems,resources=Tenant/status,verbs=get;list;watch;create;update;patch;delete

// Tenant is the Schema for the tenants API
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TenantList contains a list of Tenant
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}
