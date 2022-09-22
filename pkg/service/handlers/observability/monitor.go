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

package observability

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-version"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type MonitorCollector struct {
	Service string `json:"service"` // 服务名
	Port    string `json:"port"`    // 端口名
	Path    string `json:"path"`    // 采集路径
}

// GetMonitorCollector 监控采集器详情
// @Tags        Observability
// @Summary     监控采集器详情
// @Description 监控采集器详情
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                            true "cluster"
// @Param       namespace path     string                                            true "namespace"
// @Param       service   query    string                                            true "服务名"
// @Success     200       {object} handlers.ResponseStruct{Data=MonitorCollector} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor [get]
// @Security    JWT
func (h *ObservabilityHandler) GetMonitorCollector(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	svcname := c.Query("service")

	ret := MonitorCollector{
		Service: svcname,
	}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		sm := v1.ServiceMonitor{}
		if err := cli.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      svcname,
		}, &sm); err != nil {
			return err
		}
		svc := corev1.Service{}
		if err := cli.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      svcname,
		}, &svc); err != nil {
			return err
		}

		if !labels.SelectorFromSet(sm.Spec.Selector.MatchLabels).Matches(labels.Set(svc.Labels)) {
			log.Warnf("selector on servicemonitor %s doesn't match service labels", svcname)
			return nil
		}

		if len(sm.Spec.Endpoints) == 0 {
			log.Warnf("endpoint on servicemonitor %s is null", svcname)
			return nil
		}
		ret.Port = sm.Spec.Endpoints[0].Port
		ret.Path = sm.Spec.Endpoints[0].Path
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// MonitorCollectorStatus 监控采集器状态
// @Tags        Observability
// @Summary     监控采集器状态
// @Description 监控采集器状态
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                         true "cluster"
// @Param       namespace path     string                                         true "namespace"
// @Param       service   query    string                                         true "服务名"
// @Success     200       {object} handlers.ResponseStruct{Data=promv1.ActiveTarget} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/status [get]
// @Security    JWT
func (h *ObservabilityHandler) MonitorCollectorStatus(c *gin.Context) {
	scrapTarget := fmt.Sprintf("serviceMonitor/%s/%s/0", c.Param("namespace"), c.Query("service"))
	var ret promv1.ActiveTarget
	if err := h.Execute(c.Request.Context(), c.Param("cluster"), func(ctx context.Context, cli agents.Client) error {
		targets, err := cli.Extend().PrometheusTargets(ctx)
		if err != nil {
			return err
		}
		for _, v := range targets.Active {
			if v.ScrapePool == scrapTarget {
				ret = v
				return nil
			}
		}
		return fmt.Errorf("scrap target %s not found", scrapTarget)
	}); err != nil {
		log.Error(err, "get scrap target status")
		ret.Health = promv1.HealthUnknown
		ret.LastError = err.Error()
		ret.ScrapePool = scrapTarget
	}

	handlers.OK(c, ret)
}

// AddOrUpdateMonitorCollector 添加/更新监控采集器
// @Tags        Observability
// @Summary     添加/更新监控采集器
// @Description 添加/更新监控采集器
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       form      body     MonitorCollector                     true "采集器内容"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor [post]
// @Security    JWT
func (h *ObservabilityHandler) AddOrUpdateMonitorCollector(c *gin.Context) {
	req := MonitorCollector{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	// 以url上的c为准
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	h.SetAuditData(c, "创建", "监控采集器", req.Service)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		svc := corev1.Service{}
		if err := cli.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      req.Service,
		}, &svc); err != nil {
			return err
		}

		found := false
		for _, v := range svc.Spec.Ports {
			if v.Name == req.Port {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("port %s not found in service %s", req.Port, svc.Name)
		}

		sm := v1.ServiceMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      req.Service,
			},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, cli, &sm, func() error {
			sm.Spec = v1.ServiceMonitorSpec{
				Selector: *metav1.SetAsLabelSelector(svc.Labels),
				NamespaceSelector: v1.NamespaceSelector{
					Any:        false,
					MatchNames: []string{namespace},
				},
				Endpoints: []v1.Endpoint{{
					Port:        req.Port,
					HonorLabels: true,
					Interval:    "30s",
					Path:        req.Path,
				}},
			}
			return nil
		})
		if err != nil {
			return err
		}

		if svc.Labels == nil {
			svc.Labels = make(map[string]string)
		}
		svc.Labels[gems.LabelMonitorCollector] = gems.StatusEnabled
		return cli.Update(ctx, &svc)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

// DeleteMonitorCollector 删除监控采集器
// @Tags        Observability
// @Summary     删除监控采集器
// @Description 删除监控采集器
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       service   query    string                               true "服务名"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor [delete]
// @Security    JWT
func (h *ObservabilityHandler) DeleteMonitorCollector(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	svcname := c.Query("service")

	h.SetAuditData(c, "删除", "监控采集器", svcname)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		svc := corev1.Service{}
		if err := cli.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      svcname,
		}, &svc); err != nil {
			return err
		}

		if err := cli.Delete(ctx, &v1.ServiceMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      svcname,
			},
		}); err != nil {
			return err
		}
		delete(svc.Labels, gems.LabelMonitorCollector)
		return cli.Update(ctx, &svc)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

// ListMonitorAlertRule 监控告警规则列表
// @Tags        Observability
// @Summary     监控告警规则列表
// @Description 监控告警规则列表
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                                      true "cluster"
// @Param       namespace path     string                                                      true "namespace"
// @Success     200       {object} handlers.ResponseStruct{Data=[]prometheus.MonitorAlertRule} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts [get]
// @Security    JWT
func (h *ObservabilityHandler) ListMonitorAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	ret := []prometheus.MonitorAlertRule{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		var err error
		ret, err = cli.Extend().ListMonitorAlertRules(ctx, namespace, false, h.GetDataBase().NewPromqlTplMapperFromDB().FindPromqlTpl)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// GetMonitorAlertRule 监控告警规则详情
// @Tags        Observability
// @Summary     监控告警规则详情
// @Description 监控告警规则详情
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                                    true "cluster"
// @Param       namespace path     string                                                    true "namespace"
// @Param       name      path     string                                                    true "name"
// @Success     200       {object} handlers.ResponseStruct{Data=prometheus.MonitorAlertRule} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts/{name} [get]
// @Security    JWT
func (h *ObservabilityHandler) GetMonitorAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")

	var alerts []prometheus.MonitorAlertRule
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		var err error
		alerts, err = cli.Extend().ListMonitorAlertRules(ctx, namespace, true, h.GetDataBase().NewPromqlTplMapperFromDB().FindPromqlTpl)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	index := -1
	for i := range alerts {
		if alerts[i].Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		handlers.NotOK(c, fmt.Errorf("alert %s not found", name))
	}
	handlers.OK(c, alerts[index])
}

func (h *ObservabilityHandler) withAlertruleReq(c *gin.Context, f func(req prometheus.MonitorAlertRule) error) error {
	req := prometheus.MonitorAlertRule{}
	if err := c.BindJSON(&req); err != nil {
		return err
	}
	req.Namespace = c.Param("namespace")

	if err := prometheus.MutateMonitorAlert(&req, h.GetDataBase().FindPromqlTpl); err != nil {
		return err
	}

	if err := req.BaseAlertRule.CheckAndModify(); err != nil {
		return err
	}
	return f(req)
}

// CreateMonitorAlertRule 创建监控告警规则
// @Tags        Observability
// @Summary     创建监控告警规则
// @Description 创建监控告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       form      body     prometheus.MonitorAlertRule          true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts [post]
// @Security    JWT
func (h *ObservabilityHandler) CreateMonitorAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.withAlertruleReq(c, func(req prometheus.MonitorAlertRule) error {
		h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
		h.SetAuditData(c, "创建", "监控告警规则", req.Name)

		return h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
			// get、update、commit
			raw, err := cli.Extend().GetRawMonitorAlertResource(ctx, namespace, req.Source, h.GetDataBase().NewPromqlTplMapperFromDB().FindPromqlTpl)
			if err != nil {
				return err
			}

			// check name duplicated
			amconfigList := v1alpha1.AlertmanagerConfigList{}
			if err := cli.List(ctx, &amconfigList, client.InNamespace(namespace), client.HasLabels([]string{
				gems.LabelAlertmanagerConfigType,
			})); err != nil {
				return err
			}
			if err := checkAlertName(req.Name, amconfigList.Items); err != nil {
				return err
			}

			if err := raw.ModifyAlertRule(req, prometheus.Add); err != nil {
				return err
			}

			return cli.Extend().CommitRawMonitorAlertResource(ctx, raw)
		})
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

func checkAlertName(name string, amconfigs []*v1alpha1.AlertmanagerConfig) error {
	for _, v := range amconfigs {
		routes, err := v.Spec.Route.ChildRoutes()
		if err != nil {
			return err
		}
		for _, v := range routes {
			for _, m := range v.Matchers {
				if m.Name == prometheus.AlertNameLabel && m.Value == name {
					return fmt.Errorf("duplicated name in: %s", name)
				}
			}
		}

	}
	return nil
}

// UpdateMonitorAlertRule 修改监控告警规则
// @Tags        Observability
// @Summary     修改监控告警规则
// @Description 修改监控告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       name      path     string                               true "name"
// @Param       form      body     prometheus.MonitorAlertRule          true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts/{name} [put]
// @Security    JWT
func (h *ObservabilityHandler) UpdateMonitorAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	h.m.Lock()
	defer h.m.Unlock()
	if err := h.withAlertruleReq(c, func(req prometheus.MonitorAlertRule) error {
		h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
		h.SetAuditData(c, "更新", "监控告警规则", req.Name)

		return h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
			// get、update、commit
			raw, err := cli.Extend().GetRawMonitorAlertResource(ctx, namespace, req.Source, h.GetDataBase().NewPromqlTplMapperFromDB().FindPromqlTpl)
			if err != nil {
				return err
			}

			if err := raw.ModifyAlertRule(req, prometheus.Update); err != nil {
				return err
			}

			return cli.Extend().CommitRawMonitorAlertResource(ctx, raw)
		})
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

// DeleteMonitorAlertRule 删除AlertRule
// @Tags        Observability
// @Summary     修改监控告警规则
// @Description 修改监控告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       name      path     string                               true "name"
// @Param       source    query    string                               true "source"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts/{name} [delete]
// @Security    JWT
func (h *ObservabilityHandler) DeleteMonitorAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	source := c.Query("source")
	req := prometheus.MonitorAlertRule{
		BaseAlertRule: prometheus.BaseAlertRule{
			Namespace: namespace,
			Name:      name,
		},
	}
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	h.SetAuditData(c, "删除", "监控告警规则", req.Name)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		// get、update、commit
		raw, err := cli.Extend().GetRawMonitorAlertResource(ctx, namespace, source, h.GetDataBase().NewPromqlTplMapperFromDB().FindPromqlTpl)
		if err != nil {
			return err
		}

		if err := raw.ModifyAlertRule(req, prometheus.Delete); err != nil {
			return err
		}

		if err := cli.Extend().CommitRawMonitorAlertResource(ctx, raw); err != nil {
			return err
		}

		// 清理silence规则
		return deleteSilenceIfExist(ctx, namespace, name, cli)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

const exporterRepo = "kubegems"

// ExporterSchema 获取exporter的schema
// @Tags        Observability
// @Summary     获取exporter的schema
// @Description 获取exporter的schema
// @Accept      json
// @Produce     json
// @Param       name path     string                               true "exporter app name"
// @Success     200  {object} handlers.ResponseStruct{Data=object} "resp"
// @Router      /v1/observability/monitor/exporters/{name}/schema [get]
// @Security    JWT
func (h *ObservabilityHandler) ExporterSchema(c *gin.Context) {
	name := c.Param("name")

	index, err := h.ChartmuseumClient.ListChartVersions(c.Request.Context(), exporterRepo, name)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	findMaxChartVersion := func() (string, error) {
		maxVersion, _ := version.NewVersion("0.0.0")
		for _, v := range *index {
			thisVersion, err := version.NewVersion(v.Version)
			if err != nil {
				return "", err
			}
			if thisVersion.GreaterThan(maxVersion) {
				maxVersion = thisVersion
			}
		}
		return maxVersion.Original(), nil
	}

	maxVersion, err := findMaxChartVersion()
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	chartfiles, err := h.ChartmuseumClient.GetChartBufferedFiles(c.Request.Context(), exporterRepo, name, maxVersion)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	var schema, values string
	for _, v := range chartfiles {
		if v.Name == "values.schema.json" {
			schema = base64.StdEncoding.EncodeToString(v.Data)
		} else if v.Name == "values.yaml" {
			values = base64.StdEncoding.EncodeToString(v.Data)
		}
	}

	handlers.OK(c, gin.H{
		"values.schema.json": schema,
		"values.yaml":        values,
		"app":                name,
		"version":            maxVersion,
		"repo":               strings.TrimSuffix(h.AppStoreOpt.Addr, "/") + "/" + exporterRepo,
	})
}
