package worker

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/kubegems/gems/pkg/kubeclient"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/utils/agents"
	"github.com/kubegems/gems/pkg/utils/argo"
	"github.com/kubegems/gems/pkg/utils/database"
	"github.com/kubegems/gems/pkg/utils/exporter"
	"github.com/kubegems/gems/pkg/utils/git"
	_ "github.com/kubegems/gems/pkg/utils/kube"
	"github.com/kubegems/gems/pkg/utils/pprof"
	"github.com/kubegems/gems/pkg/utils/prometheus"
	"github.com/kubegems/gems/pkg/utils/redis"
	"github.com/kubegems/gems/pkg/worker/collector"
	"github.com/kubegems/gems/pkg/worker/dump"
	"github.com/kubegems/gems/pkg/worker/resourcelist"
	"github.com/kubegems/gems/pkg/worker/task"
	"golang.org/x/sync/errgroup"
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
	models.InitRedis(rediscli)
	databasecli, err := database.NewDatabase(options.Mysql)
	if err != nil {
		return nil, err
	}
	// agent client
	agentclientset, err := agents.NewClientSet(databasecli, options.System)
	if err != nil {
		return nil, err
	}
	// git
	gitprovider, err := git.NewProvider(options.Git)
	if err != nil {
		return nil, err
	}

	// argo
	argocli, err := argo.NewClient(ctx, options.Argo, agentclientset)
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

	// fixme: 初始化kubeclient,如果不调用 kubeclient 的静态方法就可以移除了
	kubeclient.Init(deps.Agentscli)

	collector.Init(deps.Argocli, deps.Databse)
	exporter.SetNamespace("gems_worker")
	exporter.RegisterCollector("cluster", true, collector.NewClusterCollector())
	exporter.RegisterCollector("environment", true, collector.NewEnvironmentCollector())
	exporter.RegisterCollector("user", true, collector.NewUserCollector())
	exporter.RegisterCollector("application", true, collector.NewApplicationCollector())
	exporterHandler := exporter.NewHandler(options.Exporter.IncludeExporterMetrics, options.Exporter.MaxRequests, log.GlobalLogger.Sugar())

	// dump
	dump := &dump.Dump{Options: options.Dump, DB: deps.Databse}
	dump.Start()

	// resource cache
	cache := resourcelist.NewResourceCache(deps.Databse)
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
		return task.Run(ctx, deps.Redis, deps.Databse, deps.Git, deps.Argocli, options.Appstore)
	})
	return eg.Wait()
}
