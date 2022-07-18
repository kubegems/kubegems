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
	"context"
	"net/http"
	"sync"
)

// using get pathvars from context.Context returns map[string]string{}
var ContextKeyPathVars = struct{ name string }{name: "path variables"}

//  ServeMux is a http.ServeMux like library,but support path variable
type ServeMux struct {
	mu      sync.RWMutex
	matcher matcher
}

func NewServeMux() *ServeMux {
	return &ServeMux{matcher: matcher{root: &node{}}}
}

func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	matched, val, vars := mux.matcher.Match(r.URL.Path)
	if matched {
		r = r.WithContext(context.WithValue(r.Context(), ContextKeyPathVars, vars))
		val.(http.Handler).ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

func (mux *ServeMux) Handle(pattern string, handler http.Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	_ = mux.matcher.Register(pattern, handler)
}

func (mux *ServeMux) HandlerFunc(pattern string, handler func(w http.ResponseWriter, r *http.Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	mux.Handle(pattern, http.HandlerFunc(handler))
}
