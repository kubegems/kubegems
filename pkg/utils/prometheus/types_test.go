package prometheus

import (
	"reflect"
	"testing"
)

func TestPromqlGenerator_FindRuleContext(t *testing.T) {
	type fields struct {
		Resource   string
		Rule       string
		Unit       string
		LabelPairs map[string]string
	}
	type args struct {
		cfg *MonitorOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    RuleContext
		wantErr bool
	}{
		{
			name: "no error",
			fields: fields{
				Resource: "node",
				Rule:     "cpuTotal",
				Unit:     "short",
				LabelPairs: map[string]string{
					"node": "master1",
				},
			},
			args: args{
				cfg: DefaultMonitorOptions(),
			},
			want: RuleContext{
				ResourceDetail: ResourceDetail{
					Namespaced: false,
					ShowName:   "节点",
				},
				RuleDetail: RuleDetail{
					Expr:     "gems_node_cpu_total_cores",
					ShowName: "CPU总量",
					Labels:   []string{"node"},
					Unit:     "short",
				},
			},
			wantErr: false,
		},
		{
			name: "has extra label error",
			fields: fields{
				Resource: "node",
				Rule:     "cpuTotal",
				Unit:     "short",
				LabelPairs: map[string]string{
					"node":      "master1",
					"container": "mycontainer",
				},
			},
			args: args{
				cfg: DefaultMonitorOptions(),
			},
			want:    RuleContext{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &PromqlGenerator{
				Resource:   tt.fields.Resource,
				Rule:       tt.fields.Rule,
				LabelPairs: tt.fields.LabelPairs,
			}
			got, err := g.FindRuleContext(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("PromqlGenerator.FindRuleContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got.ResourceDetail.Rules = nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PromqlGenerator.FindRuleContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
