package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/utils/resourcequota"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceValidate) ValidateEnvironment(ctx context.Context, req admission.Request) admission.Response {
	/*
		1. 是否存在对应的租户
		2. 资源是否够
		3. LimitRange是否合法
	*/

	env := &gemsv1beta1.Environment{}
	err := r.decoder.Decode(req, env)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	switch req.Operation {
	case v1.Create:
		// 1. 检查租户是否存在,如果环境关联的租户不存在，则不允许创建环境
		if !r.tenantExist(ctx, env.Spec.Tenant) {
			return admission.Denied(fmt.Sprintf("Environment related Tenant %s not exist", env.Spec.Tenant))
		}

		// 检查租户的资源是否足够
		var tenantRq gemsv1beta1.TenantResourceQuota
		var old gemsv1beta1.Environment
		tenantRqKey := types.NamespacedName{
			Name: env.Spec.Tenant,
		}
		if err := r.Client.Get(ctx, tenantRqKey, &tenantRq); err != nil {
			if errors.IsNotFound(err) {
				return admission.Denied("Environment related Tenant has no TenantResourceQuota, can not create or update!")
			} else {
				return admission.Denied("system error, try again!")
			}
		}

		if enough, msgs := r.tenantResourceIsEnough(&tenantRq, env, &old); !enough {
			return admission.Denied(strings.Join(msgs, ";"))
		}

		// 3. 检查LimitRange是否合法
		if errmsg, invalid := resourcequota.IsLimitRangeInvalid(env.Spec.LimitRage); invalid {
			msg := fmt.Sprintf("LimitRange format error: %v", strings.Join(errmsg, ";"))
			return admission.Denied(msg)
		}
		return admission.Allowed("pass")
	case v1.Update:
		var old gemsv1beta1.Environment
		if req.Operation == v1.Update {
			if err := r.decoder.DecodeRaw(req.OldObject, &old); err != nil {
				return admission.Errored(http.StatusBadRequest, err)
			}
		}
		if env.Spec.Tenant != old.Spec.Tenant {
			return admission.Denied("field Tenant is immutable")
		}
		if !equality.Semantic.DeepEqual(env.Spec.ResourceQuota, old.Spec.ResourceQuota) {
			var tenantRq gemsv1beta1.TenantResourceQuota
			tenantRqKey := types.NamespacedName{
				Name: env.Spec.Tenant,
			}
			if err := r.Client.Get(ctx, tenantRqKey, &tenantRq); err != nil {
				if errors.IsNotFound(err) {
					return admission.Denied("Environment related Tenant has no TenantResourceQuota, can not create or update!")
				} else {
					return admission.Denied("system error, try again!")
				}
			}

			if enough, msgs := r.tenantResourceIsEnough(&tenantRq, env, &old); !enough {
				return admission.Denied(strings.Join(msgs, ";"))
			}
		}
		if errmsg, invalid := resourcequota.IsLimitRangeInvalid(env.Spec.LimitRage); invalid {
			msg := fmt.Sprintf("LimitRange format error: %v", strings.Join(errmsg, ";"))
			return admission.Denied(msg)
		}
		return admission.Allowed("pass")
	default:
		return admission.Allowed("pass")
	}
}

func (r *ResourceValidate) tenantExist(ctx context.Context, tenantName string) bool {
	if tenantName == notenant {
		return true
	}
	var tenant gemsv1beta1.Tenant
	tenantKey := types.NamespacedName{
		Name: tenantName,
	}
	if err := r.Client.Get(ctx, tenantKey, &tenant); err != nil {
		r.Log.Error(err, "failed to get tenant")
		return false
	}
	return true
}

func (r *ResourceValidate) tenantResourceIsEnough(trq *gemsv1beta1.TenantResourceQuota, env, old *gemsv1beta1.Environment) (bool, []string) {
	allocated := corev1.ResourceList{}
	if old != nil {
		oldres := old.Spec.ResourceQuota
		for _, key := range resourcequota.TenantLimitResources {
			allocatedv := trq.Status.Allocated[key]
			oldv, oexist := oldres[key]
			if oexist {
				allocatedv.Sub(oldv)
			}
			allocated[key] = allocatedv
		}
	} else {
		allocated = trq.Status.Allocated
	}
	return resourcequota.ResourceIsEnough(trq.Spec.Hard, allocated, env.Spec.ResourceQuota, resourcequota.TenantLimitResources)
}
