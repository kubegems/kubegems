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

import "time"

type AgentOptions struct {
	Listen            string        `json:"listen,omitempty"`
	ClientID          string        `json:"clientID,omitempty"`
	ClientIDSecret    string        `json:"clientIDSecret,omitempty" description:"client ID secret"`
	ManufactureFile   []string      `json:"manufactureFile,omitempty" description:"file with manufacture info in json object format"`
	ManufactureRemap  []string      `json:"manufactureRemap,omitempty" description:"remap manufacture file key to newkey,example 'foo=newfoo'"`
	Manufacture       []string      `json:"manufacture,omitempty" description:"manufacture kvs,example 'device-id=value,foo=bar'"`
	EdgeHubAddr       string        `json:"edgeHubAddr,omitempty"`
	KeepAliveInterval time.Duration `json:"keepAliveInterval,omitempty"`
}

func NewDefaultAgentOptions() *AgentOptions {
	return &AgentOptions{
		EdgeHubAddr:       "127.0.0.1:8080",
		ClientID:          "",
		Listen:            ":8080",
		KeepAliveInterval: 30 * time.Minute,
		ClientIDSecret:    "kubegems-edge-agent-id",
		ManufactureFile:   []string{"/etc/os-release"},
		ManufactureRemap:  []string{},
		Manufacture:       []string{},
	}
}
