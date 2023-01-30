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

package bundle

import "testing"

func TestPerRepoCacheDir(t *testing.T) {
	tests := []struct {
		repo    string
		basedir string
		want    string
	}{
		{
			repo:    "https://foo.com/bar",
			basedir: "/app/plugins",
			want:    "/app/plugins/foo.com/bar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			if got := PerRepoCacheDir(tt.repo, tt.basedir); got != tt.want {
				t.Errorf("PerRepoCacheDir() = %v, want %v", got, tt.want)
			}
		})
	}
}
