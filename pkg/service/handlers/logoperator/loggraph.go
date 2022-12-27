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

package logoperatorhandler

import (
	"context"
	"sort"
	"strings"

	// "fmt"
	// "net/http"
	// "net/url"
	// "sync"

	loggingv1beta1 "github.com/banzaicloud/logging-operator/pkg/sdk/logging/api/v1beta1"
	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// promemodel "github.com/prometheus/common/model"

	// corev1 "k8s.io/api/core/v1"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "src.kubegems.io/controller/gemlabels"
)

var PrimaryKeyName = "tenant_id"

type AggrValue struct {
	Min  float64
	Hour float64
	Day  float64
}

func (h *LogOperatorHandler) GetTenantNamespaces(c *gin.Context) ([]string, error) {
	var list []models.Environment
	var namespaces []string
	projectids := []uint{}
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Model(&models.Project{}).Where("tenant_id = ?", c.Param(PrimaryKeyName)).Pluck("id", &projectids).Error; err != nil {
		return nil, err
	}
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		return nil, err
	}
	cond := &handlers.PageQueryCond{
		Model:                  "Environment",
		SearchFields:           []string{"EnvironmentName"},
		PreloadFields:          []string{"Creator", "Cluster", "Project", "ResourceQuota", "Applications", "Users"},
		PreloadSensitiveFields: map[string]string{"Cluster": "id, cluster_name"},
		Where:                  []*handlers.QArgs{handlers.Args("project_id in (?)", projectids)},
	}

	_, _, _, err = query.PageList(h.GetDB().WithContext(ctx), cond, &list)

	if err != nil {
		return nil, err
	}

	for _, e := range list {
		namespaces = append(namespaces, e.Namespace)
	}
	return namespaces, nil
}

func (h *LogOperatorHandler) Flows(c *gin.Context) {
	cluster := c.Param("cluster")
	tenant := c.Param(PrimaryKeyName)
	ctx := c.Request.Context()
	allFlows := []loggingv1beta1.Flow{}
	flows := loggingv1beta1.FlowList{}
	if err := h.Execute(ctx, cluster, func(ctx context.Context, tc agents.Client) error {
		return tc.List(ctx, &flows, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(map[string]string{gemlabels.LabelTenant: tenant}))
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	allFlows = append(allFlows, flows.Items...)
	sort.Slice(allFlows, func(i, j int) bool {
		return strings.ToLower(allFlows[i].Name) < strings.ToLower(allFlows[j].Name)
	})
	handlers.OK(c, allFlows)
}

func (h *LogOperatorHandler) Outputs(c *gin.Context) {
	cluster := c.Param("cluster")
	tenant := c.Param(PrimaryKeyName)
	ctx := c.Request.Context()
	allOutputs := []loggingv1beta1.Output{}
	outputs := loggingv1beta1.OutputList{}
	if err := h.Execute(ctx, cluster, func(ctx context.Context, tc agents.Client) error {
		return tc.List(ctx, &outputs, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(map[string]string{gemlabels.LabelTenant: tenant}))
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	allOutputs = append(allOutputs, outputs.Items...)
	sort.Slice(allOutputs, func(i, j int) bool {
		return strings.ToLower(allOutputs[i].Name) < strings.ToLower(allOutputs[j].Name)
	})
	handlers.OK(c, allOutputs)
}

// TODO: uncomment when metrics api is stable
// func (h *LogOperatorHandler) Metrics(c *gin.Context) {
// 	cluster := c.Param("cluster")
// 	namespace := c.Param("namespace")
// 	flowid := c.Param("flowid")
// 	flow, err := kubeclient.GetClient().GetFlowByName(cluster, namespace, flowid, nil)
// 	if err != nil {
// 		handlers.NotOK(c, err)
// 		return
// 	}
// 	buildMetricTable := func(flow *loggingv1beta1.Flow, cluster, namespace string) (*models.MetricTable, error) {
// 		table := &models.MetricTable{}
// 		pods, err := kubeclient.GetClient().GetPodsOfNamespaces(cluster, namespace, nil)
// 		if err != nil {
// 			return nil, err
// 		}
// 		appSet := make(map[string][]corev1.Pod, 10)
// 		for _, p := range pods {
// 			if v, ok := p.Labels[gemlabels.LabelApplication]; ok {
// 				appSet[v] = append(appSet[v], p)
// 			}
// 		}
// 		if len(appSet) <= 0 {
// 			return nil, fmt.Errorf("no metadata exist, so skip metrics")
// 		}

// 		ret := make([]promemodel.Vector, 3)
// 		interval := []string{"1m", "1h", "24h"}
// 		var wg sync.WaitGroup
// 		for i := 0; i < 3; i++ {
// 			wg.Add(1)
// 			go func(index int) error {
// 				defer wg.Done()
// 				values := url.Values{}
// 				promql := fmt.Sprintf("irate(logging_entry_count{namespace=\"%s\"}[%s])", namespace, interval[index])
// 				values.Add("query", promql)
// 				u := "/custom/prometheus/v1/vector?" + values.Encode()
// 				err = kubeclient.DoRequest(http.MethodGet, cluster, u, nil, &ret[index])
// 				if err != nil {
// 					return err
// 				}
// 				return nil
// 			}(i)

// 		}
// 		wg.Wait()

// 		appAggrValue := make(map[string]map[string]float64, 10)

// 		// appSet = map[string][]corev1.Pod{
// 		// 	"test-app": {
// 		// 		corev1.Pod{
// 		// 			ObjectMeta: metav1.ObjectMeta{
// 		// 				Name: "test_pod",
// 		// 			},
// 		// 		},
// 		// 	},
// 		// }
// 		// ret = []promemodel.Vector{
// 		// 	{
// 		// 		&promemodel.Sample{
// 		// 			Metric: promemodel.Metric{
// 		// 				"pod": "test_pod",
// 		// 			},
// 		// 			Value:     123.1,
// 		// 			Timestamp: 1234567,
// 		// 		},
// 		// 	},
// 		// 	{
// 		// 		&promemodel.Sample{
// 		// 			Metric: promemodel.Metric{
// 		// 				"pod": "test_pod",
// 		// 			},
// 		// 			Value:     124.1,
// 		// 			Timestamp: 1234567,
// 		// 		},
// 		// 	},
// 		// 	{
// 		// 		&promemodel.Sample{
// 		// 			Metric: promemodel.Metric{
// 		// 				"pod": "test_pod",
// 		// 			},
// 		// 			Value:     125.1,
// 		// 			Timestamp: 1234567,
// 		// 		},
// 		// 	},
// 		// }
// 		// aggr 1m
// 		for k, v := range appSet {
// 			for _, p := range v {
// 				for _, sample := range ret[0] {
// 					if sample.Metric[promemodel.LabelName("pod")] == promemodel.LabelValue(p.Name) {
// 						if aggr, exist := appAggrValue[k]; exist {
// 							aggr["min"] += float64(sample.Value)
// 						} else {
// 							appAggrValue[k] = make(map[string]float64, 3)
// 							appAggrValue[k]["min"] += float64(sample.Value)
// 						}

// 					}
// 				}
// 				for _, sample := range ret[1] {
// 					if sample.Metric[promemodel.LabelName("pod")] == promemodel.LabelValue(p.Name) {
// 						if aggr, exist := appAggrValue[k]; exist {
// 							aggr["hour"] += float64(sample.Value)
// 						} else {
// 							appAggrValue[k] = make(map[string]float64, 3)
// 							appAggrValue[k]["hour"] += float64(sample.Value)
// 						}

// 					}
// 				}
// 				for _, sample := range ret[2] {
// 					if sample.Metric[promemodel.LabelName("pod")] == promemodel.LabelValue(p.Name) {
// 						if aggr, exist := appAggrValue[k]; exist {
// 							aggr["day"] += float64(sample.Value)
// 						} else {
// 							appAggrValue[k] = make(map[string]float64, 3)
// 							appAggrValue[k]["day"] += float64(sample.Value)
// 						}

// 					}
// 				}
// 			}
// 		}
// 		for k, v := range appAggrValue {
// 			table.Rows = append(table.Rows, models.MetricRow{
// 				AppName: k,
// 				RealTimeRate: func() float64 {
// 					if _, ok := v["min"]; ok {
// 						return v["min"]
// 					}
// 					return 0.0
// 				}(),
// 				AvgOfHour: func() float64 {
// 					if _, ok := v["hour"]; ok {
// 						return v["hour"]
// 					}
// 					return 0.0
// 				}(),
// 				AvgOfDay: func() float64 {
// 					if _, ok := v["day"]; ok {
// 						return v["day"]
// 					}
// 					return 0.0
// 				}(),
// 			})
// 		}
// 		return table, nil

// 	}
// 	table, err := buildMetricTable(flow, cluster, namespace)
// 	if err != nil {
// 		handlers.NotOK(c, err)
// 		return
// 	}
// 	handlers.OK(c, table)
// }

// func (h *LogOperatorHandler) Graph(c *gin.Context) {
// 	cluster := c.Param("cluster")
// 	namespace := c.Param("namespace")
// 	flowid := c.Param("flowid")
// 	flow, err := kubeclient.GetClient().GetFlowByName(cluster, namespace, flowid, nil)
// 	if err != nil {
// 		handlers.NotOK(c, err)
// 		return
// 	}
// 	graph := buildFlowGraph(flow)
// 	handlers.OK(c, graph)
// }

// const (
// 	virtualbox         = "filterbox"
// 	matchprefix        = "matcher-"
// 	filterprefix       = "filter-"
// 	outputprefix       = "output-"
// 	globaloutputprefix = "globaloutput-"
// 	inprefix           = "input-"
// 	outprefix          = "output-"
// 	clusteroutprefix   = "clusteroutput-"
// )

// func buildFlowGraph(flow *loggingv1beta1.Flow) *models.Graph {
// 	var nodes []*models.NodeWrapper
// 	var edges []*models.EdgeWrapper
// 	nodes = append(nodes, &models.NodeWrapper{
// 		Data: &models.NodeData{
// 			ID: virtualbox,
// 		},
// 	})
// 	for id, _ := range flow.Spec.Match {
// 		nd := &models.NodeData{
// 			ID:       fmt.Sprintf("%s%d", matchprefix, id),
// 			NodeType: string(models.Matcher),
// 		}
// 		nw := &models.NodeWrapper{
// 			Data: nd,
// 		}
// 		nodes = append(nodes, nw)
// 	}
// 	for id, _ := range flow.Spec.Filters {
// 		nd := &models.NodeData{
// 			ID:       fmt.Sprintf("%s%d", filterprefix, id),
// 			Parent:   virtualbox,
// 			NodeType: string(models.Filter),
// 		}
// 		nw := &models.NodeWrapper{
// 			Data: nd,
// 		}
// 		nodes = append(nodes, nw)
// 	}

// 	for id, _ := range flow.Spec.LocalOutputRefs {
// 		nd := &models.NodeData{
// 			ID:       fmt.Sprintf("%s%d", outputprefix, id),
// 			NodeType: string(models.Output),
// 		}
// 		nw := &models.NodeWrapper{
// 			Data: nd,
// 		}
// 		nodes = append(nodes, nw)
// 	}
// 	for id, _ := range flow.Spec.GlobalOutputRefs {
// 		nd := &models.NodeData{
// 			ID:       fmt.Sprintf("%s%d", globaloutputprefix, id),
// 			NodeType: string(models.GlobalOutput),
// 		}
// 		nw := &models.NodeWrapper{
// 			Data: nd,
// 		}
// 		nodes = append(nodes, nw)
// 	}
// 	for id, _ := range flow.Spec.Match {
// 		ed := &models.EdgeData{
// 			ID:     fmt.Sprintf("%s%d", inprefix, id),
// 			Source: fmt.Sprintf("%s%d", matchprefix, id),
// 			Target: virtualbox,
// 		}
// 		ew := &models.EdgeWrapper{
// 			Data: ed,
// 		}
// 		edges = append(edges, ew)
// 	}
// 	for id, _ := range flow.Spec.LocalOutputRefs {
// 		ed := &models.EdgeData{
// 			ID:     fmt.Sprintf("%s%d", outprefix, id),
// 			Target: fmt.Sprintf("%s%d", outputprefix, id),
// 			Source: virtualbox,
// 		}
// 		ew := &models.EdgeWrapper{
// 			Data: ed,
// 		}
// 		edges = append(edges, ew)
// 	}
// 	for id, _ := range flow.Spec.GlobalOutputRefs {
// 		ed := &models.EdgeData{
// 			ID:     fmt.Sprintf("%s%d", clusteroutprefix, id),
// 			Target: fmt.Sprintf("%s%d", globaloutputprefix, id),
// 			Source: virtualbox,
// 		}
// 		ew := &models.EdgeWrapper{
// 			Data: ed,
// 		}
// 		edges = append(edges, ew)
// 	}
// 	graph := &models.Graph{
// 		Elements: models.Elements{
// 			Nodes: nodes,
// 			Edges: edges,
// 		},
// 	}
// 	return graph
// }
