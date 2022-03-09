package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	ext_v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	gemlabels "kubegems.io/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/apis/networking"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	defaultGatewayName = "default-gateway"
	notenant           = "notenant"
	lowercase          = []rune("abcdefghijklmnopqrstuvwxyz")
)

func (r *ResourceMutate) MutateIngress(ctx context.Context, req admission.Request) admission.Response {
	ingress := &ext_v1beta1.Ingress{}
	err := r.decoder.Decode(req, ingress)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case v1.Create, v1.Update:
		// 指定ingressclass的，校验租户网关
		ingressClassName := ""
		if ingress.Spec.IngressClassName != nil {
			ingressClassName = *ingress.Spec.IngressClassName
		}

		// 默认网关创建
		// ingress生成随机域名
		var tgs gemsv1beta1.TenantGatewayList
		if err := r.Client.List(ctx, &tgs, client.MatchingLabels{
			networking.LabelIngressClass: ingressClassName,
		}); err != nil {
			log.Log.Error(err, "list gateway failed")
			return admission.Denied(fmt.Sprintf("list gateway failed: %v", err))
		}

		// 没找到对应的租户网关，不接管
		if len(tgs.Items) == 0 {
			log.Log.Info(fmt.Sprintf("no such ingressclass: [%s]", ingressClassName))
			return admission.Allowed("pass")
		}

		for i := range ingress.Spec.Rules {
			if ingress.Spec.Rules[i].Host == "" {
				ingress.Spec.Rules[i].Host = randHost(tgs.Items[0].Spec.BaseDomain)
			}
		}

		if err := CheckGatewayAndIngressProtocol(tgs.Items[0], []ext_v1beta1.Ingress{*ingress}); err != nil {
			log.Log.Error(err, "ingress and gateway protocol check failed")
			return admission.Denied(err.Error())
		}

		for j := range ingress.Spec.TLS {
			if len(ingress.Spec.TLS[j].Hosts) == 0 {
				return admission.Denied("ingress tls must specify at least one host")
			}
		}
		if ingress.Labels == nil {
			ingress.Labels = make(map[string]string)
		}
		// ingress打上 ingressClass标签
		var ns corev1.Namespace
		if err := r.Client.Get(ctx, types.NamespacedName{Name: ingress.Namespace}, &ns); err != nil {
			return admission.Denied(err.Error())
		}
		nslabel := ns.GetLabels()
		for _, label := range gemlabels.CommonLabels {
			if v, ok := nslabel[label]; ok {
				ingress.Labels[label] = v
			}
		}
		ingress.Labels[networking.LabelIngressClass] = ingressClassName

		modifyed, _ := json.Marshal(ingress)
		return admission.PatchResponseFromRaw(req.Object.Raw, modifyed)
	default:
		return admission.Allowed("pass")
	}
}

func randHost(old string) string {
	tmp := strings.Split(old, ".")
	for i := range tmp {
		if tmp[i] == "*" {
			rands := make([]rune, 5)
			for i := range rands {
				rands[i] = lowercase[rand.Intn(len(lowercase))]
			}
			tmp[i] = string(rands)
		}
	}
	return strings.Join(tmp, ".")
}
