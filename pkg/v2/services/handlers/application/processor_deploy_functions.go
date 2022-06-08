package application

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/service/handlers/noproxy"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (p *ApplicationProcessor) GetHorizontalPodAutoscaler(ctx context.Context, ref PathRef) (*v2beta1.HorizontalPodAutoscaler, error) {
	var ret *v2beta1.HorizontalPodAutoscaler

	err := p.Manifest.StoreFunc(ctx, ref, func(ctx context.Context, store GitStore) error {
		workload, err := ParseMainWorkload(ctx, store)
		if err != nil {
			return err
		}
		// type check
		switch workload.(type) {
		case *appsv1.Deployment:
		case *appsv1.StatefulSet:
		case *batchv1.Job:
		default:
			return fmt.Errorf("unsupported workload type %s", workload.GetObjectKind().GroupVersionKind())
		}

		sc := &v2beta1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      noproxy.FormatHPAName(workload.GetObjectKind().GroupVersionKind().Kind, workload.GetName()),
				Namespace: workload.GetNamespace(),
			},
		}
		if err := store.Get(ctx, client.ObjectKeyFromObject(sc), sc); err != nil {
			return err
		}
		ret = sc
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (p *ApplicationProcessor) DeleteHorizontalPodAutoscaler(ctx context.Context, ref PathRef) error {
	updatefun := func(ctx context.Context, store GitStore) error {
		hpalist := &v2beta1.HorizontalPodAutoscalerList{}
		if err := store.List(ctx, hpalist); err != nil {
			return err
		}
		for _, v := range hpalist.Items {
			_ = store.Delete(ctx, &v)
		}
		return nil
	}
	return p.Manifest.StoreUpdateFunc(ctx, ref, updatefun, "remove hpa")
}

func (p *ApplicationProcessor) SetHorizontalPodAutoscaler(ctx context.Context, ref PathRef, scalerMetrics HPAMetrics) error {
	updatefun := func(_ context.Context, store GitStore) error {
		workload, err := ParseMainWorkload(ctx, store)
		if err != nil {
			return err
		}
		// type check
		switch workload.(type) {
		case *appsv1.Deployment:
		case *appsv1.StatefulSet:
		case *batchv1.Job:
		default:
			return fmt.Errorf("unsupported workload type %s", workload.GetObjectKind().GroupVersionKind())
		}

		name := workload.GetName()
		namespace := workload.GetNamespace()
		gv := appsv1.SchemeGroupVersion
		kind := workload.GetObjectKind().GroupVersionKind().Kind

		sc := &v2beta1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      noproxy.FormatHPAName(kind, name),
				Namespace: namespace,
			},
		}
		scalerSpec := v2beta1.HorizontalPodAutoscalerSpec{
			MinReplicas: scalerMetrics.MinReplicas,
			MaxReplicas: scalerMetrics.MaxReplicas,
			ScaleTargetRef: v2beta1.CrossVersionObjectReference{
				Kind:       kind,
				Name:       name,
				APIVersion: gv.Identifier(),
			},
			Metrics: func() []v2beta1.MetricSpec {
				var metrics []v2beta1.MetricSpec
				if scalerMetrics.Cpu > 0 {
					metrics = append(metrics, v2beta1.MetricSpec{
						Type: v2beta1.ResourceMetricSourceType,
						Resource: &v2beta1.ResourceMetricSource{
							Name:                     v1.ResourceCPU,
							TargetAverageUtilization: &scalerMetrics.Cpu,
						},
					})
				}
				if scalerMetrics.Memory > 0 {
					metrics = append(metrics, v2beta1.MetricSpec{
						Type: v2beta1.ResourceMetricSourceType,
						Resource: &v2beta1.ResourceMetricSource{
							Name:                     v1.ResourceMemory,
							TargetAverageUtilization: &scalerMetrics.Memory,
						},
					})
				}
				return metrics
			}(),
		}
		_, err = controllerutil.CreateOrUpdate(ctx, store, sc, func() error {
			// update spec
			sc.Spec = scalerSpec
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	}
	return p.Manifest.StoreUpdateFunc(ctx, ref, updatefun, "update hpa")
}

func (p *ApplicationProcessor) GetReplicas(ctx context.Context, ref PathRef) (*int32, error) {
	var replicas *int32
	_ = p.Manifest.StoreFunc(ctx, ref, func(ctx context.Context, store GitStore) error {
		workload, _ := ParseMainWorkload(ctx, store)
		switch app := workload.(type) {
		case *appsv1.Deployment:
			replicas = app.Spec.Replicas
		case *appsv1.StatefulSet:
			replicas = app.Spec.Replicas
		}
		return nil
	})
	return replicas, nil
}

func (p *ApplicationProcessor) SetReplicas(ctx context.Context, ref PathRef, replicas *int32) error {
	updatefunc := func(ctx context.Context, store GitStore) error {
		workload, err := ParseMainWorkload(ctx, store)
		if err != nil {
			return err
		}
		switch app := workload.(type) {
		case *appsv1.Deployment:
			app.Spec.Replicas = replicas
			return store.Update(ctx, app)
		case *appsv1.StatefulSet:
			app.Spec.Replicas = replicas
			return store.Update(ctx, app)
		default:
			return fmt.Errorf("unsupported scale workload: %s", workload.GetResourceVersion())
		}
	}
	return p.Manifest.StoreUpdateFunc(ctx, ref, updatefunc, fmt.Sprintf("scale replicas to %v", replicas))
}
