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

	loggingv1beta1 "github.com/banzaicloud/logging-operator/pkg/sdk/logging/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/apis/gems"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *LabelInjectorMutate) CommonInjectLabel(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case v1.Create, v1.Update:
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
		case "Flow":
			obj = &loggingv1beta1.Flow{}
		case "Output":
			obj = &loggingv1beta1.Output{}
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
			r.Log.Error(err, "getmeta")
			return admission.Allowed("pass")
		}
		originlabels := metadata.GetLabels()
		if originlabels == nil {
			originlabels = map[string]string{}
		}

		ns, err := r.getAndMutateNS(ctx, req)
		if err != nil {
			r.Log.Error(err, "getAndMutateNSLabel")
			return admission.Allowed("pass")
		}

		newLabels := r.getComonLabels(ctx, ns)
		for k, v := range newLabels {
			originlabels[k] = v
		}

		metadata.SetLabels(originlabels)
		modifyed, e := json.Marshal(metadata)
		if e != nil {
			r.Log.WithName(req.Name).WithName(req.Namespace).Error(e, "error")
		}
		return admission.PatchResponseFromRaw(req.Object.Raw, modifyed)
	case v1.Delete:
		_, err := r.getAndMutateNS(ctx, req)
		if err != nil {
			r.Log.Error(err, "getAndMutateNSLabel")
			return admission.Allowed("pass")
		}
	default:
		return admission.Allowed("pass")
	}
	return admission.Allowed("pass")
}

func (r *LabelInjectorMutate) getComonLabels(ctx context.Context, ns *corev1.Namespace) map[string]string {
	ret := map[string]string{}
	nslabel := ns.GetLabels()
	for _, label := range gemlabels.CommonLabels {
		if v, exist := nslabel[label]; exist {
			ret[label] = v
		}
	}
	return ret
}

func (r *LabelInjectorMutate) getAndMutateNS(ctx context.Context, req admission.Request) (*corev1.Namespace, error) {
	var ns corev1.Namespace
	if err := r.Client.Get(ctx, types.NamespacedName{Name: req.Namespace}, &ns); err != nil {
		return nil, err
	}
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}

	switch req.Kind.Kind {
	case "Flow":
		if req.Name == "default" {
			switch req.Operation {
			case v1.Create, v1.Update:
				ns.Labels[gems.LabelLogCollector] = gems.StatusEnabled
			case v1.Delete:
				ns.Labels[gems.LabelLogCollector] = gems.StatusDisabled
			}
		}
	}
	return &ns, r.Client.Update(ctx, &ns)
}
