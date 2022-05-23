package v1beta1

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
			name: "nil map value",
			args: map[string]interface{}{
				"nested": map[string]interface{}{
					"bar": nil,
				},
			},
			want: map[string]interface{}{},
		},
		{
			name: "map with nil val",
			args: map[string]interface{}{
				"nested": map[string]interface{}{
					"foo": "",
					"var": "val",
				},
			},
			want: map[string]interface{}{
				"nested": map[string]interface{}{
					"foo": "",
					"var": "val",
				},
			},
		},
		{
			name: "remove no val",
			args: map[string]interface{}{
				"nested": map[string]interface{}{
					"foo":  "",
					"bool": false,
				},
			},
			want: map[string]interface{}{
				"nested": map[string]interface{}{
					"foo":  "",
					"bool": false,
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
