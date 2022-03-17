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
	"kubegems.io/pkg/utils/git"
	_ "kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/pprof"
	"kubegems.io/pkg/utils/prometheus/exporter"
	"kubegems.io/pkg/utils/redis"
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

	exporterHandler := exporter.NewHandler("gems_worker", map[string]exporter.Collectorfunc{})

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	eg.Go(func() error {
		return exporterHandler.Run(ctx, options.Exporter)
	})
	eg.Go(func() error {
		return task.Run(ctx, deps.Redis, deps.Databse, deps.Git, deps.Argocli, options.AppStore, deps.Agentscli)
	})
	return eg.Wait()
}
