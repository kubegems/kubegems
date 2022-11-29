// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/pointer"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/agent/indexer"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
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
	channelMap     = map[string]*models.AlertChannel{}
	nginxAnnosPath = "scripts/release-1.22-update/nginx-anno.yaml"
	tgsPath        = "scripts/release-1.22-update/tg.yaml"
	logAmcfgPath   = "scripts/release-1.22-update/log-amcfg.yaml"
	channelPath    = "scripts/release-1.22-update/channels.yaml"
	db             *database.Database
	cli            client.Client
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

	cli = c.GetClient()
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
	// updateDashboardTpls()
	updateDashboards()
	// updateReceivers()
}

type DashTmp struct {
	ID        uint
	Name      string
	Graphs    datatypes.JSON
	OldGraphs []OldGraph `gorm:"-"`
}

type OldGraph struct {
	Name            string                      `json:"name"`
	Expr            string                      `json:"expr"`
	Unit            string                      `json:"unit"`
	PromqlGenerator *prometheus.PromqlGenerator `json:"promqlGenerator"`
}

func updateDashboardTpls() {
	oldDashTpls := []*DashTmp{}
	if err := db.DB().Raw(`SELECT name, graphs FROM monitor_dashboard_tpls`).Scan(&oldDashTpls).Error; err != nil {
		panic(err)
	}

	for _, tpl := range oldDashTpls {
		if err := json.Unmarshal(tpl.Graphs, &tpl.OldGraphs); err != nil {
			panic(err)
		}
		newGraphs := prometheus.MonitorGraphs{}
		for _, graph := range tpl.OldGraphs {
			newGraphs = append(newGraphs, prometheus.MetricGraph{
				Name: tpl.Name,
				Unit: graph.Unit,
				Targets: []prometheus.Target{{
					TargetName:      "A",
					PromqlGenerator: graph.PromqlGenerator,
					Expr:            graph.Expr,
				}},
			})
		}
		if err := db.DB().Model(&models.MonitorDashboardTpl{}).Where("name= ?", tpl.Name).Update("graphs", newGraphs).Error; err != nil {
			panic(err)
		}
	}
}

func updateDashboards() {
	oldDashs := []*DashTmp{}
	if err := db.DB().Raw(`SELECT id, graphs FROM monitor_dashboards`).Scan(&oldDashs).Error; err != nil {
		panic(err)
	}

	for _, tpl := range oldDashs {
		if err := json.Unmarshal(tpl.Graphs, &tpl.OldGraphs); err != nil {
			panic(err)
		}
		newGraphs := prometheus.MonitorGraphs{}
		for _, graph := range tpl.OldGraphs {
			newGraphs = append(newGraphs, prometheus.MetricGraph{
				Name: tpl.Name,
				Unit: graph.Unit,
				Targets: []prometheus.Target{{
					TargetName:      "A",
					PromqlGenerator: graph.PromqlGenerator,
					Expr:            graph.Expr,
				}},
			})
		}
		if err := db.DB().Model(&models.MonitorDashboard{}).Where("id= ?", tpl.ID).Update("graphs", newGraphs).Error; err != nil {
			panic(err)
		}
	}
}

func updateReceivers() {
	ctx := context.TODO()
	amconfigs := v1alpha1.AlertmanagerConfigList{}
	if err := cli.List(ctx, &amconfigs, client.InNamespace(v1.NamespaceAll), client.HasLabels([]string{
		gems.LabelAlertmanagerConfigName,
	})); err != nil {
		panic(err)
	}

	for _, v := range amconfigs.Items {
		for i, rec := range v.Spec.Receivers {
			for j := range v.Spec.Receivers[i].WebhookConfigs {
				if rec.Name != models.DefaultReceiver.Name {
					v.Spec.Receivers[i].WebhookConfigs[j].SendResolved = pointer.Bool(false)
					log.Printf("cluster: %s, ns: %s, webhook rec: %s", cctx, v.Namespace, rec.Name)
				}
			}
			for j := range v.Spec.Receivers[i].EmailConfigs {
				v.Spec.Receivers[i].EmailConfigs[j].SendResolved = pointer.Bool(false)
				log.Printf("cluster: %s, ns: %s, email rec: %s", cctx, v.Namespace, rec.Name)
			}
		}
		if err := cli.Update(ctx, v); err != nil {
			panic(err)
		}
	}
}
