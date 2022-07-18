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

package prometheus

import (
	"testing"
)

func TestLogqlGenerator_ToLogql(t *testing.T) {
	type fields struct {
		Duration   string
		Match      string
		LabelPairs map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "1",
			fields: fields{
				Duration: "1m",
				Match:    "error",
				LabelPairs: map[string]string{
					"pod":       "mypod",
					"container": "mycontainer",
				},
			},
			want: `sum(count_over_time({container=~"mycontainer", pod=~"mypod", namespace="myns"} |~ "error" [1m]))without(fluentd_thread)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &LogqlGenerator{
				Duration:   tt.fields.Duration,
				Match:      tt.fields.Match,
				LabelPairs: tt.fields.LabelPairs,
			}
			if got := g.ToLogql("myns"); got != tt.want {
				t.Errorf("LogqlGenerator.ToLogql() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitQueryExpr(t *testing.T) {
	type args struct {
		logql string
	}
	tests := []struct {
		name      string
		args      args
		wantQuery string
		wantOp    string
		wantValue string
		wantHasOp bool
	}{
		{
			name: "logql",
			args: args{
				logql: `sum(count_over_time({namespace="ns", container="event-exporter"}| json | line_format "{{.metadata_namespace}}" |~ "error" [1m]))>1`,
			},
			wantQuery: `sum(count_over_time({namespace="ns", container="event-exporter"}| json | line_format "{{.metadata_namespace}}" |~ "error" [1m]))`,
			wantOp:    ">",
			wantValue: "1",
			wantHasOp: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotOp, gotValue, gotHasOp := SplitQueryExpr(tt.args.logql)
			if gotQuery != tt.wantQuery {
				t.Errorf("SplitQueryExpr() gotQuery = %v, want %v", gotQuery, tt.wantQuery)
			}
			if gotOp != tt.wantOp {
				t.Errorf("SplitQueryExpr() gotOp = %v, want %v", gotOp, tt.wantOp)
			}
			if gotValue != tt.wantValue {
				t.Errorf("SplitQueryExpr() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
			if gotHasOp != tt.wantHasOp {
				t.Errorf("SplitQueryExpr() gotHasOp = %v, want %v", gotHasOp, tt.wantHasOp)
			}
		})
	}
}
