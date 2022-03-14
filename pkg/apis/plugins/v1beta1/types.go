package v1beta1

import (
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Namespaced
// nolint: tagliatelle
type Installer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstallerSpec   `json:"spec,omitempty"`
	Status InstallerStatus `json:"status,omitempty"`
}

type InstallerSpec struct {
	CluaterName string                `json:"cluaterName,omitempty"`
	Global      InstallerSpecGlobal   `json:"global,omitempty"`
	Plugins     []InstallerSpecPlugin `json:"plugins,omitempty"`
}

type InstallerSpecGlobal struct {
	Repository      string            `json:"repository,omitempty"`
	Imagepullsecret string            `json:"imagepullsecret,omitempty"`
	Storageclass    string            `json:"storageclass,omitempty"`
	Additional      map[string]string `json:"additional,omitempty"`
}

type InstallerSpecPlugin struct {
	Enabled   bool                 `json:"enabled,omitempty"`   // enabled
	Name      string               `json:"name,omitempty"`      // plugin name,also helm chart name
	Namespace string               `json:"namespace,omitempty"` // install namespace
	Labels    map[string]string    `json:"labels,omitempty"`    // plugin labels
	Version   string               `json:"version,omitempty"`   // plugin version,alsi helm chart version
	Values    runtime.RawExtension `json:"values,omitempty"`    // plugin values, helm values
}

type InstallerStatus struct {
	Phase             string                  `json:"phase,omitempty"`
	LastReconcileTime metav1.Time             `json:"lastReconcileTime,omitempty"`
	States            []InstallerStatusStatus `json:"states,omitempty"`
}

type InstallerStatusStatus struct {
	Name              string               `json:"name,omitempty"`
	Namespace         string               `json:"namespace,omitempty"`
	Message           string               `json:"message,omitempty"`
	Version           string               `json:"version,omitempty"`
	Values            runtime.RawExtension `json:"values,omitempty"`
	CreationTimestamp metav1.Time          `json:"creationTimestamp,omitempty"`
	UpgradeTimestamp  metav1.Time          `json:"upgradeTimestamp,omitempty"`
	DeletionTimestamp *metav1.Time         `json:"deletionTimestamp,omitempty"`
	Status            release.Status       `json:"status,omitempty"` // Status is the current state of the release
	Notes             string               `json:"notes,omitempty"`  // Contains the rendered templates/NOTES.txt if available
}

//+kubebuilder:object:root=true
// nolint: tagliatelle
type InstallerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Installer `json:"items"`
}
