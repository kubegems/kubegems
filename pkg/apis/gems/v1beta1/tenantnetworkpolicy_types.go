/*
Copyright 2021 cloudminds.com.

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

type EnvironmentNetworkPolicy struct {
	Project string `json:"project,omitempty"`
	Name    string `json:"name,omitempty"`
}

type ProjectNetworkPolicy struct {
	Name string `json:"name,omitempty"`
}

// TenantNetworkPolicySpec defines the desired state of TenantNetworkPolicy
type TenantNetworkPolicySpec struct {
	Tenant                     string                     `json:"tenant,omitempty"`
	TenantIsolated             bool                       `json:"tenantIsolated,omitempty"`
	ProjectNetworkPolicies     []ProjectNetworkPolicy     `json:"projectNetworkPolicies,omitempty"`
	EnvironmentNetworkPolicies []EnvironmentNetworkPolicy `json:"environmentNetworkPolicies,omitempty"`
}

// TenantNetworkPolicyStatus defines the observed state of TenantNetworkPolicy
type TenantNetworkPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=tnetpol
//+kubebuilder:subresource:status
//+kubebuilder:rbac:groups=gems,resources=TenantNetworkPolicy,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems,resources=TenantNetworkPolicy/status,verbs=get;list;watch;create;update;patch;delete

// TenantNetworkPolicy is the Schema for the tenantnetworkpolicies API
type TenantNetworkPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantNetworkPolicySpec   `json:"spec,omitempty"`
	Status TenantNetworkPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TenantNetworkPolicyList contains a list of TenantNetworkPolicy
type TenantNetworkPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TenantNetworkPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TenantNetworkPolicy{}, &TenantNetworkPolicyList{})
}
