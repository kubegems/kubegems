package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	ext_v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	gemlabels "kubegems.io/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/controller/utils"
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
		ingressList := ext_v1beta1.IngressList{}
		if err := r.Client.List(ctx, &ingressList, client.MatchingLabels(map[string]string{
			gemlabels.LabelIngressClass: tg.Labels[gemlabels.LabelIngressClass],
		})); err != nil {
			return admission.Denied(err.Error())
		}
		if err := utils.CheckGatewayAndIngressProtocol(*tg, ingressList.Items); err != nil {
			return admission.Denied(err.Error())
		}

		return admission.Allowed("pass")
	case v1.Delete:
		tg := &gemsv1beta1.TenantGateway{}
		err := r.Client.Get(ctx, types.NamespacedName{
			Namespace: req.Namespace,
			Name:      req.Name,
		}, tg)
		if err != nil {
			return admission.Denied(err.Error())
		}

		ingressList := ext_v1beta1.IngressList{}
		if err := r.Client.List(ctx, &ingressList, client.MatchingLabels(map[string]string{
			gemlabels.LabelIngressClass: tg.Labels[gemlabels.LabelIngressClass],
		})); err != nil {
			return admission.Denied(err.Error())
		}
		if len(ingressList.Items) > 0 {
			return admission.Denied(fmt.Sprintf("网关: %s 还有关联的路由", tg.Name))
		}
		return admission.Allowed("pass")
	default:
		return admission.Allowed("pass")
	}
}
