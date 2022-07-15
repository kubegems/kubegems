// Copyright 2022 The kubegems.io Authors
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
	"kubegems.io/kubegems/pkg/utils/resourcequota"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceMutate) MutateStatefulSet(ctx context.Context, req admission.Request) admission.Response {
	sts := &appsv1.StatefulSet{}
	err := r.decoder.Decode(req, sts)
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
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: sts.Namespace, Name: "default"}, &lr); err != nil {
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
		patchResoucesIfNotExists(sts.Spec.Template.Spec.Containers, limits, requests)
		patchResoucesIfNotExists(sts.Spec.Template.Spec.InitContainers, limits, requests)
		patchEpherContainerResoucesIfNotExists(sts.Spec.Template.Spec.EphemeralContainers, limits, requests)
		modifyed, _ := json.Marshal(sts)
		return admission.PatchResponseFromRaw(req.Object.Raw, modifyed)
	default:
		return admission.Allowed("pass")
	}
}
