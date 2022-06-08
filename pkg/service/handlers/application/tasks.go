package application

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/workflow"
)

type TaskHandler struct {
	BaseHandler
	Processor *TaskProcessor
}

func NewTaskHandler(base BaseHandler) *TaskHandler {
	return &TaskHandler{
		BaseHandler: base,
		Processor: &TaskProcessor{
			workflow.NewClientFromBackend(workflow.NewRedisBackendFromClient(base.GetRedis().Client)),
		},
	}
}

// @Tags         Application
// @Summary      应用异步任务
// @Description  应用异步任务列表
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                            true   "tenaut id"
// @Param        project_id      path      int                                            true   "project id"
// @Param        environment_id  path      int                                            true   "environment_id"
// @Param        name            path      string                                         true   "application name"
// @Param        watch           query     string                                         false  "is watch sse ,sse key 为 'data'"
// @Param        limit           query     int                                            false  "限制返回的条数，返回最新的n条记录"
// @Param        type            query     string                                         false  "限制返回的任务类型，例如仅返回 部署镜像(update-image),切换模式(switch-strategy)  的任务"
// @Success      200             {object}  handlers.ResponseStruct{Data=[]workflow.Task}  "task status"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/tasks [get]
// @Security     JWT
func (h *TaskHandler) List(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		iswatch, _ := strconv.ParseBool(c.Query("watch"))
		limit, _ := strconv.Atoi(c.Query("limit"))
		kind := c.Query("type")

		tasks, err := h.Processor.ListTasks(ctx, ref, kind)
		if err != nil {
			return nil, err
		}
		if !iswatch {
			if limit != 0 && len(tasks) > limit {
				return tasks[:limit], nil
			}
			return tasks, nil
		}

		c.SSEvent("data", tasks)
		c.Writer.Flush()

		err = h.Processor.WatchTasks(ctx, ref, kind, func(_ context.Context, task *workflow.Task) error {
			//  更新 list中的task
			updated := false
			for i, item := range tasks {
				if task.UID == item.UID {
					tasks[i] = *task
					updated = true
				}
			}
			if !updated {
				// 插入队首
				tasks = append([]workflow.Task{*task}, tasks...)
				if len(tasks) > limit {
					tasks = tasks[:limit]
				}
			}
			// 仅当更新了列表中的数据时才推送
			c.SSEvent("data", tasks)
			c.Writer.Flush()

			return nil
		})
		if err != nil {
			log.Info("watch tasks closed", "err", err.Error())
		}
		return nil, nil
	})
}

// @Tags Application
// @Summary 应用列表的异步任务列表
// @Description 应用列表的异步任务列表
// @Accept json
// @Produce json
// @Param tenant_id      path  int    true "tenaut id"
// @Param project_id     path  int    true "project id"
// @Param environment_id path  int    true "environment_id"
// @Param names 		 query string false "names,逗号','分隔,限制返回结果为这些name"
// @Success 200 {object} handlers.ResponseStruct{Data=[]ApplicationTask} "task status"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/_/tasks [get]
// @Security JWT
type ApplicationTask struct {
	Name string        `json:"name"` // 应用名称
	Task workflow.Task `json:"task"` // 最新一次任务，如果不存在最新一次任务，则返回一个空任务
}

func (h *TaskHandler) BatchList(c *gin.Context) {
	names := []string{}
	if val := c.Query("names"); val != "" {
		names = strings.Split(c.Query("names"), ",")
	}

	h.NoNameRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		tasks, err := h.Processor.ListTasks(ctx, ref, "")
		if err != nil {
			return nil, err
		}

		// convert
		apptasks := map[string][]ApplicationTask{}
		for _, task := range tasks {
			appname := task.Addtionals[LabelApplication]
			if appname == "" {
				continue
			}
			if atasks, ok := apptasks[appname]; ok {
				apptasks[appname] = append(atasks, ApplicationTask{Name: appname, Task: task})
			} else {
				apptasks[appname] = []ApplicationTask{{Name: appname, Task: task}}
			}
		}

		result := []ApplicationTask{}
		for appname, v := range apptasks {
			// filter out
			if len(names) > 0 && !StringsIn(names, appname) {
				continue
			}
			// sort
			if len(v) > 0 {
				result = append(result, v[0])
			}
		}
		return result, nil
	})
}

func StringsIn(li []string, i string) bool {
	for _, v := range li {
		if v == i {
			return true
		}
	}
	return false
}
