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

package switcher

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"kubegems.io/kubegems/pkg/utils/msgbus"
)

type NotifyUser struct {
	Username     string
	UserID       uint
	RWLock       sync.RWMutex
	CurrentWatch map[string]map[string][]string
	Conn         *websocket.Conn
	wslock       sync.Mutex
	SessionID    string
}

func (nu *NotifyUser) IsWatchObject(msg *msgbus.NotifyMessage) bool {
	nu.RWLock.RLock()
	defer nu.RWLock.RUnlock()

	if msg.InvolvedObject == nil || len(nu.CurrentWatch) == 0 {
		return false
	}

	object := msg.InvolvedObject

	// cluster
	for cluster, kinds := range nu.CurrentWatch {
		if cluster == "*" || cluster == object.Cluster {
			// kind
			for kind, nms := range kinds {
				if kind == "*" || kind == object.Kind {
					// 如果是Add事件则无需继续匹配，直接发送
					if msg.EventKind == msgbus.Add {
						return true
					}

					// namespacedName
					for _, nm := range nms {
						if nm == "*" {
							return true
						}
						matchnamespace, matchname := msgbus.NamespacedNameSplit(nm)
						namespace, name := msgbus.NamespacedNameSplit(object.NamespacedName)
						// namespace
						if matchnamespace == "*" || matchnamespace == namespace {
							// name
							if matchname == "*" || matchname == name {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

func (nu *NotifyUser) SetCurrentWatch(w map[string]map[string][]string) {
	nu.RWLock.Lock()
	defer nu.RWLock.Unlock()
	nu.CurrentWatch = w
}

func (nu *NotifyUser) CloseConn() {
	nu.Conn.Close()
}

func (nu *NotifyUser) Read(into interface{}) error {
	return nu.Conn.ReadJSON(into)
}

func (nu *NotifyUser) Write(data interface{}) error {
	/*
		panic: concurrent write to websocket connection
	*/
	nu.wslock.Lock()
	defer nu.wslock.Unlock()

	return nu.Conn.WriteJSON(data)
}

func NewNotifyUser(conn *websocket.Conn, username string, userid uint) *NotifyUser {
	return &NotifyUser{
		Username:     username,
		UserID:       userid,
		Conn:         conn,
		RWLock:       sync.RWMutex{},
		CurrentWatch: msgbus.CurrentWatch{},
		SessionID:    uuid.NewString(),
	}
}
