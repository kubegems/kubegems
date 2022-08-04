// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
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
	Backend   string                      `json:"backend,omitempty"` // kind of the model deployment
	Model     ModelSpec                   `json:"model,omitempty"`
	Replicas  *int32                      `json:"replicas,omitempty"`
	Env       []corev1.EnvVar             `json:"env,omitempty"`
	Args      []string                    `json:"args,omitempty"`
	Ports     []corev1.ContainerPort      `json:"ports,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Host is the hostname of the model serving endpoint
	// automatically generated if not specified
	// +optional
	Host string `json:"host,omitempty"`
}

type ModelDeploymentStatus struct {
	URL     string `json:"url,omitempty"` // url of the model deployment serving endpoint
	Phase   Phase  `json:"phase,omitempty"`
	Message string `json:"message,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	RawStatus *runtime.RawExtension `json:"oamStatus,omitempty"`
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
	Source string `json:"source"`
	// +kubebuilder:validation:Required
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url"`
	// +kubebuilder:validation:Optional
	Prameters  []Parameter `json:"parameters"`
	ServerType string      `json:"serverType"`
	// +kubebuilder:validation:Required
	Image string `json:"image"`
	// +kubebuilder:validation:Optional
	ContainerSpecOverrride *corev1.Container `json:"containerSpecOverrride,omitempty"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

//+kubebuilder:object:root=true
type ModelDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ModelDeployment `json:"items"`
}
