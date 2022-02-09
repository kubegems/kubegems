package webhooks

import (
	"context"
	"net/http"

	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	v1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceValidate) ValidateIstioGateway(ctx context.Context, req admission.Request) admission.Response {
	gw := istiov1beta1.Gateway{}
	err := r.decoder.Decode(req, &gw)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case v1.Create, v1.Update:
		if len(gw.Spec.Selector) == 0 {
			return admission.Denied("istio gateway's selector can't be null!")
		}
		return admission.Allowed("pass")
	default:
		return admission.Allowed("pass")
	}
}
