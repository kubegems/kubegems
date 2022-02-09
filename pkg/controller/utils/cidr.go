package utils

import (
	"context"
	"errors"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _cidr string

const (
	kubesystem                     = "kube-system"
	controllerManagerContainerName = "kube-controller-manager"
	cidrPrefix                     = "--cluster-cidr="
)

var controllerManagerLabel = map[string]string{
	"component": "kube-controller-manager",
}

func GetCIDR(c client.Client) (string, error) {
	if len(_cidr) == 0 {
		return getCIDR(c)
	}
	return _cidr, nil
}

func getCIDR(c client.Client) (string, error) {
	ctx := context.Background()
	podlist := &corev1.PodList{}
	if err := c.List(ctx, podlist, &client.ListOptions{
		Namespace:     kubesystem,
		LabelSelector: labels.SelectorFromSet(controllerManagerLabel),
	}); err != nil {
		return "", err
	}
	if len(podlist.Items) == 0 {
		return "", errors.New("get cidr error, can't get apiserver pod")
	}
	pod := podlist.Items[0]

	command := []string{}
	for _, container := range pod.Spec.Containers {
		if container.Name == controllerManagerContainerName {
			command = container.Command
		}
	}
	if len(command) == 0 {
		return "", errors.New("get cidr error, can't get apiserver command")
	}
	for _, cmd := range command {
		if !strings.HasPrefix(cmd, cidrPrefix) {
			continue
		}
		left := strings.TrimPrefix(cmd, cidrPrefix)
		_cidr = left
		return _cidr, nil
	}
	return "", errors.New("get cidr error, can't get command args")
}
