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
	"reflect"
	"testing"
)

func TestParseContent(t *testing.T) {
	type args struct{}
	tests := []struct {
		name string
		data []byte
		want map[string]string
	}{
		{
			name: "",
			data: []byte("foo   1.0.0 \n bar 1.2.0 some desc \n   "),
			want: map[string]string{
				"foo": "1.0.0",
				"bar": "1.2.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseContent(tt.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseContent() = %v, want %v", got, tt.want)
			}
		})
	}
}
