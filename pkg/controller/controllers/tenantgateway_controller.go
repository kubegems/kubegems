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
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	nginx_v1alpha1 "github.com/nginxinc/nginx-ingress-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	gemlabels "kubegems.io/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/controller/handler"
	"kubegems.io/pkg/controller/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type TenantGatewayOptions struct {
	NginxImageRepo   string
	NginxImageTag    string
	NginxMetricsPort uint16
}

func DefaultTenantGatewayOptions() TenantGatewayOptions {
	return TenantGatewayOptions{
		NginxImageRepo:   "kubegems/nginx-ingress",
		NginxImageTag:    "1.11.1",
		NginxMetricsPort: 9113,
	}
}

// TenantGatewayReconciler reconciles a TenantGateway object
type TenantGatewayReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Opts     TenantGatewayOptions
}

//+kubebuilder:rbac:groups=gems.kubegems.io,resources=tenantgateways,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=gems.kubegems.io,resources=tenantgateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.nginx.org,resources=nginxingresscontrollers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=extensions,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *TenantGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		租户网关逻辑:
		1. 检查是否删除操作
		2. 删除需要删除本身及对应ingress与ingressClass资源
		3. 创建与更新需要操作对应nginxIngressController资源
		4. 无论什么操作都需要为tenanrGateway检查并添加finalizer字段
	*/
	log := r.Log.WithValues("tenantgateway", req.NamespacedName)

	var tg gemsv1beta1.TenantGateway
	if err := r.Get(ctx, req.NamespacedName, &tg); err != nil {
		log.WithName(req.Name).Error(err, "Faild to get TenantGateway")
		return ctrl.Result{}, nil
	}

	if !tg.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&tg, gemlabels.FinalizerGateway) {
			if err := r.Delete(ctx, &networkingv1beta1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: tg.Spec.IngressClass,
				},
			}); err != nil {
				log.Info("Failed to delete ingressClass")
			} else {
				log.Info("Success to delete", "ingressClass", tg.Spec.IngressClass)
			}

			// 删除tenantGateway
			controllerutil.RemoveFinalizer(&tg, gemlabels.FinalizerGateway)
			// TODO 这里通过更新的方式删除tg，会再次排队，并报错找不到tg，问题不大
			if err := r.Update(ctx, &tg); err != nil {
				log.Error(err, "failed to delete tenantGateway")
				return ctrl.Result{Requeue: true}, err
			}
			log.Info("success to delete tenantGateway")
			return ctrl.Result{}, nil
		}
	}

	found := &nginx_v1alpha1.NginxIngressController{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: gemlabels.NamespaceGateway, // gateway资源都在这里
		Name:      tg.Name,
	}, found); err != nil {
		if apierrors.IsNotFound(err) {
			// 没有nic，执行create
			nic := r.nginxIngressControllerForTenantGateway(&tg)
			if err := r.Create(ctx, nic); err != nil {
				r.Recorder.Eventf(&tg, corev1.EventTypeWarning, utils.ReasonFailedCreate, "Failed to create NginxIngressController %s: %v", nic.Name, err)
				log.Info("Error create NginxIngressController")
			}
			r.Recorder.Eventf(&tg, corev1.EventTypeNormal, utils.ReasonCreated, "Successfully create NginxIngressController %s", nic.Name)
		} else {
			r.Recorder.Eventf(found, corev1.EventTypeWarning, utils.ReasonUnknowError, "Failed to get NginxIngressController %s: %v", found.Name, err)
			log.Error(err, "Error get NginxIngressController")
		}
	} else {
		// 找到该gateway，执行更新
		if r.hasNginxIngressControllerChanged(found, &tg) {
			updated := r.updateNginxIngressController(found, &tg)
			if err := r.Update(ctx, updated); err != nil {
				r.Recorder.Eventf(&tg, corev1.EventTypeWarning, utils.ReasonFailedCreate, "Failed to update NginxIngressController %s: %v", found.Name, err)
				log.Error(err, "Error update NginxIngressController")
			} else {
				r.Recorder.Eventf(&tg, corev1.EventTypeNormal, utils.ReasonCreated, "Successfully update NginxIngressController %s", found.Name)
			}
		}
	}

	// 检查并加上finalizer
	if !controllerutil.ContainsFinalizer(&tg, gemlabels.FinalizerGateway) {
		controllerutil.AddFinalizer(&tg, gemlabels.FinalizerGateway)
		if err := r.Update(ctx, &tg); err != nil {
			log.Error(err, "failed to update tenantGateway")
			return ctrl.Result{}, nil
		}
	}

	// 最后处理status
	svc := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: gemlabels.NamespaceGateway, // gateway资源都在这里
		Name:      tg.Name,
	}, svc); err != nil {
		return ctrl.Result{}, nil
	}
	dep := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: gemlabels.NamespaceGateway, // gateway资源都在这里
		Name:      tg.Name,
	}, dep); err != nil {
		return ctrl.Result{}, nil
	}

	if !equality.Semantic.DeepEqual(svc.Spec.Ports, tg.Status.Ports) ||
		dep.Status.AvailableReplicas != tg.Status.AvailableReplicas {
		tg.Status.Ports = svc.Spec.Ports
		tg.Status.AvailableReplicas = dep.Status.AvailableReplicas
		if err := r.Status().Update(ctx, &tg); err != nil {
			log.Error(err, "failed to update tenantGateway")
			return ctrl.Result{}, nil
		}
		log.Info("success to update", "gateway status", tg.Status)
	}
	return ctrl.Result{}, nil
}

func (r *TenantGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&source.Kind{Type: &corev1.Service{}}, handler.NewServiceHandler(r.Client, r.Log)).
		Watches(&source.Kind{Type: &appsv1.Deployment{}}, handler.NewDepoymentHandler(r.Client, r.Log)).
		For(&gemsv1beta1.TenantGateway{}).
		Complete(r)
}

func (r *TenantGatewayReconciler) hasNginxIngressControllerChanged(nic *nginx_v1alpha1.NginxIngressController, tg *gemsv1beta1.TenantGateway) bool {
	// label
	if nic.Labels[gemlabels.LabelTenant] != tg.Spec.Tenant {
		return true
	}

	// OwnerReferences
	if len(nic.OwnerReferences) == 0 || nic.OwnerReferences[0].Name != tg.Name {
		return true
	}

	if strings.ToLower(nic.Spec.Type) != "deployment" {
		return true
	}

	if nic.Spec.ServiceType != string(tg.Spec.Type) {
		return true
	}

	if nic.Spec.Replicas != nil && tg.Spec.Replicas != nil && *nic.Spec.Replicas != *tg.Spec.Replicas {
		return true
	}

	if nic.Spec.IngressClass != tg.Spec.IngressClass {
		return true
	}

	// service
	if nic.Spec.Service == nil {
		nic.Spec.Service = &nginx_v1alpha1.Service{}
	}
	if tg.Spec.Service == nil {
		tg.Spec.Service = &gemsv1beta1.Service{}
	}
	if !reflect.DeepEqual(nic.Spec.Service.ExtraLabels, tg.Spec.Service.ExtraLabels) {
		return true
	}

	// image
	if nic.Spec.Image.Repository != r.Opts.NginxImageRepo || nic.Spec.Image.Tag != r.Opts.NginxImageTag {
		return true
	}

	// workload
	if nic.Spec.Workload == nil {
		nic.Spec.Workload = &nginx_v1alpha1.Workload{}
	}
	if tg.Spec.Workload == nil {
		tg.Spec.Workload = &gemsv1beta1.Workload{}
	}
	if utils.HasDifferentResources(nic.Spec.Workload.Resources, tg.Spec.Workload.Resources) {
		return true
	}
	if !reflect.DeepEqual(nic.Spec.Workload.ExtraLabels, tg.Spec.Workload.ExtraLabels) || !reflect.DeepEqual(nic.Spec.ConfigMapData, tg.Spec.ConfigMapData) {
		return true
	}

	return false
}

func (r *TenantGatewayReconciler) nginxIngressControllerForTenantGateway(tg *gemsv1beta1.TenantGateway) *nginx_v1alpha1.NginxIngressController {
	return &nginx_v1alpha1.NginxIngressController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tg.Name,
			Namespace: gemlabels.NamespaceGateway,
			Labels: map[string]string{
				gemlabels.LabelTenant:      tg.Spec.Tenant,
				gemlabels.LabelApplication: tg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(tg, gemsv1beta1.SchemeTenantGateway)},
		},
		Spec: nginx_v1alpha1.NginxIngressControllerSpec{
			Type:         "deployment",
			ServiceType:  string(tg.Spec.Type),
			Replicas:     tg.Spec.Replicas,
			IngressClass: tg.Spec.IngressClass,
			Service:      (*nginx_v1alpha1.Service)(tg.Spec.Service),
			Image: nginx_v1alpha1.Image{
				Repository: r.Opts.NginxImageRepo,
				Tag:        r.Opts.NginxImageTag,
				PullPolicy: string(corev1.PullIfNotPresent),
			},
			Workload:      (*nginx_v1alpha1.Workload)(tg.Spec.Workload),
			ConfigMapData: tg.Spec.ConfigMapData,
			Prometheus: &nginx_v1alpha1.Prometheus{
				Enable: true,
				Port:   &r.Opts.NginxMetricsPort,
			},
		},
	}
}

func (r *TenantGatewayReconciler) updateNginxIngressController(nic *nginx_v1alpha1.NginxIngressController, tg *gemsv1beta1.TenantGateway) *nginx_v1alpha1.NginxIngressController {
	nic.SetLabels(map[string]string{
		gemlabels.LabelTenant:      tg.Spec.Tenant,
		gemlabels.LabelApplication: tg.Name,
	})
	nic.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(tg, gemsv1beta1.SchemeTenantGateway)})
	nic.Spec.Type = "deployment"
	nic.Spec.ServiceType = string(tg.Spec.Type)
	nic.Spec.Replicas = tg.Spec.Replicas
	nic.Spec.IngressClass = tg.Spec.IngressClass
	nic.Spec.Service = (*nginx_v1alpha1.Service)(tg.Spec.Service)
	nic.Spec.Image = nginx_v1alpha1.Image{
		Repository: r.Opts.NginxImageRepo,
		Tag:        r.Opts.NginxImageTag,
		PullPolicy: string(corev1.PullIfNotPresent),
	}
	nic.Spec.Workload = (*nginx_v1alpha1.Workload)(tg.Spec.Workload)
	nic.Spec.ConfigMapData = tg.Spec.ConfigMapData
	nic.Spec.Prometheus = &nginx_v1alpha1.Prometheus{
		Enable: true,
		Port:   &r.Opts.NginxMetricsPort,
	}
	return nic
}
