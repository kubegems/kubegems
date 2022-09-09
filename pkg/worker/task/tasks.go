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

	"github.com/go-logr/logr"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/helm"
	"kubegems.io/kubegems/pkg/utils/redis"
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

func Run(ctx context.Context, rediscli *redis.Client,
	db *database.Database,
	gitp *git.SimpleLocalProvider,
	argocd *argo.Client,
	helmOptions *helm.Options,
	agents *agents.ClientSet,
) error {

	p := &ProcessorContext{
		server:    workflow.NewServerFromRedisClient(rediscli.Client),
		client:    workflow.NewClientFromRedisClient(rediscli.Client),
		rediscli:  rediscli,
		crontasks: []CronTask{},
		Logger:    log.FromContextOrDiscard(ctx),
	}

	// 注册支持的处理函数
	taskers := []Tasker{
		// 示例
		&SampleTasker{},
		// application 应用部署相关
		MustNewApplicationTasker(db, gitp, argocd, rediscli, agents),
		// task-archive 持久化过期任务至database
		NewTaskArchiverTasker(db, rediscli),
		// chart-sync 同步helmchart
		&HelmSyncTasker{DB: db, ChartRepoUrl: helmOptions.Addr},
		// cluster
		&ClusterSyncTasker{DB: db, cs: agents},
	}
	if err := p.RegisterTasker(taskers...); err != nil {
		return err
	}
	return p.Run(ctx)
}

type ProcessorContext struct {
	Logger    logr.Logger
	server    *workflow.Server
	client    *workflow.Client
	rediscli  *redis.Client
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

func (p *ProcessorContext) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	// 启动 worker 消费
	eg.Go(func() error {
		return p.server.Run(ctx)
	})
	// 启动定时任务
	eg.Go(func() error {
		return p.RunCronTasksWithLock(ctx)
	})
	return eg.Wait()
}

// 由于worker是多副本的，且crontask 只能在 worker上运行cron。
// 为了避免多个worker都执行，使用redis 锁选择一个worker来触发这些crontask
func (p *ProcessorContext) RunCronTasksWithLock(ctx context.Context) error {
	rs := redsync.New(goredis.NewPool(p.rediscli.Client))
	mutex := rs.NewMutex("crontask-client-lock")
	// 如果其他副本获取到锁，这里会一直阻塞
	if err := mutex.LockContext(ctx); err != nil {
		return err
	}
	defer mutex.Unlock()

	for _, crontask := range p.crontasks {
		if err := p.client.SubmitCronTask(ctx, crontask.Task, crontask.CronExp); err != nil {
			p.Logger.Error(err, "submit crontask failed", "exp", crontask.CronExp)
		}
	}

	<-ctx.Done()
	return nil
}
