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
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

type reg struct {
	method  string
	path    string
	handler gin.HandlerFunc
}

func TestRouter_Match(t *testing.T) {
	apisfunc := func(c *gin.Context) {}
	apisgvnrfunc := func(c *gin.Context) {}
	apisgvnrnfunc := func(c *gin.Context) {}
	apiscorev1configmapfunc := func(c *gin.Context) {}
	testnotfoundfunc := func(c *gin.Context) {}

	tests := []struct {
		setnotfound gin.HandlerFunc
		registered  []reg
		req         *http.Request
		want        gin.HandlerFunc
	}{
		{
			registered: []reg{
				{"GET", "/apis/{path}*", apisfunc},
				{"GET", "/apis/{group}/{version}/namespaces/{namespace}/{resource}", apisgvnrfunc},
				{"GET", "/apis/{group}/{version}/namespaces/{namespace}/{resource}/{name}", apisgvnrnfunc},
				{"GET", "/apis/core/v1/namespaces/{namespace}/configmap/{name}", apiscorev1configmapfunc},
			},
			req:  httptest.NewRequest("GET", "/apis/core/v1/namespaces/default/configmap/abc", nil),
			want: apiscorev1configmapfunc,
		},
		{
			registered: []reg{
				{"GET", "/apis/{path}*", apisfunc},
			},
			setnotfound: testnotfoundfunc,
			req:         httptest.NewRequest("GET", "/api/abc", nil),
			want:        testnotfoundfunc,
		},
		{
			registered: []reg{
				{"*", "/apis/{path}*", apisfunc},
			},
			setnotfound: testnotfoundfunc,
			req:         httptest.NewRequest("GET", "/apis/abc/def", nil),
			want:        apisfunc,
		},
		{
			registered: []reg{
				{"GET", "/apis/{path}*", apisfunc},
			},
			req:  httptest.NewRequest("GET", "/api/abc/def", nil),
			want: DefaultNotFoundHandler,
		},
	}
	for _, tt := range tests {
		t.Run(tt.req.URL.String(), func(t *testing.T) {
			m := NewRouter()
			if tt.setnotfound != nil {
				m.Notfound = tt.setnotfound
			}
			for _, reg := range tt.registered {
				if err := m.Register(reg.method, reg.path, reg.handler); err != nil {
					t.Error(err)
				}
			}
			ginc := &gin.Context{
				Request: tt.req,
				Params:  gin.Params{},
			}

			if got := m.Match(ginc); reflect.ValueOf(got).Pointer() != reflect.ValueOf(tt.want).Pointer() {
				t.Errorf("Router.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouter_MustRegister(t *testing.T) {
	ginfunc := func(c *gin.Context) { return }

	type args struct {
		path    string
		handler gin.HandlerFunc
	}
	tests := []struct {
		name       string
		registered []reg
		wantPanic  bool
	}{
		{
			registered: []reg{
				{"GET", "/apis/{path}*", ginfunc},
				{"POST", "/apis/{group}/{version}/namespaces/{namespace}/{resource}", ginfunc},
				{"DELETE", "/apis/{group}/{version}/namespaces/{namespace}/{resource}/{name}", ginfunc},
				{"PUT", "/apis/core/v1/namespaces/{namespace}/configmap/{name}", ginfunc},
				{"PATCH", "/apis/{name}*", ginfunc},
				{"GET", "/apis/{name}*", ginfunc},
			},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); tt.wantPanic && r == nil {
					t.Errorf("The code did not panic")
				}
			}()

			m := &Router{}
			for _, reg := range tt.registered {
				switch reg.method {
				case http.MethodGet:
					m.GET(reg.path, reg.handler)
				case http.MethodPost:
					m.POST(reg.path, reg.handler)
				case http.MethodPut:
					m.PUT(reg.path, reg.handler)
				case http.MethodDelete:
					m.DELETE(reg.path, reg.handler)
				}
			}
		})
	}
}
