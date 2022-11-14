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
