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

package observe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus/alertmanager/pkg/labels"
	alertmanagertypes "github.com/prometheus/alertmanager/types"
	"github.com/prometheus/prometheus/pkg/rulefmt"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/channels"
	"kubegems.io/kubegems/pkg/utils/prometheus/templates"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	alertProxyHeader = map[string]string{
		"namespace": gems.NamespaceMonitor,
		"service":   "kube-prometheus-stack-alertmanager",
		"port":      "9093",
	}
	jaegerProxyHeader = map[string]string{
		"namespace": gems.NamespaceObserve,
		"service":   "jaeger-operator-jaeger-query",
		"port":      "16686",
	}
	allNamespace = "_all"
)

type ObserveClient struct {
	agents.Client
	*gorm.DB
}

func NewClient(cli agents.Client, db *gorm.DB) *ObserveClient {
	return &ObserveClient{Client: cli, DB: db}
}

func (c *ObserveClient) ListMonitorAlertRules(ctx context.Context, namespace string, hasDetail bool, tplGetter templates.TplGetter) ([]MonitorAlertRule, error) {
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
		if err := c.List(ctx, &promeRuleList,
			client.InNamespace(namespace),
			client.HasLabels([]string{gems.LabelPrometheusRuleName}),
		); err != nil {
			return err
		}
		for _, v := range promeRuleList.Items {
			promRuleMap[client.ObjectKeyFromObject(v).String()] = v
		}
		return nil
	})
	eg.Go(func() error {
		amConfigList := monitoringv1alpha1.AlertmanagerConfigList{}
		if err := c.List(ctx, &amConfigList,
			client.InNamespace(namespace),
			client.MatchingLabels(map[string]string{gems.LabelAlertmanagerConfigType: prometheus.AlertTypeMonitor}),
		); err != nil {
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
		realTimeAlertRules, err = c.Extend().GetPromeAlertRules(ctx, "")
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	ret := []MonitorAlertRule{}
	for key, rule := range promRuleMap {
		amconfig, ok := amConfigMap[key]
		if !ok {
			log.Warnf("alertmanager config %s in cluster: %s namespace %s not found", rule.Name, c.Name(), rule.Namespace)
			continue
		}
		raw := &RawMonitorAlertResource{
			Base: &BaseAlertResource{
				AMConfig:      amconfig,
				Silences:      silenceNamespaceMap[rule.Namespace],
				ChannelGetter: models.NewChannnelMappler(c.DB).FindChannel,
			},
			PrometheusRule: rule,
			TplGetter:      tplGetter,
		}

		alerts, err := raw.ToAlerts(hasDetail)
		if err != nil {
			return nil, err
		}
		for _, v := range alerts {
			if !v.IsExtraAlert() {
				ret = append(ret, v)
			}
		}
	}

	// realtime alert rules 按照namespace+name 分组
	for i := range ret {
		ret[i].SetChannelStatus()
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

func (c *ObserveClient) ListLoggingAlertRules(ctx context.Context, namespace string, hasDetail bool) ([]LoggingAlertRule, error) {
	if namespace == allNamespace {
		namespace = v1.NamespaceAll
	}
	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gems.NamespaceLogging,
			Name:      LoggingAlertRuleCMName,
		},
	}
	amConfigList := monitoringv1alpha1.AlertmanagerConfigList{}
	configNamespaceMap := map[string]*monitoringv1alpha1.AlertmanagerConfig{}
	var allSilences []alertmanagertypes.Silence
	var realTimeAlertRules map[string]prometheus.RealTimeAlertRule

	eg := errgroup.Group{}
	eg.Go(func() error {
		return c.Client.Get(ctx, client.ObjectKeyFromObject(&cm), &cm)
	})
	eg.Go(func() error {
		return c.List(ctx, &amConfigList, client.InNamespace(namespace), client.MatchingLabels(map[string]string{
			gems.LabelAlertmanagerConfigName: prometheus.DefaultAlertCRDName,
		}))
	})
	eg.Go(func() error {
		var err error
		allSilences, err = c.ListSilences(ctx, nil, prometheus.SilenceCommentForAlertrulePrefix)
		return err
	})
	eg.Go(func() error {
		var err error
		realTimeAlertRules, err = c.Extend().GetLokiAlertRules(ctx)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// 当前namespace或者所有namespace的map
	groupNamespaceMap := map[string]*rulefmt.RuleGroups{}
	for k, v := range cm.Data {
		// skip recording rule
		if k == LokiRecordingRulesKey {
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
	ret := []LoggingAlertRule{}
	for thisNamesapce, rulegroups := range groupNamespaceMap {
		amconfig, ok := configNamespaceMap[thisNamesapce]
		if !ok {
			log.Warnf("logging alertmanager config in cluster %s namespace %s not found", c.Name(), thisNamesapce)
			continue
		}
		raw := &RawLoggingAlertRule{
			Base: &BaseAlertResource{
				AMConfig:      amconfig,
				Silences:      silenceNamespaceMap[thisNamesapce],
				ChannelGetter: models.NewChannnelMappler(c.DB).FindChannel,
			},
			ConfigMap:  &cm,
			RuleGroups: rulegroups,
		}

		alerts, err := raw.ToAlerts(hasDetail)
		if err != nil {
			return nil, err
		}
		for _, v := range alerts {
			if !v.IsExtraAlert() {
				ret = append(ret, v)
			}
		}
	}

	for i := range ret {
		ret[i].SetChannelStatus()
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

func (c *ObserveClient) getBaseAlertResource(ctx context.Context, namespace, name string) (*BaseAlertResource, error) {
	loggingAMConfig, err := c.GetOrCreateAlertmanagerConfig(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	silence, err := c.ListSilences(ctx, map[string]string{
		prometheus.AlertNamespaceLabel: namespace,
	}, prometheus.SilenceCommentForAlertrulePrefix)
	if err != nil {
		return nil, err
	}
	return &BaseAlertResource{
		AMConfig:      loggingAMConfig,
		Silences:      silence,
		ChannelGetter: models.NewChannnelMappler(c.DB).FindChannel,
	}, nil
}

// GetRawMonitorAlertResource get specified namespace's alert
func (c *ObserveClient) GetRawMonitorAlertResource(ctx context.Context, namespace, name string, tplGetter templates.TplGetter) (*RawMonitorAlertResource, error) {
	base, err := c.getBaseAlertResource(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	// copy receivers if created by appstore exporter
	if name != prometheus.DefaultAlertCRDName {
		defaultAmcfg, err := c.GetOrCreateAlertmanagerConfig(ctx, namespace, prometheus.DefaultAlertCRDName)
		if err != nil {
			return nil, errors.Wrap(err, "get default amcfg")
		}
		base.AMConfig.Spec.Receivers = defaultAmcfg.Spec.Receivers
	}
	promerule, err := c.GetOrCreatePrometheusRule(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	return &RawMonitorAlertResource{
		Base:           base,
		PrometheusRule: promerule,
		TplGetter:      tplGetter,
	}, nil
}

func (c *ObserveClient) GetRawLoggingAlertResource(ctx context.Context, namespace string) (*RawLoggingAlertRule, error) {
	base, err := c.getBaseAlertResource(ctx, namespace, prometheus.DefaultAlertCRDName)
	if err != nil {
		return nil, err
	}
	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gems.NamespaceLogging,
			Name:      LoggingAlertRuleCMName,
		},
	}
	if err := c.Client.Get(ctx, client.ObjectKeyFromObject(&cm), &cm); err != nil {
		return nil, err
	}

	raw := &RawLoggingAlertRule{
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

func (c *ObserveClient) CommitRawLoggingAlertResource(ctx context.Context, raw *RawLoggingAlertRule) error {
	bts, err := yaml.Marshal(raw.RuleGroups)
	if err != nil {
		return err
	}
	if raw.ConfigMap.Data == nil {
		raw.ConfigMap.Data = make(map[string]string)
	}
	raw.ConfigMap.Data[raw.Base.AMConfig.Namespace] = string(bts)
	if err := c.Client.Update(ctx, raw.ConfigMap); err != nil {
		return err
	}
	return c.Client.Update(ctx, raw.Base.AMConfig)
}

func (c *ObserveClient) GetOrCreateAlertmanagerConfig(ctx context.Context, namespace, name string) (*monitoringv1alpha1.AlertmanagerConfig, error) {
	aconfig := &monitoringv1alpha1.AlertmanagerConfig{}
	err := c.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, aconfig)
	if kerrors.IsNotFound(err) {
		// 初始化
		aconfig = GetBaseAlertmanagerConfig(namespace, name)
		if err := c.Extend().CheckAlertmanagerConfig(ctx, aconfig); err != nil {
			return nil, err
		}

		if err := c.Client.Create(ctx, aconfig); err != nil {
			return nil, err
		}
		return aconfig, nil
	}
	return aconfig, err
}

func (c *ObserveClient) GetOrCreatePrometheusRule(ctx context.Context, namespace, name string) (*monitoringv1.PrometheusRule, error) {
	prule := &monitoringv1.PrometheusRule{}
	err := c.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, prule)
	if kerrors.IsNotFound(err) {
		prule = GetBasePrometheusRule(namespace, name)
		if err := c.Client.Create(ctx, prule); err != nil {
			return nil, err
		}
		return prule, nil
	}
	return prule, err
}

func (c *ObserveClient) CommitRawMonitorAlertResource(ctx context.Context, raw *RawMonitorAlertResource) error {
	if err := c.Extend().CheckAlertmanagerConfig(ctx, raw.Base.AMConfig); err != nil {
		bts, _ := yaml.Marshal(raw.Base.AMConfig)
		log.Error(err, "amconfig", string(bts))
		return err
	}

	if err := c.Client.Update(ctx, raw.PrometheusRule); err != nil {
		return err
	}
	return c.Client.Update(ctx, raw.Base.AMConfig)
}

func (c *ObserveClient) CreateOrUpdateAlertEmailSecret(ctx context.Context, namespace string, receivers []AlertReceiver) error {
	emails := map[string]*channels.Email{}
	for _, rec := range receivers {
		switch v := rec.AlertChannel.ChannelConfig.ChannelIf.(type) {
		case *channels.Email:
			emails[rec.AlertChannel.ReceiverName()] = v
		}
	}

	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      channels.EmailSecretName,
			Namespace: namespace,
			Labels: map[string]string{
				channels.EmailSecretLabelKey: channels.EmailSecretLabelValue,
			},
		},
		Type: v1.SecretTypeOpaque,
	}
	_, err := controllerutil.CreateOrUpdate(ctx, c.Client, sec, func() error {
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

func (c *ObserveClient) ListSilences(ctx context.Context, labels map[string]string, commentPrefix string) ([]alertmanagertypes.Silence, error) {
	allSilences := []alertmanagertypes.Silence{}

	req := agents.Request{
		Path: "/v1/service-proxy/api/v2/silences",
		Query: func() url.Values {
			values := url.Values{}
			for k, v := range labels {
				values.Add("filter", fmt.Sprintf(`%s="%s"`, k, v))
			}
			return values
		}(),
		Headers: agents.HeadersFrom(alertProxyHeader),
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
func (c *ObserveClient) CreateOrUpdateSilenceIfNotExist(ctx context.Context, info models.AlertInfo) error {
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

	agentreq := agents.Request{
		Method:  http.MethodPost,
		Path:    "/v1/service-proxy/api/v2/silences",
		Body:    silence,
		Headers: agents.HeadersFrom(alertProxyHeader),
	}

	if err := c.DoRequest(ctx, agentreq); err != nil {
		return fmt.Errorf("create silence:%w", err)
	}
	return nil
}

// use for blacklist
func (c *ObserveClient) DeleteSilenceIfExist(ctx context.Context, info models.AlertInfo) error {
	silenceList, err := c.ListSilences(ctx, info.LabelMap, prometheus.SilenceCommentForBlackListPrefix)
	if err != nil {
		return err
	}
	switch len(silenceList) {
	case 0:
		return nil
	case 1:
		agentreq := agents.Request{
			Method:  http.MethodDelete,
			Path:    fmt.Sprintf("/v1/service-proxy/api/v2/silence/%s", silenceList[0].ID),
			Headers: agents.HeadersFrom(alertProxyHeader),
		}
		return c.DoRequest(ctx, agentreq)
	default:
		return fmt.Errorf("too many silences for alert: %v", info)
	}
}

func (c ObserveClient) SearchTrace(
	ctx context.Context,
	service string,
	start, end time.Time,
	maxDuration, minDuration string,
	limit int,
) ([]Trace, error) {
	q := url.Values{}
	q.Add("service", service)
	q.Add("start", strconv.FormatInt(start.UnixMicro(), 10))
	q.Add("end", strconv.FormatInt(end.UnixMicro(), 10))
	// q.Add("maxDuration", maxDuration)
	// q.Add("minDuration", minDuration)
	// q.Add("limit", limit)

	resp := tracesResponse{}
	req := agents.Request{
		Method:  http.MethodGet,
		Path:    "/v1/service-proxy/api/traces",
		Query:   q,
		Headers: agents.HeadersFrom(jaegerProxyHeader),
		Into:    &resp,
	}
	if err := c.DoRequest(ctx, req); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		log.Errorf("jaeger resp err: %v", resp.Errors)
		return nil, fmt.Errorf(resp.Errors[0].Msg)
	}

	var maxDur, minDur time.Duration
	var err error
	if maxDuration != "" {
		maxDur, err = time.ParseDuration(maxDuration)
		if err != nil {
			return nil, errors.Wrap(err, "parse maxDuration")
		}
	}
	if minDuration != "" {
		minDur, err = time.ParseDuration(minDuration)
		if err != nil {
			return nil, errors.Wrap(err, "parse minDuration")
		}
	}

	// 由于我们要求过滤的是trace的duration，而jaeger api过滤的是span的，所以需要我们手动过滤
	ret := make([]Trace, 0, len(resp.Data))
	for _, trace := range resp.Data {
		validTrace := true
		// 只判断第一个span
		if len(trace.Spans) > 0 {
			maxSpan := trace.Spans[0]
			if maxDuration != "" && maxSpan.Duration > uint64(maxDur.Microseconds()) {
				validTrace = false
			}
			if minDuration != "" && maxSpan.Duration < uint64(minDur.Microseconds()) {
				validTrace = false
			}
		}
		if validTrace {
			ret = append(ret, trace)
		}
	}
	// sort by starttime desc
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Spans[0].StartTime > ret[j].Spans[0].StartTime
	})
	if len(ret) > limit {
		return ret[:limit], nil
	}
	return ret, nil
}

func (c ObserveClient) GetTrace(
	ctx context.Context,
	traceID string,
) (*Trace, error) {
	resp := tracesResponse{}
	req := agents.Request{
		Method:  http.MethodGet,
		Path:    "/v1/service-proxy/api/traces/" + traceID,
		Headers: agents.HeadersFrom(jaegerProxyHeader),
		Into:    &resp,
	}
	if err := c.DoRequest(ctx, req); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		log.Errorf("jaeger resp err: %v", resp.Errors)
		return nil, fmt.Errorf(resp.Errors[0].Msg)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no trace found")
	}
	return &resp.Data[0], nil
}
