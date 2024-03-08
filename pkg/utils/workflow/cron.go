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
