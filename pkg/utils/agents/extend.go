package agents

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus/alertmanager/pkg/labels"
	alertmanagertypes "github.com/prometheus/alertmanager/types"
	prommodel "github.com/prometheus/common/model"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/pkg/apis/plugins"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/loki"
	"kubegems.io/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	alertProxyHeader = map[string]string{
		"namespace": "gemcloud-monitoring-system",
		"service":   "alertmanager",
		"port":      "9093",
	}
)

type ExtendClient struct {
	*TypedClient
}

// plugins.kubegems.io/v1alpha1
func (c *ExtendClient) ListPlugins(ctx context.Context) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	err := c.DoRequest(ctx, Request{
		Method: http.MethodGet,
		Path:   "/custom/" + plugins.GroupName + "/v1beta1/installers",
		Into:   WrappedResponse(&ret),
	})
	return ret, err
}

func (c *ExtendClient) EnablePlugin(ctx context.Context, ptype, name string) error {
	return c.DoRequest(ctx, Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/custom/%s/v1beta1/installers/%s/actions/enable?type=%s", plugins.GroupName, name, ptype),
	})
}

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
			Path:    fmt.Sprintf("/v1/service-proxy/api/v2/silences/%s", silenceList[0].ID),
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

// TODO: 使用原生prometheus api
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

func (c *ExtendClient) GetPrometheusLabelValues(ctx context.Context, match, label, start, end string) ([]string, error) {
	resp := struct {
		Labels []string    `json:"labels,omitempty"`
		Warns  interface{} `json:"warns,omitempty"`
	}{}
	values := url.Values{}
	values.Add("match", match)
	values.Add("label", label)
	values.Add("start", start)
	values.Add("end", end)
	if err := c.DoRequest(ctx, Request{
		Path:  "/custom/prometheus/v1/labelvalues",
		Query: values,
		Into:  WrappedResponse(&resp),
	}); err != nil {
		return nil, fmt.Errorf("prometheus label values failed, cluster: %s, promql: %s, label: %s, %v", c.Name, match, label, err)
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

func (c *ExtendClient) ListAllAlertRules(ctx context.Context, opts *prometheus.MonitorOptions) ([]prometheus.AlertRule, error) {
	ruleList := monitoringv1.PrometheusRuleList{}
	configList := monitoringv1alpha1.AlertmanagerConfigList{}

	configNamespaceMap := map[string]*monitoringv1alpha1.AlertmanagerConfig{}
	silenceNamespaceMap := map[string][]alertmanagertypes.Silence{}

	ret := []prometheus.AlertRule{}
	if err := c.List(ctx, &ruleList, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(prometheus.PrometheusRuleSelector)); err != nil {
		return nil, err
	}
	if err := c.List(ctx, &configList, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(prometheus.AlertmanagerConfigSelector)); err != nil {
		return nil, err
	}

	// 按照namespace分组
	for _, v := range configList.Items {
		if v.Name == prometheus.AlertmanagerConfigName {
			configNamespaceMap[v.Namespace] = v
		}
	}

	// 按照namespace分组
	allSilences, err := c.ListSilences(ctx, nil, prometheus.SilenceCommentForAlertrulePrefix)
	if err != nil {
		return nil, err
	}
	for _, silence := range allSilences {
		for _, m := range silence.Matchers {
			if m.Name == prometheus.AlertNamespaceLabel {
				silenceNamespaceMap[m.Value] = append(silenceNamespaceMap[m.Value], silence)
			}
		}
	}

	for _, rule := range ruleList.Items {
		if rule.Name == prometheus.PrometheusRuleName {
			amconfig, ok := configNamespaceMap[rule.Namespace]
			if !ok {
				log.Warnf("alertmanager config %s in namespace %s not found", rule.Name, rule.Namespace)
				continue
			}
			raw := &prometheus.RawAlertResource{
				PrometheusRule:     rule,
				AlertmanagerConfig: amconfig,
				Silences:           silenceNamespaceMap[rule.Namespace],
				MonitorOptions:     opts,
			}

			alerts, err := raw.ToAlerts(false)
			if err != nil {
				return nil, err
			}
			ret = append(ret, alerts...)
		}
	}

	return ret, nil
}

// GetRawAlertResource get specified namespace's alert
func (c *ExtendClient) GetRawAlertResource(ctx context.Context, namespace string, opts *prometheus.MonitorOptions) (*prometheus.RawAlertResource, error) {
	amcfg, err := c.GetOrCreateAlertmanagerConfig(ctx, namespace)
	if err != nil {
		return nil, err
	}

	promerule, err := c.GetOrCreatePrometheusRule(ctx, namespace)
	if err != nil {
		return nil, err
	}

	silence, err := c.ListSilences(ctx, map[string]string{
		prometheus.AlertNamespaceLabel: namespace,
	}, prometheus.SilenceCommentForAlertrulePrefix)
	if err != nil {
		return nil, err
	}

	return &prometheus.RawAlertResource{
		AlertmanagerConfig: amcfg,
		PrometheusRule:     promerule,
		Silences:           silence,
		MonitorOptions:     opts,
	}, nil
}

func (c *ExtendClient) GetOrCreateAlertmanagerConfig(ctx context.Context, namespace string) (*monitoringv1alpha1.AlertmanagerConfig, error) {
	aconfig := &monitoringv1alpha1.AlertmanagerConfig{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: prometheus.AlertmanagerConfigName}, aconfig)
	if kerrors.IsNotFound(err) {
		// 初始化
		aconfig = prometheus.GetBaseAlertmanagerConfig(namespace)
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

func (c *ExtendClient) GetOrCreatePrometheusRule(ctx context.Context, namespace string) (*monitoringv1.PrometheusRule, error) {
	prule := &monitoringv1.PrometheusRule{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: prometheus.PrometheusRuleName}, prule)
	if kerrors.IsNotFound(err) {
		prule = prometheus.GetBasePrometheusRule(namespace)
		if err := c.Create(ctx, prule); err != nil {
			return nil, err
		}
		return prule, nil
	}
	return prule, err
}

func (c *ExtendClient) CommitRawAlertResource(ctx context.Context, raw *prometheus.RawAlertResource) error {
	if err := c.CheckAlertmanagerConfig(ctx, raw.AlertmanagerConfig); err != nil {
		return err
	}

	if err := c.Update(ctx, raw.PrometheusRule); err != nil {
		return err
	}
	return c.Update(ctx, raw.AlertmanagerConfig)
}

func (c *ExtendClient) ListMetricTargets(ctx context.Context, namespace string) ([]*prometheus.MetricTarget, error) {
	pms := monitoringv1.PodMonitorList{}
	sms := monitoringv1.ServiceMonitorList{}
	g := errgroup.Group{}
	g.Go(func() error {
		return c.List(ctx, &pms, client.InNamespace(namespace))
	})
	g.Go(func() error {
		return c.List(ctx, &sms, client.InNamespace(namespace))
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	ret := []*prometheus.MetricTarget{}
	for _, v := range pms.Items {
		ret = append(ret, prometheus.ConvertToMetricTarget(v))
	}
	for _, v := range sms.Items {
		ret = append(ret, prometheus.ConvertToMetricTarget(v))
	}
	sort.Slice(ret, func(i, j int) bool {
		return strings.ToLower(ret[i].Name) < strings.ToLower(ret[j].Name)
	})
	return ret, nil
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
