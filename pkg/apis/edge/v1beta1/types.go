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
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status"
// +kubebuilder:printcolumn:name="RegisterAddress",type="string",JSONPath=".spec.register.address",description="Hub address for register"
// +kubebuilder:printcolumn:name="Token",type="string",JSONPath=".spec.register.bootstrapToken",description="Token used for register"
// +kubebuilder:printcolumn:name="LastOnline",type="string",JSONPath=".status.tunnel.lastOnlineTimestamp",description="CreationTimestamp of the bundle"
type EdgeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EdgeClusterSpec   `json:"spec,omitempty"`
	Status            EdgeClusterStatus `json:"status,omitempty"`
}

type EdgeClusterSpec struct {
	Register RegisterInfo `json:"register,omitempty"`
}

type RegisterInfo struct {
	HubName        string       `json:"hubName,omitempty"`        // register on hub
	ExpiresAt      *metav1.Time `json:"expiresAt,omitempty"`      // edge certs expires at,default 1 year
	Image          string       `json:"image,omitempty"`          // edge certs
	BootstrapToken string       `json:"bootstrapToken,omitempty"` // edge token
	Certs          *Certs       `json:"certs,omitempty"`          // pre generated certs
}

type EdgePhase string

const (
	EdgePhaseWaiting = "Waiting"
	EdgePhaseOnline  = "Online"
	EdgePhaseOffline = "Offline"
)

type EdgeClusterStatus struct {
	Phase       EdgePhase         `json:"phase,omitempty"`
	Register    RegisterStatus    `json:"register,omitempty"`
	Tunnel      TunnelStatus      `json:"tunnel,omitempty"`
	Manufacture ManufactureStatus `json:"manufacture,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status"
// +kubebuilder:printcolumn:name="Address",type="string",JSONPath=".status.address",description="Hub address for register"
// +kubebuilder:printcolumn:name="LastOnline",type="string",JSONPath=".status.tunnel.lastOnlineTimestamp",description="CreationTimestamp of the bundle"
type EdgeHub struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EdgeHubSpec   `json:"spec,omitempty"`
	Status            EdgeHubStatus `json:"status,omitempty"`
}
type EdgeHubSpec struct{}

type EdgeHubStatus struct {
	Phase       EdgePhase         `json:"phase,omitempty"`
	Address     string            `json:"address,omitempty"` // address of the hub
	Tunnel      TunnelStatus      `json:"tunnel,omitempty"`
	Manufacture ManufactureStatus `json:"manufacture,omitempty"`
}

type Certs struct {
	CA   []byte `json:"ca,omitempty"`
	Cert []byte `json:"cert,omitempty"`
	Key  []byte `json:"key,omitempty"`
}

type RegisterStatus struct {
	LastRegister      *metav1.Time `json:"lastRegister,omitempty"`
	LastRegisterToken string       `json:"lastRegisterToken,omitempty"`
	URL               string       `json:"url,omitempty"`
}

type TunnelStatus struct {
	Connected            bool         `json:"connected,omitempty"`
	LastOnlineTimestamp  *metav1.Time `json:"lastOnlineTimestamp,omitempty"`
	LastOfflineTimestamp *metav1.Time `json:"lastOfflineTimestamp,omitempty"`
}

type ManufactureStatus map[string]string

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
type EdgeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeCluster `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
type EdgeHubList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeHub `json:"items"`
}

var _ = SchemeBuilder.Register(
	&EdgeCluster{},
	&EdgeClusterList{},
	&EdgeHub{},
	&EdgeHubList{},
)
