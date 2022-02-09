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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Service defines the Service for the Ingress Controller.
type Service struct {
	// Specifies extra labels of the service.
	ExtraLabels map[string]string `json:"extraLabels,omitempty"`
}

type Workload struct {
	// Specifies resource request and limit of the nginx container
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Specifies extra labels of the workload(deployment) of nginx.
	// +kubebuilder:validation:Optional
	ExtraLabels map[string]string `json:"extraLabels,omitempty"`
}

// TenantGatewaySpec defines the desired state of TenantGateway
type TenantGatewaySpec struct {
	// Type 负载均衡类型
	Type corev1.ServiceType `json:"type"` // NodePort or LoadBalancer
	// Replicas 负载均衡实例数
	Replicas *int32 `json:"replicas"`
	// Tenant 租户名
	Tenant string `json:"tenant"`
	// IngressClass 用以区分nginx作用域
	IngressClass string `json:"ingressClass"`
	// The service of the Ingress controller.
	// +kubebuilder:validation:Optional
	// +nullable
	Service *Service `json:"service"`
	// The Workload of the Ingress controller.
	// +kubebuilder:validation:Optional
	// +nullable
	Workload *Workload `json:"workload"`
	// Initial values of the Ingress Controller ConfigMap.
	// Check https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/configmap-resource/ for
	// more information about possible values.
	// +kubebuilder:validation:Optional
	// +nullable
	ConfigMapData map[string]string `json:"configMapData,omitempty"`
	// BaseDomain is a record to auto generate domain in ingress.
	// +kubebuilder:validation:Optional
	BaseDomain string `json:"baseDomain"`
}

// TenantGatewayStatus defines the observed state of TenantGateway
type TenantGatewayStatus struct {
	// ActAvailableReplicasive nginx deployment 正常的pod数
	AvailableReplicas int32 `json:"availableReplicas"`
	// NodePort nginx service 占用的ports
	Ports []corev1.ServicePort `json:"ports"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=tgw
//+kubebuilder:subresource:status
//+kubebuilder:rbac:groups=gems,resources=TenantGateway,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems,resources=TenantGateway/status,verbs=get;list;watch;create;update;patch;delete

// TenantGateway is the Schema for the tenantgateways API
type TenantGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantGatewaySpec   `json:"spec,omitempty"`
	Status TenantGatewayStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TenantGatewayList contains a list of TenantGateway
type TenantGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TenantGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TenantGateway{}, &TenantGatewayList{})
}
