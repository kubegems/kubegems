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
