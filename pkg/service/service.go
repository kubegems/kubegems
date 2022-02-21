package service

import (
	"context"
	"fmt"

	kialiconfig "github.com/kiali/kiali/config"
	"golang.org/x/sync/errgroup"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/service/options"
	"kubegems.io/pkg/service/routers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/git"
	_ "kubegems.io/pkg/utils/kube" // 用于 AddToSchema
	"kubegems.io/pkg/utils/pprof"
	"kubegems.io/pkg/utils/prometheus"
	"kubegems.io/pkg/utils/prometheus/collector"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/tracing"
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
	agentclientset, err := agents.NewClientSet(db, options.System)
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

func Run(ctx context.Context, options *options.Options) error {
	ctx = log.NewContext(ctx, log.LogrLogger)
	deps, err := prepareDependencies(ctx, options)
	if err != nil {
		return fmt.Errorf("failed init dependencies: %v", err)
	}

	// 测试模式需要初始化数据
	if options.TestMode {
		if err := models.MigrateDatabaseAndInitData(options.Mysql, options.Redis); err != nil {
			return err
		}
	}

	// 初始化数据库中的系统配置
	models.InitConfig(deps.Databse.DB())

	// 依赖的kiali库用到，需要初始化
	// FIXME: 我们用到的配置较少，初始化时填入我们的配置，如
	// AppLabelName、InjectionLabelName、VersionLabelName、IstioIdentityDomain
	// 目前没啥问题
	kialiconfig.Set(kialiconfig.NewConfig())

	router := &routers.Router{
		Opts:     options,
		Agents:   deps.Agentscli,
		Argo:     deps.Argocli,
		Database: deps.Databse,
		Redis:    deps.Redis,
	}
	// run
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return router.Run(ctx)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	eg.Go(func() error {
		exporter.SetNamespace("gems_server")
		exporter.RegisterCollector("request", true, collector.NewRequestCollector) // http exporter
		exporterHandler := exporter.NewHandler(options.Exporter.IncludeExporterMetrics, options.Exporter.MaxRequests, log.GlobalLogger.Sugar())
		// 启动prometheus exporter
		return prometheus.RunExporter(ctx, options.Exporter, exporterHandler)
	})
	return eg.Wait()
}
