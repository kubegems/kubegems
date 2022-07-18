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

func TestCompileSection(t *testing.T) {
	tests := []struct {
		pattern    string
		want       []Element
		wantErr    bool
		wantErrStr string
	}{
		{
			pattern: "a{v}-b{f}-g*",
			want: []Element{
				{kind: ElementKindConst, param: "a"},
				{kind: ElementKindVariable, param: "v"},
				{kind: ElementKindConst, param: "-b"},
				{kind: ElementKindVariable, param: "f"},
				{kind: ElementKindConst, param: "-g"},
				{kind: ElementKindStar},
			},
		},
		{
			pattern: "{v}{f*}**-g*",
			want: []Element{
				{kind: ElementKindVariable, param: "v"},
				{kind: ElementKindVariable, param: "f*"},
				{kind: ElementKindConst, param: "**-g"},
				{kind: ElementKindStar},
			},
		},
		{
			pattern: "{v}{}{f*}**-g*",
			want: []Element{
				{kind: ElementKindVariable, param: "v"},
				{kind: ElementKindVariable, param: ""},
				{kind: ElementKindVariable, param: "f*"},
				{kind: ElementKindConst, param: "**-g"},
				{kind: ElementKindStar},
			},
		},
		{
			pattern:    "{hello",
			wantErr:    true,
			wantErrStr: "invalid char [o] in [{hello] at position 6: variable defination not closed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got, err := CompileSection(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileSection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && tt.wantErrStr != err.Error() {
				t.Errorf("CompileSection() error.Error() = %v, wantErrStr %v", err.Error(), tt.wantErrStr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CompileSection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchSection(t *testing.T) {
	type want struct {
		matched       bool
		matchthelefts bool
		vars          map[string]string
	}
	tests := []struct {
		pattern string
		tomatch []string
		want    want
	}{
		{
			pattern: "pre{name}suf",
			tomatch: []string{"prehellosuf"},
			want:    want{matched: true, vars: map[string]string{"name": "hello"}},
		},
		{

			pattern: "pre{name}suf",
			tomatch: []string{"presuf"}, // 假设 name 为空时，不被匹配
		},
		{
			pattern: "pre{name}",
			tomatch: []string{"presuf", "/", "data"},
			want:    want{matched: true, vars: map[string]string{"name": "suf"}},
		},
		{
			pattern: "pre*",
			tomatch: []string{"prehellosuf", "/", "anything"},
			want:    want{matched: true, matchthelefts: true, vars: map[string]string{}},
		},
		{
			pattern: "pre{name}*",
			tomatch: []string{"prehellosuf", "/", "anything"},
			want:    want{matched: true, matchthelefts: true, vars: map[string]string{"name": "hellosuf/anything"}},
		},
		{
			pattern: "pre",
			tomatch: []string{"prehellosuf"},
		},
		{
			pattern: "empty",
			tomatch: []string{},
		},
		{
			pattern: "apis",
			tomatch: []string{"ap2is"},
		},
		{
			pattern: "apis",
			tomatch: []string{"api"},
		},
		{
			pattern: "{a}{b}",
			tomatch: []string{"tom"},
			want:    want{matched: true, vars: map[string]string{"b": "tom"}},
		},
		{
			pattern: "{a}:{b}",
			tomatch: []string{"tom:cat"},
			want:    want{matched: true, vars: map[string]string{"a": "tom", "b": "cat"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			compiled := MustCompileSection(tt.pattern)
			matched, next, vars := MatchSection(compiled, tt.tomatch)
			if matched != tt.want.matched {
				t.Errorf("MatchSection() matched = %v, want %v", matched, tt.want.matched)
			}
			if next != tt.want.matchthelefts {
				t.Errorf("MatchSection() next = %v, want %v", next, tt.want.matchthelefts)
			}
			if !reflect.DeepEqual(vars, tt.want.vars) {
				t.Errorf("MatchSection() vars = %v, want %v", vars, tt.want.vars)
			}
		})
	}
}

func TestMustCompileSection(t *testing.T) {
	tests := []struct {
		pattern   string
		wantPanic bool
		want      []Element
	}{
		{
			pattern:   "{",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			defer func() {
				if r := recover(); tt.wantPanic && r == nil {
					t.Errorf("The code did not panic")
				}
			}()
			if got := MustCompileSection(tt.pattern); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MustCompileSection() = %v, want %v", got, tt.want)
			}
		})
	}
}
