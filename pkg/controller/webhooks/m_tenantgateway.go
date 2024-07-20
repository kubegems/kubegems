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
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/apis/networking"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceMutate) MutateTenantGateway(ctx context.Context, req admission.Request) admission.Response {
	tg := &gemsv1beta1.TenantGateway{}
	err := r.decoder.Decode(req, tg)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	switch req.Operation {
	case v1.Create, v1.Update:
		tgDefault(tg, r.Repo, "v1.3.0")
		modifyed, _ := json.Marshal(tg)
		return admission.PatchResponseFromRaw(req.Object.Raw, modifyed)
	default:
		return admission.Allowed("pass")
	}
}

func tgDefault(tg *gemsv1beta1.TenantGateway, repo, tag string) {
	defaultReplicas := int32(1)
	if tg.Labels == nil {
		tg.Labels = make(map[string]string)
	}
	if tg.Labels[gemlabels.LabelTenant] == "" {
		tg.Labels[gemlabels.LabelTenant] = tg.Spec.Tenant
	}
	if tg.Labels[networking.LabelIngressClass] == "" {
		tg.Labels[networking.LabelIngressClass] = tg.Name + "-" + tg.Spec.Tenant
	}

	if tg.Spec.Replicas == nil || *tg.Spec.Replicas <= 0 {
		tg.Spec.Replicas = &defaultReplicas
	}

	if tg.Spec.Type == "" {
		tg.Spec.Type = corev1.ServiceTypeNodePort
	}

	if tg.Spec.IngressClass == "" {
		tg.Spec.IngressClass = tg.Name + "-" + tg.Spec.Tenant
	}

	// nginx的pod上需添加租户信息，用于日志收集
	if tg.Spec.Workload == nil {
		tg.Spec.Workload = &gemsv1beta1.Workload{}
	}
	if tg.Spec.Workload.ExtraLabels == nil {
		tg.Spec.Workload.ExtraLabels = make(map[string]string)
	}
	if tg.Spec.Workload.ExtraLabels[gemlabels.LabelTenant] == "" {
		tg.Spec.Workload.ExtraLabels[gemlabels.LabelTenant] = tg.Spec.Tenant
	}
	if tg.Spec.Workload.ExtraLabels[gemlabels.LabelApplication] == "" {
		tg.Spec.Workload.ExtraLabels[gemlabels.LabelApplication] = tg.Name
	}
	if tg.Spec.Workload.ExtraLabels[gemlabels.LabelGatewayType] == "" {
		tg.Spec.Workload.ExtraLabels[gemlabels.LabelGatewayType] = "ingress-nginx" // 监控的servicemonitor 筛选 pod
	}
	if tg.Spec.BaseDomain == "" {
		tg.Spec.BaseDomain = "*.kubegems.io"
	}

	if tg.Spec.Image == nil {
		tg.Spec.Image = &gemsv1beta1.Image{}
	}
	if tg.Spec.Image.Repository == "" {
		tg.Spec.Image.Repository = repo
	}
	if tg.Spec.Image.Tag == "" {
		tg.Spec.Image.Tag = tag
	}
	if tg.Spec.Image.PullPolicy == "" {
		tg.Spec.Image.PullPolicy = corev1.PullIfNotPresent
	}
}
