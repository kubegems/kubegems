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

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
			gotcontent, _ := json.MarshalIndent(got, "", "  ")
			fmt.Printf("GenerateSchema() = %s", gotcontent)
		})
	}
}
