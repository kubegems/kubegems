package msgbus

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/kubegems/gems/pkg/kubeclient"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/msgbus/api"
	"github.com/kubegems/gems/pkg/msgbus/applications"
	"github.com/kubegems/gems/pkg/msgbus/options"
	"github.com/kubegems/gems/pkg/msgbus/switcher"
	"github.com/kubegems/gems/pkg/msgbus/tasks"
	"github.com/kubegems/gems/pkg/msgbus/workloads"
	"github.com/kubegems/gems/pkg/utils/agents"
	"github.com/kubegems/gems/pkg/utils/argo"
	"github.com/kubegems/gems/pkg/utils/database"
	"github.com/kubegems/gems/pkg/utils/pprof"
	"github.com/kubegems/gems/pkg/utils/redis"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, options *options.Options) error {
	// prepare
	deps, err := prepareDependencies(ctx, options)
	if err != nil {
		return fmt.Errorf("init dependencies failed: %v", err)
	}

	// errgroup .WithContext(ctx) 返回的 ctx 会在任意协程返回 error时取消，其他正常退出
	// 若要在发生错误时正常退出，所有routine 都需要能够正确处理 ctx Done() 并平滑退出
	eg, ctx := errgroup.WithContext(deps.Ctx)

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
	Ctx             context.Context
	Database        *database.Database
	Argo            *argo.Client
	AgentsClientSet *agents.ClientSet
	Redis           *redis.Client
	Switcher        *switcher.MessageSwitcher
}

func prepareDependencies(ctx context.Context, options *options.Options) (*Dependencies, error) {
	log.Update(options.DebugMode, options.LogLevel)
	logger, err := log.NewLogger(options.LogLevel, options.DebugMode)
	if err != nil {
		return nil, err
	}
	ctx = logr.NewContext(ctx, logger)

	// 初始化Redis实例
	rediscli, err := redis.NewClient(options.Redis)
	if err != nil {
		return nil, err
	}

	// 初始化Mysql实例
	models.InitRedis(rediscli) // 模型的hook中需要redis
	db, err := database.NewDatabase(options.Mysql, log.GlobalLogger)
	if err != nil {
		return nil, err
	}

	// 初始化 agent 客户端
	agentclientset, err := agents.NewClientSet(db, options.System)
	if err != nil {
		return nil, err
	}
	// fixme: 初始化kubeclient,如果不调用 kubeclient 的静态方法就可以移除了
	kubeclient.Init(agentclientset)

	// argo 客户端
	argocli, err := argo.NewClient(ctx, options.Argo, agentclientset)
	if err != nil {
		return nil, fmt.Errorf("初始化argocd client错误 %v", err)
	}

	// switcher 实例
	switcher := switcher.NewMessageSwitch(ctx, db)

	deps := &Dependencies{
		Ctx:             ctx,
		Database:        db,
		Argo:            argocli,
		AgentsClientSet: agentclientset,
		Redis:           rediscli,
		Switcher:        switcher,
	}
	return deps, nil
}
