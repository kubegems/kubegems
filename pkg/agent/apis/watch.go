package apis

import (
	"context"
	"fmt"
	"net/http"
	"sync"

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
	Cache       cache.Cache
	Connections sync.Map
}

// websocket conn 不支持并发,当前场景需要读写锁
type SyncConn struct {
	conn *websocket.Conn
	lock sync.RWMutex
}

func NewWatcher(c cache.Cache) *Watcher {
	return &Watcher{
		Cache:       c,
		Connections: sync.Map{},
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
	sessionID := uuid.NewString()
	w.join(conn, sessionID)
}

func (w *Watcher) send(obj interface{}) []string {
	failed := []string{}
	w.Connections.Range(func(k, c interface{}) bool {
		sconn := c.(*SyncConn)
		sconn.lock.Lock()
		defer sconn.lock.Unlock()
		e := sconn.conn.WriteJSON(obj)
		if e != nil {
			failed = append(failed, k.(string))
		}
		return true
	})
	return failed
}

func (w *Watcher) removeFailed(failed []string) {
	for _, k := range failed {
		w.Connections.Delete(k)
	}
}

func (w *Watcher) DispatchMessage(obj interface{}) {
	failedSessions := w.send(obj)
	w.removeFailed(failedSessions)
}

func (w *Watcher) join(conn *websocket.Conn, sessionid string) {
	sconn := &SyncConn{
		conn: conn,
		lock: sync.RWMutex{},
	}
	w.Connections.Store(sessionid, sconn)
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
