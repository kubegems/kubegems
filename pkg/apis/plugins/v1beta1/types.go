package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status of the plugin"
// +kubebuilder:printcolumn:name="InstallNamespace",type="string",JSONPath=".status.installNamespace",description="Install Namespace of the plugin"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Version of the plugin"
// +kubebuilder:printcolumn:name="UpgradeTimestamp",type="date",JSONPath=".status.upgradeTimestamp",description="UpgradeTimestamp of the plugin"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp of the plugin"
type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginSpec   `json:"spec,omitempty"`
	Status PluginStatus `json:"status,omitempty"`
}

type PluginSpec struct {
	Enabled          bool         `json:"enabled,omitempty"`          // enabled
	Kind             PluginKind   `json:"kind,omitempty"`             // plugin kind, e.g. "helm","kustomize","native".
	InstallNamespace string       `json:"installNamespace,omitempty"` // plugin install namespace, same with metadata.namespace if empty.
	Dependencies     []Dependency `json:"dependencies,omitempty"`     // dependencies on other plugins
	// +kubebuilder:pruning:PreserveUnknownFields
	Values  runtime.RawExtension `json:"values,omitempty"`  // plugin values, helm values
	Version string               `json:"version,omitempty"` // plugin version,also helm chart version
	Repo    string               `json:"repo,omitempty"`    // plugin repo url,optional
	Path    string               `json:"path,omitempty"`    // plugin repo path,optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Resources []runtime.RawExtension `json:"resources,omitempty"` // inline resources
}

type Dependency struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Version   string `json:"version,omitempty"`
}

type PluginStatus struct {
	Phase            PluginPhase `json:"phase,omitempty"`            // Phase is the current state of the release
	Message          string      `json:"message,omitempty"`          // Message is the message associated with the status
	InstallNamespace string      `json:"installNamespace,omitempty"` // plugin installNamespace,if empty use .metadata.namespace
	// +kubebuilder:pruning:PreserveUnknownFields
	Values            runtime.RawExtension `json:"values,omitempty"`
	Version           string               `json:"version,omitempty"`
	CreationTimestamp metav1.Time          `json:"creationTimestamp,omitempty"`
	UpgradeTimestamp  metav1.Time          `json:"upgradeTimestamp,omitempty"`
	DeletionTimestamp *metav1.Time         `json:"deletionTimestamp,omitempty"`
	Notes             string               `json:"notes,omitempty"` // Contains the rendered templates/NOTES.txt if available
}

//+kubebuilder:object:root=true
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Plugin `json:"items"`
}

type PluginKind string

const (
	PluginKindHelm      PluginKind = "helm"
	PluginKindKustomize PluginKind = "kustomize"
	PluginKindTemplate  PluginKind = "template"
	PluginKindInline    PluginKind = "inline"
)

type PluginPhase string

const (
	PluginPhaseNone      PluginPhase = "None" // No phase specified, plugin is not installed or removed
	PluginPhaseInstalled PluginPhase = "Installed"
	PluginPhaseFailed    PluginPhase = "Failed"
	PluginPhaseRemoved   PluginPhase = "Removed"
)
