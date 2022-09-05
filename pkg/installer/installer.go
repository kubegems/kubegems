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

	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/installer/api"
	"kubegems.io/kubegems/pkg/installer/controller"
	"kubegems.io/kubegems/pkg/log"
)

type Options struct {
	Controller *controller.Options `json:",inline"`
	API        *api.Options        `json:",inline"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Controller: controller.NewDefaultOptions(),
		API:        api.DefaultOptions(),
	}
}

func Run(ctx context.Context, options *Options) error {
	ctx = log.NewContext(ctx, log.LogrLogger)

	eg, ctx := errgroup.WithContext(ctx)
	// eg.Go(func() error {
	// 	return controller.Run(ctx, options.Controller)
	// })
	eg.Go(func() error {
		return api.Run(ctx, options.API, options.Controller.PluginsDir)
	})
	return eg.Wait()
}
