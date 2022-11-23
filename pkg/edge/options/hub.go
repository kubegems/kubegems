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

import "kubegems.io/kubegems/pkg/edge/tunnel"

type HubOptions struct {
	Listen           string `json:"listen,omitempty"`
	ListenGrpc       string `json:"listenGrpc,omitempty"`
	Host             string `json:"host,omitempty" validate:"required"`
	ServerID         string `json:"serverID,omitempty" validate:"required"`
	TLS              *TLS   `json:"tls,omitempty"`
	CurrentNamespace string `json:"currentNamespace,omitempty"`
	EdgeServerAddr   string `json:"edgeServerAddr,omitempty"`
}

func NewDefaultHub() *HubOptions {
	return &HubOptions{
		Listen:           ":8080",
		ListenGrpc:       ":50051",
		TLS:              NewDefaultTLS(),
		ServerID:         tunnel.RandomServerID("hub-"),
		CurrentNamespace: "",
		EdgeServerAddr:   "127.0.0.1:50052",
	}
}
