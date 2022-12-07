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

package tunnel

import (
	"encoding/json"
	"time"
)

const (
	PacketKindData    PacketKind = iota // data or as a ack
	PacketKindConnect                   // handshake and auth
	PacketKindOpen                      // open connection
	PacketKindClose                     // close connect/stream
	PacketKindRoute                     // route update
)

type PacketKind int

type Packet struct {
	Kind    PacketKind
	Src     string
	Dest    string
	SrcCID  int64
	DestCID int64
	Data    []byte
	Error   string
}

type RouteUpdateKind int

const (
	RouteUpdateKindInvalid RouteUpdateKind = iota
	RouteUpdateKindReferesh
	RouteUpdateKindOnline
	RouteUpdateKindOffline
)

type Annotations map[string]string

type PacketDataRoute struct {
	Kind        RouteUpdateKind        `json:"kind,omitempty"`
	Annotations Annotations            `json:"annotations,omitempty"`
	Peers       map[string]Annotations `json:"peers,omitempty"`
}

type PacketDataConnect struct {
	Token string `json:"token,omitempty"`
}

type PacketDataOpen struct {
	Network string        `json:"network,omitempty"`
	Address string        `json:"address,omitempty"`
	Timeout time.Duration `json:"timeout,omitempty"`
}

func PacketEncode(data any) []byte {
	raw, _ := json.Marshal(data)
	return raw
}

func PacketDecode[T any](data []byte) T {
	ret := new(T)
	_ = json.Unmarshal(data, ret)
	return *ret
}
