package observability

import (
	"testing"

	"kubegems.io/pkg/utils/prometheus"
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
			gotQuery, gotOp, gotValue, gotHasOp := prometheus.SplitLogql(tt.args.logql)
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
