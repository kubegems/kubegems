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
	"time"

	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/utils/workflow"
)

// 用于转移超过时间的任务记录至database
type TaskArchiverTasker struct {
	Databse *database.Database
	taskcli *workflow.Client
	Redis   *redis.Client
}

func NewTaskArchiverTasker(databse *database.Database, redis *redis.Client) *TaskArchiverTasker {
	return &TaskArchiverTasker{
		taskcli: workflow.NewClientFromRedisClient(redis.Client),
		Databse: databse,
		Redis:   redis,
	}
}

func (t *TaskArchiverTasker) ArchiveOutdated(ctx context.Context) error {
	// list all tasks
	tasks, err := t.taskcli.ListTasks(ctx, "", "")
	if err != nil {
		return err
	}
	log := log.FromContextOrDiscard(ctx)
	for _, task := range tasks {
		needarchive := time.Since(task.CreationTimestamp.Time) > 5*24*time.Hour // 5 days
		if needarchive {
			// todo : 存储历史任务
			log.Info("expired task", "name", task.Name, "creationtimestamp", task.CreationTimestamp)
			// 删除该记录
			if err := t.taskcli.RemoveTask(ctx, task.Group, task.Name, task.UID); err != nil {
				log.Error(err, "remove expired task")
			}
		}
	}
	return nil
}

const RemoveConsumerDuration = 5 * time.Minute

// RemoveOffline 删除长时间不活跃的worker正在处理的任务
func (t *TaskArchiverTasker) RemoveOffline(ctx context.Context) error {
	// 查看长时间未连接的worker
	streamingkey := "/workflow/submit"

	log := log.FromContextOrDiscard(ctx)

	// https://redis.io/commands/xinfo-consumers
	consumers, err := t.Redis.Client.XInfoConsumers(ctx, streamingkey, workflow.DefaultGroup).Result()
	if err != nil {
		return err
	}
	for _, consumer := range consumers {
		// milliseconds
		offlinetime := time.Duration(consumer.Idle) * time.Microsecond

		log.Info("consumer status", "name", consumer.Name, "idle", offlinetime)

		if offlinetime > RemoveConsumerDuration {
			// todo: 转移任务

			// 删除 consumer
			// https://redis.io/commands/xgroup-delconsumer
			if _, err := t.Redis.Client.XGroupDelConsumer(ctx, streamingkey, workflow.DefaultGroup, consumer.Name).Result(); err != nil {
				log.Error(err, "remove expired consumer")
				return err
			}
		}
	}
	return nil
}

const TaskFunction_ArchiveTasks = "task-archive"

func (t *TaskArchiverTasker) ProvideFuntions() map[string]interface{} {
	return map[string]interface{}{
		TaskFunction_ArchiveTasks: t.ArchiveOutdated,
	}
}

func (t *TaskArchiverTasker) Crontasks() map[string]Task {
	return map[string]Task{
		"@every 1h": {
			Name:  "task-archive",
			Group: "tasks",
			Steps: []workflow.Step{
				{
					Name:     "archive",
					Function: TaskFunction_ArchiveTasks,
				},
			},
		},
	}
}
