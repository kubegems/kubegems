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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceValidate) ValidateTenantNetworkPolicy(ctx context.Context, req admission.Request) admission.Response {
	tnetpol := &gemsv1beta1.TenantNetworkPolicy{}
	key := types.NamespacedName{
		Name: req.Name,
	}
	switch req.Operation {
	case v1.Create, v1.Update:
		// TODO:
		return admission.Allowed("pass")
	case v1.Delete:
		if err := r.Client.Get(ctx, key, tnetpol); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		owner := metav1.GetControllerOf(tnetpol)
		if owner != nil {
			tenant := &gemsv1beta1.Tenant{}
			if err := r.Client.Get(ctx, types.NamespacedName{Name: owner.Name}, tenant); err != nil {
				if errors.IsNotFound(err) {
					return admission.Allowed("pass")
				}
			}
			return admission.Denied(fmt.Sprintf("can not delete tenantNetworkPolicy %s, it's belong to %s/%s", req.Name, owner.Kind, owner.Name))
		}
		return admission.Allowed("pass")
	default:
		return admission.Allowed("pass")
	}
}
