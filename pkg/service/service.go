package service

import (
	"context"
	"fmt"

	kialiconfig "github.com/kiali/kiali/config"
	"golang.org/x/sync/errgroup"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/options"
	"kubegems.io/pkg/service/routers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/git"
	_ "kubegems.io/pkg/utils/kube" // 用于 AddToSchema
	"kubegems.io/pkg/utils/pprof"
	"kubegems.io/pkg/utils/prometheus/exporter"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/tracing"
)

type Dependencies struct {
	Options          *options.Options
	Redis            *redis.Client
	Databse          *database.Database
	Argocli          *argo.Client
	Git              *git.SimpleLocalProvider
	Agentscli        *agents.ClientSet
	DyConfigProvider options.DynamicConfigurationProviderIface
}

func prepareDependencies(ctx context.Context, opts *options.Options) (*Dependencies, error) {
	// logger
	log.SetLevel(opts.LogLevel)

	// tracing
	tracing.SetGlobal(ctx)

	// redis
	rediscli, err := redis.NewClient(opts.Redis)
	if err != nil {
		return nil, err
	}

	// database
	db, err := database.NewDatabase(opts.Mysql)
	if err != nil {
		return nil, err
	}
	// agents
	agentclientset, err := agents.NewClientSet(db)
	if err != nil {
		return nil, err
	}
	// git
	gitprovider, err := git.NewProvider(opts.Git)
	if err != nil {
		return nil, err
	}
	// argo
	argocli, err := argo.NewClient(ctx, opts.Argo)
	if err != nil {
		return nil, err
	}

	dyConfigProvider := options.NewDatabaseDynamicConfigurationProvider(db.DB())

	deps := &Dependencies{
		Redis:            rediscli,
		Databse:          db,
		Argocli:          argocli,
		Git:              gitprovider,
		Agentscli:        agentclientset,
		DyConfigProvider: dyConfigProvider,
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

	router := &routers.Router{
		Opts:     opts,
		Agents:   deps.Agentscli,
		Argo:     deps.Argocli,
		Database: deps.Databse,
		Redis:    deps.Redis,
		DyConfig: deps.DyConfigProvider,
	}

	exporterHandler := exporter.NewHandler("gems_server", map[string]exporter.Collectorfunc{
		"request":     exporter.NewRequestCollector(),
		"cluster":     exporter.NewClusterCollector(deps.Agentscli, deps.Databse),
		"environment": exporter.NewEnvironmentCollector(deps.Databse),
		"user":        exporter.NewUserCollector(deps.Databse),
		"application": exporter.NewApplicationCollector(deps.Argocli),
	})

	// run
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return router.Run(ctx, opts.System)
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
