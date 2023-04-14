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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	edgev1beta1 "kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CheckStatus(ctx context.Context, status *edgev1beta1.EdgeTaskResourceStatus, edgeresource client.Object, edgecli client.Client) {
	switch typedobj := edgeresource.(type) {
	case *appsv1.Deployment:
		checkDeploymentPodsReady(ctx, status, typedobj, edgecli)
	}
}

func checkDeploymentPodsReady(ctx context.Context, status *edgev1beta1.EdgeTaskResourceStatus, deployment *appsv1.Deployment, edgecli client.Client) {
	log := logr.FromContextOrDiscard(ctx)
	// fill images in status
	fillImages(status, deployment)
	if deployment.Status.ObservedGeneration != deployment.Generation {
		status.Ready = false
		status.Message = "deployment not observed"
		return
	}
	desireReplicas := pointer.Int32Deref(deployment.Spec.Replicas, 1)
	currentReplicas := deployment.Status.AvailableReplicas
	if desireReplicas == currentReplicas {
		// deployment 会在 pod running 的时候就认为 ready，对于running即crashloopbackoff的pod，deployment会某一瞬间认为ready
		// 有两种解决方案：
		// - 为 pod 设置 readinessProbe
		// - 为 deployment 设置 minReadySeconds
		status.Ready = true
		status.Message = ""
		return
	}
	status.Ready = false
	status.Message = fmt.Sprintf("replicas not ready: %d/%d", currentReplicas, desireReplicas)
	// events will be remove after status is ready to avoid too many events in k8
	// get depevents for deployment
	if false {
		eventlist := &corev1.EventList{}
		if err := edgecli.List(ctx, eventlist,
			client.InNamespace(deployment.Namespace),
			client.MatchingFields{"involvedObject.uid": string(deployment.UID)},
		); err != nil {
			log.Error(err, "failed to list events")
			return
		}
		status.Events = EdgeTaskResourceEventsFromEvents(eventlist.Items)
	} else {
		status.Events = nil
	}

	// found deployment pods
	pods := &corev1.PodList{}
	if err := edgecli.List(ctx, pods,
		client.InNamespace(deployment.Namespace),
		client.MatchingLabels(deployment.Spec.Selector.MatchLabels),
	); err != nil {
		log.Error(err, "failed to list pods")
		return
	}
	// fill events for those pods
	podsStatus := make([]edgev1beta1.EdgeTaskPodStatus, len(pods.Items))
	for i, pod := range pods.Items {
		val := edgev1beta1.EdgeTaskPodStatus{
			Name:   pod.Name,
			Status: pod.Status,
		}
		if false {
			eventlist := &corev1.EventList{}
			if err := edgecli.List(ctx, eventlist, client.InNamespace(deployment.Namespace), client.MatchingFields{"involvedObject.uid": string(pod.UID)}); err != nil {
				log.Error(err, "failed to list events")
			}
			val.Events = EdgeTaskResourceEventsFromEvents(eventlist.Items)
		} else {
			val.Events = nil
		}
		podsStatus[i] = val
	}
	status.PodsStatus = podsStatus
}

func fillImages(status *edgev1beta1.EdgeTaskResourceStatus, deployment *appsv1.Deployment) {
	images := []string{}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}
	status.Images = images
}
