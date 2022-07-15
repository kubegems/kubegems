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

package route

import (
	"reflect"
	"testing"
)

func Test_matcher_Register(t *testing.T) {
	matcher := &matcher{root: &node{}}

	type args struct {
		pattern string
		val     interface{}
	}
	tests := []struct {
		args    args
		wantErr bool
	}{
		{
			args:    args{pattern: "/api/{path}*", val: "-"},
			wantErr: false,
		},
		{
			args:    args{pattern: "/api/{path}*", val: "-"},
			wantErr: true,
		},
		{
			args:    args{pattern: "/api/{name}*", val: "-"},
			wantErr: true,
		},
		{
			args:    args{pattern: "/api/{name}/{path}", val: "-"},
			wantErr: false,
		},
		{
			args:    args{pattern: "/api/{name/{path}", val: "-"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.args.pattern, func(t *testing.T) {
			if err := matcher.Register(tt.args.pattern, tt.args.val); (err != nil) != tt.wantErr {
				t.Errorf("matcher.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_matcher_Match(t *testing.T) {
	tests := []struct {
		registered []string
		req        string
		matched    bool
		wantMatch  string
		vars       map[string]string
	}{
		{
			registered: []string{
				"/api/v1",
				"/api/v{a}*",
				"/apis",
				"/api/{a}/{b}/{c}",
				"/api/{a}",
				"/api/{path}*",
			},
			req:       "/api/v1/g/v/k",
			matched:   true,
			wantMatch: "/api/v{a}*",
			vars: map[string]string{
				"a": "1/g/v/k",
			},
		},
		{
			registered: []string{
				"/v1/service-proxy/{realpath}*",
				"/v1/{group}/{version}/{resource}",
			},
			req:       "/v1/service-proxy/js/t2.js",
			matched:   true,
			wantMatch: "/v1/service-proxy/{realpath}*",
			vars: map[string]string{
				"realpath": "js/t2.js",
			},
		},
		{
			registered: []string{
				"/v1/{group}/{version}/{resource}/{name}",
				"/v1/{group}/{version}/configmap/{name}",
			},
			req:       "/v1/core/v1/configmap/abc",
			matched:   true,
			wantMatch: "/v1/{group}/{version}/configmap/{name}",
			vars: map[string]string{
				"group":   "core",
				"version": "v1",
				"name":    "abc",
			},
		},
		{
			registered: []string{
				"/api/v{a}*",
				"/api/{a}/{b}/{c}",
				"/api/{path}*",
			},
			req:       "/api/v2/v/k",
			matched:   true,
			wantMatch: "/api/{a}/{b}/{c}",
			vars: map[string]string{
				"a": "v2",
				"b": "v",
				"c": "k",
			},
		},
		{
			registered: []string{
				"/api/s",
			},
			req:     "/api",
			matched: false,
			vars:    map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.req, func(t *testing.T) {
			m := &matcher{root: &node{}}
			for _, v := range tt.registered {
				if err := m.Register(v, v); err != nil {
					t.Error(err)
				}
			}

			matched, val, vars := m.Match(tt.req)
			if matched != tt.matched {
				t.Errorf("matcher.Match() matched = %v, want %v", matched, tt.matched)
			}

			if tt.wantMatch != "" && !reflect.DeepEqual(val, tt.wantMatch) {
				t.Errorf("matcher.Match() val = %v, want %v", val, tt.wantMatch)
			}

			if !reflect.DeepEqual(vars, tt.vars) {
				t.Errorf("matcher.Match() vars = %v, want %v", vars, tt.vars)
			}
		})
	}
}

func Test_sortSectionMatches(t *testing.T) {
	tests := []struct {
		name     string
		sections []*node
		want     []*node
	}{
		{
			name: "",
			sections: []*node{
				{key: MustCompileSection("{var}")},
				{key: MustCompileSection("abc")},
			},
			want: []*node{
				{key: MustCompileSection("abc")},
				{key: MustCompileSection("{var}")},
			},
		},
		{
			name: "",
			sections: []*node{
				{key: MustCompileSection("abc*")},
				{key: MustCompileSection("abc")},
			},
			want: []*node{
				{key: MustCompileSection("abc")},
				{key: MustCompileSection("abc*")},
			},
		},
		{
			name: "",
			sections: []*node{
				{key: MustCompileSection("{var}")},
				{key: MustCompileSection("abc*")},
			},
			want: []*node{
				{key: MustCompileSection("{var}")},
				{key: MustCompileSection("abc*")},
			},
		},
		{
			name: "",
			sections: []*node{
				{key: MustCompileSection("{var}")},
				{key: MustCompileSection("abc{var}")},
				{key: MustCompileSection("abc")},
			},
			want: []*node{
				{key: MustCompileSection("abc")},
				{key: MustCompileSection("abc{var}")},
				{key: MustCompileSection("{var}")},
			},
		},
		// {
		// 	name: "",
		// 	sections: []*node{
		// 		{key: MustCompileSection("abc")},
		// 		{key: MustCompileSection("*")},
		// 		{key: MustCompileSection("abc{var}")},
		// 		{key: MustCompileSection("abc*")},
		// 		{key: MustCompileSection("{var}")},
		// 		{key: MustCompileSection("abc{var*}")},
		// 	},
		// 	want: []*node{
		// 		{key: MustCompileSection("abc")},
		// 		{key: MustCompileSection("abc{var}")},
		// 		{key: MustCompileSection("{var}")},
		// 		{key: MustCompileSection("abc{var}*")},
		// 		{key: MustCompileSection("abc*")},
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortSectionMatches(tt.sections)
			if !reflect.DeepEqual(tt.sections, tt.want) {
				t.Errorf("sortSectionMatches() got = %v, want %v", tt.sections, tt.want)
			}
		})
	}
}
