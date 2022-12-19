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
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

const mysqlport = 3306

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	cfg, cli := setupClient()
	_ = cli

	selector := labels.SelectorFromSet(labels.Set{"app.kubernetes.io/name": "mysql"}).String()
	listenport, err := kube.PortForward(ctx, cfg, "kubegems", selector, mysqlport)
	if err != nil {
		panic(err)
	}
	// find mysql password
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "kubegems-mysql", Namespace: "kubegems"}}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		panic(err)
	}
	mysqlrootpassword := string(secret.Data["mysql-root-password"])
	db, err := database.NewDatabase(&database.Options{
		Addr:      fmt.Sprintf("localhost:%d", listenport),
		Username:  "root",
		Password:  mysqlrootpassword,
		Database:  "kubegems",
		Collation: "utf8mb4_unicode_ci",
	})
	if err != nil {
		panic(err)
	}

	// migrate models
	log.Print("migrating mysql models schema")
	if err := models.MigrateModels(db.DB()); err != nil {
		panic(err)
	}

	// updateDashboardTpls(db)
	updateDashboards(db)
	// updateReceivers(ctx, cli, rest.Host)

	log.Print("migrating kubegems plugins")
	MigratePlugins(ctx, cli)
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

func updateDashboardTpls(db *database.Database) {
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
				Name: graph.Name,
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

func updateDashboards(db *database.Database) {
	oldDashs := []*DashTmp{}
	if err := db.DB().Raw(`SELECT id, graphs FROM monitor_dashboards`).Scan(&oldDashs).Error; err != nil {
		panic(err)
	}

	for _, oldDash := range oldDashs {
		if err := json.Unmarshal(oldDash.Graphs, &oldDash.OldGraphs); err != nil {
			panic(err)
		}
		newGraphs := prometheus.MonitorGraphs{}
		for _, graph := range oldDash.OldGraphs {
			newGraphs = append(newGraphs, prometheus.MetricGraph{
				Name: graph.Name,
				Unit: graph.Unit,
				Targets: []prometheus.Target{{
					TargetName:      "A",
					PromqlGenerator: graph.PromqlGenerator,
					Expr:            graph.Expr,
				}},
			})
		}
		if err := db.DB().Model(&models.MonitorDashboard{}).Where("id= ?", oldDash.ID).Update("graphs", newGraphs).Error; err != nil {
			panic(err)
		}
	}
}

func updateReceivers(ctx context.Context, cli client.Client, cctx string) {
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
