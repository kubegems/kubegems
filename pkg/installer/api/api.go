package api

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"kubegems.io/kubegems/pkg/utils/httputil/apiutil"
	"kubegems.io/kubegems/pkg/utils/route"
)

type Options struct {
	Listen    string `json:"listen,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

func DefaultOptions() *Options {
	return &Options{
		Listen:    ":8080",
		Namespace: "kubegems-local",
	}
}

func Run(ctx context.Context, options *Options, cachedir string) error {
	pm, err := NewPluginManager(options.Namespace, cachedir)
	if err != nil {
		return err
	}
	modules := []apiutil.RestModule{&PluginsAPI{manager: pm}}

	server := http.Server{
		Addr:    options.Listen,
		Handler: apiutil.NewRestfulAPI("", nil, modules),
	}
	go func() {
		<-ctx.Done()
		server.Close()
	}()

	log := logr.FromContextOrDiscard(ctx)
	log.Info("listening", "addr", server.Addr)

	return server.ListenAndServe()
}

type PluginsAPI struct {
	manager *pluginManager
}

func (o *PluginsAPI) RegisterRoute(rg *route.Group) {
	rg.AddSubGroup(
		route.NewGroup("/plugins").AddRoutes(
			route.GET("").To(o.ListPlugins),
			route.GET("/{name}").To(o.GetPlugin),
			route.PUT("/{name}").To(o.EnablePlugin),
			route.DELETE("/{name}").To(o.RemovePlugin),
		),
		route.NewGroup("/repos").AddRoutes(
			route.POST("").To(o.RepoAdd),
			route.GET("").To(o.RepoList),
			route.POST("/{name}").To(o.RepoUpdate).Accept("*/*"),
			route.DELETE("/{name}").To(o.RepoRemove),
		),
	)
}
