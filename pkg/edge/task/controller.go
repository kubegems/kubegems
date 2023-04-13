// Copyright 2023 The kubegems.io Authors
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

package task

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	edgev1beta1 "kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/installer/utils"
	"kubegems.io/kubegems/pkg/utils/generic"
	"kubegems.io/kubegems/pkg/utils/kube"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	AnnotationEdgeTaskResourcesHash = "edge.kubegems.io/resources-hash"
	AnnotationEdgeTaskNameNamespace = "edge.kubegems.io/edge-task"
)

const (
	IndexFieldEdgeTaskStatusPhase         = "status.phase"
	IndexFieldEdgeTaskSpecEdgeClusterName = "spec.edgeClusterName"
	IndexFieldEdgeClusterStatusPhase      = "status.phase"
)

type Reconciler struct {
	client.Client
	EdgeClients *EdgeClientsHolder
}

// nolint: forcetypeassert
func (r *Reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options *Options) error {
	mgr.GetFieldIndexer().IndexField(ctx, &edgev1beta1.EdgeTask{}, IndexFieldEdgeTaskStatusPhase, func(rawObj client.Object) []string {
		return []string{string(rawObj.(*edgev1beta1.EdgeTask).Status.Phase)}
	})
	mgr.GetFieldIndexer().IndexField(ctx, &edgev1beta1.EdgeTask{}, IndexFieldEdgeTaskSpecEdgeClusterName, func(rawObj client.Object) []string {
		return []string{rawObj.(*edgev1beta1.EdgeTask).Spec.EdgeClusterName}
	})
	mgr.GetFieldIndexer().IndexField(ctx, &edgev1beta1.EdgeCluster{}, IndexFieldEdgeClusterStatusPhase, func(rawObj client.Object) []string {
		return []string{string(rawObj.(*edgev1beta1.EdgeCluster).Status.Phase)}
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgev1beta1.EdgeTask{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: options.MaxConcurrentReconciles}).
		Watches(&source.Kind{Type: &edgev1beta1.EdgeCluster{}}, EdgeClusterTrigger(ctx, mgr.GetClient())).
		Watches(r.EdgeClients.SourceFunc(ctx, mgr.GetClient()), nil). // watch edge clusters' resources change
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContextOrDiscard(ctx)
	edgeTask := &edgev1beta1.EdgeTask{}
	if err := r.Get(ctx, req.NamespacedName, edgeTask); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	ctx = logr.NewContext(ctx, log)

	log.Info("reconcile edge task")

	// chceck deletion
	if edgeTask.GetDeletionTimestamp() != nil {
		err := r.remove(ctx, edgeTask)
		_ = r.Status().Update(ctx, edgeTask)
		return ctrl.Result{}, err
	}
	// add finalizer if not exist
	if !controllerutil.ContainsFinalizer(edgeTask, edgev1beta1.EdgeTaskFinalizer) {
		controllerutil.AddFinalizer(edgeTask, edgev1beta1.EdgeTaskFinalizer)
		log.Info("add finalizer")
		if err := r.Update(ctx, edgeTask); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	// sync
	err := r.apply(ctx, edgeTask)
	_ = r.Status().Update(ctx, edgeTask)
	if err != nil {
		log.Error(err, "apply edge task")
	}
	return ctrl.Result{}, err
}

func (r *Reconciler) remove(ctx context.Context, edgeTask *edgev1beta1.EdgeTask) error {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("remove edge task")
	// check finalizer
	if !controllerutil.ContainsFinalizer(edgeTask, edgev1beta1.EdgeTaskFinalizer) {
		log.Info("edge task finalizer has been removed, skip cleanup")
		return nil
	}
	// wait for the edge cluster to be online if managed resources not empty
	if len(edgeTask.Status.ResourcesStatus) > 0 {
		donext, err := r.stageWaitForEdgeCluster(ctx, edgeTask)
		if !donext {
			return err
		}
		// remove the edge resources
		if err := r.removeResources(ctx, edgeTask); err != nil {
			UpdateEdgeTaskCondition(edgeTask, edgev1beta1.EdgeTaskCondition{
				Type:    edgev1beta1.EdgeTaskConditionTypeCleaned,
				Status:  corev1.ConditionFalse,
				Reason:  "RemoveResourcesFailed",
				Message: err.Error(),
			})
			return err
		}
	}
	UpdateEdgeTaskCondition(edgeTask, edgev1beta1.EdgeTaskCondition{
		Type:   edgev1beta1.EdgeTaskConditionTypeCleaned,
		Status: corev1.ConditionTrue,
		Reason: "RemoveResourcesSucceed",
	})
	log.Info("remove edge resources succeed")

	// update the edge task status
	_ = r.Status().Update(ctx, edgeTask)
	// remove the finalizer
	controllerutil.RemoveFinalizer(edgeTask, edgev1beta1.EdgeTaskFinalizer)
	log.Info("remove finalizer")
	return r.Update(ctx, edgeTask)
}

func (r *Reconciler) apply(ctx context.Context, edgeTask *edgev1beta1.EdgeTask) error {
	// render edge task resources
	resources, donext, err := r.stageRenderResources(ctx, edgeTask)
	if !donext {
		return err
	}
	// wait for the edge cluster to be online
	if donext, err := r.stageWaitForEdgeCluster(ctx, edgeTask); !donext {
		return err
	}
	// stage apply resources
	if err := r.stageApplyResources(ctx, edgeTask, resources); err != nil {
		return err
	}
	// wait for the edge task to be completed
	if err := r.stageCheckResource(ctx, edgeTask); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) stageRenderResources(ctx context.Context, edgeTask *edgev1beta1.EdgeTask) ([]*unstructured.Unstructured, bool, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("edgetask", edgeTask.Name, "namespace", edgeTask.Namespace)
	log.V(5).Info("stage render edge task resources")
	// render edge task resources
	resources, err := ParseResources(edgeTask.Spec.Resources)
	if err != nil {
		UpdateEdgeTaskCondition(edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypePrepared,
			Status:  corev1.ConditionFalse,
			Reason:  "RenderResourcesFailed",
			Message: err.Error(),
		})
		log.Error(err, "render edge task resources")
		return nil, false, nil // return nil to avoid requeue
	}
	UpdateEdgeTaskCondition(edgeTask, edgev1beta1.EdgeTaskCondition{
		Type:   edgev1beta1.EdgeTaskConditionTypePrepared,
		Status: corev1.ConditionTrue,
		Reason: "RenderResourcesSucceed",
	})
	return resources, true, nil
}

func (r *Reconciler) stageWaitForEdgeCluster(ctx context.Context, edgeTask *edgev1beta1.EdgeTask) (bool, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("edgetask", edgeTask.Name, "namespace", edgeTask.Namespace)
	log.V(5).Info("stage wait for edge cluster")
	// wait for the edge cluster to be online
	edgeclustername := edgeTask.Spec.EdgeClusterName
	if edgeclustername == "" {
		edgeclustername = edgeTask.Name
	}
	edgeCluster := &edgev1beta1.EdgeCluster{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: edgeTask.Namespace, Name: edgeclustername}, edgeCluster); err != nil {
		UpdateEdgeTaskCondition(edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypeOnline,
			Status:  corev1.ConditionFalse,
			Reason:  "EdgeClusterNotFound",
			Message: err.Error(),
		})
		log.Error(err, "get edge cluster")
		return false, err
	}
	if edgeCluster.Status.Phase != edgev1beta1.EdgePhaseOnline {
		log.Info("edge cluster is not online", "edgecluster", edgeCluster.Name, "phase", edgeCluster.Status.Phase)
		UpdateEdgeTaskCondition(edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypeOnline,
			Status:  corev1.ConditionFalse,
			Reason:  "EdgeClusterNotOnline",
			Message: "edge cluster is not online",
		})
		return false, nil // return nil to avoid requeue,wait edge cluster online trigger reconcile
	} else {
		UpdateEdgeTaskCondition(edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:   edgev1beta1.EdgeTaskConditionTypeOnline,
			Status: corev1.ConditionTrue,
			Reason: "EdgeClusterOnline",
		})
		return true, nil
	}
}

func (r *Reconciler) stageApplyResources(ctx context.Context, edgetask *edgev1beta1.EdgeTask, resources []*unstructured.Unstructured) error {
	log := logr.FromContextOrDiscard(ctx)
	if edgetask.Generation == edgetask.Status.ObservedGeneration {
		log.V(5).Info("edgetask generation not changed, skip apply resources")
		return nil
	}
	log.Info("edgetask generation changed, apply resources")
	// inject edge task metadata to resources
	// when those annotated reousrce changed, the corresponding edge task will be requeued
	for _, resource := range resources {
		InjectEdgeTask(resource, edgetask)
	}
	// check should update
	if resourceStatus, err := r.applyResources(ctx, edgetask, resources); err != nil {
		edgetask.Status.ResourcesStatus = resourceStatus
		UpdateEdgeTaskCondition(edgetask, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypeDistributed,
			Status:  corev1.ConditionFalse,
			Reason:  "ApplyResourcesFailed",
			Message: err.Error(),
		})
		return fmt.Errorf("apply resources: %w", err)
	} else {
		edgetask.Status.ResourcesStatus = resourceStatus
		edgetask.Status.ObservedGeneration = edgetask.Generation // update observed generation
		UpdateEdgeTaskCondition(edgetask, edgev1beta1.EdgeTaskCondition{
			Type:   edgev1beta1.EdgeTaskConditionTypeDistributed,
			Status: corev1.ConditionTrue,
			Reason: "ApplyResourcesSucceed",
		})
		return nil
	}
}

func (r *Reconciler) applyResources(
	ctx context.Context,
	task *edgev1beta1.EdgeTask,
	resources []*unstructured.Unstructured,
) ([]edgev1beta1.EdgeTaskResourceStatus, error) {
	cli, err := r.EdgeClients.Get(task.Spec.EdgeClusterName)
	if err != nil {
		return nil, fmt.Errorf("get edge client: %w", err)
	}
	diff := utils.Diff(convertlist(task.Status.ResourcesStatus), resources)
	applyresult, err := (&utils.Apply{Client: cli}).SyncDiff(ctx, diff, utils.NewDefaultSyncOptions())
	appliedresourcestatus := convertlistfrom(applyresult)
	if err != nil {
		return appliedresourcestatus, fmt.Errorf("sync resources: %w", err)
	}
	return appliedresourcestatus, nil
}

func (r *Reconciler) removeResources(ctx context.Context, task *edgev1beta1.EdgeTask) error {
	cli, err := r.EdgeClients.Get(task.Spec.EdgeClusterName)
	if err != nil {
		return fmt.Errorf("get edge client: %w", err)
	}
	diff := utils.Diff(convertlist(task.Status.ResourcesStatus), nil)
	if _, err = (&utils.Apply{Client: cli}).SyncDiff(ctx, diff, utils.NewDefaultSyncOptions()); err != nil {
		return fmt.Errorf("sync resources: %w", err)
	}
	return nil
}

func (r *Reconciler) stageCheckResource(ctx context.Context, task *edgev1beta1.EdgeTask) error {
	log := logr.FromContextOrDiscard(ctx)
	log.V(5).Info("check resources")
	cli, err := r.EdgeClients.Get(task.Spec.EdgeClusterName)
	if err != nil {
		UpdateEdgeTaskCondition(task, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypeAvailable,
			Status:  corev1.ConditionUnknown,
			Reason:  "EdgeClientNotReady",
			Message: fmt.Sprintf("unable get edge client: %v", err),
		})
		log.Error(err, "get edge client")
		return fmt.Errorf("get edge client: %w", err)
	}
	notreadycount, totalcount := 0, len(task.Status.ResourcesStatus)
	for i := range task.Status.ResourcesStatus {
		status := &task.Status.ResourcesStatus[i]
		obj := newObjFrom(status)
		if err := cli.Get(ctx, client.ObjectKey{Name: status.Name, Namespace: status.Namespace}, obj); err != nil {
			status.Exists = false
			status.Ready = false
			status.Message = err.Error()
		} else {
			status.Exists = true
			status.Ready = true // assume it is ready on exist
			status.Message = ""
			CheckStatus(ctx, status, obj, cli)
		}
		if !status.Ready {
			notreadycount++
		}
	}
	if notreadycount == 0 {
		log.Info("all resources ready")
		task.Status.Phase = edgev1beta1.EdgeTaskPhaseRunning
		UpdateEdgeTaskCondition(task, edgev1beta1.EdgeTaskCondition{
			Type:   edgev1beta1.EdgeTaskConditionTypeAvailable,
			Status: corev1.ConditionTrue,
			Reason: "AllResourcesReady",
		})
	} else {
		task.Status.Phase = edgev1beta1.EdgeTaskPhaseWaiting
		UpdateEdgeTaskCondition(task, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypeAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "Waiting",
			Message: fmt.Sprintf("waiting for resources ready: %d/%d", totalcount-notreadycount, totalcount),
		})
	}
	return nil // always return nil to avoid requeue
}

func newObjFrom(status *edgev1beta1.EdgeTaskResourceStatus) client.Object {
	gvk := schema.FromAPIVersionAndKind(status.APIVersion, status.Kind)
	obj, err := kube.GetScheme().New(gvk)
	if err != nil {
		// fallback to unstructured
		obj = &unstructured.Unstructured{}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
	}
	// nolint: forcetypeassert
	cliobj := obj.(client.Object)
	return cliobj
}

func findStatusOf(task *edgev1beta1.EdgeTask, obj client.Object) (int, *edgev1beta1.EdgeTaskResourceStatus) {
	for i, resource := range task.Status.ResourcesStatus {
		if resource.Kind == obj.GetObjectKind().GroupVersionKind().Kind &&
			resource.APIVersion == obj.GetObjectKind().GroupVersionKind().GroupVersion().String() &&
			resource.Name == obj.GetName() && resource.Namespace == obj.GetNamespace() {
			return i, &task.Status.ResourcesStatus[i]
		}
	}
	return -1, nil
}

func convertlist(list []edgev1beta1.EdgeTaskResourceStatus) []utils.ManagedResource {
	return generic.MapList(list, func(item edgev1beta1.EdgeTaskResourceStatus) utils.ManagedResource {
		return utils.ManagedResource{
			Kind: item.Kind, APIVersion: item.APIVersion, Name: item.Name, Namespace: item.Namespace,
		}
	})
}

func convertlistfrom(list []utils.ManagedResource) []edgev1beta1.EdgeTaskResourceStatus {
	return generic.MapList(list, func(item utils.ManagedResource) edgev1beta1.EdgeTaskResourceStatus {
		return edgev1beta1.EdgeTaskResourceStatus{
			Kind: item.Kind, APIVersion: item.APIVersion, Name: item.Name, Namespace: item.Namespace,
		}
	})
}
