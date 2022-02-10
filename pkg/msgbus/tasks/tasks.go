package tasks

import (
	"context"

	"kubegems.io/pkg/log"
	"kubegems.io/pkg/msgbus/switcher"
	"kubegems.io/pkg/service/handlers/application"
	"kubegems.io/pkg/utils/msgbus"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/retry"
	"kubegems.io/pkg/utils/workflow"
)

type TaskProducer struct {
	Bus             *switcher.MessageSwitcher
	ApplicationTask *application.TaskProcessor
}

func RunTasksCollector(ctx context.Context, ms *switcher.MessageSwitcher, redis *redis.Client) error {
	task := &TaskProducer{
		Bus: ms,
		ApplicationTask: &application.TaskProcessor{
			Workflowcli: workflow.NewClientFromRedisClient(redis.Client),
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
				Group:   "gems.cloudminds.com",
				Kind:    "Task",
				NamespacedName: msgbus.NamespacedNameFrom(
					task.Addtionals[application.AnnotationNamespace],
					task.Addtionals[application.ArgoLabelApplication],
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
