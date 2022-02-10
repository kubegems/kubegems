package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	v1 "k8s.io/api/core/v1"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/controller/utils"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceMutate) MutateTenantResourceQuota(ctx context.Context, req admission.Request) admission.Response {
	trq := &gemsv1beta1.TenantResourceQuota{}

	err := r.decoder.Decode(req, trq)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	defaultrq := utils.GetDefaultTeantResourceQuota()
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
