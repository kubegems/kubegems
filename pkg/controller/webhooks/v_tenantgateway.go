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
	"strings"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/apis/networking"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceValidate) ValidateTenantGateway(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case v1.Create, v1.Update:
		tg := &gemsv1beta1.TenantGateway{}
		err := r.decoder.Decode(req, tg)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if errs := validation.IsDNS1035Label(tg.Name); len(errs) > 0 {
			return admission.Denied(strings.Join(errs, ", "))
		}

		// 非删除操作才检测租户
		if tg.ObjectMeta.DeletionTimestamp.IsZero() {
			// 1. 检查租户是否存在,如果环境关联的租户不存在，则不允许创建
			if !r.tenantExist(ctx, tg.Spec.Tenant) {
				return admission.Denied(fmt.Sprintf("Gateway related Tenant %s not exist", tg.Spec.Tenant))
			}
		}

		if tg.Spec.Type != "" && tg.Spec.Type != corev1.ServiceTypeNodePort && tg.Spec.Type != corev1.ServiceTypeLoadBalancer {
			return admission.Denied(fmt.Sprintf("Gateway type %s not valid, must be NodePort or LoadBalancer", tg.Spec.Type))
		}

		// 必须有通配符*
		hasWildcard := false
		for _, v := range strings.Split(tg.Spec.BaseDomain, ".") {
			if v == "*" {
				hasWildcard = true
			}
		}
		if !hasWildcard {
			return admission.Denied(fmt.Sprintf("Gateway baseDomain %s must has a wildcard '*'", tg.Spec.BaseDomain))
		}

		// 校验gateway、ingress是否同步
		ingressList := networkingv1.IngressList{}
		if err := r.Client.List(ctx, &ingressList, client.MatchingLabels(map[string]string{
			networking.LabelIngressClass: tg.Labels[networking.LabelIngressClass],
		})); err != nil {
			return admission.Denied(err.Error())
		}
		if err := CheckGatewayAndIngressProtocol(*tg, ingressList.Items); err != nil {
			return admission.Denied(err.Error())
		}

		return admission.Allowed("pass")
	default:
		return admission.Allowed("pass")
	}
}
