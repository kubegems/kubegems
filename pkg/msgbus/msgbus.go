package msgbus

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/msgbus/api"
	"kubegems.io/pkg/msgbus/applications"
	"kubegems.io/pkg/msgbus/options"
	"kubegems.io/pkg/msgbus/switcher"
	"kubegems.io/pkg/msgbus/tasks"
	"kubegems.io/pkg/msgbus/workloads"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/pprof"
	"kubegems.io/pkg/utils/redis"
)

func Run(ctx context.Context, options *options.Options) error {
	ctx = log.NewContext(ctx, log.LogrLogger)
	// prepare
	deps, err := prepareDependencies(ctx, options)
	if err != nil {
		return fmt.Errorf("init dependencies failed: %v", err)
	}

	// errgroup .WithContext(ctx) 返回的 ctx 会在任意协程返回 error时取消，其他正常退出
	// 若要在发生错误时正常退出，所有routine 都需要能够正确处理 ctx Done() 并平滑退出
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return api.RunGinServer(ctx, options, deps.Database, deps.Redis, deps.Switcher)
	})
	eg.Go(func() error {
		return workloads.RunWorkloadCollector(ctx, deps.AgentsClientSet, deps.Switcher)
	})
	eg.Go(func() error {
		return applications.RunApplicationCollector(ctx, deps.Switcher, deps.Argo)
	})
	eg.Go(func() error {
		return tasks.RunTasksCollector(ctx, deps.Switcher, deps.Redis)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}

type Dependencies struct {
	Database        *database.Database
	Argo            *argo.Client
	AgentsClientSet *agents.ClientSet
	Redis           *redis.Client
	Switcher        *switcher.MessageSwitcher
}

func prepareDependencies(ctx context.Context, options *options.Options) (*Dependencies, error) {
	log.SetLevel(options.LogLevel)

	// 初始化Redis实例
	rediscli, err := redis.NewClient(options.Redis)
	if err != nil {
		return nil, err
	}

	// 初始化Mysql实例
	db, err := database.NewDatabase(options.Mysql)
	if err != nil {
		return nil, err
	}

	// 初始化 agent 客户端
	agentclientset, err := agents.NewClientSet(db, options.Agent)
	if err != nil {
		return nil, err
	}

	// argo 客户端
	argocli, err := argo.NewClient(ctx, options.Argo)
	if err != nil {
		return nil, fmt.Errorf("初始化argocd client错误 %v", err)
	}

	// switcher 实例
	switcher := switcher.NewMessageSwitch(ctx, db)

	deps := &Dependencies{
		Database:        db,
		Argo:            argocli,
		AgentsClientSet: agentclientset,
		Redis:           rediscli,
		Switcher:        switcher,
	}
	return deps, nil
}
