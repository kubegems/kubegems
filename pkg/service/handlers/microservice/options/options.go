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

package options

import "kubegems.io/kubegems/pkg/apis/gems"

type MicroserviceOptions struct {
	KialiName         string `json:"kialiName,omitempty"`
	KialiNamespace    string `json:"kialiNamespace,omitempty"`
	GatewayNamespace  string `json:"gatewayNamespace,omitempty"`
	IstioNamespace    string `json:"istioNamespace,omitempty"`
	IstioOperatorName string `json:"istioOperatorName,omitempty"`
}

func NewDefaultOptions() *MicroserviceOptions {
	return &MicroserviceOptions{
		KialiName:         "kiali",
		KialiNamespace:    "istio-system",
		GatewayNamespace:  gems.NamespaceGateway,
		IstioNamespace:    "istio-system",
		IstioOperatorName: "kubegems-istio",
	}
}
