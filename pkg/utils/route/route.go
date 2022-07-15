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

	"github.com/gin-gonic/gin"
)

type Router struct {
	methods  map[string]matcher
	Notfound gin.HandlerFunc
}

func NewRouter() *Router {
	return &Router{}
}

var DefaultNotFoundHandler = gin.WrapF(http.NotFound)

func (m *Router) GET(path string, handler gin.HandlerFunc) {
	m.MustRegister(http.MethodGet, path, handler)
}

func (m *Router) POST(path string, handler gin.HandlerFunc) {
	m.MustRegister(http.MethodPost, path, handler)
}

func (m *Router) PUT(path string, handler gin.HandlerFunc) {
	m.MustRegister(http.MethodPut, path, handler)
}

func (m *Router) DELETE(path string, handler gin.HandlerFunc) {
	m.MustRegister(http.MethodDelete, path, handler)
}

func (m *Router) PATCH(path string, handler gin.HandlerFunc) {
	m.MustRegister(http.MethodPatch, path, handler)
}

func (m *Router) ANY(path string, handler gin.HandlerFunc) {
	_ = m.Register(http.MethodGet, path, handler)
	_ = m.Register(http.MethodPost, path, handler)
	_ = m.Register(http.MethodPut, path, handler)
	_ = m.Register(http.MethodPatch, path, handler)
	_ = m.Register(http.MethodDelete, path, handler)
	_ = m.Register(http.MethodHead, path, handler)
	_ = m.Register(http.MethodOptions, path, handler)
	_ = m.Register(http.MethodTrace, path, handler)
	_ = m.Register(http.MethodConnect, path, handler)
}

func (m *Router) MustRegister(method, path string, handler gin.HandlerFunc) {
	if err := m.Register(method, path, handler); err != nil {
		panic(err)
	}
}

func (m *Router) Register(method, path string, handler gin.HandlerFunc) error {
	if m.methods == nil {
		m.methods = map[string]matcher{}
	}
	methodreg, ok := m.methods[method]
	if !ok {
		methodreg = matcher{root: &node{}}
		m.methods[method] = methodreg
	}
	return methodreg.Register(path, handler)
}

func (m *Router) Match(c *gin.Context) gin.HandlerFunc {
	// always match * method
	if regs, ok := m.methods["*"]; ok {
		if matched, val, vars := regs.Match(c.Request.URL.Path); matched {
			for k, v := range vars {
				c.Params = append(c.Params, gin.Param{Key: k, Value: v})
			}
			return val.(gin.HandlerFunc)
		}
	}

	if regs, ok := m.methods[c.Request.Method]; ok {
		if matched, val, vars := regs.Match(c.Request.URL.Path); matched {
			for k, v := range vars {
				c.Params = append(c.Params, gin.Param{Key: k, Value: v})
			}
			return val.(gin.HandlerFunc)
		}
	}

	return func() gin.HandlerFunc {
		if m.Notfound == nil {
			return DefaultNotFoundHandler
		}
		return m.Notfound
	}()
}
