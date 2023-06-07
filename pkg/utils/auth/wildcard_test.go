// Copyright 2023 The kubegems.io Authors
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

package auth

import "testing"

func TestWildcardMatchSections(t *testing.T) {
	tests := []struct {
		expr  string
		perm  string
		match bool
	}{
		{expr: "", perm: "zoo:cats:tom:get", match: false},
		{expr: "zoo:cats:tom:get", perm: "", match: false},
		{expr: "zoo:cats:tom:*", perm: "zoo:cats:tom:get", match: true},
		{expr: "zoo:cats:*:get,list", perm: "zoo:cats:tom:remove", match: false},
		{expr: "zoo:cats:*:get,list", perm: "zoo:remove", match: false},
		{expr: "zoo:*", perm: "zoo:cats:tom:remove", match: false},
		{expr: "zoo:**:some-garbage", perm: "zoo:cats:tom:remove", match: true},
		{expr: "zoo:**", perm: "zoo:cats:tom:remove", match: true},
		{expr: "zoo:list:*:*", perm: "zoo:list", match: true},
		{expr: "zoo:list:*:abc", perm: "zoo:list", match: false},
		{expr: "zoo:list:**", perm: "zoo:list", match: true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			if got := WildcardMatchSections(tt.expr, tt.perm); got != tt.match {
				t.Errorf("WildcardMatchSections() = %v, want %v", got, tt.match)
			}
		})
	}
}
