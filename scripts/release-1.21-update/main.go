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

package main

import (
	"context"

	v1beta1 "github.com/banzaicloud/logging-operator/pkg/sdk/logging/api/v1beta1"
	"github.com/banzaicloud/logging-operator/pkg/sdk/logging/model/filter"
	mv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/agent/indexer"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/prometheus"
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
	updatepromrules(cli)
	updateFlows(cli)
}

// alert manager config's default receiver
func updateAMConfig(cli client.Client) {
	ctx := context.TODO()
	amCfgs := v1alpha1.AlertmanagerConfigList{}
	if err := cli.List(ctx, &amCfgs, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(map[string]string{
		// "alertmanagerConfig": "gemcloud",
	})); err != nil {
		panic(err)
	}

	for _, v := range amCfgs.Items {
		if v.Namespace == "gemcloud-monitoring-system" || v.Namespace == gems.NamespaceMonitor {
			continue
		}

		todel := v1alpha1.AlertmanagerConfig{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: v.Namespace,
				Name:      v.Name,
			},
		}

		v.Name = prometheus.DefaultAlertCRDName
		v.Labels = map[string]string{
			gems.LabelAlertmanagerConfigName: v.Name,
		}

		for i := range v.Spec.Receivers {
			if v.Spec.Receivers[i].Name == prometheus.DefaultReceiverName {
				for j, w := range v.Spec.Receivers[i].WebhookConfigs {
					if *w.URL == "https://gems-agent.gemcloud-system:8041/alert" {
						v.Spec.Receivers[i].WebhookConfigs[j].URL = &prometheus.DefaultReceiverURL
					}
				}
			}
		}
		v.ResourceVersion = ""
		if err := cli.Create(ctx, v); err != nil {
			panic(err)
		}
		if err := cli.Delete(ctx, &todel); err != nil {
			panic(err)
		}
		log.Infof("update amconfig succeed: %s %s\n", v.Namespace, v.Name)
	}
}

// istio version and gateway
func updatepromrules(cli client.Client) {
	ctx := context.TODO()
	prules := mv1.PrometheusRuleList{}
	if err := cli.List(ctx, &prules, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(map[string]string{
		// "prometheusRule": "gemcloud",
	})); err != nil {
		panic(err)
	}

	for _, v := range prules.Items {
		if v.Namespace == "gemcloud-monitoring-system" || v.Namespace == gems.NamespaceMonitor {
			continue
		}
		todel := mv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: v.Namespace,
				Name:      v.Name,
			},
		}
		v.Name = prometheus.DefaultAlertCRDName
		v.Labels = map[string]string{
			gems.LabelPrometheusRuleType: "monitor",
			gems.LabelPrometheusRuleName: v.Name,
		}

		v.ResourceVersion = ""
		if err := cli.Create(ctx, v); err != nil {
			panic(err)
		}
		if err := cli.Delete(ctx, &todel); err != nil {
			panic(err)
		}
		log.Infof("update promrule succeed: %s %s\n", v.Namespace, v.Name)
	}
}

func updateFlows(cli client.Client) {
	prometheusFilter := func(flow string) *filter.PrometheusConfig {
		return &filter.PrometheusConfig{
			Labels: filter.Label{
				"container": "$.kubernetes.container_name",
				"namespace": "$.kubernetes.namespace_name",
				"node":      "$.kubernetes.host",
				"pod":       "$.kubernetes.pod_name",
				"flow":      flow,
			},
			Metrics: []filter.MetricSection{
				{
					Name: "gems_logging_flow_records_total",
					Type: "counter",
					Desc: "Total number of log entries collected by this each flow",
				},
			},
		}
	}
	ctx := context.TODO()
	olds := v1beta1.FlowList{}
	if err := cli.List(ctx, &olds, client.InNamespace(v1.NamespaceAll)); err != nil {
		panic(err)
	}
	for _, old := range olds.Items {
		if old.Name == "default" {
			// ns
			ns := v1.Namespace{}
			if err := cli.Get(ctx, types.NamespacedName{Name: old.Namespace}, &ns); err != nil {
				panic(err)
			}
			if ns.Labels == nil {
				ns.Labels = make(map[string]string)
			}
			ns.Labels[gems.LabelLogCollector] = gems.StatusEnabled
			if err := cli.Update(ctx, &ns); err != nil {
				panic(err)
			}
			log.Infof("update namespace succeed: %s \n", ns.Name)

			// flow filter
			filters := old.Spec.Filters
			for i := range old.Spec.Filters {
				if filters[i].Prometheus != nil {
					filters[i].Prometheus = prometheusFilter(old.Name)
				}
			}

			// flow output
			defaultGlobalOutput := "kubegems-container-console-output"
			found := false
			for _, v := range old.Spec.GlobalOutputRefs {
				if v == defaultGlobalOutput {
					found = true
					break
				}
			}
			if !found {
				old.Spec.GlobalOutputRefs = append(old.Spec.GlobalOutputRefs, defaultGlobalOutput)
			}
			if err := cli.Update(ctx, &old); err != nil {
				panic(err)
			}
			log.Infof("update flow succeed: %s %s\n", old.Namespace, old.Name)
		}
	}
}
