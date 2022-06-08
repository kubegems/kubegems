package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	v1 "k8s.io/api/admission/v1"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/utils/resourcequota"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceMutate) MutateEnvironment(ctx context.Context, req admission.Request) admission.Response {
	env := &gemsv1beta1.Environment{}
	err := r.decoder.Decode(req, env)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case v1.Create, v1.Update:
		envDefault(env)

		modifyed, _ := json.Marshal(env)
		return admission.PatchResponseFromRaw(req.Object.Raw, modifyed)
	default:
		return admission.Allowed("pass")
	}
}

func envDefault(env *gemsv1beta1.Environment) {
	if env.Spec.LimitRage == nil {
		env.Spec.LimitRage = resourcequota.GetDefaultEnvironmentLimitRange()
	}
	if len(env.Spec.LimitRageName) == 0 {
		env.Spec.LimitRageName = resourcequota.DefaultLimitRangeName
	}
	defaultResourceQuota := resourcequota.GetDefaultEnvironmentResourceQuota()
	if env.Spec.ResourceQuota == nil {
		env.Spec.ResourceQuota = defaultResourceQuota
	} else {
		for k, v := range defaultResourceQuota {
			if _, exist := env.Spec.ResourceQuota[k]; !exist {
				env.Spec.ResourceQuota[k] = v
			}
		}
	}
	if len(env.Spec.ResourceQuotaName) == 0 {
		env.Spec.ResourceQuotaName = resourcequota.DefaultResourceQuotaName
	}
	if env.Annotations == nil {
		env.Annotations = make(map[string]string)
	}
}
