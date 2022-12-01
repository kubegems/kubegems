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
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/agent/ws"
	"kubegems.io/kubegems/pkg/apis/gems"

	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/webtty"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubectlOptions struct {
	DebugImage      string `json:"image,omitempty"`
	Namespace       string `json:"namespace,omitempty"`
	PodSelector     string `json:"podSelector,omitempty"`
	UseLocalKubectl bool   `json:"useLocalKubectl,omitempty"`
}

func NewDefaultKubectlOptions() *KubectlOptions {
	return &KubectlOptions{
		Namespace: kube.LocalNamespaceOrDefault(gems.NamespaceLocal),
		PodSelector: labels.SelectorFromSet(
			labels.Set{
				"app.kubernetes.io/name": "kubegems-agent-kubectl",
			}).String(),
		DebugImage:      "registry.cn-beijing.aliyuncs.com/kubegems/debug-tools:latest",
		UseLocalKubectl: false,
	}
}

var DefaultExecCommand = []string{
	"/bin/sh",
	"-c",
	"export LINES=20; export COLUMNS=100; TERM=xterm-256color; export TERM; [ -x /bin/bash ] && ([ -x /usr/bin/script ] && /usr/bin/script -q -c /bin/bash /dev/null || exec /bin/bash) || exec /bin/sh",
}

type KubectlHandler struct {
	cluster cluster.Interface
	options *KubectlOptions
}

// ExecContainer 调试容器(websocket)
// @Tags        Agent.V1
// @Summary     调试容器(websocket)
// @Description 调试容器(websocket)
// @Param       cluster    path     string true  "cluster"
// @Param       namespace  path     string true  "namespace"
// @Param       name       path     string true  "pod name"
// @Param       container  query    string true  "container"
// @Param       stream     query    string true  "must be true"
// @Param       agentiamge query    string false "agentimage"
// @Param       debugimage query    string false "debugimage"
// @Param       forkmode   query    string false "forkmode"
// @Success     200        {object} object "ws"
// @Router      /v1/proxy/cluster/{cluster}/custom/core/v1/{namespace}/pods/{name}/actions/debug [get]
// @Security    JWT
func (h *KubectlHandler) DebugPod(c *gin.Context) {
	cmd := []string{
		"kubectl",
		"-n",
		c.Param("namespace"),
		"debug",
		c.Param("name"),
		"--image",
		paramFromHeaderOrQuery(c, "debugimage", h.options.DebugImage),
		"--image-pull-policy=IfNotPresent",
		"-it",
		"--",
		"/start.sh",
	}
	// replace with https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod/#ephemeral-container
	ExecuteAtLocalOrRemoteKubectlPodWithWebsocket(c.Writer, c.Request, cmd, h.cluster, h.options)
}

// ExecKubectl kubectl
// @Tags        Agent.V1
// @Summary     kubectl
// @Description kubectl
// @Param       cluster path     string true "cluster"
// @Param       stream  query    string true "stream must be true"
// @Success     200     {object} object "ws"
// @Router      /v1/proxy/cluster/{cluster}/custom/system/v1/kubectl [get]
// @Security    JWT
func (h *KubectlHandler) ExecKubectl(c *gin.Context) {
	ExecuteAtLocalOrRemoteKubectlPodWithWebsocket(c.Writer, c.Request, DefaultExecCommand, h.cluster, h.options)
}

func ExecuteAtLocalOrRemoteKubectlPodWithWebsocket(
	resp http.ResponseWriter, req *http.Request,
	commands []string,
	cluster cluster.Interface,
	options *KubectlOptions,
) {
	RunWebSocketStream(resp, req, func(ctx context.Context, stream remotecommand.StreamOptions) error {
		if options.UseLocalKubectl {
			return webtty.Exec(req.Context(), commands[0], commands[1:], stream)
		}
		selector, err := labels.Parse(options.PodSelector)
		if err != nil {
			return err
		}
		// execute command via exec subresource in kubectl pod
		poname, err := selectPod(ctx, cluster.GetClient(), options.Namespace, selector)
		if err != nil {
			return err
		}
		pe := PodCmdExecutor{
			Cluster: cluster,
			Pod: client.ObjectKey{
				Namespace: options.Namespace,
				Name:      poname,
			},
			PodExecOptions: v1.PodExecOptions{
				Stdin:   true,
				Stdout:  true,
				Stderr:  true,
				TTY:     true,
				Command: commands,
			},
			StreamOptions: stream,
		}
		return pe.Execute(ctx)
	})
}

func selectPod(ctx context.Context, cli client.Client, namespace string, selector labels.Selector) (string, error) {
	polist := &v1.PodList{}
	if err := cli.List(ctx, polist, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return "", fmt.Errorf("list pods: %w", err)
	}
	available := []string{}
	for _, po := range polist.Items {
		if po.Status.Phase == v1.PodRunning {
			available = append(available, po.Name)
		}
	}
	if len(available) == 0 {
		return "", errors.New("no pods available")
	}
	return available[rand.Intn(len(available))], nil
}

func RunWebSocketStream(
	resp http.ResponseWriter, req *http.Request,
	fun func(ctx context.Context, stream remotecommand.StreamOptions) error,
) {
	conn, err := ws.InitWebsocket(resp, req)
	if err != nil {
		response.ServerError(resp, err)
		return
	}
	defer conn.WsClose()
	stream := &ws.StreamHandler{WsConn: conn, ResizeEvent: make(chan remotecommand.TerminalSize)}
	streamOptions := remotecommand.StreamOptions{
		Stdin:             stream,
		Stdout:            stream,
		Stderr:            stream,
		TerminalSizeQueue: stream,
		Tty:               true,
	}
	if err := fun(req.Context(), streamOptions); err != nil {
		_ = conn.WsWrite(websocket.TextMessage, []byte("websocket stream error: "+err.Error()))
		// todo: improve this hack
		<-time.AfterFunc(time.Duration(3)*time.Second, func() {
			conn.WsClose()
		}).C
	}
}

type PodCmdExecutor struct {
	Cluster        cluster.Interface
	Pod            client.ObjectKey
	PodExecOptions v1.PodExecOptions
	StreamOptions  remotecommand.StreamOptions
}

// Execute
func (pe PodCmdExecutor) Execute(ctx context.Context) error {
	req := pe.Cluster.Kubernetes().CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(pe.Pod.Namespace).
		Name(pe.Pod.Name).
		SubResource("exec").
		VersionedParams(
			&pe.PodExecOptions,
			scheme.ParameterCodec,
		)
	executor, err := remotecommand.NewSPDYExecutor(pe.Cluster.Config(), "POST", req.URL())
	if err != nil {
		return err
	}
	// TODO: call executor.StreamWithContext() instead of executor.Stream() after client-go upgrade to v1.24
	return executor.Stream(pe.StreamOptions)
}
