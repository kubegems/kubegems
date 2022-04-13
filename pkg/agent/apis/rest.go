package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/pagination"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type REST struct {
	client  client.Client
	cluster cluster.Interface
}

type GVKN struct {
	Action string
	schema.GroupVersionKind
	Namespace     string
	Resource      string
	Name          string
	Labels        map[string]string
	LabelSelector string
}

// @Tags Agent.V1
// @Summary  获取 none namespaced scope workload
// @Description 获取 none namespaced scope workload
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/{resource}/{name} [get]
// @Security JWT
func _() {}

// @Tags Agent.V1
// @Summary 获取namespaced scope workload
// @Description 获取namespaced scope workload
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Param namespace path string true "namespace"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [get]
// @Security JWT
func (h *REST) Get(c *gin.Context) {
	obj, gvkn, err := h.readObject(c, false)
	if err != nil {
		NotOK(c, err)
		return
	}
	if err = h.client.Get(c.Request.Context(),
		types.NamespacedName{Namespace: gvkn.Namespace, Name: gvkn.Name}, obj); err != nil {
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

// @Tags Agent.V1
// @Summary  获取 none namespaced scope workload  list
// @Description 获取 none namespaced scope workload  list
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Success 200 {object} handlers.ResponseStruct{Data=[]object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/{resource} [get]
// @Security JWT
func _() {}

// @Tags Agent.V1
// @Summary 获取namespaced scope workload  list
// @Description 获取namespaced scope workload  list
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param namespace path string true "namespace"
// @Param watch query bool true "watch"
// @Success 200 {object} handlers.ResponseStruct{Data=[]object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource} [get]
// @Security JWT
func (h *REST) List(c *gin.Context) {
	list, gvkn, err := h.listObject(c)
	if err != nil {
		NotOK(c, err)
		return
	}
	listOptions := &client.ListOptions{
		Namespace:     gvkn.Namespace,
		LabelSelector: parseSels(gvkn.Labels, gvkn.LabelSelector),
	}
	if err := h.client.List(c.Request.Context(), list, listOptions); err != nil {
		NotOK(c, err)
		return
	}
	items, err := ExtractList(list)
	if err != nil {
		NotOK(c, err)
		return
	}
	pageData := pagination.NewTypedSearchSortPageResourceFromContext(c, items)
	if iswatch, _ := strconv.ParseBool(c.Param("watch")); iswatch {
		// list
		c.SSEvent("data", pageData)
		c.Writer.Flush()
		// watch
		WatchEvents(c, h.cluster, list, listOptions)
		return
	} else {
		OK(c, pageData)
		return
	}
}

func ExtractList(obj runtime.Object) ([]client.Object, error) {
	itemsPtr, err := meta.GetItemsPtr(obj)
	if err != nil {
		return nil, err
	}
	items, err := conversion.EnforcePtr(itemsPtr)
	if err != nil {
		return nil, err
	}
	list := make([]client.Object, items.Len())
	for i := range list {
		raw := items.Index(i)
		switch item := raw.Interface().(type) {
		case client.Object:
			list[i] = item
		default:
			var found bool
			if list[i], found = raw.Addr().Interface().(client.Object); !found {
				return nil, fmt.Errorf("%v: item[%v]: Expected object, got %#v(%s)", obj, i, raw.Interface(), raw.Kind())
			}
		}
	}
	return list, nil
}

func WatchEvents(c *gin.Context, cluster cluster.Interface, list client.ObjectList, opts ...client.ListOption) error {
	// watch
	ctx, cancelFunc := context.WithCancel(c.Request.Context())
	defer cancelFunc()

	go func() {
		<-c.Writer.CloseNotify()
		cancelFunc()
	}()

	onEvent := func(e watch.Event) error {
		c.SSEvent("data", e.Object)
		c.Writer.Flush()
		return nil
	}

	if err := cluster.Watch(ctx, list, onEvent, opts...); err != nil {
		log.
			WithField("watch", list.GetObjectKind().GroupVersionKind().GroupKind().String()).
			Warn(err.Error())
	}
	return nil
}

// @Tags Agent.V1
// @Summary  创建namespaced scope workload
// @Description 创建namespaced scope workload
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Param data body object true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/{resource}/{name} [post]
// @Security JWT
func _() {}

// @Tags Agent.V1
// @Summary 创建 none namespaced scope workload
// @Description 创建 none namespaced scope workload
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Param namespace path string true "namespace"
// @Param data body object true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [post]
// @Security JWT
func (h *REST) Create(c *gin.Context) {
	obj, _, err := h.readObject(c, true)
	if err != nil {
		NotOK(c, err)
		return
	}
	if err := h.client.Create(c.Request.Context(), obj); err != nil {
		log.Warnf("create object failed: %v", err)
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

// @Tags Agent.V1
// @Summary  创建none namespaced scope workload
// @Description 创建none amespaced scope workload
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Param data body object true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/{resource}/{name} [put]
// @Security JWT
func _() {}

// @Tags Agent.V1
// @Summary 创建namespaced scope workload
// @Description 创建namespaced scope workload
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Param namespace path string true "namespace"
// @Param data body object true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [put]
// @Security JWT
func (h *REST) Update(c *gin.Context) {
	obj, _, err := h.readObject(c, true)
	if err != nil {
		NotOK(c, err)
		return
	}
	if err := h.client.Update(c.Request.Context(), obj); err != nil {
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

// @Tags Agent.V1
// @Summary  创建none namespaced scope workload
// @Description 创建none namespaced scope workload
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/{resource}/{name} [delete]
// @Security JWT
func _() {}

// @Tags Agent.V1
// @Summary 创建namespaced scope workload
// @Description 创建namespaced scope workload
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Param namespace path string true "namespace"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [delete]
// @Security JWT
func (h *REST) Delete(c *gin.Context) {
	obj, _, err := h.readObject(c, false)
	if err != nil {
		NotOK(c, err)
		return
	}
	if err := h.client.Delete(c.Request.Context(), obj); err != nil {
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

// @Tags Agent.V1
// @Summary  创建none namespaced scope workload
// @Description 创建none namespaced scope workload
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Param data body object true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/{resource}/{name} [patch]
// @Security JWT
func _() {}

// @Tags Agent.V1
// @Summary 创建namespaced scope workload
// @Description 创建namespaced scope workload
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Param namespace path string true "namespace"
// @Param data body object true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [patch]
// @Security JWT
func (h *REST) Patch(c *gin.Context) {
	obj, _, err := h.readObject(c, false)
	if err != nil {
		NotOK(c, err)
		return
	}

	var patch client.Patch

	switch patchtype := types.PatchType(c.Request.Header.Get("Content-Type")); patchtype {
	// 依旧支持使用原生的patch类型
	case types.MergePatchType, types.ApplyPatchType, types.JSONPatchType, types.StrategicMergePatchType:
		patchdata, _ := io.ReadAll(c.Request.Body)
		defer c.Request.Body.Close()

		patch = client.RawPatch(patchtype, patchdata)
	default:
		// TODO: move to patch type : types.JSONPatchType
		// 默认是获取整个对象进行patch
		exist, ok := obj.DeepCopyObject().(client.Object)
		if !ok {
			NotOK(c, fmt.Errorf("%T is not a client.Object", obj))
			return
		}
		// read obj
		if err := json.NewDecoder(c.Request.Body).Decode(obj); err != nil {
			NotOK(c, err)
			return
		}
		// get exists
		if err = h.client.Get(c.Request.Context(), client.ObjectKeyFromObject(obj), exist); err != nil {
			NotOK(c, err)
			return
		}
		// 所有类型全部都使用 json merge，要求client端传完整的对象数据
		// 因不使用 strategic patch 不需要具体类型，可以使用 unstructured
		patch = &kube.JsonPatchType{From: exist}
	}

	if err := h.client.Patch(c.Request.Context(), obj, patch); err != nil {
		NotOK(c, err)
		return
	} else {
		OK(c, obj)
	}
}

type scaleForm struct {
	Replicas int32 `json:"replicas"`
}

// @Tags Agent.V1
// @Summary  nonamespace 扩缩容
// @Description 扩缩容
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Param data body object true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/{resource}/{name}/actions/scale [patch]
// @Security JWT
func _() {}

// @Tags Agent.V1
// @Summary  扩缩容
// @Description 扩缩容
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param group path string true "group"
// @Param version path string true "version"
// @Param resource path string true "resoruce"
// @Param name path string true "name"
// @Param namespace path string true "namespace"
// @Param data body scaleForm true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "counter"
// @Router /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name}/actions/scale [patch]
// @Security JWT
func (h *REST) Scale(c *gin.Context) {
	gvkn, err := h.parseGVKN(c)
	if err != nil {
		NotOK(c, err)
		return
	}
	formdata := scaleForm{}
	if e := c.BindJSON(&formdata); e != nil {
		NotOK(c, e)
		return
	}

	patch := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"replicas": formdata.Replicas,
			},
		},
	}
	patch.SetGroupVersionKind(gvkn.GroupVersionKind)
	patch.SetName(gvkn.Name)
	patch.SetNamespace(gvkn.Namespace)

	if err := h.client.Patch(c, patch, client.Merge); err != nil {
		NotOK(c, err)
	} else {
		OK(c, patch)
	}
}

func (r *REST) parseGVKN(c *gin.Context) (GVKN, error) {
	group := c.Param("group")
	if group == "core" {
		group = ""
	}
	namespace := c.Param("namespace")
	if namespace == "_all" || namespace == "_" {
		namespace = ""
	}
	gvkn := GVKN{
		GroupVersionKind: schema.GroupVersionKind{
			Group:   group,
			Version: c.Param("version"),
		},
		Namespace:     namespace,
		Name:          c.Param("name"),
		Resource:      c.Param("resource"),
		Labels:        c.QueryMap("labels"),
		LabelSelector: c.Query("labelSelector"),
	}
	gvk, err := r.client.RESTMapper().KindFor(gvkn.GroupVersion().WithResource(gvkn.Resource))
	if err != nil {
		return GVKN{}, err
	}
	gvkn.GroupVersionKind = gvk
	return gvkn, nil
}

// parse map to labelselector
// eg:
//     label1__in: a,b,c
//     label2__notin: a,b,c
//     label3__exist: any
//     label4__notexist: any
func parseSels(mapSelector map[string]string, selector string) labels.Selector {
	var sel labels.Selector
	if len(selector) > 0 {
		sel, err := labels.Parse(selector)
		if err == nil {
			return sel
		}
	}
	sel = labels.NewSelector()
	for k, v := range mapSelector {
		// nolint: nestif
		if !strings.Contains(k, "__") {
			if req, err := labels.NewRequirement(k, selection.Equals, []string{v}); err == nil {
				sel = sel.Add(*req)
			}
		} else {
			seps := strings.Split(k, "__")
			lenth := len(seps)
			key := strings.Join(seps[:lenth-1], "__")
			op := seps[lenth-1]
			switch op {
			case "exist":
				if req, err := labels.NewRequirement(key, selection.Exists, []string{}); err == nil {
					sel = sel.Add(*req)
				}
			case "neq":
				if req, err := labels.NewRequirement(key, selection.NotEquals, []string{v}); err == nil {
					sel = sel.Add(*req)
				}
			case "notexist":
				if req, err := labels.NewRequirement(key, selection.DoesNotExist, []string{}); err == nil {
					sel = sel.Add(*req)
				}
			case "in":
				if req, err := labels.NewRequirement(key, selection.In, strings.Split(v, ",")); err == nil {
					sel = sel.Add(*req)
				}
			case "notin":
				if req, err := labels.NewRequirement(key, selection.NotIn, strings.Split(v, ",")); err == nil {
					sel = sel.Add(*req)
				}
			}
		}
	}
	return sel
}

func (r *REST) readObject(c *gin.Context, readbody bool) (client.Object, GVKN, error) {
	gvkn, err := r.parseGVKN(c)
	if err != nil {
		return nil, GVKN{}, err
	}
	// try decode using typed ObjectList first
	runobj, err := r.client.Scheme().New(gvkn.GroupVersionKind)
	if err != nil {
		// fallback to unstructured.Unstructured
		runobj = &unstructured.Unstructured{}
	}
	obj, ok := runobj.(client.Object)
	if !ok {
		// fallback to unstructured.Unstructured
		obj = &unstructured.Unstructured{}
	}
	if readbody {
		if err := json.NewDecoder(c.Request.Body).Decode(&obj); err != nil {
			return nil, gvkn, apierrors.NewBadRequest(err.Error())
		}
		defer c.Request.Body.Close()
	}
	// override name/namespace in body if set in url
	if objns := obj.GetNamespace(); objns != "" && objns != gvkn.Namespace {
		return obj, gvkn,
			apierrors.NewBadRequest(
				fmt.Sprintf("namespace in path %s is different with in body %s", gvkn.Namespace, objns),
			)
	}
	if gvkn.Name != "" {
		obj.SetName(gvkn.Name)
	}
	obj.GetObjectKind().SetGroupVersionKind(gvkn.GroupVersionKind)
	obj.SetNamespace(gvkn.Namespace)
	return obj, gvkn, nil
}

func (r *REST) listObject(c *gin.Context) (client.ObjectList, GVKN, error) {
	gvkn, err := r.parseGVKN(c)
	if err != nil {
		return nil, GVKN{}, err
	}
	if !strings.HasSuffix(gvkn.Kind, "List") {
		gvkn.GroupVersionKind.Kind = gvkn.GroupVersionKind.Kind + "List"
	}
	// try decode using typed ObjectList first
	runlist, err := r.client.Scheme().New(gvkn.GroupVersionKind)
	if err != nil {
		// fallback to unstructured.UnstructuredList
		runlist = &unstructured.UnstructuredList{}
	}
	objlist, ok := runlist.(client.ObjectList)
	if !ok {
		// fallback to unstructured.UnstructuredList
		objlist = &unstructured.UnstructuredList{}
	}
	objlist.GetObjectKind().SetGroupVersionKind(gvkn.GroupVersionKind)
	return objlist, gvkn, nil
}
