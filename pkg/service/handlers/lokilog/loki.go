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

package lokilog

import (
	"context"
	"net/url"
	"strings"

	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/loki"
)

type LokiCli = LogHandler

func (c LokiCli) LokiQuery(ctx context.Context, cluster string, query map[string]string) (*loki.QueryResponseData, error) {
	url := formatURL(nil, nil, query, "/custom/loki/v1/query")
	ret := &loki.QueryResponseData{}

	err := c.query(ctx, cluster, url, ret)
	return ret, err
}

func (c LokiCli) LokiQueryRange(ctx context.Context, cluster string, query map[string]string) (*loki.QueryResponseData, error) {
	url := formatURL(nil, nil, query, "/custom/loki/v1/queryrange")
	ret := &loki.QueryResponseData{}

	err := c.query(ctx, cluster, url, ret)
	return ret, err
}

func (c LokiCli) LokiLabels(ctx context.Context, cluster string, query map[string]string) ([]string, error) {
	url := formatURL(nil, nil, query, "/custom/loki/v1/labels")
	ret := []string{}

	err := c.query(ctx, cluster, url, &ret)
	return ret, err
}

func (c LokiCli) LokiLabelValues(ctx context.Context, cluster string, label string, query map[string]string) ([]string, error) {
	query["label"] = label
	url := formatURL(nil, nil, query, "/custom/loki/v1/labelvalues")
	ret := []string{}

	err := c.query(ctx, cluster, url, &ret)
	return ret, err
}

func (c LokiCli) LokiSeries(ctx context.Context, cluster string, query map[string]string) (interface{}, error) {
	url := formatURL(nil, nil, query, "/custom/loki/v1/series")
	ret := []interface{}{}

	err := c.query(ctx, cluster, url, &ret)
	return ret, err
}

func (c LokiCli) query(ctx context.Context, cluster string, path string, into interface{}) error {
	return c.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.DoRequest(ctx, agents.Request{
			Path: path,
			Into: agents.WrappedResponse(into),
		})
	})
}

func formatURL(args, labelsel, query map[string]string, ptn string) string {
	base := ptn
	for key, value := range args {
		base = strings.ReplaceAll(base, "{"+key+"}", value)
	}
	qs := url.Values{}
	for qk, qv := range labelsel {
		qs.Set("labels["+qk+"]", qv)
	}
	for qk, qv := range query {
		qs.Set(qk, qv)
	}
	u := url.URL{
		Path:     base,
		RawQuery: qs.Encode(),
	}
	return u.String()
}
