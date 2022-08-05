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

package gitserver

import (
	"context"
	"net/http"
)

type Options struct {
	Listen            string // http server listen address
	UseGitHTTPBackend bool   // use git httpbackend CGI to serve git http requests
}

func NewDefaultOptions() *Options {
	return &Options{
		Listen:            ":8080",
		UseGitHTTPBackend: false,
	}
}

type Server struct {
	GitBase string
	LFS     LFSMetaManager
}

func (s *Server) Run(ctx context.Context, opts *Options) error {
	if opts == nil {
		opts = NewDefaultOptions()
	}
	httpserver := &http.Server{
		Addr:    opts.Listen,
		Handler: s.routes(s.LFS != nil, opts.UseGitHTTPBackend),
	}
	go func() {
		<-ctx.Done()
		httpserver.Shutdown(ctx)
	}()
	return httpserver.ListenAndServe()
}
