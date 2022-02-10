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
