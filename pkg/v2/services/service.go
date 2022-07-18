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

package services

import (
	"context"
	"fmt"

	kialiconfig "github.com/kiali/kiali/config"
	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	_ "kubegems.io/kubegems/pkg/utils/kube" // 用于 AddToSchema
	"kubegems.io/kubegems/pkg/utils/pprof"
	"kubegems.io/kubegems/pkg/utils/prometheus/exporter"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/utils/tracing"
	"kubegems.io/kubegems/pkg/v2/model/validate"
	"kubegems.io/kubegems/pkg/v2/services/options"
)

type Dependencies struct {
	Options   *options.Options
	Redis     *redis.Client
	Databse   *database.Database
	Argocli   *argo.Client
	Git       *git.SimpleLocalProvider
	Agentscli *agents.ClientSet
}

func prepareDependencies(ctx context.Context, options *options.Options) (*Dependencies, error) {
	// logger
	log.SetLevel(options.LogLevel)

	// tracing
	tracing.SetGlobal(ctx)

	// redis
	rediscli, err := redis.NewClient(options.Redis)
	if err != nil {
		return nil, err
	}
	// database
	db, err := database.NewDatabase(options.Mysql)
	if err != nil {
		return nil, err
	}
	// agents
	agentclientset, err := agents.NewClientSet(db)
	if err != nil {
		return nil, err
	}
	// git
	gitprovider, err := git.NewProvider(options.Git)
	if err != nil {
		return nil, err
	}
	// argo
	argocli, err := argo.NewClient(ctx, options.Argo)
	if err != nil {
		return nil, err
	}
	deps := &Dependencies{
		Redis:     rediscli,
		Databse:   db,
		Argocli:   argocli,
		Git:       gitprovider,
		Agentscli: agentclientset,
	}
	return deps, nil
}

func Run(ctx context.Context, opts *options.Options) error {

	ctx = log.NewContext(ctx, log.LogrLogger)
	deps, err := prepareDependencies(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed init dependencies: %v", err)
	}

	// 依赖的kiali库用到，需要初始化
	// FIXME: 我们用到的配置较少，初始化时填入我们的配置，如
	// AppLabelName、InjectionLabelName、VersionLabelName、IstioIdentityDomain
	// 目前没啥问题
	kialiconfig.Set(kialiconfig.NewConfig())

	validate.InitValidator()
	rest := NewRest(deps, opts)

	exporterHandler := exporter.NewHandler("gems_server", map[string]exporter.Collectorfunc{
		"request": exporter.NewRequestCollector(),
	})

	// run
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return RunRest(ctx, opts.System, rest)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	eg.Go(func() error {
		// 启动prometheus exporter
		return exporterHandler.Run(ctx, opts.Exporter)
	})
	return eg.Wait()
}
