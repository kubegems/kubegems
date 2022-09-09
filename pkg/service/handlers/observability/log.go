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
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	v1beta1 "github.com/banzaicloud/logging-operator/pkg/sdk/logging/api/v1beta1"
	"github.com/banzaicloud/logging-operator/pkg/sdk/logging/model/filter"
	"github.com/gin-gonic/gin"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/slice"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultGlobalOutput = "kubegems-container-console-output"
)

var (
	prometheusFilter = func(flow string) *filter.PrometheusConfig {
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
	geoIPFilter = func(keys string) *filter.GeoIP {
		return &filter.GeoIP{
			GeoipLookupKeys: keys,
			Records: []filter.Record{
				{
					"city":         fmt.Sprintf(`${city.names.en["%s"]}`, keys),
					"latitude":     fmt.Sprintf(`${location.latitude["%s"]}`, keys),
					"longitude":    fmt.Sprintf(`${location.longitude["%s"]}`, keys),
					"country":      fmt.Sprintf(`${country.iso_code["%s"]}`, keys),
					"country_name": fmt.Sprintf(`${country.names.en["%s"]}`, keys),
					"postal_code":  fmt.Sprintf(`${postal.code["%s"]}`, keys),
				},
			},
		}
	}
	throttleRecordModifierFilter = &filter.RecordModifier{
		Records: []filter.Record{
			{
				"throttle_group_key": "${record['kubernetes']['namespace_name']+record['kubernetes']['pod_name']}",
			},
		},
	}
	throttleFilter = func(limit int) *filter.Throttle {
		return &filter.Throttle{
			GroupKey:                 "throttle_group_key",
			GroupBucketLimit:         limit,
			GroupBucketPeriodSeconds: 60,
		}
	}
)

// NamespaceLogCollector namespace级日志采集器
// @Tags        Observability
// @Summary     namespace级日志采集器
// @Description namespace级日志采集器
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       enable    query    bool                                 true "是否启用日志采集"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/logging [put]
// @Security    JWT
func (h *ObservabilityHandler) NamespaceLogCollector(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	enable, _ := strconv.ParseBool(c.Query("enable"))

	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		defaultFlow := v1beta1.Flow{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "default",
			},
		}
		if enable {
			_, err := controllerutil.CreateOrUpdate(ctx, cli, &defaultFlow, func() error {
				defaultFlow.Spec = v1beta1.FlowSpec{
					Filters: []v1beta1.Filter{
						{
							Prometheus: prometheusFilter(defaultFlow.Name),
						},
					},
					GlobalOutputRefs: []string{defaultGlobalOutput},
				}
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			if err := cli.Delete(ctx, &defaultFlow); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

var applables = []string{
	"app",                     // istio app label
	gems.LabelApplication,     // kubegems app label
	"apps.kubernetes.io/name", // k8s app label
}

type AppInfo struct {
	AppLabel    string `json:"appLabel"`
	CollectedBy string `json:"collectedBy"` // 由哪个flow采集的日志
}

// ListLogApps 获取支持日志采集的应用及标签
// @Tags        Observability
// @Summary     获取支持日志采集的应用及标签
// @Description 获取支持日志采集的应用及标签
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                           true "cluster"
// @Param       namespace path     string                                           true "namespace"
// @Success     200       {object} handlers.ResponseStruct{Data=map[string]AppInfo} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/logging/apps [get]
// @Security    JWT
func (h *ObservabilityHandler) ListLogApps(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	podList := corev1.PodList{}
	flowList := v1beta1.FlowList{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		if err := cli.List(ctx, &podList, client.InNamespace(namespace)); err != nil {
			return err
		}
		return cli.List(ctx, &flowList, client.InNamespace(namespace))
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	ret := getAppsLogStatus(podList, flowList)
	handlers.OK(c, ret)
}

type LogCollector struct {
	Apps           map[string]string `json:"apps"` // 要采集的应用, appname-applabel key-value
	Outputs        []string          `json:"outputs"`
	ClusterOutputs []string          `json:"clusterOutputs"`
	EnableMetrics  bool              `json:"enableMetrics"` // 是否启用日志采集监控
	PluginConfig   `json:"pluginConfig"`
}

type PluginConfig struct {
	Throttle        int      `json:"throttle"`        // 日志条目限速, 条/10s
	GeoIPLookupKeys []string `json:"geoIPLookupKeys"` // GeoIP keys
}

// AddAppLogCollector 应用级日志采集器
// @Tags        Observability
// @Summary     应用级日志采集器
// @Description 应用级日志采集器
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       form      body     LogCollector                         true "采集器内容"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/logging/apps [post]
// @Security    JWT
func (h *ObservabilityHandler) AddAppLogCollector(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	req := LogCollector{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}

	rand.Seed(time.Now().UnixNano())
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		defaultFlow := v1beta1.Flow{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      fmt.Sprintf("appflow-%s", string(utils.RandomRune(4, utils.RuneKindLower))),
			},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, cli, &defaultFlow, func() error {
			defaultFlow.Spec.Filters = nil
			defaultFlow.Spec = v1beta1.FlowSpec{
				LocalOutputRefs:  req.Outputs,
				GlobalOutputRefs: req.ClusterOutputs,
			}
			if req.EnableMetrics {
				defaultFlow.Spec.Filters = append(defaultFlow.Spec.Filters, v1beta1.Filter{
					Prometheus: prometheusFilter(defaultFlow.Name),
				})
			}
			if len(req.PluginConfig.GeoIPLookupKeys) > 0 {
				defaultFlow.Spec.Filters = append(defaultFlow.Spec.Filters, v1beta1.Filter{
					GeoIP: geoIPFilter(strings.Join(req.GeoIPLookupKeys, ", ")),
				})
			}
			if req.PluginConfig.Throttle > 0 {
				defaultFlow.Spec.Filters = append(defaultFlow.Spec.Filters, v1beta1.Filter{
					RecordModifier: throttleRecordModifierFilter,
				})
				defaultFlow.Spec.Filters = append(defaultFlow.Spec.Filters, v1beta1.Filter{
					Throttle: throttleFilter(req.Throttle),
				})
			}
			if len(req.Apps) == 0 {
				return i18n.Errorf(c, "can't add log collector, must specify at least one app")
			}

			podList := corev1.PodList{}
			flowList := v1beta1.FlowList{}
			if err := cli.List(ctx, &podList, client.InNamespace(namespace)); err != nil {
				return err
			}
			if err := cli.List(ctx, &flowList, client.InNamespace(namespace)); err != nil {
				return err
			}
			logstatus := getAppsLogStatus(podList, flowList)

			appnames := []string{}
			for appname, applabel := range req.Apps {
				if !slice.ContainStr(applables, applabel) {
					return i18n.Errorf(c, "app label %s is not valid, must be one of %v", applabel, applables)
				}
				if status, ok := logstatus[appname]; ok {
					if status.CollectedBy != "" {
						return i18n.Errorf(c, "app %s has been collected by flow %s", appname, status.CollectedBy)
					}
				}
				defaultFlow.Spec.Match = append(defaultFlow.Spec.Match, v1beta1.Match{
					Select: &v1beta1.Select{
						Labels: map[string]string{
							applabel: appname,
						},
					},
				})
				appnames = append(appnames, appname)
			}
			if defaultFlow.Labels == nil {
				defaultFlow.Labels = make(map[string]string)
			}
			defaultFlow.Labels[gems.LabelLogCollector] = strings.Join(appnames, ", ")
			return nil
		})
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

func getAppsLogStatus(podList corev1.PodList, flowList v1beta1.FlowList) map[string]*AppInfo {
	ret := map[string]*AppInfo{}
	for _, pod := range podList.Items {
		if pod.Labels != nil {
			for _, applabel := range applables {
				if appname, ok := pod.Labels[applabel]; ok {
					ret[appname] = &AppInfo{AppLabel: applabel}
					break
				}
			}
		}
	}
	for _, flow := range flowList.Items {
		for _, selector := range flow.Spec.Match {
			for _, appname := range selector.Select.Labels {
				if appinfo, ok := ret[appname]; ok {
					appinfo.CollectedBy = flow.Name
				}
			}
		}
	}
	return ret
}

// ListLoggingAlertRule 日志告警规则列表
// @Tags        Observability
// @Summary     日志告警规则列表
// @Description 日志告警规则列表
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                                      true "cluster"
// @Param       namespace path     string                                                      true "namespace"
// @Success     200       {object} handlers.ResponseStruct{Data=[]prometheus.LoggingAlertRule} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/logging/alerts [get]
// @Security    JWT
func (h *ObservabilityHandler) ListLoggingAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	ret := []prometheus.LoggingAlertRule{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		var err error
		ret, err = cli.Extend().ListLoggingAlertRules(ctx, namespace, false)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// GetLoggingAlertRule 日志告警规则详情
// @Tags        Observability
// @Summary     日志告警规则详情
// @Description 日志告警规则详情
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                                    true "cluster"
// @Param       namespace path     string                                                    true "namespace"
// @Param       name      path     string                                                    true "name"
// @Success     200       {object} handlers.ResponseStruct{Data=prometheus.LoggingAlertRule} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/logging/alerts/{name} [get]
// @Security    JWT
func (h *ObservabilityHandler) GetLoggingAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")

	alertrules := []prometheus.LoggingAlertRule{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		var err error
		alertrules, err = cli.Extend().ListLoggingAlertRules(ctx, namespace, true)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	index := -1
	for i := range alertrules {
		if alertrules[i].Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		handlers.NotOK(c, i18n.Errorf(c, "alert rule %s not found", name))
	}
	handlers.OK(c, alertrules[index])
}

// CreateLoggingAlertRule 创建日志告警规则
// @Tags        Observability
// @Summary     创建日志告警规则
// @Description 创建日志告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       form      body     prometheus.LoggingAlertRule          true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/logging/alerts [post]
// @Security    JWT
func (h *ObservabilityHandler) CreateLoggingAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	req := prometheus.LoggingAlertRule{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	req.Namespace = namespace
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	action := i18n.Sprintf(context.TODO(), "create")
	module := i18n.Sprintf(context.TODO(), "log alert rule")
	h.SetAuditData(c, action, module, req.Name)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		if err := req.CheckAndModify(); err != nil {
			return err
		}
		raw, err := cli.Extend().GetRawLoggingAlertResource(ctx, namespace)
		if err != nil {
			return err
		}

		// check name duplicated in log alert
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

		if err := raw.ModifyLoggingAlertRule(req, prometheus.Add); err != nil {
			return err
		}
		return cli.Extend().CommitRawLoggingAlertResource(ctx, raw)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// CreateLoggingAlertRule 更新日志告警规则
// @Tags        Observability
// @Summary     更新日志告警规则
// @Description 更新日志告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       form      body     prometheus.LoggingAlertRule          true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/logging/alerts/{name} [put]
// @Security    JWT
func (h *ObservabilityHandler) UpdateLoggingAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	req := prometheus.LoggingAlertRule{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	req.Namespace = namespace
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	action := i18n.Sprintf(context.TODO(), "update")
	module := i18n.Sprintf(context.TODO(), "log alert rule")
	h.SetAuditData(c, action, module, req.Name)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		if err := req.CheckAndModify(); err != nil {
			return err
		}
		raw, err := cli.Extend().GetRawLoggingAlertResource(ctx, namespace)
		if err != nil {
			return err
		}
		if err := raw.ModifyLoggingAlertRule(req, prometheus.Update); err != nil {
			return err
		}
		return cli.Extend().CommitRawLoggingAlertResource(ctx, raw)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteLoggingAlertRule 删除日志告警规则
// @Tags        Observability
// @Summary     删除日志告警规则
// @Description 删除日志告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       name      path     string                               true "name"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/logging/alerts/{name} [delete]
// @Security    JWT
func (h *ObservabilityHandler) DeleteLoggingAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")

	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "log alert rule")
	h.SetAuditData(c, action, module, name)
	req := prometheus.LoggingAlertRule{
		BaseAlertRule: prometheus.BaseAlertRule{
			Namespace: namespace,
			Name:      name,
		},
	}
	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		raw, err := cli.Extend().GetRawLoggingAlertResource(ctx, namespace)
		if err != nil {
			return err
		}
		if err := raw.ModifyLoggingAlertRule(req, prometheus.Delete); err != nil {
			return err
		}
		if err := cli.Extend().CommitRawLoggingAlertResource(ctx, raw); err != nil {
			return err
		}
		return deleteSilenceIfExist(ctx, namespace, name, cli)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}
