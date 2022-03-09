package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	v1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/pkg/utils/resourcequota"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceMutate) MutateDaemonSet(ctx context.Context, req admission.Request) admission.Response {
	ds := &appsv1.DaemonSet{}
	err := r.decoder.Decode(req, ds)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	limits := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("250m"),
		corev1.ResourceMemory: resource.MustParse("512Gi"),
	}
	requests := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("10m"),
		corev1.ResourceMemory: resource.MustParse("10Mi"),
	}
	var limitRangeItems []corev1.LimitRangeItem
	var lr corev1.LimitRange
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ds.Namespace, Name: "default"}, &lr); err != nil {
		limitRangeItems = resourcequota.GetDefaultEnvironmentLimitRange()
	} else {
		limitRangeItems = lr.Spec.Limits
	}
	for _, item := range limitRangeItems {
		if item.Type == corev1.LimitTypeContainer {
			limits = item.Default
			requests = item.DefaultRequest
		}
	}
	switch req.Operation {
	case v1.Create, v1.Update:
		patchResoucesIfNotExists(ds.Spec.Template.Spec.Containers, limits, requests)
		patchResoucesIfNotExists(ds.Spec.Template.Spec.InitContainers, limits, requests)
		patchEpherContainerResoucesIfNotExists(ds.Spec.Template.Spec.EphemeralContainers, limits, requests)
		modifyed, _ := json.Marshal(ds)
		return admission.PatchResponseFromRaw(req.Object.Raw, modifyed)
	default:
		return admission.Allowed("pass")
	}
}
