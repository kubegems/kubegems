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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/observe"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/channels"
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
		sm := monitoringv1.ServiceMonitor{}
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
		return i18n.Errorf(c, "scrap target %s not found", scrapTarget)
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

	action := i18n.Sprintf(context.TODO(), "create")
	module := i18n.Sprintf(context.TODO(), "monitoring collector")
	h.SetAuditData(c, action, module, req.Service)
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
			return i18n.Errorf(c, "port %s not found in Service %s", req.Port, svc.Name)
		}

		sm := monitoringv1.ServiceMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      req.Service,
			},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, cli, &sm, func() error {
			sm.Spec = monitoringv1.ServiceMonitorSpec{
				Selector: *metav1.SetAsLabelSelector(svc.Labels),
				NamespaceSelector: monitoringv1.NamespaceSelector{
					Any:        false,
					MatchNames: []string{namespace},
				},
				Endpoints: []monitoringv1.Endpoint{{
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

	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "monitoring collector")
	h.SetAuditData(c, action, module, svcname)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		svc := corev1.Service{}
		if err := cli.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      svcname,
		}, &svc); err != nil {
			return err
		}

		if err := cli.Delete(ctx, &monitoringv1.ServiceMonitor{
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
// @Param       cluster   path     string                                                                   true  "cluster"
// @Param       namespace path     string                                                                   true  "namespace"
// @Param       preload   query    string                                                                   false "choices (Receivers, Receivers.AlertChannel)"
// @Param       search    query    string                                                                   false "search in (name, expr)"
// @Param       state     query    string                                                                   false "告警状态筛选(inactive, pending, firing)"
// @Param       page      query    int                                                                      false "page"
// @Param       size      query    int                                                                      false "size"
// @Success     200       {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.AlertRule}} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts [get]
// @Security    JWT
func (h *ObservabilityHandler) ListMonitorAlertRule(c *gin.Context) {
	ret, err := h.listAlertRules(c, prometheus.AlertTypeMonitor)
	if err != nil {
		handlers.NotOK(c, err)
	}
	handlers.OK(c, ret)
}

// GetMonitorAlertRule 监控告警规则详情
// @Tags        Observability
// @Summary     监控告警规则详情
// @Description 监控告警规则详情
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                         true "cluster"
// @Param       namespace path     string                                         true "namespace"
// @Param       name      path     string                                         true "name"
// @Success     200       {object} handlers.ResponseStruct{Data=models.AlertRule} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts/{name} [get]
// @Security    JWT
func (h *ObservabilityHandler) GetMonitorAlertRule(c *gin.Context) {
	ret, err := h.getAlertRule(c, prometheus.AlertTypeMonitor)
	if err != nil {
		handlers.NotOK(c, err)
	}
	handlers.OK(c, ret)
}

func (h *ObservabilityHandler) withMonitorAlertReq(c *gin.Context, f func(req observe.MonitorAlertRule) error) error {
	req := observe.MonitorAlertRule{}
	if err := c.BindJSON(&req); err != nil {
		return err
	}
	req.Namespace = c.Param("namespace")
	for _, v := range req.BaseAlertRule.Receivers {
		if err := h.GetDB().WithContext(c.Request.Context()).First(v.AlertChannel).Error; err != nil {
			return err
		}
	}

	if err := observe.MutateMonitorAlert(&req, h.GetDataBase().FindPromqlTpl); err != nil {
		return err
	}
	return f(req)
}

func syncEmailSecret(ctx context.Context, cli agents.Client, alertrule *models.AlertRule) error {
	emails := map[string]*channels.Email{}
	for _, rec := range alertrule.Receivers {
		switch v := rec.AlertChannel.ChannelConfig.ChannelIf.(type) {
		case *channels.Email:
			emails[rec.AlertChannel.ReceiverName()] = v
		}
	}
	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      channels.EmailSecretName,
			Namespace: alertrule.Namespace,
			Labels:    channels.EmailSecretLabel,
		},
		Type: v1.SecretTypeOpaque,
	}
	_, err := controllerutil.CreateOrUpdate(ctx, cli, sec, func() error {
		if sec.Data == nil {
			sec.Data = make(map[string][]byte)
		}
		for recName, v := range emails {
			sec.Data[channels.EmailSecretKey(recName, v.From)] = []byte(v.AuthPassword) // 不需要encode
		}
		return nil
	})
	return err
}

func syncPrometheusRule(ctx context.Context, cli agents.Client, alertrule *models.AlertRule) error {
	prule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: alertrule.Namespace,
			Name:      alertrule.Name,
			Labels: map[string]string{
				gems.LabelPrometheusRuleType: prometheus.AlertTypeMonitor,
				gems.LabelPrometheusRuleName: alertrule.Name,
			},
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, cli, prule, func() error {
		rg := monitoringv1.RuleGroup{Name: alertrule.Name}
		for _, level := range alertrule.AlertLevels {
			rule := monitoringv1.Rule{
				Alert: alertrule.Name,
				Expr:  intstr.FromString(fmt.Sprintf("%s%s%s", alertrule.Expr, level.CompareOp, level.CompareValue)),
				For:   alertrule.For,
				Labels: map[string]string{
					prometheus.AlertNamespaceLabel: alertrule.Namespace,
					prometheus.AlertNameLabel:      alertrule.Name,
					prometheus.SeverityLabel:       level.Severity,
				},
				Annotations: map[string]string{
					prometheus.MessageAnnotationsKey: alertrule.Message,
					prometheus.ValueAnnotationKey:    prometheus.ValueAnnotationExpr,
				},
			}
			rg.Rules = append(rg.Rules, rule)
		}
		prule.Spec.Groups = []monitoringv1.RuleGroup{rg}
		return nil
	})
	return err
}

func syncAlertmanagerConfig(ctx context.Context, cli agents.Client, alertrule *models.AlertRule) error {
	// alertmanagerconfig
	amcfg := &v1alpha1.AlertmanagerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: alertrule.Namespace,
			Name:      alertrule.Name,
			Labels: map[string]string{
				gems.LabelAlertmanagerConfigType: prometheus.AlertTypeMonitor,
				gems.LabelAlertmanagerConfigName: alertrule.Name,
			},
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, cli, amcfg, func() error {
		amcfg.Spec = v1alpha1.AlertmanagerConfigSpec{
			Route: &v1alpha1.Route{
				Receiver:      prometheus.NullReceiverName,
				GroupBy:       []string{prometheus.AlertNamespaceLabel, prometheus.AlertNameLabel},
				GroupWait:     "30s",
				GroupInterval: "30s",
				Routes:        []apiextensionsv1.JSON{},
			},
			Receivers: []v1alpha1.Receiver{
				prometheus.NullReceiver,
			},
			InhibitRules: []v1alpha1.InhibitRule{},
		}
		for _, rec := range alertrule.Receivers {
			rawRouteData, _ := json.Marshal(v1alpha1.Route{
				Receiver:       rec.AlertChannel.ReceiverName(),
				RepeatInterval: rec.Interval,
				Continue:       true,
				Matchers: []v1alpha1.Matcher{
					{
						Name:  prometheus.AlertNamespaceLabel,
						Value: alertrule.Namespace,
					},
					{
						Name:  prometheus.AlertNameLabel,
						Value: alertrule.Name,
					},
				},
			})
			// receiver
			amcfg.Spec.Receivers = append(amcfg.Spec.Receivers, rec.AlertChannel.ToAlertmanagerReceiver())
			// route
			amcfg.Spec.Route.Routes = append(amcfg.Spec.Route.Routes, apiextensionsv1.JSON{Raw: rawRouteData})
		}
		// inhibit label
		if len(alertrule.InhibitLabels) > 0 {
			amcfg.Spec.InhibitRules = append(amcfg.Spec.InhibitRules, v1alpha1.InhibitRule{
				SourceMatch: []v1alpha1.Matcher{
					{
						Name:  prometheus.AlertNamespaceLabel,
						Value: alertrule.Namespace,
					},
					{
						Name:  prometheus.AlertNameLabel,
						Value: alertrule.Name,
					},
					{
						Name:  prometheus.SeverityLabel,
						Value: prometheus.SeverityCritical,
						Regex: false,
					},
				},
				TargetMatch: []v1alpha1.Matcher{
					{
						Name:  prometheus.AlertNamespaceLabel,
						Value: alertrule.Namespace,
					},
					{
						Name:  prometheus.AlertNameLabel,
						Value: alertrule.Name,
					},
					{
						Name:  prometheus.SeverityLabel,
						Value: prometheus.SeverityError,
						Regex: false,
					},
				},
				Equal: append(alertrule.InhibitLabels, prometheus.AlertNamespaceLabel, prometheus.AlertNameLabel),
			})
		}
		return nil
	})
	return err
}

func (h *ObservabilityHandler) syncMonitorAlertRule(ctx context.Context, alertrule *models.AlertRule) error {
	cli, err := h.GetAgents().ClientOf(ctx, alertrule.Cluster)
	if err != nil {
		return err
	}

	if err := syncEmailSecret(ctx, cli, alertrule); err != nil {
		return errors.Wrap(err, "sync secret failed")
	}
	if err := syncPrometheusRule(ctx, cli, alertrule); err != nil {
		return errors.Wrap(err, "sync prometheusrule failed")
	}
	if err := syncAlertmanagerConfig(ctx, cli, alertrule); err != nil {
		return errors.Wrap(err, "sync alertmanagerconfig failed")
	}
	return nil
}

func (h *ObservabilityHandler) deleteMonitorAlertRule(ctx context.Context, alertrule *models.AlertRule) error {
	cli, err := h.GetAgents().ClientOf(ctx, alertrule.Cluster)
	if err != nil {
		return err
	}
	if err := cli.Delete(ctx, &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: alertrule.Namespace,
			Name:      alertrule.Name,
		},
	}); err != nil && !kerrors.IsNotFound(err) {
		return err
	}
	if err := cli.Delete(ctx, &v1alpha1.AlertmanagerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: alertrule.Namespace,
			Name:      alertrule.Name,
		},
	}); err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	return deleteSilenceIfExist(ctx, alertrule.Namespace, alertrule.Name, cli)
}

// CreateMonitorAlertRule 创建监控告警规则
// @Tags        Observability
// @Summary     创建监控告警规则
// @Description 创建监控告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       form      body     models.AlertRule                     true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts [post]
// @Security    JWT
func (h *ObservabilityHandler) CreateMonitorAlertRule(c *gin.Context) {
	req, err := h.getAlertRuleReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	ctx := c.Request.Context()
	h.SetExtraAuditDataByClusterNamespace(c, req.Cluster, req.Namespace)
	action := i18n.Sprintf(ctx, "create")
	module := i18n.Sprintf(ctx, "monitor alert rule")
	h.SetAuditData(c, action, module, req.Name)

	if err := h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		allRules := []models.AlertRule{}
		if err := tx.Find(&allRules, "cluster = ? and namespace = ? and name = ?", req.Cluster, req.Namespace, req.Name).Error; err != nil {
			return err
		}
		if len(allRules) > 0 {
			return errors.Errorf("alert rule %s is already exist", req.Name)
		}
		if err := tx.Create(req).Error; err != nil {
			return err
		}
		return h.syncMonitorAlertRule(ctx, req)
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
		for _, route := range routes {
			for _, m := range route.Matchers {
				if m.Name == prometheus.AlertNameLabel && m.Value == name {
					return i18n.Errorf(context.TODO(), "duplicated name in: %s", name)
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
// @Param       form      body     models.AlertRule                     true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts/{name} [put]
// @Security    JWT
func (h *ObservabilityHandler) UpdateMonitorAlertRule(c *gin.Context) {
	req, err := h.getAlertRuleReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	ctx := c.Request.Context()
	h.SetExtraAuditDataByClusterNamespace(c, req.Cluster, req.Namespace)
	action := i18n.Sprintf(ctx, "update")
	module := i18n.Sprintf(ctx, "monitor alert rule")
	h.SetAuditData(c, action, module, req.Name)

	if err := h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := updateReceiversInDB(req, tx); err != nil {
			return errors.Wrap(err, "update receivers")
		}
		if err := tx.Select("expr", "for", "message", "inhibit_labels", "alert_levels", "promql_generator").
			Updates(req).Error; err != nil {
			return err
		}
		return h.syncMonitorAlertRule(ctx, req)
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
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts/{name} [delete]
// @Security    JWT
func (h *ObservabilityHandler) DeleteMonitorAlertRule(c *gin.Context) {
	req := &models.AlertRule{
		Cluster:   c.Param("cluster"),
		Namespace: c.Param("namespace"),
		Name:      c.Param("name"),
	}

	ctx := c.Request.Context()
	h.SetExtraAuditDataByClusterNamespace(c, req.Cluster, req.Namespace)
	action := i18n.Sprintf(ctx, "delete")
	module := i18n.Sprintf(ctx, "monitor alert rule")
	h.SetAuditData(c, action, module, req.Name)

	if err := h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(req, "cluster = ? and namespace = ? and name = ?", req.Cluster, req.Namespace, req.Name).Error; err != nil {
			return err
		}
		if err := tx.Delete(req).Error; err != nil {
			return err
		}
		return h.deleteMonitorAlertRule(ctx, req)
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
