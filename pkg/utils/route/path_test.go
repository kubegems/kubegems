package route

import (
	"reflect"
	"testing"
)

func TestParsePathTokens(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "normal",
			args: args{
				path: "/apis/v1/abc",
			},
			want: []string{"/", "apis", "/", "v1", "/", "abc"},
		},
		{
			name: "normal2",
			args: args{
				path: "apis/v1/abc",
			},
			want: []string{"apis", "/", "v1", "/", "abc"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParsePathTokens(tt.args.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePathTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompilePathPattern(t *testing.T) {
	tests := []struct {
		pattern string
		want    [][]Element
		wantErr bool
	}{
		{
			pattern: "/api/v{version}/name*",
			want: [][]Element{
				{{kind: ElementKindSplit}},
				{{kind: ElementKindConst, param: "api"}},
				{{kind: ElementKindSplit}},
				{{kind: ElementKindConst, param: "v"}, {kind: ElementKindVariable, param: "version"}},
				{{kind: ElementKindSplit}},
				{{kind: ElementKindConst, param: "name"}, {kind: ElementKindStar}},
			},
		},
		{
			pattern: "/api/v{version}/{name}*",
			want: [][]Element{
				{{kind: ElementKindSplit}},
				{{kind: ElementKindConst, param: "api"}},
				{{kind: ElementKindSplit}},
				{{kind: ElementKindConst, param: "v"}, {kind: ElementKindVariable, param: "version"}},
				{{kind: ElementKindSplit}},
				{{kind: ElementKindVariable, param: "name"}, {kind: ElementKindStar}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got, err := CompilePathPattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompilePathPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CompilePathPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}
