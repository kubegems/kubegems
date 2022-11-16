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
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/agent/indexer"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/observe"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/channels"
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
	channelMap     = map[string]*models.AlertChannel{}
	nginxAnnosPath = "scripts/release-1.22-update/nginx-anno.yaml"
	tgsPath        = "scripts/release-1.22-update/tg.yaml"
	logAmcfgPath   = "scripts/release-1.22-update/log-amcfg.yaml"
	channelPath    = "scripts/release-1.22-update/channels.yaml"
	db             *database.Database
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
	db, err = database.NewDatabase(&database.Options{
		Addr:      "10.12.32.41:3306",
		Username:  "root",
		Password:  "X69KdO15T8", // dev
		Database:  "kubegems",
		Collation: "utf8mb4_unicode_ci",
	})
	if err != nil {
		panic(err)
	}

	bts, _ := os.ReadFile(channelPath)
	if err := yaml.Unmarshal(bts, &channelMap); err != nil {
		panic(err)
	}
	// getAnno(cli)
	// updateAnno(cli)
	// updatePromrule(cli)
	// updateGateway(cli)
	// mergeLogMonitorReceiver(cli)
	// getChannels(cli)
	// saveChannels(cli)
	updateReceivers(cli)
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

func mergeLogMonitorReceiver(cli client.Client) {
	ctx := context.TODO()
	amConfigs := v1alpha1.AlertmanagerConfigList{}
	if err := cli.List(ctx, &amConfigs, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(map[string]string{
		gems.LabelAlertmanagerConfigName: "kubegems-default-logging-alert-rule",
	})); err != nil {
		panic(err)
	}

	for _, v := range amConfigs.Items {
		monitorAMCfg, err := getOrCreateAlertmanagerConfig(cli, ctx, v.Namespace, prometheus.DefaultAlertCRDName)
		if err != nil {
			panic(err)
		}
		monRecMap := map[string]v1alpha1.Receiver{}
		for _, v := range monitorAMCfg.Spec.Receivers {
			monRecMap[v.Name] = v
		}
		for _, logRec := range v.Spec.Receivers {
			if _, ok := monRecMap[logRec.Name]; !ok {
				monitorAMCfg.Spec.Receivers = append(monitorAMCfg.Spec.Receivers, logRec)
				log.Printf("namespace %s append receiver %s", v.Namespace, logRec.Name)
			}
		}
		for _, route := range v.Spec.Route.Routes {
			monitorAMCfg.Spec.Route.Routes = append(monitorAMCfg.Spec.Route.Routes, route)
			log.Printf("namespace %s append route %s", v.Namespace, string(route.String()))
		}

		if err := cli.Update(ctx, monitorAMCfg); err != nil {
			panic(err)
		}
		log.Printf("namespace %s merge finished", v.Namespace)
		if err := cli.Delete(ctx, v); err != nil {
			panic(err)
		}
	}
	bts, _ := yaml.Marshal(amConfigs)
	os.WriteFile(logAmcfgPath, bts, os.ModeAppend)
}

func getOrCreateAlertmanagerConfig(cli client.Client, ctx context.Context, namespace, name string) (*v1alpha1.AlertmanagerConfig, error) {
	aconfig := &v1alpha1.AlertmanagerConfig{}
	err := cli.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, aconfig)
	if kerrors.IsNotFound(err) {
		// 初始化
		aconfig = observe.GetBaseAlertmanagerConfig(namespace, name)
		if err := cli.Create(ctx, aconfig); err != nil {
			return nil, err
		}
		return aconfig, nil
	}
	return aconfig, err
}

func getChannels(cli client.Client) {
	ctx := context.TODO()
	amconfigs := v1alpha1.AlertmanagerConfigList{}
	if err := cli.List(ctx, &amconfigs, client.InNamespace(v1.NamespaceAll), client.HasLabels([]string{
		gems.LabelAlertmanagerConfigName,
	})); err != nil {
		panic(err)
	}

	for _, v := range amconfigs.Items {
		ns := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: v.Namespace,
			},
		}
		cli.Get(ctx, client.ObjectKeyFromObject(&ns), &ns)
		tenant := models.Tenant{}
		if err := db.DB().First(&tenant, "tenant_name = ?", ns.Labels[gems.LabelTenant]).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				panic(err)
			}
		}
		for _, rec := range v.Spec.Receivers {
			var tenantID *uint
			if rec.Name == "gemcloud-default-webhook" {
				continue
			}
			if tenant.ID == 0 {
				tenantID = nil
			} else {
				tenantID = &tenant.ID
			}
			for _, wc := range rec.WebhookConfigs {
				if _, ok := channelMap[webhookString(wc, tenantID)]; !ok {
					if strings.Contains(*wc.URL, "alertproxy") {
						u, _ := url.Parse(*wc.URL)
						channelMap[webhookString(wc, tenantID)] = &models.AlertChannel{
							Name: rec.Name,
							ChannelConfig: channels.ChannelConfig{
								ChannelIf: &channels.Feishu{
									ChannelType: channels.TypeFeishu,
									URL:         u.Query().Get("url"),
									At:          u.Query().Get("at"),
									SignSecret:  u.Query().Get("signSecret"),
								},
							},
							TenantID: tenantID,
						}
					} else {
						channelMap[webhookString(wc, tenantID)] = &models.AlertChannel{
							Name: rec.Name,
							ChannelConfig: channels.ChannelConfig{
								ChannelIf: &channels.Webhook{
									ChannelType: channels.TypeWebhook,
									URL:         *wc.URL,
								},
							},
							TenantID: tenantID,
						}
					}
				}
			}
			for i, ec := range rec.EmailConfigs {
				if _, ok := channelMap[emailString(ec, tenantID)]; !ok {
					sec := v1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: v.Namespace,
							Name:      channels.EmailSecretName,
						},
					}
					cli.Get(ctx, client.ObjectKeyFromObject(&sec), &sec)
					channelMap[emailString(ec, tenantID)] = &models.AlertChannel{
						Name: rec.Name,
						ChannelConfig: channels.ChannelConfig{
							ChannelIf: &channels.Email{
								ChannelType:  channels.TypeEmail,
								SMTPServer:   ec.Smarthost,
								RequireTLS:   *rec.EmailConfigs[i].RequireTLS,
								From:         ec.From,
								To:           ec.To,
								AuthPassword: string(sec.Data[channels.EmailSecretKey(rec.Name, ec.From)]),
							},
						},
						TenantID: tenantID,
					}
				}
			}
		}
	}

	id := 2
	for _, v := range channelMap {
		if v.Name == "gemcloud-default-webhook" {
			continue
		}
		v.ID = uint(id)
		id++
	}
	bts, _ := yaml.Marshal(channelMap)
	os.WriteFile(channelPath, bts, os.ModeAppend)
}

func saveChannels(cli client.Client) {
	for _, v := range channelMap {
		if v.Name == "gemcloud-default-webhook" || v.Name == models.DefaultChannel.Name {
			continue
		}
		if err := db.DB().Debug().Create(v).Error; err != nil {
			panic(err)
		}
	}
}

func updateReceivers(cli client.Client) {
	ctx := context.TODO()
	amconfigs := v1alpha1.AlertmanagerConfigList{}
	if err := cli.List(ctx, &amconfigs, client.InNamespace(v1.NamespaceAll), client.HasLabels([]string{
		gems.LabelAlertmanagerConfigName,
	})); err != nil {
		panic(err)
	}

	for _, v := range amconfigs.Items {
		ns := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: v.Namespace,
			},
		}
		cli.Get(ctx, client.ObjectKeyFromObject(&ns), &ns)
		tenant := models.Tenant{}
		if err := db.DB().First(&tenant, "tenant_name = ?", ns.Labels[gems.LabelTenant]).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				panic(err)
			}
		}
		oldRecNameMap := map[string]string{}
		secData := map[string][]byte{}

		// receivers
		for recID, rec := range v.Spec.Receivers {
			var tenantID *uint
			if len(rec.EmailConfigs) == 0 && len(rec.WebhookConfigs) == 0 {
				continue
			}
			if rec.Name == "gemcloud-default-webhook" {
				v.Spec.Receivers[recID] = models.DefaultReceiver
				continue
			}
			if tenant.ID == 0 {
				tenantID = nil
			} else {
				tenantID = &tenant.ID
			}
			for _, wc := range rec.WebhookConfigs {
				ch, ok := channelMap[webhookString(wc, tenantID)]
				if !ok {
					log.Fatalf("cluster: %s, ns: %s, rec: %s channel not found", cctx, v.Namespace, rec.Name)
				}
				v.Spec.Receivers[recID] = ch.ChannelConfig.ToReceiver(ch.ReceiverName())
				oldRecNameMap[rec.Name] = ch.ReceiverName()
				log.Printf("ns: %s, amcfg: %s, old rec: %s, new rec: %s", v.Namespace, v.Name, rec.Name, ch.ReceiverName())
			}
			for _, ec := range rec.EmailConfigs {
				ch, ok := channelMap[emailString(ec, tenantID)]
				if !ok {
					log.Fatalf("cluster: %s, ns: %s, rec: %s channel not found", cctx, v.Namespace, rec.Name)
				}
				v.Spec.Receivers[recID] = ch.ChannelConfig.ToReceiver(ch.ReceiverName())
				oldRecNameMap[rec.Name] = ch.ReceiverName()
				secData[channels.EmailSecretKey(ch.ReceiverName(), ec.From)] = []byte(ch.ChannelConfig.ChannelIf.(*channels.Email).AuthPassword)
				log.Printf("ns: %s, amcfg: %s, old rec: %s, new rec: %s", v.Namespace, v.Name, rec.Name, ch.ReceiverName())
			}
		}

		// routes
		routes, err := v.Spec.Route.ChildRoutes()
		if err != nil {
			panic(err)
		}
		newRoutes := []extv1.JSON{}
		for _, route := range routes {
			if route.Receiver == "gemcloud-default-webhook" {
				route.Receiver = models.DefaultReceiver.Name
			} else {
				newRecName, ok := oldRecNameMap[route.Receiver]
				if !ok {
					log.Fatalf("cluster: %s, ns: %s, route rec: %s not found", cctx, v.Namespace, route.Receiver)
				}
				route.Receiver = newRecName
			}
			bts, _ := json.Marshal(route)
			newRoutes = append(newRoutes, extv1.JSON{Raw: bts})
		}
		v.Spec.Route.Routes = newRoutes
		if err := cli.Update(ctx, v); err != nil {
			panic(err)
		}

		// secret
		sec := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: v.Namespace,
				Name:      channels.EmailSecretName,
			},
		}
		if err := cli.Get(ctx, client.ObjectKeyFromObject(&sec), &sec); err != nil {
			log.Println(err)
			continue
		}
		sec.Data = secData
		if err := cli.Update(ctx, &sec); err != nil {
			panic(err)
		}
	}
	bts, _ := yaml.Marshal(amconfigs)
	os.WriteFile("a.yaml", bts, os.ModeAppend)
}

func webhookString(w v1alpha1.WebhookConfig, id *uint) string {
	if id == nil {
		return fmt.Sprintf("%s-null", *w.URL)
	}
	return fmt.Sprintf("%s-%d", *w.URL, *id)
}

func emailString(e v1alpha1.EmailConfig, id *uint) string {
	ret, _ := json.Marshal(e)
	if id == nil {
		return fmt.Sprintf("%s-null", string(ret))
	}
	return fmt.Sprintf("%s-%d", string(ret), *id)
}
