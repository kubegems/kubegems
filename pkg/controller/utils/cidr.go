package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCIDRs(c client.Client) ([]string, error) {
	ctx := context.Background()
	ret := []string{}
	nodeList := &corev1.NodeList{}
	if err := c.List(ctx, nodeList, &client.ListOptions{}); err != nil {
		return ret, err
	}
	for _, node := range nodeList.Items {
		ret = append(ret, node.Spec.PodCIDRs...)
	}
	return ret, nil
}
