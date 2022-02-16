package application

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/argoproj/argo-rollouts/pkg/apiclient/rollout"
	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/gin-gonic/gin"
	istioclinetworkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/stream"
	"kubegems.io/pkg/utils/workflow"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	istioclinetworkingv1alpha3.AddToScheme(scheme.Scheme)
	rolloutsv1alpha1.AddToScheme(scheme.Scheme)
}

type ImageDetails struct {
	Running string // 意为当前 argo 实际正在运行的版本
	Publish string // 意为需要更新到的版本，argo更新版本会失败，所以两个版本会存在差异
}

// @Tags StrategyDeployment
// @Summary 获取当前的应用更新策略
// @Description 获取部署
// @Accept json
// @Produce json
// @Param tenant_id      	path  	int    	true "tenautid"
// @Param project_id     	path  	int    	true "proid"
// @param environment_id 	path  	int	  	true "envid"
// @Param name 				path  	string	true "applicationname"
// @Success 200 {object} handlers.ResponseStruct{Data=DeploymentStrategyWithImages} "deploy"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/strategydeploy [get]
// @Security JWT
// 通过 argo app 将部署策略设置至，在sync是根据策略进行修改或者 rollout创建
func (h *ApplicationHandler) GetStrategyDeployment(c *gin.Context) {
	h.LocalCliFunc(c, nil, func(ctx context.Context, store GitStore, ref PathRef) (interface{}, error) {
		// 发布策略需要从两个地方获取。
		// 1. 如果存在同名的 argo rollout 资源，则认为是使用的argo。
		// 2. 如果不存在argo rollout，则从 deployment 中获取。

		dm, err := h.ApplicationProcessor.Get(ctx, ref)
		if err != nil {
			return nil, err
		}
		strategyWithDeployment, err := ParseUpdateStrategyAndDeployment(ctx, store)
		if err != nil {
			return nil, err
		}
		// ret := strategyWithDeployment.Strategy
		ret := DeploymentStrategyWithImages{
			DeployImages: ConvertDeploiedManifestToView(*dm),
			Strategy:     strategyWithDeployment.Strategy,
		}
		return ret, nil
	}, "")
}

// @Tags StrategyDeployment
// @Summary 使用更新策略更新应用
// @Description 使用更新策略更新应用
// @Accept json
// @Produce json
// @Param tenant_id      	path  	int    	true "tenaut id"
// @Param project_id     	path  	int    	true "project id"
// @param environment_id 	path	int 	true "environment id"
// @Param name 				path  	string	true "applicationname"
// @Param body              body    DeploymentStrategyWithImages true "reqbody"
// @Success 200 {object} 	handlers.ResponseStruct{Data=DeploymentStrategy} "-"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/strategydeploy [post]
// @Security JWT
//
// 开始灰度发布流程
// 灰度发布流程目前有如下几种：
// - 滚动更新，deployment rollingupdate 策略。
// - 重新创建，deployment recreated 策略。
// - 灰度发布，rollout canary 策略。
// - 蓝绿发布，rollout bluegreen 策略。
func (h *ApplicationHandler) EnableStrategyDeployment(c *gin.Context) {
	strategy := &DeploymentStrategyWithImages{}
	h.NamedRefFunc(c, strategy, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// 审计
		h.SetAuditData(c, "策略更新", "应用", ref.Name)

		steps := []workflow.Step{
			{
				Name:     "configuration",
				Function: TaskFunction_Application_PrepareDeploymentStrategy,
				Args:     workflow.ArgsOf(ref, strategy),
			},
			{
				Name:     "sync",
				Function: TaskFunction_Application_Sync,
				Args:     workflow.ArgsOf(ref),
			},
			// {
			// 	Name:     "wait-sync",
			// 	Function: TaskFunction_Application_WaitSync,
			// 	Args:     workflow.ArgsOf(ref),
			// },
		}
		if err := h.Task.Processor.SubmitTask(ctx, ref, "strategy-deploy", steps); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

// @Tags StrategyDeployment
// @Summary 切换更新策略
// @Description 切换更新策略
// @Accept json
// @Produce json
// @Param tenant_id      	path  	int    	true "tenaut id"
// @Param project_id     	path  	int    	true "project id"
// @param environment_id 	path	int 	true "environment id"
// @Param name 				path  	string	true "applicationname"
// @Success 200 {object} 	handlers.ResponseStruct{Data=DeploymentStrategy} "-"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/strategyswitch [post]
// @Security JWT
//
// 开始灰度发布流程
// 灰度发布流程目前有如下几种：
// - 滚动更新，deployment rollingupdate 策略。
// - 重新创建，deployment recreated 策略。
// - 灰度发布，rollout canary 策略。
// - 蓝绿发布，rollout bluegreen 策略。

// SwitchStrategy 用于在不同类型的发布策略中切换，切换时创建异步任务
func (h *ApplicationHandler) SwitchStrategy(c *gin.Context) {
	strategy := &DeploymentStrategy{}
	h.NamedRefFunc(c, strategy, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// 审计
		h.SetAuditData(c, "切换", "部署策略", ref.Name)

		steps := []workflow.Step{
			{
				Name:     "configuration",
				Function: TaskFunction_Application_PrepareDeploymentStrategy,
				Args:     workflow.ArgsOf(ref, DeploymentStrategyWithImages{Strategy: *strategy}),
			},
			{
				Name:     "sync",
				Function: TaskFunction_Application_Sync,
				Args:     workflow.ArgsOf(ref),
			},
			// {
			// 	Name:     "wait-sync",
			// 	Function: TaskFunction_Application_WaitSync,
			// 	Args:     workflow.ArgsOf(ref),
			// },
		}
		// 根据策略选择等待目标
		switch strategy.Type {
		case BlueGreenDeploymentStrategyType, CanaryDeploymentStrategyType:
			steps = append(steps, workflow.Step{
				Name:     "wait-rollouts",
				Function: TaskFunction_Application_WaitRollouts,
				Args:     workflow.ArgsOf(ref),
			})
		}

		if err := h.Task.Processor.SubmitTask(ctx, ref, "switch-strategy", steps); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

type DeploymentStrategyWithDeployment struct {
	Strategy   DeploymentStrategy
	Deployment *appsv1.Deployment
}

// @Tags StrategyDeployment
// @Summary 获取支持的灰度分析
// @Description 获取支持的灰度分析
// @Accept json
// @Produce json
// @Param tenant_id      	path  	int    	true "tenaut id"
// @Param project_id     	path  	int    	true "project id"
// @Param name 				path  	string	true "applicationname"
// @param environment_id 	path	int 	true "environment id"
// @Success 200 {object} 	handlers.ResponseStruct{Data=v1alpha1.AnalysisTemplate} "-"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/analysistemplate [get]
// @Security JWT
func (h *ApplicationHandler) ListAnalysisTemplate(c *gin.Context) {
	h.RemoteCliFunc(c, nil, func(ctx context.Context, cli agents.Client, _ string, _ PathRef) (interface{}, error) {
		analysisTemplateList := &rolloutsv1alpha1.ClusterAnalysisTemplateList{}
		if err := cli.List(ctx, analysisTemplateList); err != nil {
			return nil, err
		}
		return analysisTemplateList.Items, nil
	})
}

// @Tags StrategyDeployment
// @Summary 更新过程中的实时状态
// @Description 更新中的实时状态
// @Accept json
// @Produce json
// @Param tenant_id      	path  	int    	true "tenaut id"
// @Param project_id     	path  	int    	true "project id"
// @param environment_id 	path 	int 	true "environment id"
// @Param name 				path  	string	true "applicationname"
// @param watch				query	bool	false "watch 则返回 ssevent"
// @Success 200 {object} 	handlers.ResponseStruct{Data=rollout.RolloutInfo} "-"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/application/{name}/strategydeploystatus [get]
// @Security JWT
func (h *ApplicationHandler) StrategyDeploymentStatus(c *gin.Context) {
	h.LocalAndRemoteCliFunc(c, nil, func(ctx context.Context, store GitStore, cli agents.Client, namespace string, _ PathRef) (interface{}, error) {
		deployment, err := ParseMainDeployment(ctx, store)
		if err != nil {
			return nil, err
		}

		rolloutresource := &rolloutsv1alpha1.Rollout{ObjectMeta: metav1.ObjectMeta{Namespace: deployment.Namespace, Name: deployment.Name}}
		if err := store.Get(ctx, client.ObjectKeyFromObject(rolloutresource), rolloutresource); err != nil {
			if !errors.IsNotFound(err) {
				return nil, err
			}
			rolloutresource = nil
		}

		var pathtemplate string
		if rolloutresource != nil {
			pathtemplate = "/custom/argoproj.io/v1alpha1/namespaces/%s/rollouts/%s/actions/info?watch=%t"
		} else {
			pathtemplate = "/custom/argoproj.io/v1alpha1/namespaces/%s/rollouts/%s/actions/depinfo?watch=%t"
		}

		name := deployment.Name
		if iswatch, _ := strconv.ParseBool(c.Query("watch")); iswatch {
			// watch rollout info
			resp, err := cli.DoRawRequest(ctx, agents.Request{
				Method: http.MethodGet,
				Path:   fmt.Sprintf(pathtemplate, namespace, name, true),
			})
			if err != nil {
				return nil, err
			}
			receiver := stream.StartReceiver(resp.Body)

			item := &rollout.RolloutInfo{}
			c.Stream(func(_ io.Writer) bool {
				item.Reset()
				if err := receiver.Recieve(item); err != nil {
					return false
				}
				c.SSEvent("data", item)
				return true
			})
			return nil, nil
		} else {
			// get rollout info
			path := fmt.Sprintf(pathtemplate, namespace, name)
			item := &rollout.RolloutInfo{}

			req := agents.Request{
				Method: http.MethodGet,
				Path:   path,
				Into:   &handlers.ResponseStruct{Data: item},
			}
			if err := cli.DoRequest(ctx, req); err != nil {
				return nil, err
			}
			return item, nil
		}
	}, "")
}

type (
	LocalAndRemoteStoreFunc func(ctx context.Context, local GitStore, remote agents.Client, namespace string, ref PathRef) (interface{}, error)
	RemoteStoreFunc         func(ctx context.Context, remote agents.Client, namespace string, ref PathRef) (interface{}, error)
	LocalStoreFunc          func(ctx context.Context, store GitStore, ref PathRef) (interface{}, error)
)

func (h *ApplicationHandler) RemoteCliFunc(c *gin.Context, body interface{}, fun RemoteStoreFunc) {
	h.NamedRefFunc(c, body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		cluster, namespace := ClusterNamespaceFromCtx(ctx)
		p, err := h.Agents.ClientOf(ctx, cluster)
		if err != nil {
			return nil, err
		}
		remotecli := p
		return fun(ctx, remotecli, namespace, ref)
	})
}

func (h *ApplicationHandler) LocalAndRemoteCliFunc(c *gin.Context, body interface{}, fun LocalAndRemoteStoreFunc, msg string) {
	h.RemoteCliFunc(c, body, func(ctx context.Context, remote agents.Client, namespace string, ref PathRef) (interface{}, error) {
		var data interface{}
		updategfsfunc := func(ctx context.Context, store GitStore) error {
			got, err := fun(ctx, store, remote, namespace, ref)
			if err != nil {
				return err
			}
			data = got
			return nil
		}
		// 写入编排并同步
		if err := h.Manifest.StoreUpdateFunc(ctx, ref, updategfsfunc, msg); err != nil {
			return nil, err
		}
		// 同步
		if msg != "" {
			if err := h.ApplicationProcessor.Sync(ctx, ref); err != nil {
				return nil, err
			}
		}
		return data, nil
	})
}

func (h *ApplicationHandler) LocalCliFunc(c *gin.Context, body interface{}, fun LocalStoreFunc, msg string) {
	var data interface{}
	h.NamedRefFunc(c, body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		updategfsfunc := func(ctx context.Context, store GitStore) error {
			got, err := fun(ctx, store, ref)
			if err != nil {
				return err
			}
			data = got
			return nil
		}

		_ = h.Manifest.StoreUpdateFunc(ctx, ref, updategfsfunc, msg)

		// 同步
		if msg != "" {
			if err := h.ApplicationProcessor.Sync(ctx, ref); err != nil {
				return nil, err
			}
		}
		return data, nil
	})
}
