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

package installer

import (
	"context"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/apis/plugins"
	"kubegems.io/kubegems/pkg/installer/api"
	"kubegems.io/kubegems/pkg/installer/controller"
	"kubegems.io/kubegems/pkg/utils/pprof"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Options struct {
	Controller *controller.Options `json:",inline"`
	API        *api.Options        `json:",inline"`
	PluginsDir string              `json:"pluginsDir" description:"where plugins cached in"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Controller: controller.NewDefaultOptions(),
		API:        api.DefaultOptions(),
		PluginsDir: plugins.KubegemsPluginsCachePath,
	}
}

func Run(ctx context.Context, options *Options) error {
	ctx = logr.NewContext(ctx, zap.New(zap.UseDevMode(true)))
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return controller.Run(ctx, options.Controller, options.PluginsDir)
	})
	eg.Go(func() error {
		return api.Run(ctx, options.API, options.PluginsDir)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}
