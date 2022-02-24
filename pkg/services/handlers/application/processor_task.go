package application

import (
	"context"
	"fmt"
	"path"

	"kubegems.io/pkg/utils/workflow"
)

const (
	TaskAddtionalKeyCommiter = "committer"
)

type TaskProcessor struct {
	Workflowcli *workflow.Client
}

func (p *TaskProcessor) SubmitTask(ctx context.Context, ref PathRef, typ string, steps []workflow.Step) error {
	cluster, namespace := ClusterNamespaceFromCtx(ctx)
	task := workflow.Task{
		Name:  TaskNameOf(ref, typ),
		Group: TaskGroupApplication,
		Steps: steps,
		Addtionals: map[string]string{
			"ref":                    string(ref.JsonStringBase64()),
			"type":                   typ,                         // 用于前端，区分各个任务类型
			TaskAddtionalKeyCommiter: AuthorFromContext(ctx).Name, // 用于在异步任务中拿到 committer 在更改编排时带入
			AnnotationCluster:        cluster,                     // 以下三个 用于msgbus中按照 cluster namespace name 进行消息分发
			AnnotationNamespace:      namespace,
			LabelApplication:         ref.Name,
		},
	}
	return p.Workflowcli.SubmitTask(ctx, task)
}

func (p *TaskProcessor) ListTasks(ctx context.Context, ref PathRef, typ string) ([]workflow.Task, error) {
	tasks, err := p.Workflowcli.ListTasks(ctx, TaskGroupApplication, TaskNameOf(ref, typ))
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (p *TaskProcessor) GetTaskLatest(ctx context.Context, ref PathRef, typ string) (*workflow.Task, error) {
	tasks, err := p.Workflowcli.ListTasks(ctx, TaskGroupApplication, TaskNameOf(ref, typ))
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found for task %s", ref.Name)
	}
	return &tasks[0], nil
}

func (p *TaskProcessor) WatchTasks(ctx context.Context, ref PathRef,
	typ string, callback func(ctx context.Context, task *workflow.Task) error) error {
	return p.Workflowcli.WatchTasks(ctx, TaskGroupApplication, TaskNameOf(ref, typ), callback)
}

func TaskNameOf(ref PathRef, taskname string) string {
	if ref.IsEmpty() {
		return ""
	}
	ret := path.Join(ref.Tenant, ref.Project, ref.Env, ref.Name)
	ret = ret + "/" + taskname
	return ret
}
