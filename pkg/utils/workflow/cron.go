// Copyright 2024 The kubegems.io Authors
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

package workflow

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"kubegems.io/kubegems/pkg/log"
)

type CronClient struct {
	Client
	crontab *cron.Cron
}

func NewCronSubmiter(client Client) *CronClient {
	c := &CronClient{
		Client:  client,
		crontab: cron.New(),
	}
	go c.crontab.Run()
	return c
}

func (s *CronClient) SubmitCronTask(ctx context.Context, task Task, crontabexp string) error {
	log := log.FromContextOrDiscard(ctx).WithValues("task", task, "cron", crontabexp)
	log.Info("register cron task")
	_, err := s.crontab.AddFunc(crontabexp, func() {
		log.Info("trigger a cron task run", "now", time.Now())
		if err := s.Client.SubmitTask(ctx, task); err != nil {
			log.Error(err, "run crontab task failed")
		}
	})
	return err
}
