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
	"fmt"
	"net/http"

	v1 "k8s.io/api/admission/v1"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceValidate) ValidateTenant(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case v1.Delete:
		var envs gemsv1beta1.EnvironmentList
		r.Client.List(ctx, &envs)
		if len(envs.Items) > 0 {
			return admission.Denied(fmt.Sprintf("current tenant have %v Environments in this cluster, please delete these Environments first!", req.Name))
		}
		return admission.Allowed("pass")
	case v1.Update:
		oldtenant := &gemsv1beta1.Tenant{}
		newtenant := &gemsv1beta1.Tenant{}
		nerr := r.decoder.DecodeRaw(req.Object, newtenant)
		olderr := r.decoder.DecodeRaw(req.OldObject, oldtenant)
		if nerr != nil {
			return admission.Errored(http.StatusBadRequest, nerr)
		}
		if olderr != nil {
			return admission.Errored(http.StatusBadRequest, olderr)
		}
		if oldtenant.Spec.TenantName != newtenant.Spec.TenantName {
			return admission.Denied("Field \"tenantName\" is immutable")
		}
		return admission.Allowed("pass")
	default:
		return admission.Allowed("pass")
	}
}
