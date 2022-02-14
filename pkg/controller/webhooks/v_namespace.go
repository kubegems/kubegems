package webhooks

import (
	"context"
	"fmt"
	"net/http"

	gemsv1beta1 "github.com/kubegems/gems/pkg/apis/gems/v1beta1"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ResourceValidate) ValidateNamespace(ctx context.Context, req admission.Request) admission.Response {
	ns := &corev1.Namespace{}
	key := types.NamespacedName{
		Name: req.Name,
	}
	if err := r.Client.Get(ctx, key, ns); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	switch req.Operation {
	case v1.Delete:
		owner := metav1.GetControllerOf(ns)
		if owner != nil {
			env := &gemsv1beta1.Environment{}
			if err := r.Client.Get(ctx, types.NamespacedName{Name: owner.Name}, env); err != nil {
				if errors.IsNotFound(err) {
					return admission.Allowed("pass")
				}
			}
			return admission.Denied(fmt.Sprintf("can not delete namespace %s, it's belong to %s/%s", req.Name, owner.Kind, owner.Name))
		}
		return admission.Allowed("pass")
	default:
		return admission.Allowed("pass")
	}
}
