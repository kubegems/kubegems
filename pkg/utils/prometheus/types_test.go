package prometheus

import (
	"reflect"
	"testing"
)

func TestBaseQueryParams_FindRuleContext(t *testing.T) {
	type fields struct {
		Resource   string
		Rule       string
		Unit       string
		LabelPairs map[string]string
	}
	type args struct {
		cfg GemsMetricConfig
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
				Unit:     "core",
				LabelPairs: map[string]string{
					"host": "master1",
				},
			},
			args: args{
				cfg: DefaultMetricConfigContent(),
			},
			want: RuleContext{
				ResourceDetail: ResourceDetail{
					Namespaced: false,
					ShowName:   "节点",
				},
				RuleDetail: RuleDetail{
					Expr:     "gems_node_cpu_total_cores",
					ShowName: "CPU总量",
					Units:    []string{"core", "mcore"},
					Labels:   []string{"host"},
				},
			},
			wantErr: false,
		},
		{
			name: "has extra label error",
			fields: fields{
				Resource: "node",
				Rule:     "cpuTotal",
				Unit:     "core",
				LabelPairs: map[string]string{
					"host":      "master1",
					"container": "mycontainer",
				},
			},
			args: args{
				cfg: DefaultMetricConfigContent(),
			},
			want:    RuleContext{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := &BaseQueryParams{
				Resource:   tt.fields.Resource,
				Rule:       tt.fields.Rule,
				Unit:       tt.fields.Unit,
				LabelPairs: tt.fields.LabelPairs,
			}
			got, err := params.FindRuleContext(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("BaseQueryParams.FindRuleContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got.ResourceDetail.Rules = nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BaseQueryParams.FindRuleContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
