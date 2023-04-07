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

package agent

import "time"

const (
	ClientIDSecret           = "kubegems-edge-agent-id"
	DefaultKeepAliveInterval = 30 * time.Minute
)

type Options struct {
	Listen            string        `json:"listen,omitempty"`
	DeviceID          string        `json:"deviceID,omitempty" description:"device id in kubegems edge,use random generated client-id by default"`
	DeviceIDKey       string        `json:"deviceIDKey,omitempty" description:"use value of key as device-id in manufacture"`
	ManufactureFile   []string      `json:"manufactureFile,omitempty" description:"file with manufacture info in json object format"`
	ManufactureRemap  []string      `json:"manufactureRemap,omitempty" description:"remap manufacture file key to newkey,example 'newkey=existskey'"`
	Manufacture       []string      `json:"manufacture,omitempty" description:"manufacture kvs,example 'some-key=value,foo=bar'"`
	EdgeHubAddr       string        `json:"edgeHubAddr,omitempty"`
	KeepAliveInterval time.Duration `json:"keepAliveInterval,omitempty"`
	TLS               *ClientTLS    `json:"tls,omitempty" description:"skip server tls verify"`
}

type ClientTLS struct {
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

func NewDefaultOptions() *Options {
	return &Options{
		EdgeHubAddr:       "127.0.0.1:8080",
		Listen:            ":8080",
		DeviceID:          "",
		DeviceIDKey:       "",
		KeepAliveInterval: DefaultKeepAliveInterval,
		ManufactureFile:   []string{"/etc/os-release"},
		ManufactureRemap:  []string{},
		Manufacture:       []string{},
		TLS:               &ClientTLS{},
	}
}
