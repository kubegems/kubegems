package controllers

import (
	"reflect"
	"testing"
)


func TestRemoveNulls(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "",
			args: map[string]interface{}{
				"nested": map[string]interface{}{
					"foo": "",
				},
			},
			want: map[string]interface{}{},
		},
		{
			name: "",
			args: map[string]interface{}{
				"nested": map[string]interface{}{
					"foo": "",
					"var": "val",
				},
			},
			want: map[string]interface{}{
				"nested": map[string]interface{}{
					"var": "val",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RemoveNulls(tt.args)
			if !reflect.DeepEqual(tt.args, tt.want) {
				t.Errorf("RemoveNulls() = %v, want %v", tt.args, tt.want)
			}
		})
	}
}
