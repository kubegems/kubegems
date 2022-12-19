package kube

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"

	"emperror.dev/errors"
	"github.com/argoproj/argo-cd/v2/util/io"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func PortForward(ctx context.Context, config *rest.Config, namespace string, podLabelSelector string, targetPort int) (int, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return -1, err
	}
	podlist, err := clientSet.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{LabelSelector: podLabelSelector})
	if err != nil {
		return -1, err
	}
	if len(podlist.Items) == 0 {
		return -1, fmt.Errorf("no pods selected with label selector: %s", podLabelSelector)
	}
	pod := podlist.Items[0]

	url := clientSet.CoreV1().RESTClient().Post().Resource("pods").
		Namespace(pod.Namespace).Name(pod.Name).
		SubResource("portforward").URL()

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return -1, errors.Wrap(err, "Could not create round tripper")
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)

	// random select a unused port using port number 0
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return -1, err
	}
	tmpaddr, _ := ln.Addr().(*net.TCPAddr)
	port := tmpaddr.Port
	io.Close(ln)

	readyChan := make(chan struct{})
	errChan := make(chan error)

	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", port, targetPort)}, ctx.Done(), readyChan, out, errOut)
	if err != nil {
		return -1, err
	}
	go func() {
		if err = forwarder.ForwardPorts(); err != nil {
			errChan <- err
		}
	}()
	select {
	case err = <-errChan:
		return -1, err
	case <-readyChan:
	}
	if len(errOut.String()) != 0 {
		return -1, fmt.Errorf(errOut.String())
	}
	return port, nil
}
