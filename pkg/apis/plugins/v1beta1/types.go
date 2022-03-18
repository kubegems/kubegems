package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Namespaced
//+kubebuilder:subresource:status
type Installer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstallerSpec   `json:"spec,omitempty"`
	Status InstallerStatus `json:"status,omitempty"`
}

type InstallerSpec struct {
	CluaterName string `json:"cluaterName,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	Global  runtime.RawExtension  `json:"global,omitempty"`
	Plugins []InstallerSpecPlugin `json:"plugins,omitempty"`
}

type InstallerSpecGlobal struct {
	Repository      string            `json:"repository,omitempty"`
	Imagepullsecret string            `json:"imagepullsecret,omitempty"`
	Storageclass    string            `json:"storageclass,omitempty"`
	Additional      map[string]string `json:"additional,omitempty"`
}

type InstallerSpecPluginKind string

const (
	InstallerSpecPluginKindHelm      InstallerSpecPluginKind = "helm"
	InstallerSpecPluginKindKustomize InstallerSpecPluginKind = "kustomize"
	InstallerSpecPluginKindNative    InstallerSpecPluginKind = "native"
)

type InstallerSpecPlugin struct {
	Enabled   bool                    `json:"enabled,omitempty"`   // enabled
	Kind      InstallerSpecPluginKind `json:"kind,omitempty"`      // plugin kind, e.g. "helm","kustomize","native".
	Name      string                  `json:"name,omitempty"`      // plugin name,also helm chart name
	Namespace string                  `json:"namespace,omitempty"` // install namespace
	Labels    map[string]string       `json:"labels,omitempty"`    // plugin labels
	Version   string                  `json:"version,omitempty"`   // plugin version,alsi helm chart version

	// +kubebuilder:pruning:PreserveUnknownFields
	Values runtime.RawExtension `json:"values,omitempty"` // plugin values, helm values
}

type InstallerStatus struct {
	Phase             string                  `json:"phase,omitempty"`
	LastReconcileTime metav1.Time             `json:"lastReconcileTime,omitempty"`
	States            []InstallerStatusStatus `json:"states,omitempty"`
}

type Status string

// NOTE: Make sure to update cmd/helm/status.go when adding or modifying any of these statuses.
const (
	// StatusNotInstalled indicates that a release is not installed.
	StatusNotInstall Status = "notinstall"
	// StatusUnknown indicates that a release is in an uncertain state.
	StatusUnknown Status = "unknown"
	// StatusDeployed indicates that the release has been pushed to Kubernetes.
	StatusDeployed Status = "deployed"
	// StatusUninstalled indicates that a release has been uninstalled from Kubernetes.
	StatusUninstalled Status = "uninstalled"
	// StatusSuperseded indicates that this release object is outdated and a newer one exists.
	StatusSuperseded Status = "superseded"
	// StatusFailed indicates that the release was not successfully deployed.
	StatusFailed Status = "failed"
	// StatusUninstalling indicates that a uninstall operation is underway.
	StatusUninstalling Status = "uninstalling"
	// StatusPendingInstall indicates that an install operation is underway.
	StatusPendingInstall Status = "pending-install"
	// StatusPendingUpgrade indicates that an upgrade operation is underway.
	StatusPendingUpgrade Status = "pending-upgrade"
	// StatusPendingRollback indicates that an rollback operation is underway.
	StatusPendingRollback Status = "pending-rollback"
)

type InstallerStatusStatus struct {
	Name      string                  `json:"name,omitempty"`
	Namespace string                  `json:"namespace,omitempty"`
	Kind      InstallerSpecPluginKind `json:"kind,omitempty"`   // plugin kind, e.g. "helm","kustomize","native".
	Status    Status                  `json:"status,omitempty"` // Status is the current state of the release
	Message   string                  `json:"message,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	Values            runtime.RawExtension `json:"values,omitempty"`
	Version           string               `json:"version,omitempty"`
	CreationTimestamp metav1.Time          `json:"creationTimestamp,omitempty"`
	UpgradeTimestamp  metav1.Time          `json:"upgradeTimestamp,omitempty"`
	DeletionTimestamp *metav1.Time         `json:"deletionTimestamp,omitempty"`
	Notes             string               `json:"notes,omitempty"` // Contains the rendered templates/NOTES.txt if available
}

//+kubebuilder:object:root=true
type InstallerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Installer `json:"items"`
}
