package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/utils/proxy"
	"github.com/kubegems/gems/pkg/utils/route"
	"github.com/kubegems/gems/pkg/utils/stream"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// https://github.com/kubernetes/apiserver/blob/master/pkg/server/config.go#L362
const (
	MaxRequestBodyBytes = 3 * 1024 * 1024
	RoutePrefix         = "/internal"
)

type ClientRest struct {
	Cli client.Client
}

func (h *ClientRest) Register(r *route.Router) {
	r.GET("/internal/{group}/{version}/{kind}/{name}", h.Get)
	r.GET("/internal/{group}/{version}/namespaces/{namespace}/{kind}/{name}", h.Get)

	r.GET("/internal/{group}/{version}/{kind}", h.List)
	r.GET("/internal/{group}/{version}/namespaces/{namespace}/{kind}", h.List)

	r.POST("/internal/{group}/{version}/{kind}", h.Create)
	r.POST("/internal/{group}/{version}/namespaces/{namespace}/{kind}", h.Create)

	r.PUT("/internal/{group}/{version}/{kind}/{name}", h.Update)
	r.PUT("/internal/{group}/{version}/namespaces/{namespace}/{kind}/{name}", h.Update)

	r.PATCH("/internal/{group}/{version}/{kind}/{name}", h.Patch)
	r.PATCH("/internal/{group}/{version}/namespaces/{namespace}/{kind}/{name}", h.Patch)

	r.DELETE("/internal/{group}/{version}/{kind}/{name}", h.Delete)
	r.DELETE("/internal/{group}/{version}/namespaces/{namespace}/{kind}/{name}", h.Delete)

	r.GET("/internal/{group}/{version}/namespaces/{namespace}/{kind}/{name}/portforward", h.PortForward)

	r.GET("/internal/core/v1/namespaces/{namespace}/{kind}/{name}:{port}/proxy/{proxypath}*", h.Proxy)
	r.GET("/internal/core/v1/namespaces/{namespace}/{kind}/{name}:{port}/proxy/", h.Proxy)
}

func (h *ClientRest) Get(c *gin.Context) {
	gvkn := h.parseGVKN(c)

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvkn.GroupVersionKind)
	ctx := c.Request.Context()
	if err := h.Cli.Get(ctx, client.ObjectKey{Namespace: gvkn.Namespace, Name: gvkn.Name}, obj); err != nil {
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

const (
	ListOptionLabelSelector = "label-selector"
	ListOptionFieldSelector = "field-selector"
	ListOptionLimit         = "limit"
	ListOptionContinue      = "continue"
)

func (h *ClientRest) List(c *gin.Context) {
	gvkn := h.parseGVKN(c)

	labelSelectorStr := c.Query(ListOptionLabelSelector)
	labelSelector, _ := labels.Parse(labelSelectorStr)

	fieldSelectorStr := c.Query(ListOptionFieldSelector)
	fieldSelector, _ := fields.ParseSelector(fieldSelectorStr)

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvkn.GroupVersionKind)

	limit, _ := strconv.Atoi(c.Query(ListOptionLimit))

	listOptions := &client.ListOptions{
		Namespace:     gvkn.Namespace,
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
		Limit:         int64(limit),
		Continue:      c.Query(ListOptionContinue),
	}

	if iswatch, _ := strconv.ParseBool(c.Query("watch")); iswatch {
		if watchablecli, ok := h.Cli.(client.WithWatch); ok {
			watcher, err := watchablecli.Watch(c.Request.Context(), list, listOptions)
			if err != nil {
				NotOK(c, err)
				return
			}

			defer watcher.Stop()

			pusher, err := stream.StartPusher(c.Writer)
			if err != nil {
				NotOK(c, err)
				return
			}
			// send stream
			for {
				select {
				case e, ok := <-watcher.ResultChan():
					if !ok {
						return
					}
					if err := pusher.Push(e); err != nil {
						return
					}
				case <-c.Request.Context().Done():
					return
				}
			}
		} else {
			NotOK(c, apierrors.NewServiceUnavailable("client not watchable"))
		}
	} else {
		if err := h.Cli.List(c.Request.Context(), list, listOptions); err != nil {
			NotOK(c, err)
			return
		}
		OK(c, list)
		return
	}
}

func NotOK(c *gin.Context, err error) {
	// 增加针对 apierrors 状态码适配
	statuserr := &apierrors.StatusError{}
	if !errors.As(err, &statuserr) {
		statuserr = apierrors.NewBadRequest(err.Error())
	}
	c.AbortWithStatusJSON(int(statuserr.Status().Code), statuserr.ErrStatus)
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

func (h *ClientRest) Create(c *gin.Context) {
	gvkn := h.parseGVKN(c)

	content := map[string]interface{}{}
	if err := json.NewDecoder(c.Request.Body).Decode(&content); err != nil {
		NotOK(c, err)
		return
	}
	defer c.Request.Body.Close()

	options := &client.CreateOptions{}

	obj := &unstructured.Unstructured{Object: content}
	obj.SetGroupVersionKind(gvkn.GroupVersionKind)
	obj.SetNamespace(gvkn.Namespace)

	if err := h.Cli.Create(c.Request.Context(), obj, options); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, obj)
}

func (h *ClientRest) Update(c *gin.Context) {
	gvkn := h.parseGVKN(c)

	content := map[string]interface{}{}
	if err := json.NewDecoder(c.Request.Body).Decode(&content); err != nil {
		NotOK(c, err)
		return
	}
	defer c.Request.Body.Close()

	issubresource, _ := strconv.ParseBool(c.Query("subresource"))

	options := &client.UpdateOptions{}

	obj := &unstructured.Unstructured{Object: content}
	obj.SetGroupVersionKind(gvkn.GroupVersionKind)
	obj.SetNamespace(gvkn.Namespace)
	obj.SetName(gvkn.Name)

	if issubresource {
		if err := h.Cli.Status().Update(c.Request.Context(), obj, options); err != nil {
			NotOK(c, err)
			return
		}
	} else {
		if err := h.Cli.Update(c.Request.Context(), obj, options); err != nil {
			NotOK(c, err)
			return
		}
	}
	OK(c, obj)
}

const (
	DeleteOptionDeletionPropagation = "deletion-propagation"
	DeleteOptionGracePeriod         = "grace-period-seconds"
)

func (h *ClientRest) Delete(c *gin.Context) {
	gvkn := h.parseGVKN(c)

	options := &client.DeleteOptions{
		PropagationPolicy: func() *metav1.DeletionPropagation {
			if policy := metav1.DeletionPropagation(c.Query(DeleteOptionDeletionPropagation)); policy != "" {
				return &policy
			}
			return nil
		}(),
		GracePeriodSeconds: func() *int64 {
			if seconds := c.Query(DeleteOptionGracePeriod); seconds != "" {
				sec, _ := strconv.Atoi(seconds)
				return pointer.Int64(int64(sec))
			}
			return nil
		}(),
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvkn.GroupVersionKind)
	obj.SetNamespace(gvkn.Namespace)
	obj.SetName(gvkn.Name)

	if err := h.Cli.Delete(c.Request.Context(), obj, options); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, obj)
}

const (
	PatchOptionForce = "force"
)

func (h *ClientRest) Patch(c *gin.Context) {
	gvkn := h.parseGVKN(c)

	var obj client.Object
	var patch client.Patch

	patchtype := types.PatchType(c.Request.Header.Get("Content-Type"))

	issubresource, _ := strconv.ParseBool(c.Query("subresource"))

	data, err := io.ReadAll(&io.LimitedReader{R: c.Request.Body, N: MaxRequestBodyBytes})
	if err != nil {
		NotOK(c, err)
		return
	}

	options := &client.PatchOptions{
		Force: func() *bool {
			if b := c.Query(PatchOptionForce); b != "" {
				bl, _ := strconv.ParseBool(b)
				return pointer.Bool(bl)
			}
			return nil
		}(),
	}

	obj = &unstructured.Unstructured{}
	obj.GetObjectKind().SetGroupVersionKind(gvkn.GroupVersionKind)
	obj.SetName(gvkn.Name)
	obj.SetNamespace(gvkn.Namespace)

	patch = client.RawPatch(patchtype, data)
	if issubresource {
		if err := h.Cli.Status().Patch(c.Request.Context(), obj, patch, options); err != nil {
			NotOK(c, err)
			return
		}
	} else {
		if err := h.Cli.Patch(c.Request.Context(), obj, patch, options); err != nil {
			NotOK(c, err)
			return
		}
	}

	OK(c, obj)
}

func (h *ClientRest) PortForward(c *gin.Context) {
	gvkn := h.parseGVKN(c)

	// must core v1
	if gvkn.Group != "" || gvkn.Version != "v1" {
		NotOK(c, fmt.Errorf("unsupported group: %s", gvkn.GroupVersionKind.GroupVersion()))
		return
	}

	port, err := strconv.Atoi(c.Query("port"))
	if err != nil {
		NotOK(c, err)
		return
	}

	ctx := c.Request.Context()

	process := func() error {
		var target string
		switch gvkn.Kind {
		case "Pod":
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: gvkn.Name, Namespace: gvkn.Namespace}}
			if err := h.Cli.Get(ctx, client.ObjectKeyFromObject(pod), pod); err != nil {
				return err
			}
			if pod.Status.Phase != corev1.PodRunning {
				return fmt.Errorf("pod %s is not running", pod.Name)
			}
			// pod: {pod-ip}.
			target = fmt.Sprintf("%s:%d", pod.Status.PodIP, port)
		case "Service", "":
			// see: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
			// svc: {svcname}.{namespace}.svc
			target = fmt.Sprintf("%s.%s.svc:%d", gvkn.Name, gvkn.Namespace, port)
		}

		// dial tcp
		tcpproxy, err := proxy.NewTCPProxy(target, -1)
		if err != nil {
			return err
		}

		source, _, err := c.Writer.Hijack()
		if err != nil {
			return fmt.Errorf("unable hijack http connection: %v", err)
		}

		if err := tcpproxy.ServeConn(source); err != nil {
			log.Errorf("copy connection error: %v", err)
			// already hijacked,return nil avoid http response
			return nil
		}
		return nil
	}
	if err := process(); err != nil {
		NotOK(c, err)
		return
	}
	// do nothing
}

type GVKN struct {
	schema.GroupVersionKind
	Namespace string
	Resource  string
	Name      string
}

func (r *ClientRest) parseGVKN(c *gin.Context) GVKN {
	gvkn := GVKN{
		GroupVersionKind: schema.GroupVersionKind{
			Group: func() string {
				if group := c.Param("group"); group != "core" {
					return group
				}
				return ""
			}(),
			Version: c.Param("version"),
			Kind:    c.Param("kind"),
		},
		Namespace: c.Param("namespace"),
		Name:      c.Param("name"),
	}
	return gvkn
}

func (h *ClientRest) Proxy(c *gin.Context) {
	gvkn := h.parseGVKN(c)

	port, err := strconv.Atoi(c.Param("port"))
	if err != nil {
		NotOK(c, err)
		return
	}
	proxypath := "/" + c.Param("proxypath")
	ctx := c.Request.Context()

	process := func() error {
		var host string
		switch strings.ToLower(gvkn.Kind) {
		case "pod":
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: gvkn.Name, Namespace: gvkn.Namespace}}
			if err := h.Cli.Get(ctx, client.ObjectKeyFromObject(pod), pod); err != nil {
				return err
			}
			if pod.Status.Phase != corev1.PodRunning {
				return fmt.Errorf("pod %s is not running", pod.Name)
			}
			// pod: {pod-ip}.
			host = fmt.Sprintf("%s:%d", pod.Status.PodIP, port)
		case "service", "":
			// see: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
			// svc: {svcname}.{namespace}.svc
			host = fmt.Sprintf("%s.%s.svc:%d", gvkn.Name, gvkn.Namespace, port)
		}

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = host
				req.URL.Path = proxypath
				req.URL.RawPath = proxypath
				req.Host = host
			},
		}

		proxy.ServeHTTP(c.Writer, c.Request)
		return nil
	}
	if err := process(); err != nil {
		NotOK(c, err)
		return
	}
}
