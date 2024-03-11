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

package tasks

import (
	"context"

	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/apps/application"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/msgbus/switcher"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"kubegems.io/kubegems/pkg/utils/retry"
	"kubegems.io/kubegems/pkg/utils/workflow"
)

type TaskProducer struct {
	Bus             *switcher.MessageSwitcher
	ApplicationTask *application.TaskProcessor
}

func RunTasksCollector(ctx context.Context, ms *switcher.MessageSwitcher) error {
	task := &TaskProducer{
		Bus: ms,
		ApplicationTask: &application.TaskProcessor{
			Workflowcli: workflow.NewDefaultRemoteClient(),
		},
	}
	return task.Run(ctx)
}

func (p *TaskProducer) Run(ctx context.Context) error {
	return retry.Always(func() error {
		return p.runapplicationtasks(ctx)
	})
}

func (p *TaskProducer) runapplicationtasks(ctx context.Context) error {
	if err := p.ApplicationTask.WatchTasks(ctx, application.PathRef{}, "", func(_ context.Context, task *workflow.Task) error {
		// skip
		if len(task.Addtionals) == 0 {
			return nil
		}
		msg := &msgbus.NotifyMessage{
			MessageType: msgbus.Changed,
			EventKind:   msgbus.Update,
			InvolvedObject: &msgbus.InvolvedObject{
				Cluster: task.Addtionals[application.AnnotationCluster],
				Group:   gems.GroupName,
				Kind:    "Task",
				NamespacedName: msgbus.NamespacedNameFrom(
					task.Addtionals[application.AnnotationNamespace],
					task.Addtionals[application.LabelApplication],
				),
			},
			Content: task,
		}
		p.Bus.DispatchMessage(msg)
		return nil
	}); err != nil {
		log.Error(err, "task producer failed to watch")
		return err
	}
	return nil
}
