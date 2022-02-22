package apis

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/agent/ws"
	"kubegems.io/pkg/service/handlers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DebugAgentNamespace = "debug-tools"
	DebugAgentImage     = "kubegems/debug-agent:latest"
	DebugToolsImage     = "kubegems/debug-tools:latest"
)

// ExecContainer 调试容器(websocket)
// @Tags Agent.V1
// @Summary 调试容器(websocket)
// @Description 调试容器(websocket)
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "pod name"
// @Param container query string true "container"
// @Param stream query string true "must be true"
// @Param agentiamge query string false "agentimage"
// @Param debugimage query string false "debugimage"
// @Param forkmode query string false "forkmode"
// @Success 200 {object} object "ws"
// @Router /v1/proxy/cluster/{cluster}/custom/core/v1/{namespace}/pods/{name}/actions/debug [get]
// @Security JWT
func (h *PodHandler) DebugPod(c *gin.Context) {
	conn, err := ws.InitWebsocket(c.Writer, c.Request)
	if err != nil {
		_ = conn.WsWrite(websocket.TextMessage, []byte("init websocket connection error"))
		conn.WsClose()
		return
	}
	handler := &ws.StreamHandler{WsConn: conn, ResizeEvent: make(chan remotecommand.TerminalSize)}
	exec, err := h.getDebug(c)
	if err != nil {
		log.Printf("Upgrade Websocket Faield: %s", err.Error())
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
		_ = conn.WsWrite(websocket.TextMessage, []byte("init websocket stream error "+err.Error()))
		<-time.AfterFunc(time.Duration(3)*time.Second, func() {
			conn.WsClose()
		}).C
		return
	}
}

func (h *PodHandler) getDebug(c *gin.Context) (remotecommand.Executor, error) {
	namespace := c.Param("namespace")
	pod := c.Param("name")

	debugImage := getDefaultHeader(c, "debugimage", h.debugoptions.Image)
	command := []string{
		"kubectl",
		"-n",
		namespace,
		"debug",
		pod,
		"--image",
		debugImage,
		"--image-pull-policy=IfNotPresent",
		"-it",
		"--",
		"/start.sh",
	}
	poname, err := getKubectlContainer(c.Request.Context(), h.cluster.GetClient(), h.debugoptions)
	if err != nil {
		return nil, err
	}
	req := h.cluster.Kubernetes().
		CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(h.debugoptions.Namespace).
		Name(poname).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: h.debugoptions.Container,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)
	ex, err := remotecommand.NewSPDYExecutor(h.cluster.Config(), "POST", req.URL())
	return ex, err
}

type KubectlHandler struct {
	cluster      cluster.Interface
	debugoptions *DebugOptions
}

// ExecKubectl kubectl
// @Tags Agent.V1
// @Summary kubectl
// @Description kubectl
// @Param cluster path string true "cluster"
// @Param stream query string  true "stream must be true"
// @Success 200 {object} object "ws"
// @Router /v1/proxy/cluster/{cluster}/custom/system/v1/kubectl [get]
// @Security JWT
func (h *KubectlHandler) ExecKubectl(c *gin.Context) {
	conn, err := ws.InitWebsocket(c.Writer, c.Request)
	if err != nil {
		_ = conn.WsWrite(websocket.TextMessage, []byte("init websocket connection error"))
		conn.WsClose()
		return
	}
	handler := &ws.StreamHandler{WsConn: conn, ResizeEvent: make(chan remotecommand.TerminalSize)}
	exec, err := h.getKubectl(c)
	if err != nil {
		log.Printf("Upgrade Websocket Faield: %s", err.Error())
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
		_ = conn.WsWrite(websocket.TextMessage, []byte("init websocket stream error "+err.Error()))
		<-time.AfterFunc(time.Duration(3)*time.Second, func() {
			conn.WsClose()
		}).C
		return
	}
}

func (h *KubectlHandler) getKubectl(c *gin.Context) (remotecommand.Executor, error) {
	command := []string{
		"/bin/sh",
		"-c",
		"export LINES=20; export COLUMNS=100; TERM=xterm-256color; export TERM; [ -x /bin/bash ] && ([ -x /usr/bin/script ] && /usr/bin/script -q -c /bin/bash /dev/null || exec /bin/bash) || exec /bin/sh",
	}
	poname, err := getKubectlContainer(c.Request.Context(), h.cluster.GetClient(), h.debugoptions)
	if err != nil {
		return nil, err
	}
	req := h.cluster.Kubernetes().
		CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(h.debugoptions.Namespace).
		Name(poname).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			// 若不设置使用默认container
			Command: command,
			Stdin:   true,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)
	return remotecommand.NewSPDYExecutor(h.cluster.Config(), "POST", req.URL())
}

func getKubectlContainer(ctx context.Context, ctl client.Client, debug *DebugOptions) (string, error) {
	namespace := debug.Namespace

	polist := &v1.PodList{}
	sel, err := labels.Parse(debug.PodSelector)
	if err != nil {
		return "", err
	}

	if err := ctl.List(ctx, polist, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
		return "", fmt.Errorf("failed to get kubectl container %v", err)
	}
	if len(polist.Items) == 0 {
		return "", fmt.Errorf("failed to get kubectl container")
	}
	var poname string

	randI := rand.New(rand.NewSource(time.Now().Unix()))
	randI.Shuffle(len(polist.Items), func(i, j int) { polist.Items[i], polist.Items[j] = polist.Items[j], polist.Items[i] })
	for _, po := range polist.Items {
		if po.Status.Phase == v1.PodRunning {
			poname = po.GetName()
			break
		}
	}
	if len(poname) == 0 {
		return poname, fmt.Errorf("can't find kubectl container")
	}
	return poname, nil
}
