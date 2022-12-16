package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-openapi/spec"
)

func TestGenerateSchema(t *testing.T) {
	type args struct {
		chartpath string
	}
	tests := []struct {
		name    string
		args    args
		want    *spec.Schema
		wantErr bool
	}{
		{
			name: "",
			args: args{
				chartpath: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valuescontent, err := os.ReadFile(filepath.Join(tt.args.chartpath, "values.yaml"))
			if err != nil {
				t.Error(err)
				return
			}
			got, err := GenerateSchema(valuescontent)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				gotcontent, _ := json.MarshalIndent(got, "", "  ")
				wantcontent, _ := json.MarshalIndent(tt.want, "", "  ")
				t.Errorf("GenerateSchema() = %s, want %s", gotcontent, wantcontent)
			}
		})
	}
}
