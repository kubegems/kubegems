package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
type EdgeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EdgeClusterSpec   `json:"spec,omitempty"`
	Status            EdgeClusterStatus `json:"status,omitempty"`
}

type EdgeClusterSpec struct{}

type EdgeClusterStatus struct {
	Phase       string                    `json:"phase,omitempty"`
	Register    EdgeClusterStatusRegister `json:"register,omitempty"`
	Manufacture ManufactureStatus         `json:"manufacture,omitempty"`
}

type EdgeClusterStatusRegister struct {
	LastRegister metav1.Time `json:"lastRegister,omitempty"`
	LastReporr   metav1.Time `json:"lastReporr,omitempty"`
}

type ManufactureStatus map[string]string

// +kubebuilder:object:root=true
type EdgeClusterCridential struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EdgeClusterCridentialSpec   `json:"spec,omitempty"`
	Status            EdgeClusterCridentialStatus `json:"status,omitempty"`
}

type EdgeClusterCridentialSpec struct {
	Token string `json:"token,omitempty"`
}

type EdgeClusterCridentialStatus struct {
	Expire metav1.Time `json:"expire,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
type EdgeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeCluster `json:"items"`
}

// +kubebuilder:object:root=true
type EdgeClusterCridentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeCluster `json:"items"`
}

var _ = SchemeBuilder.Register(
	&EdgeCluster{},
	&EdgeClusterList{},
	&EdgeClusterCridential{},
	&EdgeClusterCridentialList{},
)
