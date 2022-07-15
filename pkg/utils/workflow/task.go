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
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// task 存储路径为  /{prefix}/{task-name}/{uid}
// Task 任务，一个task为一个整体性的任务，其下可以包含子任务，分支任务，嵌套任务， 触发其他任务等。
// 任务支持重试
// 支持失败策略
// task 之间可以进行值传递
// task 可以设置为同步执行
// task 可以设置为定时执行

type Task struct {
	UID               string            `json:"uid,omitempty"`
	Name              string            `json:"name,omitempty"`  // 任务名称，例如 更新镜像，同步数据等。
	Group             string            `json:"group,omitempty"` // 任务类型分组
	Steps             []Step            `json:"steps,omitempty"`
	CreationTimestamp metav1.Time       `json:"creationTimestamp,omitempty"`
	Addtionals        map[string]string `json:"addtionals,omitempty"` // 额外信息
	Status            *TaskStatus       `json:"status,omitempty"`
}

type Step struct {
	Name     string        `json:"name,omitempty"`
	Function string        `json:"function,omitempty"` // 任务所使用的 函数/组件/插件
	Args     []interface{} `json:"args,omitempty"`     // 对应的参数
	SubSteps []Step        `json:"subSteps,omitempty"` // 子任务
	Status   *TaskStatus   `json:"status,omitempty"`
}

type jsonArgsTask struct {
	UID               string            `json:"uid,omitempty"`
	Name              string            `json:"name,omitempty"`
	Group             string            `json:"group,omitempty"`
	Steps             []*jsonArgsStep   `json:"steps,omitempty"`
	CreationTimestamp metav1.Time       `json:"creationTimestamp,omitempty"`
	Addtionals        map[string]string `json:"addtionals,omitempty"` // 额外信息
	Status            TaskStatus        `json:"status,omitempty"`
}

type jsonArgsStep struct {
	Name     string            `json:"name,omitempty"`
	Function string            `json:"function,omitempty"`
	Args     []json.RawMessage `json:"args,omitempty"`
	SubSteps []*jsonArgsStep   `json:"subSteps,omitempty"`
	Status   TaskStatus        `json:"status,omitempty"`
	Timeout  time.Duration     `json:"timeout,omitempty"` // 任务执行超时
}

func ArgsOf(args ...interface{}) []interface{} {
	return args
}

type TaskStatusCode string

const (
	TaskStatusPending TaskStatusCode = "Pending"
	TaskStatusRunning TaskStatusCode = "Running"
	TaskStatusSuccess TaskStatusCode = "Success"
	TaskStatusError   TaskStatusCode = "Error"
)

type TaskStatus struct {
	StartTimestamp  metav1.Time    `json:"startTimestamp,omitempty"`
	FinishTimestamp metav1.Time    `json:"finishTimestamp,omitempty"`
	Status          TaskStatusCode `json:"status,omitempty"`
	Result          []interface{}  `json:"result,omitempty"`
	Executer        string         `json:"executer,omitempty"`
	Message         string         `json:"message,omitempty"`
}
