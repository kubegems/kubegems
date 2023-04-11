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
	edgev1beta1 "kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateStatus(ctx context.Context, status *edgev1beta1.EdgeTaskResourceStatus, edgeresource client.Object, edgecli client.Client) {
	switch typedobj := edgeresource.(type) {
	case *appsv1.Deployment:
		checkDeploymentPodsReady(ctx, status, typedobj, edgecli)
	}
}

func checkDeploymentPodsReady(ctx context.Context, status *edgev1beta1.EdgeTaskResourceStatus, deployment *appsv1.Deployment, edgecli client.Client) {
	log := logr.FromContextOrDiscard(ctx)
	// fill images in status
	fillImages(status, deployment)
	if deployment.Status.ReadyReplicas == deployment.Status.Replicas {
		status.Ready = true
		status.Message = ""
		// events will be remove after status is ready to avoid too many events in k8s
		return
	}
	status.Message = fmt.Sprintf("replicas not ready: %d/%d", deployment.Status.ReadyReplicas, deployment.Status.Replicas)
	status.Ready = false

	// get depevents for deployment
	eventlist := &corev1.EventList{}
	if err := edgecli.List(ctx, eventlist,
		client.InNamespace(deployment.Namespace),
		client.MatchingFields{"involvedObject.uid": string(deployment.UID)},
	); err != nil {
		log.Error(err, "failed to list events")
		return
	}
	status.Events = EdgeTaskResourceEventsFromEvents(eventlist.Items)

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
		if err := edgecli.List(ctx, eventlist, client.InNamespace(deployment.Namespace), client.MatchingFields{"involvedObject.uid": string(pod.UID)}); err != nil {
			log.Error(err, "failed to list events")
		}
		val := edgev1beta1.EdgeTaskPodStatus{
			Name:   pod.Name,
			Status: pod.Status,
			Events: EdgeTaskResourceEventsFromEvents(eventlist.Items),
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
