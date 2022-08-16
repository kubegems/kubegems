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

package agents

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus/alertmanager/pkg/labels"
	alertmanagertypes "github.com/prometheus/alertmanager/types"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/rulefmt"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/apis/plugins"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/loki"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	alertProxyHeader = map[string]string{
		"namespace": gems.NamespaceMonitor,
		"service":   "kube-prometheus-stack-alertmanager",
		"port":      "9093",
	}
	allNamespace = "_all"
)

type ExtendClient struct {
	*TypedClient
}

// plugins.kubegems.io/v1alpha1
// Depracated: use 'gemsplugin.ListPlugins' instead
func (c *ExtendClient) ListPlugins(ctx context.Context) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	err := c.DoRequest(ctx, Request{
		Method: http.MethodGet,
		Path:   "/custom/" + plugins.GroupName + "/v1beta1/installers",
		Into:   WrappedResponse(&ret),
	})
	return ret, err
}

// Depracated: use 'gemsplugin.EnablePlugin' instead
func (c *ExtendClient) EnablePlugin(ctx context.Context, ptype, name string) error {
	return c.DoRequest(ctx, Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/custom/%s/v1beta1/installers/%s/actions/enable?type=%s", plugins.GroupName, name, ptype),
	})
}

// Depracated: use 'gemsplugin.EnablePlugin' instead
func (c *ExtendClient) DisablePlugin(ctx context.Context, ptype, name string) error {
	return c.DoRequest(ctx, Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/custom/%s/v1beta1/installers/%s/actions/disable?type=%s", plugins.GroupName, name, ptype),
	})
}

// statistics.system/v1
func (c *ExtendClient) ClusterWorkloadStatistics(ctx context.Context, ret interface{}) error {
	return c.DoRequest(ctx, Request{
		Path: "/custom/statistics.system/v1/workloads",
		Into: WrappedResponse(ret),
	})
}

func (c *ExtendClient) ClusterResourceStatistics(ctx context.Context, ret interface{}) error {
	return c.DoRequest(ctx, Request{
		Path: "/custom/statistics.system/v1/resources",
		Into: WrappedResponse(ret),
	})
}

// health.system/v1
func (c *ExtendClient) Healthy(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.DoRequest(ctx, Request{Path: "/healthz"})
}

func (c *ExtendClient) ListSilences(ctx context.Context, labels map[string]string, commentPrefix string) ([]alertmanagertypes.Silence, error) {
	allSilences := []alertmanagertypes.Silence{}

	req := Request{
		Path: "/v1/service-proxy/api/v2/silences",
		Query: func() url.Values {
			values := url.Values{}
			for k, v := range labels {
				values.Add("filter", fmt.Sprintf(`%s="%s"`, k, v))
			}
			return values
		}(),
		Headers: HeadersFrom(alertProxyHeader),
		Into:    &allSilences,
	}

	if err := c.DoRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("list silence by %v, %w", labels, err)
	}

	// 只返回活跃的
	ret := []alertmanagertypes.Silence{}
	if commentPrefix == "" {
		for _, v := range allSilences {
			if v.Status.State == alertmanagertypes.SilenceStateActive {
				ret = append(ret, v)
			}
		}
	} else {
		for _, v := range allSilences {
			if v.Status.State == alertmanagertypes.SilenceStateActive &&
				strings.HasPrefix(v.Comment, commentPrefix) {
				ret = append(ret, v)
			}
		}
	}
	return ret, nil
}

// use for blacklist
func (c *ExtendClient) CreateOrUpdateSilenceIfNotExist(ctx context.Context, info models.AlertInfo) error {
	silenceList, err := c.ListSilences(ctx, info.LabelMap, prometheus.SilenceCommentForBlackListPrefix)
	if err != nil {
		return err
	}
	convertAlertInfoToSilence := func(info models.AlertInfo) alertmanagertypes.Silence {
		ret := alertmanagertypes.Silence{
			StartsAt:  *info.SilenceStartsAt,
			EndsAt:    *info.SilenceEndsAt,
			UpdatedAt: *info.SilenceUpdatedAt,
			CreatedBy: info.SilenceCreator,
			Comment:   fmt.Sprintf("%s%s", prometheus.SilenceCommentForBlackListPrefix, info.Fingerprint), // comment存指纹，以便取出时做匹配
			Matchers:  make(labels.Matchers, len(info.LabelMap)),
		}
		index := 0
		for k, v := range info.LabelMap {
			ret.Matchers[index] = &labels.Matcher{
				Type:  labels.MatchEqual,
				Name:  k,
				Value: v,
			}
			index++
		}
		return ret
	}

	silence := convertAlertInfoToSilence(info)
	switch len(silenceList) {
	case 0:
		break
	case 1:
		silence.ID = silenceList[0].ID
	default:
		return fmt.Errorf("too many silences for alert: %v", info)
	}

	agentreq := Request{
		Method:  http.MethodPost,
		Path:    "/v1/service-proxy/api/v2/silences",
		Body:    silence,
		Headers: HeadersFrom(alertProxyHeader),
	}

	if err := c.DoRequest(ctx, agentreq); err != nil {
		return fmt.Errorf("create silence:%w", err)
	}
	return nil
}

// use for blacklist
func (c *ExtendClient) DeleteSilenceIfExist(ctx context.Context, info models.AlertInfo) error {
	silenceList, err := c.ListSilences(ctx, info.LabelMap, prometheus.SilenceCommentForBlackListPrefix)
	if err != nil {
		return err
	}
	switch len(silenceList) {
	case 0:
		return nil
	case 1:
		agentreq := Request{
			Method:  http.MethodDelete,
			Path:    fmt.Sprintf("/v1/service-proxy/api/v2/silence/%s", silenceList[0].ID),
			Headers: HeadersFrom(alertProxyHeader),
		}
		return c.DoRequest(ctx, agentreq)
	default:
		return fmt.Errorf("too many silences for alert: %v", info)
	}
}

func (c *ExtendClient) CheckAlertmanagerConfig(ctx context.Context, data *monitoringv1alpha1.AlertmanagerConfig) error {
	return c.DoRequest(ctx, Request{
		Method: http.MethodPost,
		Path:   "/custom/alertmanager/v1/alerts/_/actions/check",
		Body:   data,
	})
}

func (c *ExtendClient) GetPromeAlertRules(ctx context.Context, name string) (map[string]prometheus.RealTimeAlertRule, error) {
	ret := map[string]prometheus.RealTimeAlertRule{}
	if err := c.DoRequest(ctx, Request{
		Path: fmt.Sprintf("/custom/prometheus/v1/alertrule?name=%s", name),
		Into: WrappedResponse(&ret),
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *ExtendClient) GetLokiAlertRules(ctx context.Context) (map[string]prometheus.RealTimeAlertRule, error) {
	ret := map[string]prometheus.RealTimeAlertRule{}
	if err := c.DoRequest(ctx, Request{
		Path: "/custom/loki/v1/alertrule",
		Into: WrappedResponse(&ret),
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *ExtendClient) GetPrometheusLabelNames(ctx context.Context, matchs []string, start, end string) ([]string, error) {
	resp := struct {
		Labels []string    `json:"labels,omitempty"`
		Warns  interface{} `json:"warns,omitempty"`
	}{}
	values := url.Values{}
	for _, v := range matchs {
		values.Add("match", v)
	}
	values.Add("start", start)
	values.Add("end", end)
	if err := c.DoRequest(ctx, Request{
		Path:  "/custom/prometheus/v1/labelnames",
		Query: values,
		Into:  WrappedResponse(&resp),
	}); err != nil {
		return nil, fmt.Errorf("prometheus label names failed, cluster: %s, matchs: %v, %v", c.Name, matchs, err)
	}

	return resp.Labels, nil
}

func (c *ExtendClient) GetPrometheusLabelValues(ctx context.Context, matchs []string, label, start, end string) ([]string, error) {
	resp := struct {
		Labels []string    `json:"labels,omitempty"`
		Warns  interface{} `json:"warns,omitempty"`
	}{}
	values := url.Values{}
	for _, v := range matchs {
		values.Add("match", v)
	}
	values.Add("label", label)
	values.Add("start", start)
	values.Add("end", end)
	if err := c.DoRequest(ctx, Request{
		Path:  "/custom/prometheus/v1/labelvalues",
		Query: values,
		Into:  WrappedResponse(&resp),
	}); err != nil {
		return nil, fmt.Errorf("prometheus label values failed, cluster: %s, matchs: %v, label: %s, %v", c.Name, matchs, label, err)
	}

	return resp.Labels, nil
}

func (c *ExtendClient) PrometheusQueryRange(ctx context.Context, query, start, end, step string) (prommodel.Matrix, error) {
	ret := prommodel.Matrix{}
	values := url.Values{}
	values.Add("query", query)
	values.Add("start", start)
	values.Add("end", end)
	values.Add("step", step)
	if err := c.DoRequest(ctx, Request{
		Path:  "/custom/prometheus/v1/matrix",
		Query: values,
		Into:  WrappedResponse(&ret),
	}); err != nil {
		return nil, fmt.Errorf("prometheus query range failed, cluster: %s, promql: %s, %v", c.Name, query, err)
	}

	for i := range ret {
		if len(ret[i].Metric) == 0 {
			ret[i].Metric = prommodel.Metric{
				"__name__": prommodel.LabelValue(query),
			}
		}
	}
	return ret, nil
}

func (c *ExtendClient) PrometheusVector(ctx context.Context, query string) (prommodel.Vector, error) {
	ret := prommodel.Vector{}
	values := url.Values{}
	values.Add("query", query)
	if err := c.DoRequest(ctx, Request{
		Path:  "/custom/prometheus/v1/vector",
		Query: values,
		Into:  WrappedResponse(&ret),
	}); err != nil {
		return nil, fmt.Errorf("prometheus vector failed, cluster: %s, promql: %s, %v", c.Name, query, err)
	}
	return ret, nil
}

func (c *ExtendClient) PrometheusTargets(ctx context.Context) (*promv1.TargetsResult, error) {
	ret := promv1.TargetsResult{}
	if err := c.DoRequest(ctx, Request{
		Path:  "/custom/prometheus/v1/targets",
		Query: nil,
		Into:  WrappedResponse(&ret),
	}); err != nil {
		return nil, errors.Wrapf(err, "get prometheus targets from cluster: %s", c.Name)
	}
	return &ret, nil
}

func (c *ExtendClient) ListMonitorAlertRules(ctx context.Context, namespace string, hasDetail bool) ([]prometheus.MonitorAlertRule, error) {
	if namespace == allNamespace {
		namespace = v1.NamespaceAll
	}

	promRuleMap := map[string]*monitoringv1.PrometheusRule{}
	amConfigMap := map[string]*monitoringv1alpha1.AlertmanagerConfig{}
	silenceNamespaceMap := map[string][]alertmanagertypes.Silence{}
	var realTimeAlertRules map[string]prometheus.RealTimeAlertRule

	eg := errgroup.Group{}
	eg.Go(func() error {
		promeRuleList := monitoringv1.PrometheusRuleList{}
		if err := c.List(ctx, &promeRuleList, client.InNamespace(namespace), client.MatchingLabels(map[string]string{
			gems.LabelPrometheusRuleType: prometheus.AlertTypeMonitor,
		})); err != nil {
			return err
		}
		for _, v := range promeRuleList.Items {
			promRuleMap[client.ObjectKeyFromObject(v).String()] = v
		}
		return nil
	})
	eg.Go(func() error {
		amConfigList := monitoringv1alpha1.AlertmanagerConfigList{}
		if err := c.List(ctx, &amConfigList, client.InNamespace(namespace), client.MatchingLabels(map[string]string{
			gems.LabelAlertmanagerConfigType: prometheus.AlertTypeMonitor,
		})); err != nil {
			return nil
		}
		for _, v := range amConfigList.Items {
			amConfigMap[client.ObjectKeyFromObject(v).String()] = v
		}
		return nil
	})
	eg.Go(func() error {
		allSilences, err := c.ListSilences(ctx, nil, prometheus.SilenceCommentForAlertrulePrefix)
		if err != nil {
			return err
		}
		// silence 按照namespace分组
		for _, silence := range allSilences {
			for _, m := range silence.Matchers {
				if m.Name == prometheus.AlertNamespaceLabel {
					silenceNamespaceMap[m.Value] = append(silenceNamespaceMap[m.Value], silence)
				}
			}
		}
		return nil
	})
	eg.Go(func() error {
		var err error
		realTimeAlertRules, err = c.GetPromeAlertRules(ctx, "")
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	ret := []prometheus.MonitorAlertRule{}
	for key, rule := range promRuleMap {
		amconfig, ok := amConfigMap[key]
		if !ok {
			log.Warnf("alertmanager config %s in namespace %s not found", rule.Name, rule.Namespace)
			continue
		}
		raw := &prometheus.RawMonitorAlertResource{
			Base: &prometheus.BaseAlertResource{
				AMConfig: amconfig,
				Silences: silenceNamespaceMap[rule.Namespace],
			},
			PrometheusRule: rule,
		}

		alerts, err := raw.ToAlerts(hasDetail)
		if err != nil {
			return nil, err
		}
		ret = append(ret, alerts...)
	}

	// realtime alert rules 按照namespace+name 分组
	for i := range ret {
		key := prometheus.RealTimeAlertKey(ret[i].Namespace, ret[i].Name)
		if promRule, ok := realTimeAlertRules[key]; ok {
			ret[i].State = promRule.State
			if hasDetail {
				tmp := realTimeAlertRules[key]
				sort.Sort(&tmp)
				ret[i].RealTimeAlerts = tmp.Alerts
			}
		} else {
			ret[i].State = "inactive"
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return strings.ToLower(ret[i].Name) < strings.ToLower(ret[j].Name)
	})

	return ret, nil
}

func (c *ExtendClient) ListLoggingAlertRules(ctx context.Context, namespace string, hasDetail bool) ([]prometheus.LoggingAlertRule, error) {
	if namespace == allNamespace {
		namespace = v1.NamespaceAll
	}
	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gems.NamespaceLogging,
			Name:      prometheus.LoggingAlertRuleCMName,
		},
	}
	amConfigList := monitoringv1alpha1.AlertmanagerConfigList{}
	configNamespaceMap := map[string]*monitoringv1alpha1.AlertmanagerConfig{}
	var allSilences []alertmanagertypes.Silence
	var realTimeAlertRules map[string]prometheus.RealTimeAlertRule

	eg := errgroup.Group{}
	eg.Go(func() error {
		return c.Get(ctx, client.ObjectKeyFromObject(&cm), &cm)
	})
	eg.Go(func() error {
		return c.List(ctx, &amConfigList, client.InNamespace(namespace), client.MatchingLabels(map[string]string{
			gems.LabelAlertmanagerConfigName: prometheus.LoggingAlertCRDName,
		}))
	})
	eg.Go(func() error {
		var err error
		allSilences, err = c.ListSilences(ctx, nil, prometheus.SilenceCommentForAlertrulePrefix)
		return err
	})
	eg.Go(func() error {
		var err error
		realTimeAlertRules, err = c.GetLokiAlertRules(ctx)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// 当前namespace或者所有namespace的map
	groupNamespaceMap := map[string]*rulefmt.RuleGroups{}
	for k, v := range cm.Data {
		// skip recording rule
		if k == prometheus.LokiRecordingRulesKey {
			continue
		}
		if namespace != v1.NamespaceAll && namespace != k {
			continue
		}
		groups := rulefmt.RuleGroups{}
		if err := yaml.Unmarshal([]byte(v), &groups); err != nil {
			return nil, errors.Wrapf(err, "decode log rulegroups")
		}
		groupNamespaceMap[k] = &groups
	}

	// amconfig 按照namespace分组
	for _, v := range amConfigList.Items {
		configNamespaceMap[v.Namespace] = v
	}

	// silence 按照namespace分组
	silenceNamespaceMap := map[string][]alertmanagertypes.Silence{}
	for _, silence := range allSilences {
		for _, m := range silence.Matchers {
			if m.Name == prometheus.AlertNamespaceLabel {
				silenceNamespaceMap[m.Value] = append(silenceNamespaceMap[m.Value], silence)
			}
		}
	}

	// realtime alert rules 按照namespace+name 分组
	ret := []prometheus.LoggingAlertRule{}
	for thisNamesapce, rulegroups := range groupNamespaceMap {
		amconfig, ok := configNamespaceMap[thisNamesapce]
		if !ok {
			log.Warnf("alertmanager in namespace %s not found", thisNamesapce)
			continue
		}
		raw := &prometheus.RawLoggingAlertRule{
			Base: &prometheus.BaseAlertResource{
				AMConfig: amconfig,
				Silences: silenceNamespaceMap[thisNamesapce],
			},
			ConfigMap:  &cm,
			RuleGroups: rulegroups,
		}

		alerts, err := raw.ToAlerts(hasDetail)
		if err != nil {
			return nil, err
		}
		ret = append(ret, alerts...)
	}

	for i := range ret {
		key := prometheus.RealTimeAlertKey(ret[i].Namespace, ret[i].Name)
		if promRule, ok := realTimeAlertRules[key]; ok {
			ret[i].State = promRule.State
			if hasDetail {
				tmp := realTimeAlertRules[key]
				sort.Sort(&tmp)
				ret[i].RealTimeAlerts = tmp.Alerts
			}
		} else {
			ret[i].State = "inactive"
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return strings.ToLower(ret[i].Name) < strings.ToLower(ret[j].Name)
	})

	return ret, nil
}

func (c *ExtendClient) getBaseAlertResource(ctx context.Context, namespace, amconfigName string) (*prometheus.BaseAlertResource, error) {
	loggingAMConfig, err := c.GetOrCreateAlertmanagerConfig(ctx, namespace, amconfigName)
	if err != nil {
		return nil, err
	}
	silence, err := c.ListSilences(ctx, map[string]string{
		prometheus.AlertNamespaceLabel: namespace,
	}, prometheus.SilenceCommentForAlertrulePrefix)
	if err != nil {
		return nil, err
	}
	return &prometheus.BaseAlertResource{
		AMConfig: loggingAMConfig,
		Silences: silence,
	}, nil
}

// GetRawMonitorAlertResource get specified namespace's alert
func (c *ExtendClient) GetRawMonitorAlertResource(ctx context.Context, namespace, name string) (*prometheus.RawMonitorAlertResource, error) {
	base, err := c.getBaseAlertResource(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	promerule, err := c.GetOrCreatePrometheusRule(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	return &prometheus.RawMonitorAlertResource{
		Base:           base,
		PrometheusRule: promerule,
	}, nil
}

func (c *ExtendClient) GetRawLoggingAlertResource(ctx context.Context, namespace string) (*prometheus.RawLoggingAlertRule, error) {
	base, err := c.getBaseAlertResource(ctx, namespace, prometheus.LoggingAlertCRDName)
	if err != nil {
		return nil, err
	}
	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gems.NamespaceLogging,
			Name:      prometheus.LoggingAlertRuleCMName,
		},
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(&cm), &cm); err != nil {
		return nil, err
	}

	raw := &prometheus.RawLoggingAlertRule{
		Base:       base,
		ConfigMap:  &cm,
		RuleGroups: &rulefmt.RuleGroups{},
	}
	groupstr, ok := cm.Data[namespace]
	if ok {
		groups := rulefmt.RuleGroups{}
		if err := yaml.Unmarshal([]byte(groupstr), &groups); err != nil {
			return nil, errors.Wrapf(err, "decode log rulegroups")
		}
		raw.RuleGroups = &groups
	}

	return raw, nil
}

func (c *ExtendClient) CommitRawLoggingAlertResource(ctx context.Context, raw *prometheus.RawLoggingAlertRule) error {
	bts, err := yaml.Marshal(raw.RuleGroups)
	if err != nil {
		return err
	}
	if raw.ConfigMap.Data == nil {
		raw.ConfigMap.Data = make(map[string]string)
	}
	raw.ConfigMap.Data[raw.Base.AMConfig.Namespace] = string(bts)
	if err := c.Update(ctx, raw.ConfigMap); err != nil {
		return err
	}
	return c.Update(ctx, raw.Base.AMConfig)
}

func (c *ExtendClient) GetOrCreateAlertmanagerConfig(ctx context.Context, namespace string, name string) (*monitoringv1alpha1.AlertmanagerConfig, error) {
	aconfig := &monitoringv1alpha1.AlertmanagerConfig{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, aconfig)
	if kerrors.IsNotFound(err) {
		// 初始化
		aconfig = prometheus.GetBaseAlertmanagerConfig(namespace, name)
		if err := c.CheckAlertmanagerConfig(ctx, aconfig); err != nil {
			return nil, err
		}

		if err := c.Create(ctx, aconfig); err != nil {
			return nil, err
		}
		return aconfig, nil
	}
	return aconfig, err
}

func (c *ExtendClient) GetOrCreatePrometheusRule(ctx context.Context, namespace, name string) (*monitoringv1.PrometheusRule, error) {
	prule := &monitoringv1.PrometheusRule{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, prule)
	if kerrors.IsNotFound(err) {
		prule = prometheus.GetBasePrometheusRule(namespace, name)
		if err := c.Create(ctx, prule); err != nil {
			return nil, err
		}
		return prule, nil
	}
	return prule, err
}

func (c *ExtendClient) CommitRawMonitorAlertResource(ctx context.Context, raw *prometheus.RawMonitorAlertResource) error {
	if err := c.CheckAlertmanagerConfig(ctx, raw.Base.AMConfig); err != nil {
		bts, _ := yaml.Marshal(raw.Base.AMConfig)
		log.Error(err, "amconfig", string(bts))
		return err
	}

	if err := c.Update(ctx, raw.PrometheusRule); err != nil {
		return err
	}
	return c.Update(ctx, raw.Base.AMConfig)
}

func (c *ExtendClient) LokiQuery(ctx context.Context, logql string) (loki.QueryResponseData, error) {
	ret := loki.QueryResponseData{}
	values := url.Values{}
	values.Add("query", logql)
	if err := c.DoRequest(ctx, Request{
		Path:  "/custom/loki/v1/query",
		Query: values,
		Into:  WrappedResponse(&ret),
	}); err != nil {
		return ret, fmt.Errorf("loki query failed, cluster: %s, logql: %s, %v", c.Name, logql, err)
	}
	return ret, nil
}

func (c *ExtendClient) CreateOrUpdateAlertEmailSecret(ctx context.Context, namespace string, rec *prometheus.ReceiverConfig) error {
	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheus.EmailSecretName,
			Namespace: namespace,
			Labels:    prometheus.EmailSecretLabel,
		},
		Type: v1.SecretTypeOpaque,
	}
	_, err := controllerutil.CreateOrUpdate(ctx, c.TypedClient, sec, func() error {
		if sec.Data == nil {
			sec.Data = make(map[string][]byte)
		}
		for _, v := range rec.EmailConfigs {
			sec.Data[prometheus.EmailSecretKey(rec.Name, v.From)] = []byte(v.AuthPassword) // 不需要encode
		}
		return nil
	})
	return err
}

func (c *ExtendClient) DeleteAlertEmailSecret(ctx context.Context, namespace string, rec monitoringv1alpha1.Receiver) error {
	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheus.EmailSecretName,
			Namespace: namespace,
		},
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(sec), sec); err != nil {
		return err
	}
	for _, v := range rec.EmailConfigs {
		delete(sec.Data, prometheus.EmailSecretKey(rec.Name, v.From))
	}
	return c.Update(ctx, sec)
}

func (c *ExtendClient) ListReceivers(ctx context.Context, namespace, scope, search string) ([]prometheus.ReceiverConfig, error) {
	if namespace == allNamespace {
		namespace = v1.NamespaceAll
	} else {
		var amname string
		if scope == prometheus.AlertTypeMonitor {
			amname = prometheus.MonitorAlertCRDName
		} else {
			amname = prometheus.LoggingAlertCRDName
		}
		_, err := c.GetOrCreateAlertmanagerConfig(ctx, namespace, amname)
		if err != nil {
			return nil, err
		}
	}

	configlist := &monitoringv1alpha1.AlertmanagerConfigList{}
	if err := c.List(ctx, configlist, client.MatchingLabels(map[string]string{
		gems.LabelAlertmanagerConfigType: scope,
	}), client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	emailsecretlist := &v1.SecretList{}
	if err := c.List(ctx, emailsecretlist, client.MatchingLabels(prometheus.EmailSecretLabel), client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	secretNamespaceMap := map[string]*v1.Secret{}
	for i, v := range emailsecretlist.Items {
		if v.Name == prometheus.EmailSecretName {
			secretNamespaceMap[v.Namespace] = &emailsecretlist.Items[i]
		}
	}

	ret := []prometheus.ReceiverConfig{}
	for _, config := range configlist.Items {
		for _, rec := range config.Spec.Receivers {
			if rec.Name != prometheus.NullReceiverName {
				// 隐藏介入中心创建的默认接收器
				if (config.Name != prometheus.MonitorAlertCRDName && config.Name != prometheus.LoggingAlertCRDName) &&
					rec.Name == prometheus.DefaultReceiverName {
					continue
				}
				if search == "" || (search != "" && strings.Contains(rec.Name, search)) {
					ret = append(ret, prometheus.ToGemsReceiver(rec, config.Namespace, config.Name, secretNamespaceMap[config.Namespace]))
				}
			}
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return strings.ToLower(ret[i].Name) < strings.ToLower(ret[j].Name)
	})
	return ret, nil
}

func (c *ExtendClient) CreateReceiver(ctx context.Context, rec prometheus.ReceiverConfig) error {
	if err := rec.Precheck(); err != nil {
		return err
	}
	aconfig, err := c.GetOrCreateAlertmanagerConfig(ctx, rec.Namespace, rec.Source)
	if err != nil {
		return err
	}
	if err := c.CreateOrUpdateAlertEmailSecret(ctx, rec.Namespace, &rec); err != nil {
		return err
	}

	receiver := prometheus.ToAlertmanagerReceiver(rec)
	if err := prometheus.ModifyReceiver(ctx, aconfig, &receiver, prometheus.Add); err != nil {
		return err
	}
	return c.Update(ctx, aconfig)
}

func (c *ExtendClient) UpdateReceiver(ctx context.Context, rec prometheus.ReceiverConfig) error {
	if err := rec.Precheck(); err != nil {
		return err
	}
	aconfig, err := c.GetOrCreateAlertmanagerConfig(ctx, rec.Namespace, rec.Source)
	if err != nil {
		return err
	}
	if err := c.CreateOrUpdateAlertEmailSecret(ctx, rec.Namespace, &rec); err != nil {
		return err
	}

	receiver := prometheus.ToAlertmanagerReceiver(rec)
	if err := prometheus.ModifyReceiver(ctx, aconfig, &receiver, prometheus.Update); err != nil {
		return err
	}
	return c.Update(ctx, aconfig)
}

func (c *ExtendClient) DeleteReceiver(ctx context.Context, rec prometheus.ReceiverConfig) error {
	if err := rec.Precheck(); err != nil {
		return err
	}
	rawRec := monitoringv1alpha1.Receiver{Name: rec.Name}
	aconfig, err := c.GetOrCreateAlertmanagerConfig(ctx, rec.Namespace, rec.Source)
	if err != nil {
		return err
	}
	if err := prometheus.ModifyReceiver(ctx, aconfig, &rawRec, prometheus.Delete); err != nil {
		return err
	}
	if err := c.DeleteAlertEmailSecret(ctx, rec.Namespace, rawRec); err != nil {
		return err
	}
	return c.Update(ctx, aconfig)
}
