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

package api

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"kubegems.io/kubegems/pkg/installer/pluginmanager"
	"kubegems.io/kubegems/pkg/utils/httputil/apiutil"
	"kubegems.io/kubegems/pkg/utils/route"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	Listen    string `json:"listen,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

func DefaultOptions() *Options {
	return &Options{
		Listen:    ":8080",
		Namespace: "kubegems-installer",
	}
}

func Run(ctx context.Context, options *Options, cachedir string) error {
	pm, err := pluginmanager.DefaultPluginManager(cachedir)
	if err != nil {
		return err
	}
	modules := []apiutil.RestModule{&PluginsAPI{PM: pm}}

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
	PM *pluginmanager.PluginManager
}

func NewPluginsAPI(cli client.Client) *PluginsAPI {
	return &PluginsAPI{PM: &pluginmanager.PluginManager{Client: cli}}
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
			route.GET("/{name}").To(o.RepoGet),
			route.POST("/{name}").To(o.RepoUpdate).Accept("*/*"),
			route.DELETE("/{name}").To(o.RepoRemove),
		),
	)
}
