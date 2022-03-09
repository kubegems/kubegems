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

func (r *ResourceMutate) MutateDeployment(ctx context.Context, req admission.Request) admission.Response {
	dep := &appsv1.Deployment{}
	err := r.decoder.Decode(req, dep)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	limits := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("512m"),
		corev1.ResourceMemory: resource.MustParse("1Gi"),
	}
	requests := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("10m"),
		corev1.ResourceMemory: resource.MustParse("10Mi"),
	}
	var limitRangeItems []corev1.LimitRangeItem
	var lr corev1.LimitRange
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: dep.Namespace, Name: "default"}, &lr); err != nil {
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
		patchResoucesIfNotExists(dep.Spec.Template.Spec.Containers, limits, requests)
		patchResoucesIfNotExists(dep.Spec.Template.Spec.InitContainers, limits, requests)
		patchEpherContainerResoucesIfNotExists(dep.Spec.Template.Spec.EphemeralContainers, limits, requests)
		modifyed, _ := json.Marshal(dep)
		return admission.PatchResponseFromRaw(req.Object.Raw, modifyed)
	default:
		return admission.Allowed("pass")
	}
}
