package task

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/util/workqueue"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
	edgev1beta1 "kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/installer/utils"
	"kubegems.io/kubegems/pkg/utils/generic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	AnnotationEdgeTaskResourcesHash = "edge.kubegems.io/resources-hash"
)

const (
	IndexFieldEdgeTaskStatusPhase    = "status.phase"
	IndexFieldEdgeClusterStatusPhase = "status.phase"
)

type Reconciler struct {
	client.Client
	EdgeClients *EdgeClientsHolder
}

// nolint: forcetypeassert
func (r *Reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, concurrent int) error {
	mgr.GetFieldIndexer().IndexField(ctx, &edgev1beta1.EdgeTask{}, IndexFieldEdgeTaskStatusPhase, func(rawObj client.Object) []string {
		return []string{string(rawObj.(*edgev1beta1.EdgeTask).Status.Phase)}
	})
	mgr.GetFieldIndexer().IndexField(ctx, &edgev1beta1.EdgeCluster{}, IndexFieldEdgeClusterStatusPhase, func(rawObj client.Object) []string {
		return []string{string(rawObj.(*edgev1beta1.EdgeCluster).Status.Phase)}
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgev1beta1.EdgeTask{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: concurrent}).
		Watches(&source.Kind{Type: &edgev1beta1.EdgeCluster{}}, EdgeClusterTrigger(ctx, mgr.GetClient())).
		Watches(r.EdgeClients.SourceFunc(), nil). // watch edge clusters' resources change
		Complete(r)
}

func EdgeClusterTrigger(ctx context.Context, cli client.Client) handler.EventHandler {
	log := logr.FromContextOrDiscard(ctx)
	return handler.Funcs{
		UpdateFunc: func(ue event.UpdateEvent, rli workqueue.RateLimitingInterface) {
			// once the edgecluster is coming online, we need to trigger the uncompleted tasks for it
			if ue.ObjectNew.GetDeletionTimestamp() != nil {
				return
			}
			previous, ok := ue.ObjectOld.(*edgev1beta1.EdgeCluster)
			if !ok {
				return
			}
			current, ok := ue.ObjectNew.(*edgev1beta1.EdgeCluster)
			if !ok {
				return
			}
			log.WithValues("edgecluster", current.Name, "namespace", current.Namespace)
			// in case of the edgecluster is coming online from other status
			// offline  || online -> online
			if current.Status.Phase != edgev1beta1.EdgePhaseOnline || previous.Status.Phase == current.Status.Phase {
				return
			}
			log.Info("edgecluster status changed", "old", previous.Status.Phase, "new", current.Status.Phase)
			log.Info("edgecluster is coming online, trigger the uncompleted tasks for it")
			task := &edgev1beta1.EdgeTask{}
			if err := cli.Get(ctx, client.ObjectKeyFromObject(current), task); err != nil {
				log.Error(err, "failed to get edge task")
				return
			}
			if task.Status.Phase == edgev1beta1.EdgeTaskPhaseRunning {
				return
			}
			log.Info("trigger edge task", "name", task.Name, "namespace", task.Namespace)
			rli.Add(ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
		},
	}
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
		return ctrl.Result{}, r.remove(ctx, edgeTask)
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
	log.Info("apply edge task")
	if err := r.apply(ctx, edgeTask); err != nil {
		log.Error(err, "apply edge task")
		return ctrl.Result{}, err // err will be requeued
	}
	return ctrl.Result{}, nil
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
		if err := r.stageWaitForEdgeCluster(ctx, edgeTask); err != nil {
			return err
		}
	}
	// remove the edge resources
	if err := r.removeResources(ctx, edgeTask); err != nil {
		_ = r.UpdateEdgeTaskCondition(ctx, edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypeCleaned,
			Status:  corev1.ConditionFalse,
			Reason:  "RemoveResourcesFailed",
			Message: err.Error(),
		})
		log.Error(err, "remove edge task")
		return err
	}
	r.UpdateEdgeTaskCondition(ctx, edgeTask, edgev1beta1.EdgeTaskCondition{
		Type:   edgev1beta1.EdgeTaskConditionTypeCleaned,
		Status: corev1.ConditionTrue,
		Reason: "RemoveResourcesSucceed",
	})
	log.Info("remove edge resources succeed")

	// remove the finalizer
	controllerutil.RemoveFinalizer(edgeTask, edgev1beta1.EdgeTaskFinalizer)
	log.Info("remove finalizer")
	if err := r.Update(ctx, edgeTask); err != nil {
		log.Error(err, "remove finalizer")
		return err
	}
	return nil
}

func (r *Reconciler) apply(ctx context.Context, edgeTask *edgev1beta1.EdgeTask) error {
	// render edge task resources
	resources, donext, err := r.stageRenderResources(ctx, edgeTask)
	if !donext {
		return err
	}
	// wait for the edge cluster to be online
	if err := r.stageWaitForEdgeCluster(ctx, edgeTask); err != nil {
		return err
	}
	// stage apply resources
	if err := r.stageApplyResources(ctx, edgeTask, resources); err != nil {
		return err
	}
	// wait for the edge task to be completed
	return r.stageCheckResource(ctx, edgeTask)
}

func (r *Reconciler) stageRenderResources(ctx context.Context, edgeTask *edgev1beta1.EdgeTask) ([]*unstructured.Unstructured, bool, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("edgetask", edgeTask.Name, "namespace", edgeTask.Namespace)
	log.Info("stage render edge task resources")
	// render edge task resources
	resources, err := ParseResources(ctx, edgeTask)
	if err != nil {
		r.UpdateEdgeTaskCondition(ctx, edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypePrepared,
			Status:  corev1.ConditionFalse,
			Reason:  "RenderResourcesFailed",
			Message: err.Error(),
		})
		log.Error(err, "render edge task resources")
		return nil, false, nil // return nil to avoid requeue
	} else {
		r.UpdateEdgeTaskCondition(ctx, edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:   edgev1beta1.EdgeTaskConditionTypePrepared,
			Status: corev1.ConditionTrue,
			Reason: "RenderResourcesSucceed",
		})
	}
	return resources, true, nil
}

func (r *Reconciler) stageWaitForEdgeCluster(ctx context.Context, edgeTask *edgev1beta1.EdgeTask) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("edgetask", edgeTask.Name, "namespace", edgeTask.Namespace)
	log.Info("stage wait for edge cluster")
	// wait for the edge cluster to be online
	edgeclustername := edgeTask.Spec.EdgeClusterName
	if edgeclustername == "" {
		edgeclustername = edgeTask.Name
	}
	edgeCluster := &edgev1beta1.EdgeCluster{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: edgeTask.Namespace, Name: edgeclustername}, edgeCluster); err != nil {
		edgeTask.Status.Phase = edgev1beta1.EdgeTaskPhaseWaiting
		r.UpdateEdgeTaskCondition(ctx, edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypeOnline,
			Status:  corev1.ConditionFalse,
			Reason:  "EdgeClusterNotFound",
			Message: err.Error(),
		})
		return fmt.Errorf("get edge cluster: %w", err)
	}
	if edgeCluster.Status.Phase != edgev1beta1.EdgePhaseOnline {
		edgeTask.Status.Phase = edgev1beta1.EdgeTaskPhaseWaiting
		r.UpdateEdgeTaskCondition(ctx, edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypeOnline,
			Status:  corev1.ConditionFalse,
			Reason:  "EdgeClusterNotOnline",
			Message: "edge cluster is not online",
		})
		return fmt.Errorf("edge cluster %s is not online", edgeCluster.Name)
	}
	r.UpdateEdgeTaskCondition(ctx, edgeTask, edgev1beta1.EdgeTaskCondition{
		Type:   edgev1beta1.EdgeTaskConditionTypeOnline,
		Status: corev1.ConditionTrue,
		Reason: "EdgeClusterOnline",
	})
	return nil
}

func (r *Reconciler) stageApplyResources(ctx context.Context, edgeTask *edgev1beta1.EdgeTask, resources []*unstructured.Unstructured) error {
	log := logr.FromContextOrDiscard(ctx)
	if edgeTask.Annotations == nil {
		edgeTask.Annotations = map[string]string{}
	}
	resourceshash := HashResources(resources)
	previoushash := edgeTask.Annotations[AnnotationEdgeTaskResourcesHash]
	if previoushash == resourceshash {
		log.Info("same hash of resources, skip apply", "hash", resourceshash)
		return nil
	}

	RemoveEdgeTaskCondition(&edgeTask.Status, edgev1beta1.EdgeTaskConditionTypeAvailable)

	log.Info("hash of resources changed", "previous", previoushash, "current", resourceshash)
	// check should update
	if resourceStatus, err := r.applyResources(ctx, edgeTask, resources); err != nil {
		edgeTask.Status.ResourcesStatus = resourceStatus
		r.UpdateEdgeTaskCondition(ctx, edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypeDistributed,
			Status:  corev1.ConditionFalse,
			Reason:  "ApplyResourcesFailed",
			Message: err.Error(),
		})
		return fmt.Errorf("apply resources: %w", err)
	} else {
		edgeTask.Status.ResourcesStatus = resourceStatus
		r.UpdateEdgeTaskCondition(ctx, edgeTask, edgev1beta1.EdgeTaskCondition{
			Type:   edgev1beta1.EdgeTaskConditionTypeDistributed,
			Status: corev1.ConditionTrue,
			Reason: "ApplyResourcesSucceed",
		})
		edgeTask.Annotations[AnnotationEdgeTaskResourcesHash] = resourceshash
		if err = r.Update(ctx, edgeTask); err != nil {
			log.Error(err, "update edge task status")
		}
	}
	return nil
}

func ParseResources(ctx context.Context, edgeTask *edgev1beta1.EdgeTask) ([]*unstructured.Unstructured, error) {
	unstructedlist := []*unstructured.Unstructured{}
	for i, resource := range edgeTask.Spec.Resources {
		list, err := utils.SplitYAML(resource.Raw)
		if err != nil {
			return nil, fmt.Errorf("split resource on index %d : %w", i, err)
		}
		unstructedlist = append(unstructedlist, list...)
	}
	return unstructedlist, nil
}

func ParseResourcesTyped(ctx context.Context, edgeTask *edgev1beta1.EdgeTask, schema *runtime.Scheme) ([]client.Object, error) {
	objects, err := ParseResources(ctx, edgeTask)
	if err != nil {
		return nil, nil
	}
	return utils.ConvertToTypedList(objects, schema), nil
}

func HashResources(obj any) string {
	hasher := fnv.New32()
	hashutil.DeepHashObject(hasher, obj)
	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum32()))
}

func (r *Reconciler) applyResources(ctx context.Context, edgeTask *edgev1beta1.EdgeTask, resources []*unstructured.Unstructured,
) ([]edgev1beta1.EdgeTaskResourceStatus, error) {
	edgecli, err := r.EdgeClients.Get(edgeTask.Name)
	if err != nil {
		return nil, fmt.Errorf("get edge client: %w", err)
	}
	diff := utils.Diff(convertlist(edgeTask.Status.ResourcesStatus), resources)
	applier := utils.Apply{Client: edgecli}
	applyresult, err := applier.SyncDiff(ctx, diff, utils.NewDefaultSyncOptions())
	appliedresourcestatus := convertlistfrom(applyresult)
	if err != nil {
		return appliedresourcestatus, fmt.Errorf("sync resources: %w", err)
	}
	return appliedresourcestatus, nil
}

func (r *Reconciler) removeResources(ctx context.Context, edgeTask *edgev1beta1.EdgeTask) error {
	edgecli, err := r.EdgeClients.Get(edgeTask.Name)
	if err != nil {
		return fmt.Errorf("get edge client: %w", err)
	}
	diff := utils.Diff(convertlist(edgeTask.Status.ResourcesStatus), nil)
	applier := utils.Apply{Client: edgecli}
	_, err = applier.SyncDiff(ctx, diff, utils.NewDefaultSyncOptions())
	if err != nil {
		return fmt.Errorf("sync resources: %w", err)
	}
	return nil
}

func (r *Reconciler) stageCheckResource(ctx context.Context, task *edgev1beta1.EdgeTask) error {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("check resources")
	if _, cond := GetEdgeTaskCondition(&task.Status, edgev1beta1.EdgeTaskConditionTypeAvailable); cond == nil {
		r.UpdateEdgeTaskCondition(ctx, task, edgev1beta1.EdgeTaskCondition{
			Type:   edgev1beta1.EdgeTaskConditionTypeAvailable,
			Status: corev1.ConditionFalse,
			Reason: "Waiting",
		})
	}
	edgecli, err := r.EdgeClients.Get(task.Name)
	if err != nil {
		log.Error(err, "get edge client")
		return r.UpdateEdgeTaskCondition(ctx, task, edgev1beta1.EdgeTaskCondition{
			Type:    edgev1beta1.EdgeTaskConditionTypeAvailable,
			Status:  corev1.ConditionUnknown,
			Reason:  "EdgeClientNotReady",
			Message: fmt.Sprintf("unable get edge client: %v", err),
		})
	}
	allReady := true
	for i := range task.Status.ResourcesStatus {
		status := &task.Status.ResourcesStatus[i]
		gvk := schema.FromAPIVersionAndKind(status.APIVersion, status.Kind)
		obj, err := edgecli.Scheme().New(gvk)
		if err != nil {
			// fallback to unstructured
			obj = &unstructured.Unstructured{}
			obj.GetObjectKind().SetGroupVersionKind(gvk)
		}
		// nolint: forcetypeassert
		cliobj := obj.(client.Object)
		if err := edgecli.Get(ctx, client.ObjectKey{Name: status.Name, Namespace: status.Namespace}, cliobj); err != nil {
			status.Exists = false
			status.Ready = false
			status.Message = err.Error()
		} else {
			status.Exists = true
			status.Ready = true // assume it is ready on exist
			status.Message = "resource exist"
		}
		UpdateStatus(ctx, status, cliobj, edgecli)
		if !status.Ready {
			allReady = false
		}
	}
	if allReady {
		log.Info("all resources ready")
		task.Status.Phase = edgev1beta1.EdgeTaskPhaseRunning
		return r.UpdateEdgeTaskCondition(ctx, task, edgev1beta1.EdgeTaskCondition{
			Type:   edgev1beta1.EdgeTaskConditionTypeAvailable,
			Status: corev1.ConditionTrue,
			Reason: "AllResourcesReady",
		})
	} else {
		task.Status.Phase = edgev1beta1.EdgeTaskPhaseWaiting
		return r.Client.Status().Update(ctx, task)
	}
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
