/*
Copyright 2021 kubegems.io.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	nginxv1beta1 "kubegems.io/ingress-nginx-operator/api/v1beta1"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

//+kubebuilder:rbac:groups=gems.kubegems.io,resources=tenantgateways,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=gems.kubegems.io,resources=tenantgateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.kubegems.io,resources=nginxingresscontrollers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=extensions,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// TenantGatewayReconciler reconciles a TenantGateway object
type TenantGatewayReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *TenantGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&source.Kind{Type: &corev1.Service{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(r.ServeicePredicate())).
		Watches(&source.Kind{Type: &appsv1.Deployment{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(r.DeploymentPredicate())).
		For(&gemsv1beta1.TenantGateway{}).
		Complete(r)
}

func (r *TenantGatewayReconciler) ServeicePredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			newSvc, okn := e.ObjectNew.(*corev1.Service)
			oldSvc, oko := e.ObjectOld.(*corev1.Service)
			if !okn || !oko {
				return false
			}
			if newSvc.Namespace != gemlabels.NamespaceGateway {
				return false
			}
			return !equality.Semantic.DeepEqual(newSvc.Spec.Ports, oldSvc.Spec.Ports)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (r *TenantGatewayReconciler) DeploymentPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldDep, oko := e.ObjectOld.(*appsv1.Deployment)
			newDep, okn := e.ObjectNew.(*appsv1.Deployment)
			if !oko || !okn {
				return false
			}
			if newDep.Namespace != gemlabels.NamespaceGateway {
				return false
			}
			return !equality.Semantic.DeepEqual(oldDep.Status.AvailableReplicas, newDep.Status.AvailableReplicas)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (r *TenantGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("tenantgateway", req.NamespacedName)

	tg := &gemsv1beta1.TenantGateway{}
	if err := r.Get(ctx, req.NamespacedName, tg); err != nil {
		log.Error(err, "Faild to get TenantGateway")
		return ctrl.Result{}, nil
	}
	if !tg.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(tg, gemlabels.FinalizerGateway) {
			return ctrl.Result{}, nil
		}
		// 删除tenantGateway
		if err := r.Remove(ctx, tg); err != nil {
			log.Error(err, "failed to delete tenantGateway")
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(tg, gemlabels.FinalizerGateway)
		if err := r.Update(ctx, tg); err != nil {
			log.Error(err, "failed to delete tenantGateway")
			return ctrl.Result{Requeue: true}, err
		}
		log.Info("success to delete tenantGateway")
		return ctrl.Result{}, nil
	}
	// 检查并加上finalizer
	if !controllerutil.ContainsFinalizer(tg, gemlabels.FinalizerGateway) {
		controllerutil.AddFinalizer(tg, gemlabels.FinalizerGateway)
		if err := r.Update(ctx, tg); err != nil {
			log.Error(err, "failed to update tenantGateway")
			return ctrl.Result{}, nil
		}
	}
	if err := r.Sync(ctx, tg); err != nil {
		log.Error(err, "failed to sync tenantGateway")
		return ctrl.Result{}, err
	}
	log.Info("success to sync tenantGateway")
	return ctrl.Result{}, nil
}

func (r *TenantGatewayReconciler) Remove(ctx context.Context, tgw *gemsv1beta1.TenantGateway) error {
	ic := &nginxv1beta1.NginxIngressController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tgw.Name,
			Namespace: gemlabels.NamespaceGateway,
		},
	}
	return client.IgnoreNotFound(r.Delete(ctx, ic))
}

func (r *TenantGatewayReconciler) Sync(ctx context.Context, tg *gemsv1beta1.TenantGateway) error {
	log := r.Log.WithValues("tenantgateway", tg.Name)

	ic := &nginxv1beta1.NginxIngressController{
		ObjectMeta: metav1.ObjectMeta{Namespace: gemlabels.NamespaceGateway, Name: tg.Name},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, ic, func() error {
		return r.syncNginxIngressControllerForTenantGateway(tg, ic)
	}); err != nil {
		log.Error(err, "Error create or update NginxIngressController")
		return err
	}

	objectKey := types.NamespacedName{
		Namespace: gemlabels.NamespaceGateway, // gateway资源都在这里
		Name:      tg.Name,
	}

	// 最后处理status
	svc := &corev1.Service{}
	if err := client.IgnoreNotFound(r.Get(ctx, objectKey, svc)); err != nil {
		return nil
	}
	dep := &appsv1.Deployment{}
	if err := client.IgnoreNotFound(r.Get(ctx, objectKey, dep)); err != nil {
		return nil
	}
	original := tg.DeepCopy()
	tg.Status.Ports = svc.Spec.Ports
	tg.Status.AvailableReplicas = dep.Status.AvailableReplicas
	if !equality.Semantic.DeepEqual(original.Status, tg.Status) {
		if err := r.Status().Update(ctx, tg); err != nil {
			log.Error(err, "failed to update tenantGateway")
			return nil
		}
		log.Info("success to update", "gateway status", tg.Status)
	}
	return nil
}

func (r *TenantGatewayReconciler) syncNginxIngressControllerForTenantGateway(tg *gemsv1beta1.TenantGateway, ic *nginxv1beta1.NginxIngressController) error {
	ic.Labels = tg.Labels
	ic.Annotations = tg.Annotations

	if ic.Labels == nil {
		ic.Labels = map[string]string{}
	}
	if ic.Annotations == nil {
		ic.Annotations = map[string]string{}
	}

	ic.Labels[gemlabels.LabelTenant] = tg.Spec.Tenant
	ic.Labels[gemlabels.LabelApplication] = tg.Name

	if svc := tg.Spec.Service; svc != nil {
		ic.Spec.Service = &nginxv1beta1.Service{
			Type:             string(tg.Spec.Type),
			ExtraLabels:      svc.ExtraLabels,
			ExtraAnnotations: svc.ExtraAnnotations,
			Ports:            svc.Ports,
		}
	}
	if replicas := tg.Spec.Replicas; replicas != nil {
		ic.Spec.Replicas = replicas
	}
	if image := tg.Spec.Image; image != nil {
		ic.Spec.Image = nginxv1beta1.Image{
			Repository: image.Repository,
			Tag:        image.Tag,
			PullPolicy: image.PullPolicy,
		}
	}
	if workload := tg.Spec.Workload; workload != nil {
		ic.Spec.Workload = &nginxv1beta1.Workload{
			Resources:   workload.Resources,
			ExtraLabels: workload.ExtraLabels,
		}
	}
	if configMapData := tg.Spec.ConfigMapData; configMapData != nil {
		if ic.Spec.ConfigMapData == nil {
			ic.Spec.ConfigMapData = map[string]string{}
		}
		for k, v := range configMapData {
			ic.Spec.ConfigMapData[k] = v
		}
	}
	return ctrl.SetControllerReference(tg, ic, r.Scheme)
}
