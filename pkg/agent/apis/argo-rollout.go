package apis

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/argoproj/argo-rollouts/pkg/apiclient/rollout"
	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/argoproj/argo-rollouts/pkg/kubectl-argo-rollouts/info"
	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/kubectl/pkg/util/deployment"
	"k8s.io/utils/pointer"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/stream"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ArgoRolloutHandler struct {
	cluster cluster.Interface
}

// @Tags         Agent.V1
// @Summary      rollout info
// @Description  rollout info(argo rollout)
// @Param        cluster  path      string  true  "cluster"
// @Success      200      {object}  object  "rollout.RolloutInfo"
// @Router       /v1/proxy/cluster/{cluster}/custom/argoproj.io/v1alpha1/namespaces/{namespace}/rollouts/{name}/actions/info [get]
// @Security     JWT
func (h *ArgoRolloutHandler) GetRolloutInfo(c *gin.Context) {
	namespace, name := c.Param("namespace"), c.Param("name")

	ctx := c.Request.Context()
	cli := h.cluster.GetClient()

	if iswatch, _ := strconv.ParseBool(c.Query("watch")); iswatch {
		rolloutlist := &rolloutsv1alpha1.RolloutList{}
		ServeWatchThen(c, cli, namespace, name, rolloutlist, func(obj client.Object) (interface{}, error) {
			return GetRolloutInfo(ctx, cli, obj.GetNamespace(), obj.GetName())
		})
	} else {
		info, err := GetRolloutInfo(ctx, cli, namespace, name)
		if err != nil {
			NotOK(c, err)
			return
		}
		OK(c, info)
	}
}

// @Tags         Agent.V1
// @Summary      rollout info
// @Description  rollout info(deployment)
// @Param        cluster  path      string  true  "cluster"
// @Success      200      {object}  object  "rollout.RolloutInfo"
// @Router       /v1/proxy/cluster/{cluster}/custom/argoproj.io/v1alpha1/namespaces/{namespace}/rollouts/{name}/actions/depinfo [get]
// @Security     JWT
func (h *ArgoRolloutHandler) GetRolloutDepInfo(c *gin.Context) {
	namespace, name := c.Param("namespace"), c.Param("name")

	ctx := c.Request.Context()
	// nolint: ifshort
	cli := h.cluster.GetClient()

	if iswatch, _ := strconv.ParseBool(c.Query("watch")); iswatch {
		deploymentList := &appsv1.DeploymentList{}
		ServeWatchThen(c, cli, namespace, name, deploymentList, func(obj client.Object) (interface{}, error) {
			return GetDeploymentInfo(ctx, cli, obj.GetNamespace(), obj.GetName())
		})
	} else {
		info, err := GetDeploymentInfo(ctx, cli, namespace, name)
		if err != nil {
			NotOK(c, err)
			return
		}
		OK(c, info)
	}
}

const WatcherRefreshInterval = 5 * time.Second

func ServeWatchThen(c *gin.Context, cli client.Client, namespace, name string, watchlist client.ObjectList,
	onchangefunc func(obj client.Object) (interface{}, error)) {
	ctx := c.Request.Context()
	watchablecli, ok := cli.(client.WithWatch)
	if !ok {
		NotOK(c, fmt.Errorf("client dose't supported watch"))
		return
	}
	watcher, err := watchablecli.Watch(ctx, watchlist, client.InNamespace(namespace))
	if err != nil {
		NotOK(c, err)
		return
	}
	defer watcher.Stop()

	ticker := time.NewTicker(WatcherRefreshInterval)
	defer ticker.Stop()

	pusher, err := stream.StartPusher(c.Writer)
	if err != nil {
		NotOK(c, err)
		return
	}

	onevent := func(r client.Object) bool {
		select {
		case <-c.Writer.CloseNotify():
			return false
		default:

			info, err := onchangefunc(r)
			if err != nil {
				log.Error(err, "rollout info")
				return false
			}
			if err := pusher.Push(info); err != nil {
				log.Error(err, "push rollout info")
				return false
			}
			return true
		}
	}

	var laststate client.Object
	for {
		select {
		case e := <-watcher.ResultChan():
			if e.Type == watch.Error {
				return
			}
			if dep, ok := e.Object.(client.Object); ok {
				if dep.GetName() == name {
					laststate = dep
					if !onevent(dep) {
						return
					}
				}
			}
		case <-ticker.C:
			if !onevent(laststate) {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func GetDeploymentInfo(ctx context.Context, cli client.Client, namespace string, name string) (*rollout.RolloutInfo, error) {
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(dep), dep); err != nil {
		return nil, err
	}

	rsList := &appsv1.ReplicaSetList{}
	if err := cli.List(ctx, rsList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	rss := make([]*appsv1.ReplicaSet, len(rsList.Items))
	for i := range rsList.Items {
		rss[i] = &rsList.Items[i]
	}

	podList := &corev1.PodList{}
	if err := cli.List(ctx, podList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	pods := make([]*corev1.Pod, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &podList.Items[i]
	}

	info := &rollout.RolloutInfo{
		ObjectMeta: &metav1.ObjectMeta{
			Name:              dep.Name,
			Namespace:         dep.Namespace,
			UID:               dep.UID,
			CreationTimestamp: dep.CreationTimestamp,
			ResourceVersion:   dep.ObjectMeta.ResourceVersion,
		},
		Status: func() string {
			if dep.Status.Replicas == dep.Status.UpdatedReplicas {
				return string(rolloutsv1alpha1.RolloutPhaseHealthy)
			}
			return string(rolloutsv1alpha1.RolloutPhaseProgressing)
		}(),
		Message: func() string {
			for _, condition := range dep.Status.Conditions {
				if condition.Type == appsv1.DeploymentAvailable {
					return condition.Message
				}
			}
			return ""
		}(),
		Strategy:  string(dep.Spec.Strategy.Type),
		Ready:     dep.Status.ReadyReplicas,
		Current:   dep.Status.Replicas,
		Desired:   pointer.Int32Deref(dep.Spec.Replicas, 0),
		Updated:   dep.Status.UpdatedReplicas,
		Available: dep.Status.AvailableReplicas,
		RestartedAt: func() string {
			if cond := deployment.GetDeploymentCondition(dep.Status, appsv1.DeploymentProgressing); cond != nil {
				return cond.LastTransitionTime.String()
			}
			return ""
		}(),
		Generation:  strconv.FormatInt(dep.Status.ObservedGeneration, 10),
		ReplicaSets: info.GetReplicaSetInfo(dep.GetUID(), nil, rss, pods),
		Containers: func() []*rollout.ContainerInfo {
			ctns := []*rollout.ContainerInfo{}
			for _, c := range dep.Spec.Template.Spec.Containers {
				ctns = append(ctns, &rollout.ContainerInfo{Name: c.Name, Image: c.Image})
			}
			return ctns
		}(),
		Steps: []*rolloutsv1alpha1.CanaryStep{},
	}
	// override the ReplicaSets's reversion
	for _, item := range info.ReplicaSets {
		for _, rs := range rss {
			if rs.UID == item.ObjectMeta.UID && rs.Annotations != nil {
				rev, _ := strconv.Atoi(rs.Annotations[deployment.RevisionAnnotation])
				item.Revision = int32(rev)
			}
		}
	}
	sort.Slice(info.ReplicaSets, func(i, j int) bool {
		return info.ReplicaSets[i].Revision > info.ReplicaSets[j].Revision
	})
	return info, nil
}

func GetRolloutInfo(ctx context.Context, cli client.Client, namespace, name string) (*rollout.RolloutInfo, error) {
	// see: https://github.com/argoproj/argo-rollouts/blob/v1.1.0/pkg/kubectl-argo-rollouts/viewcontroller/viewcontroller.go#L171
	rollout := &rolloutsv1alpha1.Rollout{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(rollout), rollout); err != nil {
		return nil, err
	}
	// get all rs
	rss := &appsv1.ReplicaSetList{}
	if err := cli.List(ctx, rss, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	// all pods
	pods := &corev1.PodList{}
	if err := cli.List(ctx, pods, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	// all exp
	exps := &rolloutsv1alpha1.ExperimentList{}
	if err := cli.List(ctx, exps, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	// all analysisrun
	anss := &rolloutsv1alpha1.AnalysisRunList{}
	if err := cli.List(ctx, anss, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	return NewRolloutInfo(rollout, rss.Items, pods.Items, exps.Items, anss.Items), nil
}

func NewRolloutInfo(ro *rolloutsv1alpha1.Rollout,
	allReplicaSets []appsv1.ReplicaSet,
	allPods []corev1.Pod,
	allExperiments []rolloutsv1alpha1.Experiment,
	allARs []rolloutsv1alpha1.AnalysisRun,
) *rollout.RolloutInfo {
	return info.NewRolloutInfo(ro,
		func() []*appsv1.ReplicaSet {
			list := make([]*appsv1.ReplicaSet, len(allReplicaSets))
			for i := range allReplicaSets {
				list[i] = &allReplicaSets[i]
			}
			return list
		}(),
		func() []*corev1.Pod {
			list := make([]*corev1.Pod, len(allPods))
			for i := range allPods {
				list[i] = &allPods[i]
			}
			return list
		}(),
		func() []*rolloutsv1alpha1.Experiment {
			list := make([]*rolloutsv1alpha1.Experiment, len(allExperiments))
			for i := range allExperiments {
				list[i] = &allExperiments[i]
			}
			return list
		}(),
		func() []*rolloutsv1alpha1.AnalysisRun {
			list := make([]*rolloutsv1alpha1.AnalysisRun, len(allARs))
			for i := range allARs {
				list[i] = &allARs[i]
			}
			return list
		}())
}
