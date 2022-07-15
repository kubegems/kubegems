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
