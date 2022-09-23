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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	mv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/agent/indexer"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type NginxAnno struct {
	NewKey   string    `json:"newKey"` // empty means no need to set
	IngInfos []IngInfo `json:"ingInfos"`
}

type IngInfo struct {
	Cluster      string `json:"cluster"`
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	IngressClass string `json:"ingressClass"`
	OldValue     string `json:"oldValue"`
	NewValue     string `json:"newValue"`
}

var (
	cctx           = ""
	nginxAnnos     = make(map[string]NginxAnno)
	nginxAnnosPath = "scripts/release-1.22-update/nginx-anno.yaml"
	tgsPath        = "scripts/release-1.22-update/tg.yaml"
)

func main() {
	cfg := clientcmdapi.Config{}
	tmp, err := os.ReadFile("/home/slt/.kube/config")
	if err != nil {
		panic(err)
	}
	yaml.Unmarshal(tmp, &cfg)
	cctx = cfg.CurrentContext

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

	bts, _ := os.ReadFile(nginxAnnosPath)
	if err := yaml.Unmarshal(bts, &nginxAnnos); err != nil {
		panic(err)
	}
	// getAnno(cli)
	updateAnno(cli)
	updatePromrule(cli)
	updateGateway(cli)
}

// alert manager config's default receiver
func getAnno(cli client.Client) {
	ctx := context.TODO()
	ingresses := networkingv1.IngressList{}
	if err := cli.List(ctx, &ingresses, client.InNamespace(v1.NamespaceAll)); err != nil {
		panic(err)
	}

	for _, ing := range ingresses.Items {
		if ing.Spec.IngressClassName != nil {
			for k, v := range ing.Annotations {
				if strings.HasPrefix(k, "nginx.org") {
					infos := nginxAnnos[k].IngInfos
					infos = append(infos, IngInfo{
						Cluster:      cctx,
						Namespace:    ing.Namespace,
						Name:         ing.Name,
						IngressClass: *ing.Spec.IngressClassName,
						OldValue:     v,
					})
				}
			}
		}
	}

	bts, _ := yaml.Marshal(nginxAnnos)
	os.WriteFile(nginxAnnosPath, bts, os.ModeAppend)
}

func updateAnno(cli client.Client) {
	ctx := context.TODO()
	for _, v := range nginxAnnos {
		for _, info := range v.IngInfos {
			if v.NewKey != "" && info.Cluster == cctx {
				ing := networkingv1.Ingress{}
				if err := cli.Get(ctx, types.NamespacedName{
					Namespace: info.Namespace,
					Name:      info.Name,
				}, &ing); err != nil {
					panic(err)
				}

				ing.Annotations[v.NewKey] = info.NewValue
				if err := cli.Update(ctx, &ing); err != nil {
					panic(err)
				}
			}
		}
	}
}

func updatePromrule(cli client.Client) {
	ctx := context.TODO()
	rules := mv1.PrometheusRuleList{}
	if err := cli.List(ctx, &rules, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(map[string]string{
		"prometheusrule.kubegems.io/type": "monitor",
	})); err != nil {
		panic(err)
	}

	tplGetter := models.NewPromqlTplMapperFromFile().FindPromqlTplWithoutScope
	for _, rule := range rules.Items {
		for _, g := range rule.Spec.Groups {
			for _, r := range g.Rules {
				tmp, ok := r.Annotations[prometheus.ExprJsonAnnotationKey]
				if ok {
					gen := prometheus.PromqlGenerator{}
					json.Unmarshal([]byte(tmp), &gen)
					tpl, err := tplGetter(gen.Resource, gen.Rule)
					if err != nil {
						panic(err)
					}
					gen.Scope = tpl.ScopeName
					bts, _ := json.Marshal(gen)
					r.Annotations[prometheus.ExprJsonAnnotationKey] = string(bts)

					// format message
					msg := fmt.Sprintf("%s: [cluster:{{ $externalLabels.%s }}] ", g.Name, prometheus.AlertClusterKey)
					for _, label := range tpl.Labels {
						msg += fmt.Sprintf("[%s:{{ $labels.%s }}] ", label, label)
					}
					unitValue, err := prometheus.ParseUnit(gen.Unit)
					msg += fmt.Sprintf("%s trigger alert, value: %s%s", tpl.RuleShowName, prometheus.ValueAnnotationExpr, unitValue.Show)
					r.Annotations[prometheus.MessageAnnotationsKey] = msg

					delete(r.Labels, "gems_alert_resource")
					delete(r.Labels, "gems_alert_rule")
					r.Labels[prometheus.AlertPromqlTpl] = tpl.String()
				}
			}
		}
		if err := cli.Update(ctx, rule); err != nil {
			panic(err)
		}
	}
}

func updateGateway(cli client.Client) {
	ctx := context.TODO()
	tgs := v1beta1.TenantGatewayList{}
	if err := cli.List(ctx, &tgs); err != nil {
		panic(err)
	}
	bts, _ := yaml.Marshal(tgs)
	os.WriteFile(tgsPath, bts, os.ModeAppend)

	for _, tg := range tgs.Items {
		if err := cli.Delete(ctx, &tg); err != nil {
			panic(err)
		}
	}

	time.Sleep(30 * time.Second)

	for _, tg := range tgs.Items {
		tg.ResourceVersion = ""
		tg.Spec.Image = v1beta1.Image{
			Repository: "registry.cn-beijing.aliyuncs.com/kubegems/nginx-ingress",
			Tag:        "v1.3.0",
		}
		if err := cli.Create(ctx, &tg); err != nil {
			panic(err)
		}
	}
	time.Sleep(30 * time.Second)

	for _, tg := range tgs.Items {
		svc := v1.Service{}
		if err := cli.Get(ctx, types.NamespacedName{
			Namespace: gems.NamespaceGateway,
			Name:      tg.Name,
		}, &svc); err != nil {
			panic(err)
		}

		svc.Spec.Ports = tg.Status.Ports
		if err := cli.Update(ctx, &svc); err != nil {
			panic(err)
		}
	}
}
