package application

import (
	"reflect"
	"testing"

	"sigs.k8s.io/yaml"
)

type withStatus struct {
	Status struct{} `json:"status,omitempty"`
}

func Test_removeStatusField(t *testing.T) {
	type args struct {
		originFrom interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "",
			args: args{
				originFrom: &withStatus{Status: struct{}{}},
			},
			want: []byte("{}\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := yaml.Marshal(tt.args.originFrom)
			if got := removeStatusField(content); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("removeStatusField() = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}
