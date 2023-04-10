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

package common

const (
	AnnotationKeyEdgeHubAddress = "edge.kubegems.io/edge-hub-address"
	AnnotationKeyEdgeHubCert    = "edge.kubegems.io/edge-hub-key"
	AnnotationKeyEdgeHubCA      = "edge.kubegems.io/edge-hub-ca"
	AnnotationKeyEdgeHubKey     = "edge.kubegems.io/edge-hub-cert"
	LabelKeIsyEdgeHub           = "edge.kubegems.io/is-edge-hub"

	AnnotationKeyEdgeAgentAddress           = "edge.kubegems.io/edge-agent-address"
	AnnotationKeyEdgeAgentKeepaliveInterval = "edge.kubegems.io/edge-agent-keepalive-interval"
	AnnotationKeyEdgeAgentRegisterAddress   = "edge.kubegems.io/edge-agent-register-address"
	AnnotationKeyKubernetesVersion          = "edge.kubegems.io/kubernetes-version"
	AnnotationKeyAPIserverAddress           = "edge.kubegems.io/apiserver-address"
	AnnotationKeyNodesCount                 = "edge.kubegems.io/nodes-count"
	AnnotationKeyDeviceID                   = "edge.kubegems.io/device-id"
	AnnotationKeyExternalIP                 = "edge.kubegems.io/external-ip"

	// temporary connection do not write to database
	AnnotationIsTemporaryConnect = "edge.kubegems.io/temporary-connect"

	// edge agent default address
	AnnotationValueDefaultEdgeAgentAddress = "http://127.0.0.1:8080"
)
