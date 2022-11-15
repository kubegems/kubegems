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
	PacketKindOpen                      // dial network
	PacketKindClose                     // close connect/stream
	PacketKindRoute                     // route update
)

type PacketKind int

type Packet struct {
	Kind   PacketKind
	Src    string
	Dest   string
	SrcID  int64
	DestID int64
	Data   []byte
	Error  string
}

type PeerUpdateKind string

const (
	PeerUpdateKindAdd     PeerUpdateKind = "add"
	PeerUpdateKindRemove  PeerUpdateKind = "remove"
	PeerUpdateKindRefresh PeerUpdateKind = "refresh"
)

type PacketDataRoute struct {
	Kind     PeerUpdateKind `json:"kind,omitempty"`
	SubPeers []string       `json:"subPeers,omitempty"`
}

type PacketDataConnect struct {
	Token   string      `json:"token,omitempty"`
	Options PeerOptions `json:"options,omitempty"` // peer options
}

type PacketDataDialOptions struct {
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
