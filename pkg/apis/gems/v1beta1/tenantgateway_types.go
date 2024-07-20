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

// Service defines the Service for the Ingress Controller.
type Service struct {
	// Specifies extra labels of the service.
	ExtraLabels map[string]string `json:"extraLabels,omitempty"`

	ExtraAnnotations map[string]string `json:"extraAnnotations,omitempty"`

	// Ports specifies the ports of the service.
	// +optional
	// +patchMergeKey=port
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=port
	// +listMapKey=protocol
	Ports []corev1.ServicePort `json:"ports,omitempty" patchMergeKey:"port" patchStrategy:"merge"`
}

type Workload struct {
	// Specifies resource request and limit of the nginx container
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Specifies extra labels of the workload(deployment) of nginx.
	// +kubebuilder:validation:Optional
	ExtraLabels map[string]string `json:"extraLabels,omitempty"`
}

// Image defines the Repository, Tag and ImagePullPolicy of the Ingress Controller Image.
type Image struct {
	// The repository of the image.
	Repository string `json:"repository"`
	// The tag (version) of the image.
	Tag string `json:"tag"`
	// The ImagePullPolicy of the image.
	PullPolicy corev1.PullPolicy `json:"pullPolicy"`
}

// TenantGatewaySpec defines the desired state of TenantGateway
type TenantGatewaySpec struct {
	// Type 负载均衡类型
	// +optional
	Type corev1.ServiceType `json:"type"` // NodePort or LoadBalancer
	// Replicas 负载均衡实例数
	// +optional
	Replicas *int32 `json:"replicas"`
	// Tenant 租户名
	Tenant string `json:"tenant"`
	// IngressClass 用以区分nginx作用域
	// +optional
	IngressClass string `json:"ingressClass"`
	// The image of the Ingress Controller.
	// +optional
	Image *Image `json:"image"`
	// The service of the Ingress controller.
	// +optional
	// +nullable
	Service *Service `json:"service"`
	// The Workload of the Ingress controller.
	// +optional
	// +nullable
	Workload *Workload `json:"workload"`
	// Initial values of the Ingress Controller ConfigMap.
	// Check https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/configmap-resource/ for
	// more information about possible values.
	// +optional
	// +nullable
	ConfigMapData map[string]string `json:"configMapData,omitempty"`
	// BaseDomain is a record to auto generate domain in ingress.
	// +optional
	BaseDomain string `json:"baseDomain"`
}

// TenantGatewayStatus defines the observed state of TenantGateway
type TenantGatewayStatus struct {
	// ActAvailableReplicasive nginx deployment 正常的pod数
	// +optional
	AvailableReplicas int32 `json:"availableReplicas"`
	// NodePort nginx service 占用的ports
	// +optional
	// +patchMergeKey=port
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=port
	// +listMapKey=protocol
	Ports []corev1.ServicePort `json:"ports,omitempty" patchMergeKey:"port" patchStrategy:"merge"`
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
