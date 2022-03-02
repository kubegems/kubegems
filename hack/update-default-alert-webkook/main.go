package main

import (
	"context"
	"fmt"

	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/agent/indexer"
	"kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {

	rest, err := kube.AutoClientConfig()
	if err != nil {
		panic(err)
	}

	c, err := cluster.NewCluster(rest)
	if err != nil {
		panic(err)
	}

	if err := indexer.CustomIndexPods(c.GetCache()); err != nil {
		panic(err)
	}

	ctx := context.TODO()
	go c.Start(ctx)
	c.GetCache().WaitForCacheSync(ctx)

	amCfgs := v1alpha1.AlertmanagerConfigList{}
	if err := c.GetClient().List(ctx, &amCfgs, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(prometheus.AlertmanagerConfigSelector)); err != nil {
		panic(err)
	}

	for _, v := range amCfgs.Items {
		for i := range v.Spec.Receivers {
			if v.Spec.Receivers[i].Name == prometheus.DefaultReceiverName {
				v.Spec.Receivers[i] = prometheus.DefaultReceiver
				goto Update
			}
		}
		// not found
		fmt.Printf("default receiver not found in: %s %s\n", v.Namespace, v.Name)
		v.Spec.Receivers = append(v.Spec.Receivers, prometheus.DefaultReceiver)
	Update:
		if err := c.GetClient().Update(ctx, v); err != nil {
			panic(err)
		}
		fmt.Printf("update succeed: %s %s\n", v.Namespace, v.Name)
	}
}
