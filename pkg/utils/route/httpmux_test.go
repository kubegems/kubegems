package route

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeMux_ServeHTTP(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		regs map[string]http.HandlerFunc
		args args
	}{
		{
			name: "",
			regs: map[string]http.HandlerFunc{
				"/apis": http.NotFound,
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodGet, "/apis", nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			mux := NewServeMux()
			for path, handler := range tt.regs {
				mux.HandlerFunc(path, handler)
			}
			mux.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
