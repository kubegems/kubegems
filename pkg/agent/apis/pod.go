package apis

import (
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/agent/ws"
	gemlabels "kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodHandler struct {
	cluster      cluster.Interface
	debugoptions *DebugOptions
}

// @Tags Agent.V1
// @Summary 获取Pod列表数据
// @Description 获取Pod列表数据
// @Accept json
// @Produce json
// @Param order query string false "page"
// @Param search query string false "search"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param namespace path string true "namespace"
// @Param fieldSelector query string false "fieldSelector, 只支持podstatus={xxx}格式"
// @Param cluster path string true "cluster"
// @Param topkind query string false "topkind(Deployment,StatefulSet,DaemonSet,Job,Node)"
// @Param topname query string false "topname"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]object}} "Pod"
// @Router /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pods [get]
// @Security JWT
func (h *PodHandler) List(c *gin.Context) {
	ns := c.Param("namespace")
	// 网关namespace必须是gemcloud-gateway-system
	if c.Query("topkind") == "TenantGateway" {
		ns = gemlabels.NamespaceGateway
	}
	if ns == "_all" || ns == "_" {
		ns = ""
	}

	selLabels, err := h.getControllerLabel(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	labelsMap := c.QueryMap("labels")
	for k, v := range labelsMap {
		selLabels[k] = v
	}
	sel := labels.SelectorFromSet(selLabels)

	podList := &v1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingLabelsSelector{Selector: sel},
	}
	fieldSelector, fexist := getFieldSelector(c)

	if err = h.cluster.GetClient().List(c.Request.Context(), podList, listOpts...); err != nil {
		NotOK(c, err)
		return
	}

	objects := podList.Items

	// filter pod status
	// see: issues #122
	if fexist {
		if val, ok := fieldSelector.RequiresExactMatch("phase"); ok {
			objects = filterByContainerState(val, objects)
		}
	}

	objects = filterByNodename(c, objects)
	pageData := NewPageDataFromContext(c, func(i int) SortAndSearchAble {
		return &objects[i]
	}, len(objects), objects)

	if iswatch, _ := strconv.ParseBool(c.Query("watch")); iswatch {
		// list
		c.SSEvent("data", pageData)
		c.Writer.Flush()
		// watch
		WatchEvents(c, h.cluster, podList, listOpts...)
	} else {
		OK(c, pageData)
	}
}

func (h *PodHandler) getControllerLabel(c *gin.Context) (map[string]string, error) {
	ns := c.Params.ByName("namespace")
	namespace := ns
	if ns == allNamespace {
		namespace = v1.NamespaceAll
	}
	ret := map[string]string{}
	topkind := c.Query("topkind")
	topname := c.Query("topname")
	if len(topkind) == 0 || len(topname) == 0 {
		return ret, nil
	}
	switch topkind {
	case "Deployment":
		dep := &appsv1.Deployment{}
		err := h.cluster.GetClient().Get(c.Request.Context(), types.NamespacedName{
			Namespace: namespace, Name: topname,
		}, dep)
		if err != nil {
			return nil, err
		}
		return dep.Spec.Selector.MatchLabels, nil
	case "StatefulSet":
		sts := &appsv1.StatefulSet{}
		err := h.cluster.GetClient().Get(c.Request.Context(), types.NamespacedName{
			Namespace: namespace, Name: topname,
		}, sts)
		if err != nil {
			return nil, err
		}
		return sts.Spec.Selector.MatchLabels, nil
	case "Job":
		job := &batchv1.Job{}
		err := h.cluster.GetClient().Get(c.Request.Context(), types.NamespacedName{
			Namespace: namespace, Name: topname,
		}, job)
		if err != nil {
			return nil, err
		}
		return job.Spec.Selector.MatchLabels, nil
	case "DaemonSet":
		ds := &appsv1.DaemonSet{}
		err := h.cluster.GetClient().Get(c.Request.Context(), types.NamespacedName{
			Namespace: namespace, Name: topname,
		}, ds)
		if err != nil {
			return nil, err
		}
		return ds.Spec.Selector.MatchLabels, nil
	case "TenantGateway":
		return map[string]string{"app": topname}, nil
	}
	return ret, nil
}

func filterByNodename(c *gin.Context, pods []v1.Pod) []v1.Pod {
	topkind := c.Query("topkind")
	topname := c.Query("topname")
	if topkind != "Node" || len(topname) == 0 {
		return pods
	}
	var ret []v1.Pod
	for _, pod := range pods {
		if pod.Spec.NodeName == topname {
			ret = append(ret, pod)
		}
	}
	return ret
}

func filterByContainerState(phase string, pods []v1.Pod) []v1.Pod {
	var ret []v1.Pod
	for _, pod := range pods {
		for _, container := range pod.Status.ContainerStatuses {
			switch {
			case phase == "Running" && container.State.Running != nil:
				ret = append(ret, pod)
			case phase == "NotRunning" && container.State.Waiting != nil:
				ret = append(ret, pod)
			}
		}
	}
	return ret
}

// ExecContainer 进入容器交互执行命令
// @Tags Agent.V1
// @Summary 进入容器交互执行命令(websocket)
// @Description 进入容器交互执行命令(websocket)
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param pod path string true "pod"
// @Param container query string true "container"
// @Param stream query string true "stream must be true"
// @Param token query string true "token"
// @Param shell query string false "default sh, choice(bash,ash,zsh)"
// @Success 200 {object} object "ws"
// @Router /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pods/{name}/actions/shell [get]
// @Security JWT
func (h *PodHandler) ExecPods(c *gin.Context) {
	conn, err := ws.InitWebsocket(c.Writer, c.Request)
	if err != nil {
		_ = conn.WsWrite(websocket.TextMessage, []byte("init websocket connection error"))
		return
	}
	handler := &ws.StreamHandler{WsConn: conn, ResizeEvent: make(chan remotecommand.TerminalSize)}
	exec, err := h.getExec(c)
	if err != nil {
		log.Infof("Upgrade Websocket Faield: %s", err.Error())
		handlers.NotOK(c, err)
		return
	}
	if err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             handler,
		Stdout:            handler,
		Stderr:            handler,
		TerminalSizeQueue: handler,
		Tty:               true,
	}); err != nil {
		_ = conn.WsWrite(websocket.TextMessage, []byte(err.Error()))
		return
	}
}

// GetContainerLogs 获取容器的stdout输出
// @Tags Agent.V1
// @Summary 实时获取日志STDOUT输出(websocket)
// @Description 实时获取日志STDOUT输出(websocket)
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param namespace path string true "namespace"
// @Param pod path string true "pod"
// @Param container query string true "container"
// @Param stream query string true "stream must be true"
// @Param follow query string true "follow"
// @Param tail query int false "tail line (default 1000)"
// @Success 200 {object} object "ws"
// @Router /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pods/{name}/actions/logs [get]
// @Security JWT
func (h *PodHandler) GetContainerLogs(c *gin.Context) {
	namespace := c.Param("namespace")
	pod := c.Param("name")
	container := getDefaultHeader(c, "container", "")

	ws, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Infof("Upgrade Websocket Faield: %s", err.Error())
		handlers.NotOK(c, err)
		return
	}

	tailInt, _ := strconv.Atoi(getDefaultHeader(c, "tail", "1000"))
	follow := getDefaultHeader(c, "follow", "true")
	tail := int64(tailInt)
	logopt := &v1.PodLogOptions{
		Container: container,
		Follow:    follow == "true",
		TailLines: &tail,
	}
	req := h.cluster.Kubernetes().CoreV1().Pods(namespace).GetLogs(pod, logopt)
	out, err := req.Stream(c.Request.Context())
	if err != nil {
		_ = ws.WriteMessage(websocket.TextMessage, []byte("init websocket stream error"))
		return
	}
	defer out.Close()
	writer := wsWriter{
		conn: ws,
	}
	_, _ = io.Copy(&writer, out)
}

type wsWriter struct {
	conn *websocket.Conn
}

func (w *wsWriter) Write(data []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (h *PodHandler) getExec(c *gin.Context) (remotecommand.Executor, error) {
	namespace := c.Param("namespace")
	pod := c.Param("name")
	container := getDefaultHeader(c, "container", "")
	command := []string{
		"/bin/sh",
		"-c",
		"export LINES=20; export COLUMNS=100; export LANG=C.UTF-8; export TERM=xterm-256color; [ -x /bin/bash ] && exec /bin/bash || exec /bin/sh",
	}

	log.Infof("exec pod in contaler %v, raw URL %v", container, c.Request.URL)
	req := h.cluster.Kubernetes().CoreV1().RESTClient().Post().Resource("pods").Namespace(namespace).Name(pod).SubResource("exec").VersionedParams(&v1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)
	return remotecommand.NewSPDYExecutor(h.cluster.Config(), "POST", req.URL())
}
