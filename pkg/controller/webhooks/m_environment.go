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
	corev1 "k8s.io/api/core/v1"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/utils/resourcequota"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceMutate) MutateEnvironment(ctx context.Context, req admission.Request) admission.Response {
	env := &gemsv1beta1.Environment{}
	err := r.decoder.Decode(req, env)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case v1.Create, v1.Update:
		envDefault(env)

		modifyed, _ := json.Marshal(env)
		return admission.PatchResponseFromRaw(req.Object.Raw, modifyed)
	default:
		return admission.Allowed("pass")
	}
}

func envDefault(env *gemsv1beta1.Environment) {
	if env.Spec.LimitRage == nil {
		env.Spec.LimitRage = resourcequota.GetDefaultEnvironmentLimitRange()
	}
	if len(env.Spec.LimitRageName) == 0 {
		env.Spec.LimitRageName = resourcequota.DefaultLimitRangeName
	}

	if env.Spec.ResourceQuota == nil {
		env.Spec.ResourceQuota = corev1.ResourceList{}
	}
	for resourceName, defaultQuantity := range resourcequota.GetDefaultEnvironmentResourceQuota() {
		if _, exist := env.Spec.ResourceQuota[resourceName]; !exist {
			env.Spec.ResourceQuota[resourceName] = defaultQuantity
		}
	}

	if len(env.Spec.ResourceQuotaName) == 0 {
		env.Spec.ResourceQuotaName = resourcequota.DefaultResourceQuotaName
	}
	if env.Annotations == nil {
		env.Annotations = make(map[string]string)
	}
}
