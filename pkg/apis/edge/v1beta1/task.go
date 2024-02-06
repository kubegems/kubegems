// Copyright 2023 The kubegems.io Authors
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

const EdgeTaskFinalizer = "edgetask.finalizers.edge.kubegems.io"

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status"
type EdgeTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EdgeTaskSpec   `json:"spec,omitempty"`
	Status            EdgeTaskStatus `json:"status,omitempty"`
}

type EdgeTaskSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	EdgeClusterName string `json:"edgeClusterName,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Resources []runtime.RawExtension `json:"resources,omitempty"`
}

type EdgeTaskPhase string

const (
	EdgeTaskPhaseWaiting EdgeTaskPhase = "Waiting"
	EdgeTaskPhaseRunning EdgeTaskPhase = "Running"
	EdgeTaskPhaseFailed  EdgeTaskPhase = "Failed"
)

type EdgeTaskStatus struct {
	ObservedGeneration int64                    `json:"observedGeneration,omitempty"`
	Phase              EdgeTaskPhase            `json:"phase,omitempty"`
	Conditions         []EdgeTaskCondition      `json:"conditions,omitempty"`
	ResourcesHash      string                   `json:"resourcesHash,omitempty"` // observed resources hash
	ResourcesStatus    []EdgeTaskResourceStatus `json:"resourcesStatus,omitempty"`
}

type EdgeTaskResourceStatus struct {
	APIVersion string                  `json:"apiVersion,omitempty"`
	Kind       string                  `json:"kind,omitempty"`
	Name       string                  `json:"name,omitempty"`
	Namespace  string                  `json:"namespace,omitempty"`
	Images     []string                `json:"images,omitempty"` // current workload images if exists
	Exists     bool                    `json:"exists,omitempty"`
	Ready      bool                    `json:"ready,omitempty"`
	Message    string                  `json:"message,omitempty"`
	Events     []EdgeTaskResourceEvent `json:"events,omitempty"`
	Status     runtime.RawExtension    `json:"status,omitempty"`
	PodsStatus []EdgeTaskPodStatus     `json:"podsStatus,omitempty"`
}

type EdgeTaskPodStatus struct {
	Name   string                  `json:"name,omitempty"`
	Status corev1.PodStatus        `json:"status,omitempty"`
	Events []EdgeTaskResourceEvent `json:"events,omitempty"` // current resource events
}

type EdgeTaskResourceEvent struct {
	Type          string      `json:"type,omitempty"`
	Reason        string      `json:"reason,omitempty"`
	Message       string      `json:"message,omitempty"`
	Count         int32       `json:"count,omitempty"`
	LastTimestamp metav1.Time `json:"lastTimestamp,omitempty"`
}

const (
	EdgeTaskConditionTypePrepared    EdgeTaskConditionType = "Prepared"    // prepare for the resource
	EdgeTaskConditionTypeOnline      EdgeTaskConditionType = "Online"      // edge cluster online
	EdgeTaskConditionTypeDistributed EdgeTaskConditionType = "Distributed" // distributed the resource
	EdgeTaskConditionTypeAvailable   EdgeTaskConditionType = "Available"   // resources is available
	EdgeTaskConditionTypeCleaned     EdgeTaskConditionType = "Cleaned"     // resources cleanup
)

type EdgeTaskConditionType string

type EdgeTaskCondition struct {
	Type               EdgeTaskConditionType  `json:"type,omitempty"`
	Status             corev1.ConditionStatus `json:"status,omitempty"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime,omitempty"`
	LastUpdateTime     metav1.Time            `json:"lastUpdateTime,omitempty"`
	Message            string                 `json:"message,omitempty"`
	Reason             string                 `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true
type EdgeTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeTask `json:"items"`
}
