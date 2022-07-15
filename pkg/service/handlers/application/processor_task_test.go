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

package application

import "testing"

func TestTaskNameOf(t *testing.T) {
	type args struct {
		ref      PathRef
		taskname string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{
				ref:      PathRef{Tenant: "ten", Project: "proj", Env: "env", Name: "name"},
				taskname: "deploy",
			},
			want: "ten/proj/env/name/deploy",
		},
		{
			args: args{
				ref: PathRef{Tenant: "ten", Project: "proj", Env: "env"},
			},
			want: "ten/proj/env/",
		},
		{
			args: args{
				ref: PathRef{Tenant: "ten", Project: "proj"},
			},
			want: "ten/proj/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TaskNameOf(tt.args.ref, tt.args.taskname); got != tt.want {
				t.Errorf("TaskNameOf() = %v, want %v", got, tt.want)
			}
		})
	}
}
