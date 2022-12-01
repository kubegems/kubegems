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
	"time"

	"github.com/pkg/errors"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/loki"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

type ExtendClient struct {
	Name string
	*TypedClient
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

func (c *ExtendClient) ClusterStatistics(ctx context.Context, ret interface{}) error {
	return c.DoRequest(ctx, Request{
		Path: "/custom/statistics.system/v1/all",
		Into: WrappedResponse(ret),
	})
}

// health.system/v1
func (c *ExtendClient) Healthy(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.DoRequest(ctx, Request{Path: "/healthz"})
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
	log.Debugf("query range: %s", query)
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

	for _, v := range ret {
		addMetricNameLabel(v.Metric, "{}")
	}
	return ret, nil
}

func (c *ExtendClient) PrometheusVector(ctx context.Context, query string) (prommodel.Vector, error) {
	log.Debugf("query vector: %s", query)
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

	for _, v := range ret {
		addMetricNameLabel(v.Metric, "{}")
	}
	return ret, nil
}

func addMetricNameLabel(metric prommodel.Metric, name string) {
	if metric == nil {
		metric = make(prommodel.Metric)
	}
	if _, ok := metric[prommodel.MetricNameLabel]; !ok {
		metric[prommodel.MetricNameLabel] = prommodel.LabelValue(name)
	}
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
