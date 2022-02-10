package switcher

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"kubegems.io/pkg/utils/msgbus"
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
