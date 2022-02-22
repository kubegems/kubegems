package worker

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/git"
	_ "kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/pprof"
	"kubegems.io/pkg/utils/prometheus"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/worker/collector"
	"kubegems.io/pkg/worker/dump"
	"kubegems.io/pkg/worker/resourcelist"
	"kubegems.io/pkg/worker/task"
)

type Dependencies struct {
	Redis     *redis.Client
	Databse   *database.Database
	Argocli   *argo.Client
	Git       *git.SimpleLocalProvider
	Agentscli *agents.ClientSet
	Logger    logr.Logger
}

func prepareDependencies(ctx context.Context, options *Options) (*Dependencies, error) {
	// logger
	log.SetLevel(options.LogLevel)

	// redis
	rediscli, err := redis.NewClient(options.Redis)
	if err != nil {
		return nil, err
	}
	// database
	databasecli, err := database.NewDatabase(options.Mysql)
	if err != nil {
		return nil, err
	}
	// agent client
	agentclientset, err := agents.NewClientSet(databasecli)
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
	return &Dependencies{
		Redis:     rediscli,
		Databse:   databasecli,
		Argocli:   argocli,
		Git:       gitprovider,
		Agentscli: agentclientset,
	}, nil
}

func Run(ctx context.Context, options *Options) error {
	ctx = logr.NewContext(ctx, log.LogrLogger)
	deps, err := prepareDependencies(ctx, options)
	if err != nil {
		return err
	}

	collector.Init(deps.Argocli, deps.Databse)
	exporter.SetNamespace("gems_worker")
	exporter.RegisterCollector("cluster", true, collector.NewClusterCollector(deps.Agentscli))
	exporter.RegisterCollector("environment", true, collector.NewEnvironmentCollector())
	exporter.RegisterCollector("user", true, collector.NewUserCollector())
	exporter.RegisterCollector("application", true, collector.NewApplicationCollector())
	exporterHandler := exporter.NewHandler(options.Exporter.IncludeExporterMetrics, options.Exporter.MaxRequests, log.GlobalLogger.Sugar())

	// dump
	dump := &dump.Dump{Options: options.Dump, DB: deps.Databse}
	dump.Start()

	// resource cache
	cache := resourcelist.NewResourceCache(deps.Databse, deps.Agentscli)
	cache.Start()

	http.HandleFunc("/refresh", func(w http.ResponseWriter, _ *http.Request) {
		if err := cache.EnvironmentSync(); err != nil {
			w.Write([]byte(err.Error()))
		} else {
			w.Write([]byte("ok"))
		}
		if err := cache.WorkloadSync(); err != nil {
			w.Write([]byte(err.Error()))
		} else {
			w.Write([]byte("ok"))
		}
	})

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	eg.Go(func() error {
		return prometheus.RunExporter(ctx, options.Exporter, exporterHandler)
	})
	eg.Go(func() error {
		return task.Run(ctx, deps.Redis, deps.Databse, deps.Git, deps.Argocli, options.AppStore, deps.Agentscli)
	})
	return eg.Wait()
}
