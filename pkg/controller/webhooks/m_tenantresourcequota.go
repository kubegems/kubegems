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

	v1 "k8s.io/api/core/v1"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/utils/resourcequota"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceMutate) MutateTenantResourceQuota(ctx context.Context, req admission.Request) admission.Response {
	trq := &gemsv1beta1.TenantResourceQuota{}

	err := r.decoder.Decode(req, trq)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	defaultrq := resourcequota.GetDefaultTeantResourceQuota()
	if trq.Spec.Hard == nil {
		trq.Spec.Hard = make(v1.ResourceList)
	}
	for key, value := range defaultrq {
		if _, exist := trq.Spec.Hard[key]; !exist {
			trq.Spec.Hard[key] = value
		}
	}

	mrq, _ := json.Marshal(trq)
	return admission.PatchResponseFromRaw(req.Object.Raw, mrq)
}
