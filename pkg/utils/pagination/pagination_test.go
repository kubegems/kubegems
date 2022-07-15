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

package pagination

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testSortAndSearchAble struct {
	name     string
	creation time.Time
}

func (t testSortAndSearchAble) GetName() string {
	return t.name
}

func (t testSortAndSearchAble) GetCreationTimestamp() metav1.Time {
	return metav1.Time{Time: t.creation}
}

func TestSortByFunc(t *testing.T) {
	testdata := []SortAndSearchAble{
		testSortAndSearchAble{
			name:     "a",
			creation: time.Time{}.Add(1),
		},
		testSortAndSearchAble{
			name:     "b",
			creation: time.Time{}.Add(3),
		},
		testSortAndSearchAble{
			name:     "zz",
			creation: time.Time{}.Add(8),
		},
		testSortAndSearchAble{
			name:     "c",
			creation: time.Time{}.Add(5),
		},
		testSortAndSearchAble{
			name:     "ba",
			creation: time.Time{}.Add(2),
		},
		testSortAndSearchAble{
			name:     "bb",
			creation: time.Time{}.Add(1),
		},
	}

	type args struct {
		datas []SortAndSearchAble
		by    string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "by name",
			args: args{
				datas: testdata,
				by:    "nameDesc",
			},
		},
		{
			name: "by creationTimestamp desc",
			args: args{
				datas: testdata,
				by:    "time",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SortByFunc(tt.args.datas, tt.args.by)
			t.Log(tt.args.datas)
		})
	}
}
