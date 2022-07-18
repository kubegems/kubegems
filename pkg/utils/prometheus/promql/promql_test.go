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

import "testing"

func TestQuery_ToPromql(t *testing.T) {
	type fields struct {
		metric    string
		selectors []string
		sumBy     []string
		op        ComparisonOperator
		value     string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "1",
			fields: fields{
				metric:    "gems_container_cpu_usage_cores",
				selectors: []string{`namespace="gemcloud-gateway-system"`, `node=~"k8s-master2-122"`},
				sumBy:     []string{"pod"},
				op:        GreaterOrEqual,
				value:     "0",
			},
			want: `sum(gems_container_cpu_usage_cores{namespace="gemcloud-gateway-system", node=~"k8s-master2-122"})by(pod) >= 0`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &Query{
				metric:       tt.fields.metric,
				selectors:    tt.fields.selectors,
				sumBy:        tt.fields.sumBy,
				compare:      tt.fields.op,
				compareValue: tt.fields.value,
			}
			if got := q.ToPromql(); got != tt.want {
				t.Errorf("Query.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
