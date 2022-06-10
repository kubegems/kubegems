package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	bundlev1 "kubegems.io/bundle-controller/pkg/apis/bundle/v1beta1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Kind",type="string",JSONPath=".spec.kind",description="Kind of the bundle"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status of the bundle"
// +kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".status.namespace",description="Install Namespace of the bundle"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Version of the bundle"
// +kubebuilder:printcolumn:name="UpgradeTimestamp",type="date",JSONPath=".status.upgradeTimestamp",description="UpgradeTimestamp of the bundle"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp of the bundle"
type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   bundlev1.BundleSpec   `json:"spec,omitempty"`
	Status bundlev1.BundleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Plugin `json:"items"`
}
