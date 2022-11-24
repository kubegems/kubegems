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

package promql

import (
	"reflect"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

func TestQuery_AddLabelMatchers(t *testing.T) {
	type fields struct {
		Expr  parser.Expr
		round float64
	}
	type args struct {
		matchers []*labels.Matcher
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "basic promql",
			fields: fields{
				Expr: parseExpr("gems_container_cpu_usage_cores"),
			},
			args: args{
				matchers: []*labels.Matcher{
					{
						Name:  "pod",
						Value: "mypod",
					},
					{
						Name:  "container",
						Type:  labels.MatchRegexp,
						Value: "c1",
					},
				},
			},
			want: `gems_container_cpu_usage_cores{container=~"c1",pod="mypod"}`,
		},
		{
			name: "function call",
			fields: fields{
				Expr: parseExpr("time()"),
			},
			args: args{
				matchers: []*labels.Matcher{
					{
						Name:  "pod",
						Value: "mypod",
					},
				},
			},
			want: `time()`,
		},
		{
			name: "complext promql",
			fields: fields{
				Expr: parseExpr(`sum(irate(aaa{pod="pod1"}[5m]) * bbb{container="c1"}) by (container) / ccc{pod="pod2"}`),
			},
			args: args{
				matchers: []*labels.Matcher{
					{
						Name:  "pod",
						Value: "mypod",
					},
				},
			},
			want: `sum by(container) (irate(aaa{pod="mypod"}[5m]) * bbb{container="c1",pod="mypod"}) / ccc{pod="mypod"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &Query{
				Expr: tt.fields.Expr,
			}
			if got := q.AddLabelMatchers(tt.args.matchers...).String(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Query.AddLabelMatchers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func parseExpr(promql string) parser.Expr {
	ret, _ := parser.ParseExpr(promql)
	return ret
}

func TestQuery_GetVectorSelectors(t *testing.T) {
	type fields struct {
		Expr parser.Expr
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "basic promql",
			fields: fields{
				Expr: parseExpr("gems_container_cpu_usage_cores"),
			},
			want: []string{"gems_container_cpu_usage_cores"},
		},
		{
			name: "function call",
			fields: fields{
				Expr: parseExpr("time()"),
			},
			want: nil,
		},
		{
			name: "complext promql",
			fields: fields{
				Expr: parseExpr(`sum(irate(aaa{pod="pod1"}[5m]) * bbb{container="c1"}) by (container) / ccc{pod="pod2"}`),
			},
			want: []string{`aaa{pod="pod1"}`, `bbb{container="c1"}`, `ccc{pod="pod2"}`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &Query{
				Expr: tt.fields.Expr,
			}
			if got := q.GetVectorSelectors(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Query.GetVectorSelectorNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPromqlByLabels(t *testing.T) {
	type args struct {
		metric model.Metric
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "null",
			args: args{
				metric: nil,
			},
			want: "{}",
		},
		{
			name: "only labels",
			args: args{
				model.Metric{
					"a": "b",
				},
			},
			want: `{a="b"}`,
		},
		{
			name: "all",
			args: args{
				metric: model.Metric{
					"__name__": "query",
					"a":        "b",
					"c":        "d",
				},
			},
			want: `query{a="b",c="d"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PromqlByLabels(tt.args.metric); got != tt.want {
				t.Errorf("PromqlByLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}
