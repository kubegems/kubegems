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

package extend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/loki"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

func NewExtendClient(addr *url.URL, tp http.RoundTripper) *ExtendClient {
	return &ExtendClient{
		BaseAddr: addr,
		HTTPClient: &http.Client{
			Transport: tp,
		},
		tracer: otel.GetTracerProvider().Tracer("kubegems.io/kubegems"),
	}
}

type ExtendClient struct {
	BaseAddr   *url.URL
	HTTPClient *http.Client
	tracer     trace.Tracer
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
	// nolint: gomnd
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
		return nil, err
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
		return nil, err
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
		return nil, err
	}
	for _, v := range ret {
		addMetricNameLabel(v.Metric, "{}")
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
		return nil, err
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
		return nil, err
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
		return ret, err
	}
	return ret, nil
}

func WrappedResponse(intodata interface{}) *response.Response {
	return &response.Response{Data: intodata}
}

func QueryFrom(kvs map[string]string) url.Values {
	value := url.Values{}
	for k, v := range kvs {
		value.Add(k, v)
	}
	return value
}

func HeadersFrom(kvs map[string]string) http.Header {
	header := http.Header{}
	for k, v := range kvs {
		header.Add(k, v)
	}
	return header
}

type Request struct {
	Method  string
	Path    string // queries 可以放在 path 中
	Query   url.Values
	Headers http.Header
	Body    interface{}
	Into    interface{}
}

func (c *ExtendClient) DoRawRequest(ctx context.Context, clientreq Request) (*http.Response, error) {
	addr := c.BaseAddr.String() + clientreq.Path

	var body io.Reader

	switch clientreqbody := clientreq.Body.(type) {
	case []byte:
		body = bytes.NewReader(clientreqbody)
	case io.Reader:
		body = clientreqbody
	default:
		content, err := json.Marshal(clientreqbody)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(content)
	}

	userBaggage, err := baggage.Parse(fmt.Sprintf("user.name=%s", "kubegems-test"))
	if err != nil {
		otel.Handle(err)
	}

	req, err := http.NewRequestWithContext(baggage.ContextWithBaggage(ctx, userBaggage), clientreq.Method, addr, body)
	if err != nil {
		return nil, err
	}

	// headers
	for k, vs := range clientreq.Headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if clientreq.Headers.Get("Content-Type") == "" {
		req.Header.Add("Content-Type", "application/json")
	}

	// inject for propagator to do distribute tracing
	otel.GetTextMapPropagator().Inject(req.Context(), propagation.HeaderCarrier(req.Header))

	// queries
	query := req.URL.Query()
	for k, vs := range clientreq.Query {
		for _, v := range vs {
			query.Add(k, v)
		}
	}
	req.URL.RawQuery = query.Encode()

	return c.HTTPClient.Do(req)
}

func (c *ExtendClient) DoRequest(ctx context.Context, req Request) error {
	if req.Method == "" {
		req.Method = "GET"
	}

	ctx, span := c.tracer.Start(ctx,
		fmt.Sprintf("TypedClient.%s %s", req.Method, req.Path),
		trace.WithAttributes(
			attribute.String("k8s.apiserver.host", c.BaseAddr.Host),
			attribute.String("request.method", req.Method),
			attribute.String("request.path", req.Path),
			attribute.String("request.query", req.Query.Encode()),
		),
	)
	defer span.End()
	resp, err := c.DoRawRequest(ctx, req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer resp.Body.Close()

	// err
	if resp.StatusCode >= http.StatusBadRequest {
		content, _ := io.ReadAll(resp.Body) // resp body may be empty
		err := fmt.Errorf("request error: code %d, body %s", resp.StatusCode, string(content))
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// success
	if req.Into != nil {
		if err := json.NewDecoder(resp.Body).Decode(req.Into); err != nil {
			err := fmt.Errorf("decode resp: err: %w", err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
