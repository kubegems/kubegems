package webhooks

import (
	"context"
	"encoding/json"
	"errors"

	v1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"

	gemlabels "github.com/kubegems/gems/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *LabelInjectorMutate) CommonInjectLabel(ctx context.Context, req admission.Request) admission.Response {
	var obj runtime.Object
	switch req.Kind.Kind {
	case "Pod":
		obj = &corev1.Pod{}
	case "ConfigMap":
		obj = &corev1.ConfigMap{}
	case "Secret":
		obj = &corev1.Secret{}
	case "Service":
		obj = &corev1.Service{}
	case "Job":
		obj = &batchv1.Job{}
	case "CronJob":
		obj = &batchv1beta1.CronJob{}
	case "Deployment":
		obj = &appsv1.Deployment{}
	case "StatefulSet":
		obj = &appsv1.StatefulSet{}
	case "DaemonSet":
		obj = &appsv1.DaemonSet{}
	case "PersistentVolumeClaim":
		obj = &corev1.PersistentVolumeClaim{}
	default:
		r.Log.Error(errors.New("not support"), req.Kind.String())
		return admission.Allowed("pass")
	}

	if err := r.decoder.Decode(req, obj); err != nil {
		r.Log.Error(err, "failed ")
		return admission.Allowed("pass")
	}

	metadata, err := meta.Accessor(obj)
	if err != nil {
		return admission.Allowed("pass")
	}
	originlabels := metadata.GetLabels()
	if originlabels == nil {
		originlabels = map[string]string{}
	}
	newLabels := r.GetComonLabels(ctx, req.Namespace)
	for k, v := range newLabels {
		originlabels[k] = v
	}

	metadata.SetLabels(originlabels)
	switch req.Operation {
	case v1.Create, v1.Update:
		modifyed, e := json.Marshal(metadata)
		if e != nil {
			r.Log.WithName(req.Name).WithName(req.Namespace).Error(e, "error")
		}
		return admission.PatchResponseFromRaw(req.Object.Raw, modifyed)
	default:
		return admission.Allowed("pass")
	}
}

func (r *LabelInjectorMutate) GetComonLabels(ctx context.Context, namespace string) map[string]string {
	ret := map[string]string{}
	var ns corev1.Namespace
	if e := r.Client.Get(ctx, types.NamespacedName{Name: namespace}, &ns); e != nil {
		return ret
	}
	nslabel := ns.GetLabels()
	for _, label := range gemlabels.CommonLabels {
		if v, exist := nslabel[label]; exist {
			ret[label] = v
		}
	}
	return ret
}
