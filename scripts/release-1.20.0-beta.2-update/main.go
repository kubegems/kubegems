package main

import (
	"context"

	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	pkgv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/agent/indexer"
	"kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/apis/networking"
	"kubegems.io/pkg/log"
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

	cli := c.GetClient()
	updateAMConfig(cli)
	updateIstioOp(cli)
}

// alert manager config's default receiver
func updateAMConfig(cli client.Client) {
	ctx := context.TODO()
	amCfgs := v1alpha1.AlertmanagerConfigList{}
	if err := cli.List(ctx, &amCfgs, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(map[string]string{
		gems.LabelAlertmanagerConfig: prometheus.MonitorAlertmanagerConfigName,
	})); err != nil {
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
		log.Infof("default receiver not found in: %s %s\n", v.Namespace, v.Name)
		v.Spec.Receivers = append(v.Spec.Receivers, prometheus.DefaultReceiver)
	Update:
		if err := cli.Update(ctx, v); err != nil {
			panic(err)
		}
		log.Infof("update succeed: %s %s\n", v.Namespace, v.Name)
	}
}

// istio version and gateway
func updateIstioOp(cli client.Client) {
	ctx := context.TODO()
	istioOp := pkgv1alpha1.IstioOperator{}
	if err := cli.Get(ctx, types.NamespacedName{
		Namespace: "istio-system",
		Name:      "gems-istio",
	}, &istioOp); err != nil {
		log.Error(err, "get istio")
		return
	}

	istioOp.Spec.Tag = "1.11.7"

	oldGwlabel := "gems.cloudminds.com/istioGateway"
	oldVsLabel := "gems.cloudminds.com/virtualSpace"
	for _, gw := range istioOp.Spec.Components.IngressGateways {
		if v, ok := gw.Label[oldGwlabel]; ok {
			gw.Label[networking.AnnotationIstioGateway] = v
			delete(gw.Label, oldGwlabel)
		}
		if v, ok := gw.Label[oldVsLabel]; ok {
			gw.Label[networking.AnnotationVirtualSpace] = v
			delete(gw.Label, oldVsLabel)
		}
	}

	if err := cli.Update(ctx, &istioOp); err != nil {
		log.Error(err, "update istio")
		return
	}
	log.Info("update istio succeed")
}
