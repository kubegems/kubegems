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

import (
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/utils/database"
)

type ServerOptions struct {
	Listen     string           `json:"listen,omitempty"`
	Host       string           `json:"host,omitempty"`
	ListenGrpc string           `json:"listenGrpc,omitempty"`
	ServerID   string           `json:"serverID,omitempty"`
	TLS        *TLS             `json:"tls,omitempty"`
	Database   database.Options `json:"database,omitempty"`
}

func NewDefaultServer() *ServerOptions {
	return &ServerOptions{
		Listen:     ":8080",
		ListenGrpc: ":50052",
		TLS:        NewDefaultTLS(),
		ServerID:   tunnel.RandomServerID("server-"),
	}
}
