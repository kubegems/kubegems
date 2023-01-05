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

package debug

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/helm"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/redis"
)

// ApplyPortForwardingOptions using apiserver port forward port for options
func ApplyPortForwardingOptions(ctx context.Context,
	debug bool,
	dbopts *database.Options,
	redisopts *redis.Options,
	gitopts *git.Options,
	appstoreopts *helm.Options,
	argoopts *argo.Options,
) error {
	// debug mode only
	if !debug {
		return nil
	}

	rest, err := kube.AutoClientConfig()
	if err != nil {
		return err
	}
	clientSet, err := kubernetes.NewForConfig(rest)
	if err != nil {
		return err
	}

	kubegemsSec, err := clientSet.CoreV1().Secrets(gems.NamespaceSystem).Get(ctx, "kubegems-config", v1.GetOptions{})
	if err != nil {
		return err
	}

	group := &errgroup.Group{}
	// mysql
	group.Go(func() error {
		addr, err := PortForward(ctx, rest, gems.NamespaceSystem, "kubegems-mysql", 3306)
		if err != nil {
			return err
		}
		mysqlSec, err := clientSet.CoreV1().Secrets(gems.NamespaceSystem).Get(ctx, "kubegems-mysql", v1.GetOptions{})
		if err != nil {
			return err
		}
		dbopts.Addr = addr
		dbopts.Password = string(mysqlSec.Data["mysql-root-password"])
		return nil
	})

	// redis
	group.Go(func() error {
		addr, err := PortForward(ctx, rest, gems.NamespaceSystem, "kubegems-redis-master", 6379)
		if err != nil {
			return err
		}
		redisSec, err := clientSet.CoreV1().Secrets(gems.NamespaceSystem).Get(ctx, "kubegems-redis", v1.GetOptions{})
		if err != nil {
			return err
		}
		redisopts.Addr = addr
		redisopts.Password = string(redisSec.Data["redis-password"])
		return nil
	})

	// git
	group.Go(func() error {
		if gitopts == nil {
			return nil
		}
		addr, err := PortForward(ctx, rest, gems.NamespaceSystem, "kubegems-gitea-http", 3000)
		if err != nil {
			return err
		}
		gitopts.Addr = "http://" + addr
		gitopts.Username = string(kubegemsSec.Data["GIT_USERNAME"])
		gitopts.Password = string(kubegemsSec.Data["GIT_PASSWORD"])
		return nil
	})

	// chartmuseum
	group.Go(func() error {
		if appstoreopts == nil {
			return nil
		}
		addr, err := PortForward(ctx, rest, gems.NamespaceSystem, "kubegems-chartmuseum", 8080)
		if err != nil {
			return err
		}
		appstoreopts.Addr = "http://" + addr
		return nil
	})

	// argo
	group.Go(func() error {
		addr, err := PortForward(ctx, rest, gems.NamespaceSystem, "kubegems-argo-cd-server", 80)
		if err != nil {
			return err
		}
		argoSec, err := clientSet.CoreV1().Secrets(gems.NamespaceSystem).Get(ctx, "argocd-secret", v1.GetOptions{})
		if err != nil {
			return err
		}
		argoopts.Addr = "http://" + addr
		argoopts.Password = string(argoSec.Data["clearPassword"])
		return nil
	})

	// jaeger tracing
	// group.Go(func() error {
	// 	addr, err := PortForward(ctx, rest, "observability", "jaeger-collector", 14268)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	os.Setenv("JAEGER_ENDPOINT", fmt.Sprintf("http://%s/api/traces", addr))
	// 	return nil
	// })

	if err := group.Wait(); err != nil {
		return err
	}
	return nil
}

func PortForward(ctx context.Context, config *rest.Config, namespace, svcname string, targetSvcPort int) (string, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}
	// get svc's pod
	svc, err := clientSet.CoreV1().Services(namespace).Get(ctx, svcname, v1.GetOptions{})
	if err != nil {
		return "", err
	}
	// get pod port from svc spec
	var targetPodPort intstr.IntOrString
	for _, port := range svc.Spec.Ports {
		if port.Port == int32(targetSvcPort) {
			targetPodPort = port.TargetPort
		}
	}

	pods, err := clientSet.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(svc.Spec.Selector)).String(),
	})
	if err != nil {
		return "", err
	}

	var targetPod *corev1.Pod
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		targetPod = &pod
		break
	}

	if targetPod == nil {
		return "", fmt.Errorf("no pods found for svc %s/%s", svc.Namespace, svc.Name)
	}

	var targetPodPortNum int32
	for _, c := range targetPod.Spec.Containers {
		for _, p := range c.Ports {
			if p.ContainerPort == targetPodPort.IntVal || p.Name == targetPodPort.StrVal {
				targetPodPortNum = p.ContainerPort
			}
		}
	}

	url := clientSet.
		CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(targetPod.Name).
		SubResource("portforward").
		URL()

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return "", errors.Wrap(err, "could not create round tripper")
	}

	readyChan := make(chan struct{})
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	// auto assign a port
	ln, err := net.Listen("tcp", "[::]:0")
	if err != nil {
		return "", err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	// reuse port next
	io.Close(ln)

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)
	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", port, targetPodPortNum)}, ctx.Done(), readyChan, out, errOut)
	if err != nil {
		return "", fmt.Errorf("forward svc %s/%s: %w", namespace, svcname, err)
	}

	go func() {
		if err = forwarder.ForwardPorts(); err != nil {
			log.Errorf("forward svc %s/%s: %s", namespace, svcname, err.Error())
		}
	}()
	<-readyChan

	if len(errOut.String()) != 0 {
		return "", fmt.Errorf("forward svc %s/%s: %s", namespace, svcname, errOut.String())
	}
	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	log.Debugf("forward-port: service %s/%s :%d -> %s", namespace, svcname, targetSvcPort, addr)
	return addr, nil
}
