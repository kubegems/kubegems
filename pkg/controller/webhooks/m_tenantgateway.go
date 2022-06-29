package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	ingressclassgvk, err := r.Client.RESTMapper().ResourceFor(schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Resource: "ingressclasses",
	})
	if err != nil {
		return admission.Denied(fmt.Sprintf("get ingressclass gvk failed: %v", err))
	}

	// https://github.com/nginxinc/kubernetes-ingress/issues/1832
	tag := "1.11.1"
	if ingressclassgvk.Version == "v1" {
		// https://github.com/nginxinc/kubernetes-ingress/releases/tag/v2.0.0
		// This is the first version to support ingressclass v1
		// Don't upgrate it, or you'll get error like nginx.conf template not valid
		tag = "2.0.0"
	}
	r.Log.Info(fmt.Sprintf("use tag: %s because of ingressclass version is: %s", tag, ingressclassgvk.Version))

	switch req.Operation {
	case v1.Create, v1.Update:
		tgDefault(tg, r.Repo, tag)
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
	if tg.Spec.Workload.ExtraLabels["name"] == "" {
		tg.Spec.Workload.ExtraLabels["name"] = "nginx-ingress-operator" // 监控的servicemonitor 筛选 pod
	}
	if tg.Spec.BaseDomain == "" {
		tg.Spec.BaseDomain = "*.kubegems.io"
	}

	if tg.Spec.Image.Repository == "" {
		tg.Spec.Image.Repository = repo
	}
	if tg.Spec.Image.Tag == "" {
		tg.Spec.Image.Tag = tag
	}
	if tg.Spec.Image.PullPolicy == "" {
		tg.Spec.Image.PullPolicy = string(corev1.PullIfNotPresent)
	}
}
