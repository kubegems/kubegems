package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	rolloutannotations "github.com/argoproj/argo-rollouts/utils/annotations"
	istionetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istioclinetworkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"istio.io/istio/pkg/config/schema/gvk"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"
	"k8s.io/utils/pointer"
	"kubegems.io/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AnnotationGeneratedByPlatformKey           = "app.gems.cloudminds.com/is-generated-by" // 表示该资源是为了{for}自动生成的，需要在特定的时刻被清理
	AnnotationGeneratedByPlatformValueRollouts = "rollouts"                                // 表示该资源是为了rollouts而生成的
)

type DeploymentStrategyType string

const (
	RollingUpdateDeploymentStrategyType DeploymentStrategyType = DeploymentStrategyType(appsv1.RollingUpdateDeploymentStrategyType)
	RecreatDeploymentStrategyType       DeploymentStrategyType = DeploymentStrategyType(appsv1.RecreateDeploymentStrategyType)
	CanaryDeploymentStrategyType        DeploymentStrategyType = "Canary"
	BlueGreenDeploymentStrategyType     DeploymentStrategyType = "BlueGreen"
)

type RecreatDeploymentStrategy struct {
	WaitShutdown bool `json:"waitShutdown,omitempty"` // 暂未使用
}

type BlueGreenDeploymentStrategy struct {
	rolloutsv1alpha1.BlueGreenStrategy
}

type CanaryDeploymentStrategy struct {
	ExtendCanaryStrategy
}

type DeploymentStrategyWithImages struct {
	DeployImages
	Strategy DeploymentStrategy `json:"strategy,omitempty"`
}

type DeploymentStrategy struct {
	Type      DeploymentStrategyType          `json:"type,omitempty" validate:"required"` // 更新策略： in(Recreate,RollingUpdate,Canary,BlueGreen)
	Canary    *CanaryDeploymentStrategy       `json:"canary,omitempty"`
	BlueGreen *BlueGreenDeploymentStrategy    `json:"blueGreen,omitempty"`
	Recreat   *RecreatDeploymentStrategy      `json:"recreat,omitempty"`
	Rolling   *appsv1.RollingUpdateDeployment `json:"rolling,omitempty"`
}

func (p *ApplicationProcessor) WaitRollouts(ctx context.Context, ref PathRef) error {
	// 从编排中找到 rollouts
	var rollout *rolloutsv1alpha1.Rollout
	// p.Manifest.StoreUpdateFunc(ctx, ref, func(ctx context.Context, store GitStore) error {
	// 	rolloutList := &rolloutsv1alpha1.RolloutList{}
	// 	if err := store.List(ctx, rolloutList); err != nil {
	// 		return err
	// 	}
	// 	if len(rolloutList.Items) > 0 {
	// 		rollout = &rolloutList.Items[0]
	// 		return nil
	// 	} else {
	// 		return fmt.Errorf("no rollouts found in application %s", ref.Name)
	// 	}
	// }, "")

	// 找到runtime client

	envinfo, err := p.DataBase.GetEnvironmentWithCluster(ref)
	if err != nil {
		return err
	}

	cli, err := p.Agents.ClientOf(ctx, envinfo.ClusterName)
	if err != nil {
		return err
	}

	if rollout == nil {
		// 暂时使用同名 rollout
		rollout = &rolloutsv1alpha1.Rollout{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ref.Name,
				Namespace: envinfo.Namespace,
			},
		}
	}

	// 等待检查是否存在
	// 间隔 5s ，执行三次，总耗时 10s
	err = retry.OnError(wait.Backoff{Duration: 5 * time.Second, Steps: 3}, errors.IsNotFound, func() error {
		return cli.TypedClient.Get(ctx, client.ObjectKeyFromObject(rollout), rollout)
	})
	if err != nil {
		return err
	}

	rolloutsList := &rolloutsv1alpha1.RolloutList{}
	watcher, err := cli.TypedClient.Watch(ctx, rolloutsList, client.InNamespace(rollout.Namespace))
	if err != nil {
		return err
	}
	defer watcher.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watcher channel closed")
			}
			switch e.Type {
			case watch.Error:
				return fmt.Errorf("watch error: %v", e)
			default:
				if changed, ok := e.Object.(*rolloutsv1alpha1.Rollout); ok && changed.Name == rollout.Name {
					// 等待直到错误或者healthy
					switch changed.Status.Phase {
					// 成功
					case rolloutsv1alpha1.RolloutPhaseHealthy, rolloutsv1alpha1.RolloutPhasePaused:
						return nil
					// 失败
					case rolloutsv1alpha1.RolloutPhaseDegraded:
						return fmt.Errorf("rollouts failed: %s", changed.Status.Message)
					}
				}
			}
		}
	}
}

func (p *ApplicationProcessor) PrepareDeploymentStrategyWithImages(
	ctx context.Context, ref PathRef, strategy DeploymentStrategyWithImages) error {
	// 更新镜像
	if len(strategy.DeployImages.PublishImages()) != 0 || strategy.DeployImages.IstioVersion != "" {
		if err := p.Manifest.StoreFunc(ctx, ref, func(ctx context.Context, store GitStore) error {
			return UpdateContentImages(ctx, store, strategy.PublishImages(), strategy.IstioVersion)
		}); err != nil {
			return err
		}
	}
	// 更新策略
	return p.PrepareDeploymentStrategy(ctx, ref, strategy.Strategy)
}

func (p *ApplicationProcessor) PrepareDeploymentStrategy(ctx context.Context, ref PathRef, strategy DeploymentStrategy) error {
	// 创建资源
	updategfsfunc := func(ctx context.Context, store GitStore) error {
		// 寻找用于更新的deployment
		deployment, err := ParseMainDeployment(ctx, store)
		if err != nil {
			return err
		}

		// 更新deployment相关配置
		if err := p.configForDeployment(ctx, store, deployment, &strategy); err != nil {
			return err
		}
		// 更新rollout相关配置
		if err := p.configForArgoRollout(ctx, store, deployment, &strategy); err != nil {
			return err
		}
		return nil
	}
	return p.Manifest.StoreUpdateFunc(ctx, ref, updategfsfunc, fmt.Sprintf("set strategy [%s]", strategy.Type))
}

func (p *ApplicationProcessor) configForDeployment(ctx context.Context, store GitStore, dep *appsv1.Deployment, strategy *DeploymentStrategy) error {
	// 更新deployment策略
	switch stype := strategy.Type; stype {
	case RollingUpdateDeploymentStrategyType:
		if dep.Spec.Strategy.Type != appsv1.DeploymentStrategyType(stype) {
			dep.Spec.Strategy = appsv1.DeploymentStrategy{
				Type:          appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: strategy.Rolling,
			}
			return store.Update(ctx, dep)
		}
	case RecreatDeploymentStrategyType:
		if dep.Spec.Strategy.Type != appsv1.DeploymentStrategyType(stype) {
			dep.Spec.Strategy = appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			}
			return store.Update(ctx, dep)
		}
	}
	return nil
}

func ParseUpdateStrategyAndDeployment(ctx context.Context, store GitStore) (*DeploymentStrategyWithDeployment, error) {
	// 寻找 deployment
	deployment, err := ParseMainDeployment(ctx, store)
	if err != nil {
		return nil, err
	}
	// 寻找 argo rollout
	rolloutresource := &rolloutsv1alpha1.Rollout{ObjectMeta: metav1.ObjectMeta{Namespace: deployment.Namespace, Name: deployment.Name}}
	if err := store.Get(ctx, client.ObjectKeyFromObject(rolloutresource), rolloutresource); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		rolloutresource = nil
	}

	ret := &DeploymentStrategyWithDeployment{
		Deployment: deployment,
	}

	if rolloutresource != nil {
		// get from rollout
		if canary := rolloutresource.Spec.Strategy.Canary; canary != nil {
			ret.Strategy.Type = CanaryDeploymentStrategyType
			ret.Strategy.Canary = &CanaryDeploymentStrategy{
				ExtendCanaryStrategy: *ExtendCanaryStrategyFromCanaryStrategy(canary),
			}
		} else if bg := rolloutresource.Spec.Strategy.BlueGreen; bg != nil {
			ret.Strategy.Type = BlueGreenDeploymentStrategyType
			ret.Strategy.BlueGreen = &BlueGreenDeploymentStrategy{
				BlueGreenStrategy: *bg,
			}
		}
	} else {
		// get from deployment
		ret.Strategy.Type = DeploymentStrategyType(deployment.Spec.Strategy.Type)
		ret.Strategy.Rolling = deployment.Spec.Strategy.RollingUpdate
	}
	return ret, nil
}

func (p *ApplicationProcessor) configForArgoRollout(ctx context.Context, cli GitStore, dep *appsv1.Deployment, strategy *DeploymentStrategy) error {
	// argo rollout 有两种配置
	switch strategy.Type {
	// 灰度发布，灰度发布可能有多种路由策略
	case CanaryDeploymentStrategyType:
		if strategy.Canary == nil {
			// 默认
			strategy.Canary = &CanaryDeploymentStrategy{
				ExtendCanaryStrategy: ExtendCanaryStrategy{
					TrafficRouting: &RolloutTrafficRouting{
						Istio: &IstioTrafficRouting{VirtualService: IstioVirtualService{IstioVirtualService: rolloutsv1alpha1.IstioVirtualService{}}},
					},
				},
			}
		}
		if err := p.prepareCanaryRollout(ctx, cli, dep, strategy.Canary); err != nil {
			return err
		}
	// 蓝绿发布，蓝绿发布比较简单
	case BlueGreenDeploymentStrategyType:
		if strategy.BlueGreen == nil {
			strategy.BlueGreen = &BlueGreenDeploymentStrategy{}
		}
		if err := p.prepareBlugreenRollout(ctx, cli, dep, strategy.BlueGreen); err != nil {
			return err
		}
	default:
		// 其他策略就移除rollout相关
		return CleanRollout(ctx, cli)
	}
	// 创建 rollout 本身
	return p.ensureRollout(ctx, cli, dep, strategy)
}

// 取消灰度
func CleanRollout(ctx context.Context, cli GitStore) error {
	// 由 rollouts 创建的资源都加上了 annotation 按照annotation存在的进行删除
	items, err := cli.ListAll(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if annotations := item.GetAnnotations(); annotations != nil &&
			annotations[AnnotationGeneratedByPlatformKey] == AnnotationGeneratedByPlatformValueRollouts {
			if err := cli.Delete(ctx, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *ApplicationProcessor) prepareBlugreenRollout(ctx context.Context, cli GitStore, dep *appsv1.Deployment, blugreen *BlueGreenDeploymentStrategy) error {
	// ensure services
	if blugreen == nil {
		return nil
	}
	if blugreen.ActiveService == "" {
		blugreen.ActiveService = dep.GetName() + "-canary"
	}
	if blugreen.PreviewService == "" {
		blugreen.PreviewService = dep.GetName()
	}
	// 需要依赖于不同的 service,先创建service
	if err := p.ensureDeploymentServices(ctx, cli, dep, blugreen.ActiveService, blugreen.PreviewService); err != nil {
		return err
	}
	// 分析参数默认
	defaultAnalysisArgs(blugreen.PrePromotionAnalysis, blugreen.PreviewService) // 默认对新版本服务做分析
	return nil
}

const (
	AnalysisArgServiceName = "service-name"
	AnalysisArgNamespace   = "namespace"
)

func defaultAnalysisArgs(analysis *rolloutsv1alpha1.RolloutAnalysis, defaultservice string) {
	if analysis == nil {
		return
	}
	for i, arg := range analysis.Args {
		switch {
		case arg.Name == AnalysisArgServiceName && strings.TrimSpace(arg.Value) == "":
			analysis.Args[i].Value = defaultservice
		case arg.Name == AnalysisArgNamespace && strings.TrimSpace(arg.Value) == "":
			analysis.Args[i].Value = ""
			analysis.Args[i].ValueFrom = &rolloutsv1alpha1.ArgumentValueFrom{
				FieldRef: &rolloutsv1alpha1.FieldRef{
					FieldPath: "metadata.namespace",
				},
			}
		}
	}
}

func (p *ApplicationProcessor) prepareCanaryRollout(ctx context.Context, cli GitStore, dep *appsv1.Deployment, canary *CanaryDeploymentStrategy) error {
	// ensure services
	if canary == nil {
		return nil
	}
	if canary.CanaryService == "" {
		canary.CanaryService = dep.GetName() + "-canary"
	}
	if canary.StableService == "" {
		canary.StableService = dep.GetName()
	}

	isinit := len(canary.ExtendCanaryStrategy.CanaryStrategy.Steps) == 0

	if err := completeCanarySteps(ctx, &canary.ExtendCanaryStrategy.CanaryStrategy); err != nil {
		return err
	}
	// 需要依赖于不同的 service,先创建service
	if err := p.ensureDeploymentServices(ctx, cli, dep, canary.StableService, canary.CanaryService); err != nil {
		return err
	}
	// 分析参数默认
	if canary.Analysis != nil {
		defaultAnalysisArgs(&canary.Analysis.RolloutAnalysis, canary.CanaryService) // 默认对新版本服务做分析
	}

	// 判断使用的 TrafficRouting 采取不同策略
	tr := canary.TrafficRouting
	if tr == nil {
		return nil
	}
	switch {
	// istio 需要创建 vitrual service
	case tr.Istio != nil:
		return p.prepareCanaryIstioRollout(ctx, cli, dep, canary.StableService, canary.CanaryService, tr.Istio, isinit)
	default:
		// 其他策略暂时不做操作
		return nil
	}
}

// set 10% if nil or zero
const defaultStepInitWeight = 10

func completeCanarySteps(_ context.Context, canary *rolloutsv1alpha1.CanaryStrategy) error {
	if len(canary.Steps) == 0 {
		canary.Steps = append(canary.Steps, rolloutsv1alpha1.CanaryStep{SetWeight: pointer.Int32(defaultStepInitWeight)})
	}
	// add pause forever
	if len(canary.Steps) == 1 {
		canary.Steps = append(canary.Steps, rolloutsv1alpha1.CanaryStep{Pause: &rolloutsv1alpha1.RolloutPause{}})
	}
	return nil
}

// istio 策略需要创建一个virtualservice
func (p *ApplicationProcessor) prepareCanaryIstioRollout(ctx context.Context, cli GitStore,
	dep *appsv1.Deployment, stablesvcname, nextsvcname string, istio *IstioTrafficRouting, isinit bool) error {
	// 虚拟服务是要和主service同名的
	if istio.VirtualService.Name == "" {
		istio.VirtualService.Name = stablesvcname
	}

	if len(istio.VirtualService.Routes) == 0 {
		istio.VirtualService.Routes = []string{"canary"}
	}

	vsroute := istio.VirtualService.Routes[0]

	vs := &istioclinetworkingv1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      istio.VirtualService.Name,
			Namespace: dep.GetNamespace(),
			Annotations: map[string]string{
				AnnotationGeneratedByPlatformKey: AnnotationGeneratedByPlatformValueRollouts,
			},
		},
		Spec: istionetworkingv1alpha3.VirtualService{
			Hosts: []string{
				stablesvcname,
			},
			Http: func() []*istionetworkingv1alpha3.HTTPRoute {
				// 这里有两种情况
				// 1. 所有流量按照流量比例灰度
				// 2. 匹配header和/或uri的部分流量按照比例灰度，其余流量不做改变

				// case #1
				httpRoutes := []*istionetworkingv1alpha3.HTTPRoute{
					{
						Name: vsroute,
						Route: []*istionetworkingv1alpha3.HTTPRouteDestination{
							{
								Destination: &istionetworkingv1alpha3.Destination{Host: stablesvcname},
								Weight:      100,
							},
							{
								Destination: &istionetworkingv1alpha3.Destination{Host: nextsvcname},
								Weight:      0,
							},
						},
					},
					{
						Route: []*istionetworkingv1alpha3.HTTPRouteDestination{
							{
								Destination: &istionetworkingv1alpha3.Destination{Host: stablesvcname},
							},
						},
					},
				}

				// case #2
				if len(istio.VirtualService.Headers) != 0 || istio.VirtualService.Uri != nil {
					httpRoutes[0].Match = []*istionetworkingv1alpha3.HTTPMatchRequest{
						{
							Headers:       istio.VirtualService.Headers,
							Uri:           istio.VirtualService.Uri,
							IgnoreUriCase: istio.VirtualService.IgnoreUriCase,
						},
					}
				}
				return httpRoutes
			}(),
		},
	}
	existvs := &istioclinetworkingv1alpha3.VirtualService{}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(vs), existvs); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// create
		return cli.Create(ctx, vs)
	}

	updated := false
	// update existvs
	var targetFound *istionetworkingv1alpha3.HTTPRoute
	for i, v := range existvs.Spec.Http {
		if v.Name == vsroute {
			if !equality.Semantic.DeepEqual(v, vs.Spec.Http[0]) {
				existvs.Spec.Http[i] = vs.Spec.Http[0]
				updated = true
			}
			// 由于编排中忽略的是 http/0 ;所以这里需要把对应的条目移动到 0 位置
			if i != 0 {
				// [0,1,2,T,4,5,6]
				//
				// [T]
				// [T]+[0,1,2]
				// [T]+[0,1,2]+[4,5,6]
				existvs.Spec.Http = append(
					append(
						[]*istionetworkingv1alpha3.HTTPRoute{v},
						existvs.Spec.Http[:i]...),
					existvs.Spec.Http[i+1:]...)
				updated = true
			}
			targetFound = v
		}
	}
	if targetFound == nil {
		// 添加 0 条目
		existvs.Spec.Http = append([]*istionetworkingv1alpha3.HTTPRoute{vs.Spec.Http[0]}, existvs.Spec.Http...)
		updated = true
	}

	if updated {
		return cli.Update(ctx, existvs)
	}
	return nil
}

func (p *ApplicationProcessor) ensureDeploymentServices(ctx context.Context, cli GitStore, dep *appsv1.Deployment, svcnames ...string) error {
	// 寻找同名service，有则copy，否则创建
	svclist := &corev1.ServiceList{}
	if err := cli.List(ctx, svclist, client.InNamespace(dep.Namespace)); err != nil {
		return err
	}

	stablesvc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: dep.Namespace, Name: dep.Name}}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(stablesvc), stablesvc); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// create a new stable service
		stablesvc = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: dep.Namespace,
				Name:      dep.Name,
				Annotations: map[string]string{
					AnnotationGeneratedByPlatformKey: AnnotationGeneratedByPlatformValueRollouts,
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: func() map[string]string {
					filterdlabels := map[string]string{}
					for k, v := range dep.Spec.Template.Labels {
						// ignore version label
						if k == LabelIstioVersion {
							continue
						}
						filterdlabels[k] = v
					}
					return filterdlabels
				}(),
				Ports: func() []corev1.ServicePort {
					ports := []corev1.ServicePort{}
					for _, container := range dep.Spec.Template.Spec.Containers {
						for _, ctnport := range container.Ports {
							ports = append(ports, corev1.ServicePort{
								Name:       ctnport.Name,
								Protocol:   ctnport.Protocol,
								Port:       ctnport.ContainerPort,
								TargetPort: intstr.FromInt(int(ctnport.ContainerPort)),
							})
						}
					}
					return ports
				}(),
			},
		}

		// application no port specified
		if len(stablesvc.Spec.Ports) == 0 {
			return fmt.Errorf("create svc for deployment failed, %s has no ports specified", dep.Name)
		}
		if err := p.CreateIfNotExist(ctx, cli, stablesvc); err != nil {
			return err
		}
	}

	// copy svcs
	for _, svcname := range svcnames {
		if svcname == stablesvc.Name {
			continue
		}
		copyed := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   stablesvc.Namespace,
				Name:        svcname,
				Labels:      stablesvc.GetLabels(),
				Annotations: stablesvc.GetAnnotations(),
				Finalizers:  stablesvc.GetFinalizers(),
			},
			Spec: corev1.ServiceSpec{
				Ports:                 stablesvc.Spec.DeepCopy().Ports,
				Selector:              stablesvc.Spec.DeepCopy().Selector,
				Type:                  stablesvc.Spec.Type,
				SessionAffinity:       stablesvc.Spec.SessionAffinity,
				ExternalTrafficPolicy: stablesvc.Spec.ExternalTrafficPolicy,
			},
		}
		if copyed.Annotations == nil {
			copyed.Annotations = map[string]string{}
		}
		copyed.Annotations[AnnotationGeneratedByPlatformKey] = AnnotationGeneratedByPlatformValueRollouts

		if err := p.CreateIfNotExist(ctx, cli, copyed); err != nil {
			return err
		}
	}
	return nil
}

func (p *ApplicationProcessor) CreateIfNotExist(ctx context.Context, cli GitStore, obj client.Object) error {
	if err := cli.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// create
		return cli.Create(ctx, obj)
	}
	return nil
}

// 如果是为了 rollout 创建的资源，会在资源上增加一个 annotation
func (p *ApplicationProcessor) ensureRollout(ctx context.Context, cli GitStore, dep *appsv1.Deployment, stretagy *DeploymentStrategy) error {
	name, namespace := dep.GetName(), dep.GetNamespace()

	// 通过名称寻找相关的 deployment 来获得副本数
	// 目前 argo rollouts 仅支持deployment
	getReplicas := func() *int32 {
		existdeployment := &appsv1.Deployment{}
		if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, existdeployment); err != nil {
			log.WithField("rollout", name).Warnf("deployment %s for rollout not found,set replicas to default", name)
			// 如果无法获得副本数则设置为默认值
			return nil
		}
		return existdeployment.Spec.Replicas
	}

	// 根据升级策略不同对 RolloutStrategy 进行配置
	getRolloutsStreategy := func() rolloutsv1alpha1.RolloutStrategy {
		switch stretagy.Type {
		case CanaryDeploymentStrategyType:
			return rolloutsv1alpha1.RolloutStrategy{Canary: stretagy.Canary.ExtendCanaryStrategy.ToCanaryStrategy()}
		case BlueGreenDeploymentStrategyType:
			return rolloutsv1alpha1.RolloutStrategy{BlueGreen: &stretagy.BlueGreen.BlueGreenStrategy}
		default:
			return rolloutsv1alpha1.RolloutStrategy{}
		}
	}

	// 构造对应的 argorollout
	argorollout := &rolloutsv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				AnnotationGeneratedByPlatformKey: AnnotationGeneratedByPlatformValueRollouts,
			},
		},
		Spec: rolloutsv1alpha1.RolloutSpec{
			TemplateResolvedFromRef: true,
			// SelectorResolvedFromRef: true,
			Selector: dep.Spec.Selector,
			WorkloadRef: &rolloutsv1alpha1.ObjectRef{
				APIVersion: gvk.Deployment.GroupVersion(),
				Kind:       gvk.Deployment.Kind,
				Name:       name,
			},
			Strategy: getRolloutsStreategy(),
			Replicas: getReplicas(),
		},
	}

	exist := &rolloutsv1alpha1.Rollout{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(exist), exist); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// create a new rollout
		return cli.Create(ctx, argorollout)
	}
	argorollout.Spec.Replicas = exist.Spec.Replicas
	exist.Spec = argorollout.Spec
	return cli.Update(ctx, exist)
}

func (h *ApplicationProcessor) Undo(ctx context.Context, ref PathRef, targetrev string) error {
	// 查询runtime
	details, err := h.DataBase.GetEnvironmentWithCluster(ref)
	if err != nil {
		return err
	}
	runtimenamespace := details.Namespace
	cluster := details.ClusterName

	// runtime client
	cli, err := h.Agents.ClientOf(ctx, cluster)
	if err != nil {
		return err
	}
	runtimecli := cli.TypedClient

	// 更新
	updategfsfunc := func(ctx context.Context, store GitStore) error {
		// 寻找用于更新的deployment
		withdeployment, err := ParseUpdateStrategyAndDeployment(ctx, store)
		if err != nil {
			return err
		}

		name := withdeployment.Deployment.Name
		var contredby client.Object
		var revisionAnnotation string
		switch withdeployment.Strategy.Type {
		case RecreatDeploymentStrategyType, RollingUpdateDeploymentStrategyType:
			// contredby deployment
			contredby = &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: runtimenamespace, Name: name}}
			revisionAnnotation = deploymentutil.RevisionAnnotation
		case BlueGreenDeploymentStrategyType, CanaryDeploymentStrategyType:
			// contredby argo rollouts
			contredby = &rolloutsv1alpha1.Rollout{ObjectMeta: metav1.ObjectMeta{Namespace: runtimenamespace, Name: name}}
			revisionAnnotation = rolloutannotations.RevisionAnnotation
		default:
			return fmt.Errorf("not target to rollback")
		}

		// 检查回滚目标
		if err := runtimecli.Get(ctx, client.ObjectKeyFromObject(contredby), contredby); err != nil {
			return err
		}
		// found reversion replicaset
		replicasetList := &appsv1.ReplicaSetList{}
		if err := runtimecli.List(ctx, replicasetList, client.InNamespace(runtimenamespace)); err != nil {
			return err
		}
		var targetreplicaset *appsv1.ReplicaSet
		for i, replicaset := range replicasetList.Items {
			if !metav1.IsControlledBy(&replicaset, contredby) {
				continue
			}
			rev := replicaset.Annotations[revisionAnnotation]
			if rev != targetrev {
				continue
			}
			targetreplicaset = &replicasetList.Items[i]
		}
		if targetreplicaset == nil {
			return fmt.Errorf("reversion %s not found", targetrev)
		}

		// 由于回滚操作经过argo cd再下发时延迟过高
		// 需要两边同时操作
		// 1. 更新git中的文件
		// 2. 同时patch当前的deployyment/rollout以直接回滚
		patch, err := getDeploymentPatch(*targetreplicaset)
		if err != nil {
			return err
		}
		// 目前来说无论是否为  rollouts 都是 patch deployment，否则直接 patch contredby 即可
		// patch runtime
		// patchtarget := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: runtimenamespace}}
		// if err := runtimecli.Patch(ctx, patchtarget, patch); err != nil {
		// 	return err
		// }
		// patch manifest
		localpatchtarget := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: runtimenamespace}}
		if err := store.Patch(ctx, localpatchtarget, patch); err != nil {
			return err
		}
		return nil
	}
	return h.Manifest.StoreUpdateFunc(ctx, ref, updategfsfunc, fmt.Sprintf("undo rev [%s]", targetrev))
}
