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

package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"sort"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client interface {
	SubmitTask(ctx context.Context, task Task) error
	ListTasks(ctx context.Context, group, name string) ([]Task, error)
	RemoveTask(ctx context.Context, group, name string, uid string) error
	WatchTasks(ctx context.Context, group, name string, onchange func(ctx context.Context, task *Task) error) error
}

type DefaultClient struct {
	backend Backend
}

func NewClientFromBackend(backend Backend) Client {
	cli := &DefaultClient{backend: backend}
	return cli
}

func (c *DefaultClient) SubmitTask(ctx context.Context, task Task) error {
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

func (c *DefaultClient) ListTasks(ctx context.Context, group, name string) ([]Task, error) {
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

func (c *DefaultClient) RemoveTask(ctx context.Context, group, name string, uid string) error {
	keyprefix := path.Join(group, name, uid)
	return c.backend.Del(ctx, keyprefix)
}

func (c *DefaultClient) WatchTasks(ctx context.Context, group, name string, onchange func(ctx context.Context, task *Task) error) error {
	keyprefix := group + "/" + name
	if group == "" && name == "" {
		keyprefix = ""
	}

	return c.backend.Watch(ctx, keyprefix, func(ctx context.Context, _ string, val []byte) error {
		if len(val) == 0 {
			// is delete
			return nil
		}
		task := &Task{}
		if err := json.Unmarshal(val, task); err != nil {
			return err
		}
		return onchange(ctx, task)
	})
}
