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
			want: `sum(count_over_time({pod=~"mypod", container=~"mycontainer"} |~ 'error' [1m]))without(fluentd_thread)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &LogqlGenerator{
				Duration:   tt.fields.Duration,
				Match:      tt.fields.Match,
				LabelPairs: tt.fields.LabelPairs,
			}
			if got := g.ToLogql(); got != tt.want {
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
