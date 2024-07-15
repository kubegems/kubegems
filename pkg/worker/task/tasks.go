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

package task

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/helm"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/utils/system"
	"kubegems.io/kubegems/pkg/utils/workflow"
)

type Tasker interface {
	ProvideFuntions() map[string]interface{}
}

// 某些任务处理本身会有定时任务，可以实现该接口
type CronTasker interface {
	Crontasks() map[string]Task // cron表达式 -> 任务
}

type (
	Task     = workflow.Task
	CronTask struct {
		CronExp string
		Task    Task
	}
)

func Run(ctx context.Context,
	listen string,
	rediscli *redis.Client,
	db *database.Database,
	gitp git.Provider,
	argocd *argo.Client,
	helmOptions *helm.Options,
	agents *agents.ClientSet,
) error {
	log := logr.FromContextOrDiscard(ctx).WithName("worker")
	var backend workflow.Backend
	var lock DistributedLock
	if rediscli != nil {
		log.Info("use redis backend")
		backend = workflow.NewRedisBackendFromClient(rediscli.Client)
		lock = NewRedisLock(rediscli)
	} else {
		log.Info("use inmemory backend")
		backend = workflow.NewInmemoryBackend(ctx)
	}
	workflowcli := workflow.NewClientFromBackend(backend)
	p := &ProcessorContext{
		server:    workflow.NewServerFromBackend(backend),
		client:    workflow.NewCronSubmiter(workflowcli),
		crontasks: []CronTask{},
	}

	// 注册支持的处理函数
	taskers := []Tasker{
		// 示例
		&SampleTasker{},
		// application 应用部署相关
		MustNewApplicationTasker(db, gitp, argocd, workflowcli, agents),
		// task-archive 持久化过期任务至database
		NewTaskArchiverTasker(db, workflowcli),
		// chart-sync 同步helmchart
		&HelmSyncTasker{DB: db, ChartRepoUrl: helmOptions.Addr},
		// cluster
		&ClusterSyncTasker{DB: db, cs: agents},
		// alertrule
		&AlertRuleSyncTasker{DB: db, cs: agents},
	}
	if err := p.RegisterTasker(taskers...); err != nil {
		return err
	}
	return p.Run(ctx, listen, lock)
}

type ProcessorContext struct {
	server    *workflow.Server
	client    *workflow.CronClient
	crontasks []CronTask
}

func (p *ProcessorContext) RegisterTasker(taskers ...Tasker) error {
	for _, t := range taskers {
		// 注册定时任务
		if cront, ok := t.(CronTasker); ok {
			for cronexp, task := range cront.Crontasks() {
				p.crontasks = append(p.crontasks, CronTask{CronExp: cronexp, Task: task})
			}
		}
		// 注册支持函数
		for k, v := range t.ProvideFuntions() {
			if err := p.server.Register(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *ProcessorContext) Run(ctx context.Context, listen string, lock DistributedLock) error {
	eg, ctx := errgroup.WithContext(ctx)
	// 启动 worker 消费
	eg.Go(func() error {
		return p.server.Run(ctx)
	})
	// 启动定时任务
	eg.Go(func() error {
		return p.RunCronTasksWithLock(ctx, lock)
	})
	// 启动http服务
	eg.Go(func() error {
		return p.RunHTTP(ctx, listen)
	})
	return eg.Wait()
}

type DistributedLock interface {
	LockContext(ctx context.Context) error
	Unlock(ctx context.Context) (bool, error)
}

type RedisLock struct {
	mu *redsync.Mutex
}

func NewRedisLock(rediscli *redis.Client) *RedisLock {
	rs := redsync.New(goredis.NewPool(rediscli.Client))
	return &RedisLock{mu: rs.NewMutex("crontask-client-lock")}
}

func (r *RedisLock) LockContext(ctx context.Context) error {
	return r.mu.LockContext(ctx)
}

func (r *RedisLock) Unlock(ctx context.Context) (bool, error) {
	return r.mu.Unlock()
}

// 由于worker是多副本的，且crontask 只能在 worker上运行cron。
// 为了避免多个worker都执行，使用redis 锁选择一个worker来触发这些crontask
func (p *ProcessorContext) RunCronTasksWithLock(ctx context.Context, lock DistributedLock) error {
	log := logr.FromContextOrDiscard(ctx)
	if lock != nil {
		log.Info("try to lock crontask")
		if err := lock.LockContext(ctx); err != nil {
			return err
		}
		defer lock.Unlock(ctx)
	} else {
		log.Info("lock not found, skip crontask lock")
	}

	for _, crontask := range p.crontasks {
		if err := p.client.SubmitCronTask(ctx, crontask.Task, crontask.CronExp); err != nil {
			log.Error(err, "submit crontask failed", "exp", crontask.CronExp)
		}
	}

	<-ctx.Done()
	return nil
}

func (p *ProcessorContext) RunHTTP(ctx context.Context, addr string) error {
	h := workflow.NewRemoteClientServer(p.client)
	return system.ListenAndServeContext(ctx, addr, nil, h.Handler())
}

func (p *ProcessorContext) handler() http.Handler {
	mux := http.NewServeMux()

	return mux
}
