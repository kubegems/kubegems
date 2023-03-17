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
	"hash/fnv"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/util/workqueue"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
	edgev1beta1 "kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/installer/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

func (r *Reconciler) UpdateEdgeTaskCondition(ctx context.Context, task *edgev1beta1.EdgeTask, condition edgev1beta1.EdgeTaskCondition) error {
	status := &task.Status
	index, oldcond := GetEdgeTaskCondition(status, condition.Type)
	now := metav1.Now()
	if oldcond == nil {
		condition.LastUpdateTime = now
		condition.LastTransitionTime = now
		status.Conditions = append(status.Conditions, condition)
	} else {
		if oldcond.Status != condition.Status {
			condition.LastTransitionTime = now
		} else {
			condition.LastTransitionTime = oldcond.LastTransitionTime
		}
		status.Conditions[index] = condition
	}
	if !reflect.DeepEqual(oldcond, condition) {
		if err := r.Client.Status().Update(ctx, task); err != nil {
			logr.FromContextOrDiscard(ctx).Error(err, "update edge task condition failed")
			return err
		}
	}
	return nil
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
			// find tasks which .spec.edgeClusterName == current.Name
			enqueueTasksOfCluster(ctx, cli, current.Name, current.Namespace, rli)
		},
	}
}

func enqueueTasksOfCluster(ctx context.Context, cli client.Client, clustername string, namespace string, queue workqueue.Interface) {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("trigger tasks of edge cluster", "clustername", clustername, "namespace", namespace)
	tasks := &edgev1beta1.EdgeTaskList{}
	if err := cli.List(ctx, tasks,
		client.InNamespace(namespace),
		client.MatchingFields{IndexFieldEdgeTaskSpecEdgeClusterName: clustername},
	); err != nil {
		log.Error(err, "list edge tasks")
		return
	}
	for _, task := range tasks.Items {
		if task.Status.Phase == edgev1beta1.EdgeTaskPhaseRunning {
			continue
		}
		log.Info("trigger edge task", "name", task.Name, "namespace", task.Namespace)
		queue.Add(ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&task)})
	}
}

func GetEdgeTaskCondition(status *edgev1beta1.EdgeTaskStatus, conditionType edgev1beta1.EdgeTaskConditionType) (int, *edgev1beta1.EdgeTaskCondition) {
	if status == nil {
		return -1, nil
	}
	for i, condition := range status.Conditions {
		if condition.Type == conditionType {
			return i, &condition
		}
	}
	return -1, nil
}

func RemoveEdgeTaskCondition(status *edgev1beta1.EdgeTaskStatus, conditionType edgev1beta1.EdgeTaskConditionType) {
	if status == nil {
		return
	}
	for i, condition := range status.Conditions {
		if condition.Type == conditionType {
			status.Conditions = append(status.Conditions[:i], status.Conditions[i+1:]...)
			return
		}
	}
}

func ExtractEdgeTask(obj client.Object) (string, string) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return "", ""
	}
	parts := strings.Split(annotations[AnnotationEdgeTaskNameNamespace], "/")
	// nolint: gomnd
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func InjectEdgeTask(obj client.Object, edgetask *edgev1beta1.EdgeTask) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[AnnotationEdgeTaskNameNamespace] = edgetask.Name + "/" + edgetask.Namespace
	obj.SetAnnotations(annotations)
}

func HashResources(obj any) string {
	hasher := fnv.New32()
	hashutil.DeepHashObject(hasher, obj)
	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum32()))
}

func ParseResources(ctx context.Context, resources []runtime.RawExtension) ([]*unstructured.Unstructured, error) {
	unstructedlist := []*unstructured.Unstructured{}
	for i, resource := range resources {
		list, err := utils.SplitYAML(resource.Raw)
		if err != nil {
			return nil, fmt.Errorf("split resource on index %d : %w", i, err)
		}
		unstructedlist = append(unstructedlist, list...)
	}
	return unstructedlist, nil
}

func ParseResourcesTyped(ctx context.Context, resources []runtime.RawExtension, schema *runtime.Scheme) ([]client.Object, error) {
	objects, err := ParseResources(ctx, resources)
	if err != nil {
		return nil, nil
	}
	return utils.ConvertToTypedList(objects, schema), nil
}

func FindOwnerControllerRecursively(ctx context.Context, cli client.Client, obj client.Object) (client.Object, error) {
	owner := metav1.GetControllerOf(obj)
	if owner == nil {
		return obj, nil // no owner
	}
	ownerobj, err := cli.Scheme().New(schema.FromAPIVersionAndKind(owner.APIVersion, owner.Kind))
	if err != nil {
		return nil, err
	}
	ownercliobj, ok := ownerobj.(client.Object)
	if !ok {
		return nil, fmt.Errorf("owner object is not a client.Object")
	}
	if err := cli.Get(ctx, client.ObjectKey{Namespace: obj.GetNamespace(), Name: owner.Name}, ownercliobj); err != nil {
		return nil, err
	}
	return FindOwnerControllerRecursively(ctx, cli, ownercliobj)
}

func EdgeTaskResourceEventsFromEvents(events []corev1.Event) []edgev1beta1.EdgeTaskResourceEvent {
	ret := make([]edgev1beta1.EdgeTaskResourceEvent, len(events))
	for i, event := range events {
		ret[i] = edgev1beta1.EdgeTaskResourceEvent{
			Reason:        event.Reason,
			Message:       event.Message,
			Type:          event.Type,
			Count:         event.Count,
			LastTimestamp: event.LastTimestamp,
		}
	}
	return ret
}
