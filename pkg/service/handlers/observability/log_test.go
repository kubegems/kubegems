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
	"testing"

	"kubegems.io/kubegems/pkg/utils/prometheus"
)

func Test_splitLogql(t *testing.T) {
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
			name: "have op",
			args: args{
				logql: `sum(count_over_time({stream=~"stdout|stderr", namespace="test"} [1m]))>100`,
			},
			wantQuery: `sum(count_over_time({stream=~"stdout|stderr", namespace="test"} [1m]))`,
			wantOp:    ">",
			wantValue: "100",
			wantHasOp: true,
		},
		{
			name: "not have op",
			args: args{
				logql: `{stream=~"stdout|stderr", namespace="test"}`,
			},
			wantQuery: `{stream=~"stdout|stderr", namespace="test"}`,
			wantOp:    "",
			wantValue: "",
			wantHasOp: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotOp, gotValue, gotHasOp := prometheus.SplitQueryExpr(tt.args.logql)
			if gotQuery != tt.wantQuery {
				t.Errorf("splitLogql() gotQuery = %v, want %v", gotQuery, tt.wantQuery)
			}
			if gotOp != tt.wantOp {
				t.Errorf("splitLogql() gotOp = %v, want %v", gotOp, tt.wantOp)
			}
			if gotValue != tt.wantValue {
				t.Errorf("splitLogql() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
			if gotHasOp != tt.wantHasOp {
				t.Errorf("splitLogql() gotHasOp = %v, want %v", gotHasOp, tt.wantHasOp)
			}
		})
	}
}
