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

package apis

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

type Watcher struct {
	Cache          cache.Cache
	connectionPool ConnectionPool
}

func NewWatcher(c cache.Cache) *Watcher {
	return &Watcher{
		Cache:          c,
		connectionPool: NewConnectionPool(),
	}
}

func (w *Watcher) Start() {
	for _, gvk := range validGVK {
		ifo, _ := w.Cache.GetInformerForKind(context.Background(), gvk)
		ifo.AddEventHandler(EvtHandler{watcher: w})
	}
}

// StreamWatch watch集群的事件 [不暴露给客户端]
func (w *Watcher) StreamWatch(c *gin.Context) {
	up := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.AbortWithStatus(400)
	}
	sconn := wrapperSyncConn(conn, w.connectionPool)
	w.connectionPool.Join(sconn)
}

func (w *Watcher) DispatchMessage(message interface{}) {
	go w.connectionPool.DispatchMessage(message)
}

func (w *Watcher) Notify(obj interface{}, evt msgbus.EventKind) {
	switch iobj := obj.(type) {
	case *corev1.Pod:
		nmsg := generateNotifyMessage(podGVK, iobj.ObjectMeta, obj, evt)
		w.DispatchMessage(nmsg)
	case *appsv1.Deployment:
		nmsg := generateNotifyMessage(deployGVK, iobj.ObjectMeta, obj, evt)
		w.DispatchMessage(nmsg)
	case *appsv1.DaemonSet:
		nmsg := generateNotifyMessage(dsGVK, iobj.ObjectMeta, obj, evt)
		w.DispatchMessage(nmsg)
	case *appsv1.StatefulSet:
		nmsg := generateNotifyMessage(stsGVK, iobj.ObjectMeta, obj, evt)
		w.DispatchMessage(nmsg)
	}
}

var (
	podGVK = schema.GroupVersionKind{
		Group: corev1.SchemeGroupVersion.Group, Version: corev1.SchemeGroupVersion.Version, Kind: "Pod",
	}
	deployGVK = schema.GroupVersionKind{
		Group: appsv1.SchemeGroupVersion.Group, Version: appsv1.SchemeGroupVersion.Version, Kind: "Deployment",
	}
	stsGVK = schema.GroupVersionKind{
		Group: appsv1.SchemeGroupVersion.Group, Version: appsv1.SchemeGroupVersion.Version, Kind: "StatefulSet",
	}
	dsGVK = schema.GroupVersionKind{
		Group: appsv1.SchemeGroupVersion.Group, Version: appsv1.SchemeGroupVersion.Version, Kind: "DaemonSet",
	}
	validGVK = []schema.GroupVersionKind{
		podGVK, deployGVK, stsGVK, dsGVK,
	}
)

type EvtHandler struct {
	watcher *Watcher
}

func (eh EvtHandler) OnUpdate(oldObj interface{}, newObj interface{}) {
	eh.watcher.Notify(newObj, msgbus.Update)
}

func (eh EvtHandler) OnDelete(obj interface{}) {
	eh.watcher.Notify(obj, msgbus.Delete)
}

func (eh EvtHandler) OnAdd(obj interface{}) {
	eh.watcher.Notify(obj, msgbus.Add)
}

func generateNotifyMessage(gvk schema.GroupVersionKind, meta metav1.ObjectMeta, obj interface{}, evt msgbus.EventKind) *msgbus.NotifyMessage {
	involed := msgbus.InvolvedObject{
		Group:          gvk.Group,
		Version:        gvk.Version,
		Kind:           gvk.Kind,
		NamespacedName: getNamespacedName(meta),
	}
	nMsg := msgbus.NotifyMessage{
		InvolvedObject: &involed,
		EventKind:      evt,
		Content:        obj,
		MessageType:    msgbus.Changed,
	}
	return &nMsg
}

func getNamespacedName(meta metav1.ObjectMeta) string {
	return fmt.Sprintf("%s/%s", meta.Namespace, meta.Name)
}

// ConnectionPool websocket 连接池，自动维护连接状态
type ConnectionPool interface {
	// Join 加入连接池
	Join(conn *syncConn)
	// 从连接池中删除指定客户端
	Remove(clients ...string)
	// 开始运行连接池
	Start()
	// 分发消息
	DispatchMessage(message interface{})
}

func NewConnectionPool() ConnectionPool {
	pool := &watcherConnectionPool{
		Locker:   &sync.Mutex{},
		pool:     map[string]*syncConn{},
		stopedCh: make(chan string, 100),
	}
	pool.Start()
	return pool
}

type watcherConnectionPool struct {
	sync.Locker
	pool     map[string]*syncConn
	stopedCh chan string
}

func (p *watcherConnectionPool) Join(conn *syncConn) {
	p.Lock()
	defer p.Unlock()
	p.pool[conn.clientID] = conn
}

func (p *watcherConnectionPool) Remove(clients ...string) {
	p.Lock()
	defer p.Unlock()
	for _, client := range clients {
		delete(p.pool, client)
	}
}

func (p *watcherConnectionPool) Start() {
	go func() {
		for {
			clientId := <-p.stopedCh
			p.Remove(clientId)
		}
	}()
}

func (p *watcherConnectionPool) DispatchMessage(message interface{}) {
	unhealthy := []string{}
	for clientID, conn := range p.pool {
		err := conn.WriteJSON(message)
		if err != nil {
			unhealthy = append(unhealthy, clientID)
		}
	}
	p.Remove(unhealthy...)
}

type syncConn struct {
	sync.Locker
	conn     *websocket.Conn
	clientID string
	pool     ConnectionPool
}

func wrapperSyncConn(conn *websocket.Conn, pool ConnectionPool) *syncConn {
	clientID := uuid.NewString()
	originHandler := conn.CloseHandler()
	conn.SetCloseHandler(func(code int, message string) error {
		pool.Remove(clientID)
		return originHandler(code, message)
	})
	return &syncConn{
		Locker:   &sync.Mutex{},
		conn:     conn,
		clientID: clientID,
		pool:     pool,
	}
}

func (syncConn *syncConn) WriteJSON(message interface{}) error {
	syncConn.Lock()
	defer syncConn.Unlock()
	timeout := time.After(time.Second * 3)
	errch := make(chan error, 1)
	go func() {
		errch <- syncConn.conn.WriteJSON(message)
	}()
	select {
	case err := <-errch:
		if err != nil {
			syncConn.conn.Close()
		}
		return err
	case <-timeout:
		syncConn.conn.Close()
		return fmt.Errorf("time out")
	}
}
