package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"sort"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client struct {
	backend Backend
	crontab *cron.Cron
}

func NewClient(options *Options) *Client {
	backend := NewRedisBackend(options.Addr, options.Username, options.Password)
	return NewClientFromBackend(backend)
}

func NewClientFromRedisClient(cli *redis.Client) *Client {
	return NewClientFromBackend(NewRedisBackendFromClient(cli))
}

func NewClientFromBackend(backend Backend) *Client {
	cli := &Client{
		backend: backend,
		crontab: cron.New(),
	}
	go cli.crontab.Run()
	return cli
}

func (c *Client) SubmitCronTask(ctx context.Context, task Task, crontabexp string) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("task", task, "cron", crontabexp)
	log.Info("register cron task")
	_, err := c.crontab.AddFunc(crontabexp, func() {
		log.Info("trigger a cron task run", "now", time.Now())
		if err := c.SubmitTask(ctx, task); err != nil {
			log.Error(err, "run crontab task failed")
		}
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) SubmitTask(ctx context.Context, task Task) error {
	if task.Name == "" {
		return errors.New("empty task name")
	}
	task.CreationTimestamp = metav1.Now()
	if task.UID == "" {
		task.UID = uuid.New().String()
	}
	if task.Status == nil {
		task.Status = &TaskStatus{Status: TaskStatusPending}
	}
	content, err := json.Marshal(task)
	if err != nil {
		return err
	}

	taskjkey := path.Join(task.Group, task.Name, task.UID)
	if err := c.backend.Put(ctx, taskjkey, content); err != nil {
		return err
	}
	return c.backend.Pub(ctx, "submit", "", content)
}

func (c *Client) ListTasks(ctx context.Context, group, name string) ([]Task, error) {
	keyprefix := group + "/" + name
	if group == "" && name == "" {
		keyprefix = ""
	}

	kvs, err := c.backend.List(ctx, keyprefix)
	if err != nil {
		return nil, err
	}

	list := make([]Task, 0, len(kvs))
	for _, v := range kvs {
		task := Task{}
		_ = json.Unmarshal(v, &task)
		list = append(list, task)
	}

	// sort
	sort.Slice(list, func(i, j int) bool {
		return !list[i].CreationTimestamp.Before(&list[j].CreationTimestamp)
	})
	return list, nil
}

func (c *Client) RemoveTask(ctx context.Context, group, name string, uid string) error {
	keyprefix := path.Join(group, name, uid)
	return c.backend.Del(ctx, keyprefix)
}

func (c *Client) WatchTasks(ctx context.Context, group, name string, onchange func(ctx context.Context, task *Task) error) error {
	keyprefix := group + "/" + name
	if group == "" && name == "" {
		keyprefix = ""
	}

	return c.backend.Watch(ctx, keyprefix, func(ctx context.Context, _ string, val []byte) error {
		task := &Task{}
		if err := json.Unmarshal(val, task); err != nil {
			return err
		}
		return onchange(ctx, task)
	})
}
