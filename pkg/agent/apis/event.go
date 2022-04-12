package apis

import (
	"fmt"

	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EventHandler struct {
	C client.Client
}

// @Tags         Agent.V1
// @Summary      获取Event列表数据
// @Description  获取Event列表数据
// @Accept       json
// @Produce      json
// @Param        order      query     string                                                            false  "page"
// @Param        page       query     int                                                               false  "page"
// @Param        size       query     int                                                               false  "page"
// @Param        search     query     string                                                            false  "search"
// @Param        namespace  path      string                                                            true   "namespace"
// @Param        cluster    path      string                                                            true   "cluster"
// @Param        topkind    query     string                                                            false  "topkind"
// @Param        topname    query     string                                                            false  "topname"
// @Success      200        {object}  handlers.ResponseStruct{Data=pagination.PageData{List=[]object}}  "Event"
// @Router       /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/events [get]
// @Security     JWT
func (h *EventHandler) List(c *gin.Context) {
	ns := c.Param("namespace")
	if ns == "_all" || ns == "_" {
		ns = ""
	}

	events := &v1.EventList{}
	if err := h.C.List(c.Request.Context(), events, client.InNamespace(ns)); err != nil {
		NotOK(c, err)
		return
	}

	objects := h.filterByTopKind(c, events.Items)

	pageData := NewPageDataFromContext(c, func(i int) SortAndSearchAble {
		return &objects[i]
	}, len(objects), objects)
	OK(c, pageData)
}

func (h *EventHandler) filterByTopKind(c *gin.Context, evts []v1.Event) []v1.Event {
	topkind := c.Query("topkind")
	topname := c.Query("topname")
	if len(topkind) == 0 || len(topname) == 0 {
		return evts
	}

	ns := c.Params.ByName("namespace")

	involvedMap := map[string]bool{
		involvedObjectKindName(topkind, topname): true,
	}

	switch topkind {
	case "Deployment":
		deploy := &appsv1.Deployment{}
		err := h.C.Get(c.Request.Context(), types.NamespacedName{Namespace: ns, Name: topname}, deploy)
		if err != nil {
			goto GOTO
		}
		replicasets := &appsv1.ReplicaSetList{}
		err = h.C.List(c.Request.Context(), replicasets, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(deploy.Spec.Selector.MatchLabels),
		})
		if err != nil {
			goto GOTO
		}
		for _, rs := range replicasets.Items {
			involvedMap[involvedObjectKindName("ReplicaSet", rs.Name)] = true
		}
	case "DaemonSet":
		ds := &appsv1.DaemonSet{}
		err := h.C.Get(c.Request.Context(), types.NamespacedName{Namespace: ns, Name: topname}, ds)
		if err != nil {
			goto GOTO
		}
		involvedMap[involvedObjectKindName("DaemonSet", ds.Name)] = true
	case "StatefulSet":
		sts := &appsv1.StatefulSet{}
		err := h.C.Get(c.Request.Context(), types.NamespacedName{Namespace: ns, Name: topname}, sts)
		if err != nil {
			goto GOTO
		}
		involvedMap[involvedObjectKindName("StatefulSet", sts.Name)] = true
	}

GOTO:
	var ret []v1.Event
	for _, evt := range evts {
		if _, exist := involvedMap[involvedObjectKindName(evt.InvolvedObject.Kind, evt.InvolvedObject.Name)]; exist {
			ret = append(ret, evt)
		}
	}
	return ret
}

func involvedObjectKindName(kind, name string) string {
	return fmt.Sprintf("%s--%s", kind, name)
}
