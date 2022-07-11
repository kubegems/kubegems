package v1beta1

import (
	oamcommon "github.com/oam-dev/kubevela/apis/core.oam.dev/common"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.model.name",name="MODEL",description="Status of the resource",type=string
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="PHASE",description="Status of the resource",type=string
type ModelDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ModelDeploymentSpec   `json:"spec,omitempty"`
	Status            ModelDeploymentStatus `json:"status,omitempty"`
}

// ModelDeploymentSpec is the spec for a ModelDeployment
type ModelDeploymentSpec struct {
	Type      string                      `json:"type,omitempty"`
	Model     ModelSpec                   `json:"model,omitempty"`
	ModelPath string                      `json:"modelPath,omitempty"` // path to mount the model from store
	Replicas  *int32                      `json:"replicas,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Host is the hostname of the model serving endpoint
	// automatically generated if not specified
	// +optional
	Host string `json:"host,omitempty"`
}

type ModelDeploymentStatus struct {
	Phase     Phase               `json:"phase,omitempty"`
	Message   string              `json:"message,omitempty"`
	OAMStatus oamcommon.AppStatus `json:"oamStatus,omitempty"`
}

type Phase string

// These are the valid statuses of pods.
const (
	// PodPending means the pod has been accepted by the system, but one or more of the containers
	// has not been started. This includes time before being bound to a node, as well as time spent
	// pulling images onto the host.
	Pending Phase = "Pending"
	// PodRunning means the pod has been bound to a node and all of the containers have been started.
	// At least one container is still running or is in the process of being restarted.
	Running Phase = "Running"
	// PodFailed means that all containers in the pod have terminated, and at least one container has
	// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
	Failed Phase = "Failed"
)

type ModelSpec struct {
	// +kubebuilder:validation:Required
	Name      string `json:"name,omitempty"`
	Version   string `json:"version,omitempty"`
	URL       string `json:"url,omitempty"`
	Framework string `json:"framework,omitempty"`
	// +kubebuilder:validation:Required
	Image string `json:"image,omitempty"`
}

//+kubebuilder:object:root=true
type ModelDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ModelDeployment `json:"items"`
}
