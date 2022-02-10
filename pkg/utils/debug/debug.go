package debug

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/kubegems/gems/pkg/kube"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/service/options"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	GemSystemNamespace = "gemcloud-system"
)

// ApplyPortForwardingOptions using apiserver port forward port for options
func ApplyPortForwardingOptions(ctx context.Context, opts *options.Options) error {
	// debug mode only
	if !opts.DebugMode {
		return nil
	}

	rest, err := kube.AutoClientConfig()
	if err != nil {
		return err
	}

	group := &errgroup.Group{}

	// mysql
	group.Go(func() error {
		addr, err := PortForward(ctx, rest, GemSystemNamespace, "gems-mysql", 3306)
		if err != nil {
			return err
		}
		opts.Mysql.Addr = addr
		return nil
	})

	// redis
	group.Go(func() error {
		addr, err := PortForward(ctx, rest, GemSystemNamespace, "gems-redis", 6379)
		if err != nil {
			return err
		}
		opts.Redis.Addr = addr
		return nil
	})

	// git
	group.Go(func() error {
		addr, err := PortForward(ctx, rest, GemSystemNamespace, "gems-gitea", 3000)
		if err != nil {
			return err
		}
		opts.Git.Host = "http://" + addr
		return nil
	})

	// chartmuseum
	group.Go(func() error {
		addr, err := PortForward(ctx, rest, GemSystemNamespace, "gems-chartmuseum", 8030)
		if err != nil {
			return err
		}
		opts.Appstore.ChartRepoUrl = "http://" + addr
		return nil
	})

	// argo
	group.Go(func() error {
		addr, err := PortForward(ctx, rest, "gemcloud-workflow-system", "argocd-server", 80)
		if err != nil {
			return err
		}
		opts.Argo.Addr = "http://" + addr
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

func PortForward(ctx context.Context, config *rest.Config, namespace, svcname string, targetPort int) (string, error) {
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
	podport := 0
	for _, port := range svc.Spec.Ports {
		if port.Port == int32(targetPort) {
			podport = int(port.TargetPort.IntVal)
		}
	}
	targetPort = podport

	pods, err := clientSet.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(svc.Spec.Selector)).String(),
	})
	if err != nil {
		return "", err
	}

	podname := ""
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		podname = pod.Name
		break
	}

	if len(podname) == 0 {
		return "", fmt.Errorf("no pods found for svc %s/%s", svc.Namespace, svc.Name)
	}

	url := clientSet.
		CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(podname).
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
	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", port, targetPort)}, ctx.Done(), readyChan, out, errOut)
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
	log.Debugf("forward-port: service %s/%s :%d -> %s", namespace, svcname, targetPort, addr)
	return addr, nil
}
