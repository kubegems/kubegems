package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/utils/resourcequota"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceValidate) ValidateTenantResourceQuota(ctx context.Context, req admission.Request) admission.Response {
	trq := &gemsv1beta1.TenantResourceQuota{}
	switch req.Operation {
	case v1.Create, v1.Update:
		if err := r.decoder.DecodeRaw(req.Object, trq); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		capacity, err := r.getClusterCapacity(ctx)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		allocated, err := r.getAllocatedResource(ctx)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		var need corev1.ResourceList
		if req.Operation == v1.Create {
			need = trq.Spec.Hard
		} else {
			oldtrq := &gemsv1beta1.TenantResourceQuota{}
			if err := r.decoder.DecodeRaw(req.OldObject, oldtrq); err != nil {
				return admission.Errored(http.StatusBadRequest, err)
			}
			need = resourcequota.SubResource(oldtrq.Spec.Hard, trq.Spec.Hard)
		}
		enough, errmsg := resourcequota.ResourceEnough(capacity, allocated, need)
		if enough {
			return admission.Allowed("pass")
		}
		return admission.Denied(strings.Join(errmsg, ";"))
	case v1.Delete:
		key := types.NamespacedName{
			Name: req.Name,
		}
		if err := r.Client.Get(ctx, key, trq); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		owner := metav1.GetControllerOf(trq)
		if owner != nil {
			tenant := &gemsv1beta1.Tenant{}
			if err := r.Client.Get(ctx, types.NamespacedName{Name: owner.Name}, tenant); err != nil {
				if errors.IsNotFound(err) {
					return admission.Allowed("pass")
				}
			}
			return admission.Denied(fmt.Sprintf("can not delete tenantresourcequota %s, it's belong to %s/%s", req.Name, owner.Kind, owner.Name))
		}
		return admission.Allowed("pass")
	default:
		return admission.Allowed("pass")
	}
}

func (r *ResourceValidate) getClusterCapacity(ctx context.Context) (corev1.ResourceList, error) {
	total := corev1.ResourceList{}
	nodes := corev1.NodeList{}
	if err := r.Client.List(ctx, &nodes); err != nil {
		return total, err
	}
	for _, node := range nodes.Items {
		for key, value := range node.Status.Capacity {
			if v, exist := total[key]; exist {
				v.Add(value)
				total[key] = v
			} else {
				total[key] = value.DeepCopy()
			}
		}
	}
	return total, nil
}

func (r *ResourceValidate) getAllocatedResource(ctx context.Context) (corev1.ResourceList, error) {
	total := corev1.ResourceList{}
	trqs := gemsv1beta1.TenantResourceQuotaList{}
	if err := r.Client.List(ctx, &trqs); err != nil {
		return total, err
	}
	for _, trq := range trqs.Items {
		for key, value := range trq.Spec.Hard {
			if v, exist := total[key]; exist {
				v.Add(value)
				total[key] = v
			} else {
				total[key] = value.DeepCopy()
			}
		}
	}
	return total, nil
}
