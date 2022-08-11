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

func TestParseUnit(t *testing.T) {
	type args struct {
		unit string
	}
	tests := []struct {
		name    string
		args    args
		want    UnitValue
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				unit: "bytes-KB/s",
			},
			want: UnitValue{
				Show:  "KB/s",
				Op:    "/",
				Value: "1024",
			},
			wantErr: false,
		},
		{
			name: "2",
			args: args{
				unit: "bytes-b/s",
			},
			wantErr: true,
		},
		{
			name: "3",
			args: args{
				unit: "custom-行",
			},
			want: UnitValue{
				Show: "行",
			},
			wantErr: false,
		},
		{
			name: "4",
			args: args{
				unit: "times",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUnit(tt.args.unit)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUnit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseUnit() = %v, want %v", got, tt.want)
			}
		})
	}
}
