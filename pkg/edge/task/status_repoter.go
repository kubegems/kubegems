package task

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slices"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	edgev1beta1 "kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateStatus(ctx context.Context, status *edgev1beta1.EdgeTaskResourceStatus, edgeresource client.Object, edgecli client.Client) {
	switch typedobj := edgeresource.(type) {
	case *appsv1.Deployment:
		if typedobj.Status.ReadyReplicas != typedobj.Status.Replicas {
			status.Message = fmt.Sprintf("replicas not ready: %d/%d", typedobj.Status.ReadyReplicas, typedobj.Status.Replicas)
			status.Ready = false
			checkDeploymentPodsReady(ctx, status, typedobj, edgecli)
		} else {
			status.Ready = true
			status.Message = ""
			// status.Events = nil
		}
	}
}

func checkDeploymentPodsReady(ctx context.Context, status *edgev1beta1.EdgeTaskResourceStatus, deployment *appsv1.Deployment, edgecli client.Client) {
	log := logr.FromContextOrDiscard(ctx)
	// found deployment pods
	pods := &corev1.PodList{}
	if err := edgecli.List(ctx, pods, client.InNamespace(deployment.Namespace), client.MatchingLabels(deployment.Spec.Selector.MatchLabels)); err != nil {
		log.Error(err, "failed to list pods")
		return
	}
	// fill events for those pods
	allpodevents := []edgev1beta1.EdgeTaskResourceEvent{}
	for _, pod := range pods.Items {
		events := &corev1.EventList{}
		if err := edgecli.List(ctx, events, client.InNamespace(deployment.Namespace), client.MatchingFields{"involvedObject.uid": string(pod.UID)}); err != nil {
			log.Error(err, "failed to list events")
		}
		items := make([]edgev1beta1.EdgeTaskResourceEvent, 0, len(events.Items))
		for _, event := range events.Items {
			items = append(items, edgev1beta1.EdgeTaskResourceEvent{
				Type:          event.Type,
				Reason:        event.Reason,
				Message:       event.Message,
				Count:         event.Count,
				LastTimestamp: event.LastTimestamp,
				InvolvedObject: corev1.ObjectReference{
					APIVersion: event.InvolvedObject.APIVersion,
					Namespace:  event.InvolvedObject.Namespace,
					Name:       event.InvolvedObject.Name,
					Kind:       event.InvolvedObject.Kind,
				},
			})
		}
		allpodevents = append(allpodevents, items...)
	}
	// sort events by last timestamp
	slices.SortFunc(allpodevents, func(i, j edgev1beta1.EdgeTaskResourceEvent) bool {
		return i.LastTimestamp.Before(&j.LastTimestamp)
	})
	status.Events = allpodevents
}
