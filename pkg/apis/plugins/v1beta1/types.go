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
// +kubebuilder:printcolumn:name="Kind",type="string",JSONPath=".spec.kind",description="Kind of the bundle"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status of the bundle"
// +kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".status.namespace",description="Install Namespace of the bundle"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Version of the bundle"
// +kubebuilder:printcolumn:name="AppVersion",type="string",JSONPath=".status.appVersion",description="app version of the bundle"
// +kubebuilder:printcolumn:name="UpgradeTimestamp",type="date",JSONPath=".status.upgradeTimestamp",description="UpgradeTimestamp of the bundle"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp of the bundle"
type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginSpec   `json:"spec,omitempty"`
	Status PluginStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Plugin `json:"items"`
}
