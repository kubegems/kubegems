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
