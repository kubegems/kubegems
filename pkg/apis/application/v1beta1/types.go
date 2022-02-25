package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ApplicationSpec `json:"spec,omitempty"`
}

// ApplicationSpec
type ApplicationSpec struct {
	Remark string            `json:"remark,omitempty"` // 备注
	Kind   string            `json:"kind,omitempty"`   // 类型
	Images []string          `json:"images,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

//+kubebuilder:object:root=true
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}
