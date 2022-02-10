/*
Copyright 2021 cloudminds.com.

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
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/controller/handler"
	"kubegems.io/pkg/controller/utils"
	gemlabels "kubegems.io/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=kubegems.io,resources=tenants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubegems.io,resources=tenants/status,verbs=get;update;patch

func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		租户逻辑:
		1. 创建或者更新TenantResourceQuota
		2. 创建或者更新TenantNetworkPolicy
		3. 创建或者更新TenantGateway
	*/
	log := r.Log.WithName("Tenant").WithValues("tenant", req.Name)
	var tenant gemsv1beta1.Tenant
	if err := r.Get(ctx, req.NamespacedName, &tenant); err != nil {
		if errors.IsNotFound(err) {
			log.Info("NotFound")
			return ctrl.Result{}, nil
		} else {
			log.Info("Failed to get Tenant")
			return ctrl.Result{}, nil
		}
	}
	oref := metav1.NewControllerRef(&tenant, gemsv1beta1.SchemeTenant)

	// 删除操作，前台删除
	if !tenant.ObjectMeta.DeletionTimestamp.IsZero() {

		// 删除所有得环境
		if controllerutil.ContainsFinalizer(&tenant, gemlabels.FinalizerEnvironment) {
			env := &gemsv1beta1.Environment{}
			if err := r.DeleteAllOf(ctx, env, &client.DeleteAllOfOptions{
				ListOptions: client.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{
						gemlabels.LabelTenant: tenant.Spec.TenantName,
					}),
				},
			}); err != nil {
				log.Error(err, "failed to delete environment")
			}
			controllerutil.RemoveFinalizer(&tenant, gemlabels.FinalizerEnvironment)
		}

		// 删除TenantResourceQuota
		if controllerutil.ContainsFinalizer(&tenant, gemlabels.FinalizerResourceQuota) {
			trq := &gemsv1beta1.TenantResourceQuota{}
			key := types.NamespacedName{
				Name: tenant.Spec.TenantName,
			}
			controllerutil.RemoveFinalizer(&tenant, gemlabels.FinalizerResourceQuota)
			if err := r.Get(ctx, key, trq); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "failed to get tenantresourcequota")
				}
			} else {
				n := metav1.Now()
				trq.SetOwnerReferences(nil)
				trq.SetDeletionTimestamp(&n)
				r.Update(ctx, trq)
			}
		}
		// 删除TenantNetworkPolicy
		if controllerutil.ContainsFinalizer(&tenant, gemlabels.FinalizerNetworkPolicy) {
			tnetpol := &gemsv1beta1.TenantNetworkPolicy{}
			key := types.NamespacedName{
				Name: tenant.Spec.TenantName,
			}
			controllerutil.RemoveFinalizer(&tenant, gemlabels.FinalizerNetworkPolicy)
			if err := r.Get(ctx, key, tnetpol); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "failed to get tenantnetworkpolicy")
				}
			} else {
				n := metav1.Now()
				tnetpol.SetOwnerReferences(nil)
				tnetpol.SetDeletionTimestamp(&n)
				r.Update(ctx, tnetpol)
			}
		}

		// 删除TenantGateway
		if controllerutil.ContainsFinalizer(&tenant, gemlabels.FinalizerGateway) {
			tg := &gemsv1beta1.TenantGateway{}
			if err := r.DeleteAllOf(ctx, tg, &client.DeleteAllOfOptions{
				ListOptions: client.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{
						gemlabels.LabelTenant: tenant.Spec.TenantName,
					}),
				},
			}); err != nil {
				log.Error(err, "failed to delete tenant gateway")
			}
			controllerutil.RemoveFinalizer(&tenant, gemlabels.FinalizerGateway)
		}

		r.Update(ctx, &tenant)
		return ctrl.Result{}, nil
	}

	var changed bool
	// TenantResourceQuota
	r.handleTenantResourceQuota(&tenant, oref, ctx, log)
	if !controllerutil.ContainsFinalizer(&tenant, gemlabels.FinalizerResourceQuota) {
		controllerutil.AddFinalizer(&tenant, gemlabels.FinalizerResourceQuota)
		changed = true
	}

	//  处理NetworkPolicy
	r.handleTenantNetworkPolicy(&tenant, oref, ctx, log)
	if !controllerutil.ContainsFinalizer(&tenant, gemlabels.FinalizerNetworkPolicy) {
		controllerutil.AddFinalizer(&tenant, gemlabels.FinalizerNetworkPolicy)
		changed = true
	}

	// 添加 环境 finalizer
	if !controllerutil.ContainsFinalizer(&tenant, gemlabels.FinalizerEnvironment) {
		controllerutil.AddFinalizer(&tenant, gemlabels.FinalizerEnvironment)
		changed = true
	}

	// 处理Gateway
	if !controllerutil.ContainsFinalizer(&tenant, gemlabels.FinalizerGateway) {
		controllerutil.AddFinalizer(&tenant, gemlabels.FinalizerGateway)
		changed = true
	}

	if changed {
		if err := r.Update(ctx, &tenant); err != nil {
			msg := fmt.Sprintf("Failed to update tenant %s: %v", tenant.Name, err)
			log.Info(msg)
			return ctrl.Result{Requeue: true}, nil
		}
	}
	// 最后处理状态
	if r.handleTenantStatus(&tenant, ctx, log) {
		if err := r.Status().Update(ctx, &tenant); err != nil {
			msg := fmt.Sprintf("Failed to update tenant %s: %v", tenant.Name, err)
			log.Info(msg)
			return ctrl.Result{Requeue: true}, nil
		}
	}
	return ctrl.Result{}, nil
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gemsv1beta1.Tenant{}).
		Watches(&source.Kind{Type: &gemsv1beta1.Environment{}}, handler.NewEnvironmentHandler(r.Client, r.Log)).
		Watches(&source.Kind{Type: &gemsv1beta1.TenantResourceQuota{}}, handler.NewTenantResourceQuotaHandler(r.Client, r.Log)).
		Complete(r)
}

func (r *TenantReconciler) handleTenantStatus(tenant *gemsv1beta1.Tenant, ctx context.Context, log logr.Logger) bool {
	var envs gemsv1beta1.EnvironmentList
	if err := r.List(ctx, &envs, client.MatchingLabels{gemlabels.LabelTenant: tenant.Spec.TenantName}); err != nil {
		r.Log.Error(err, "failed to list environments")
		return false
	}
	var envNames []string
	var namespaces []string
	for _, env := range envs.Items {
		envNames = append(envNames, env.Name)
		namespaces = append(namespaces, env.Spec.Namespace)
	}

	envSame := utils.StringArrayEqual(tenant.Status.Environments, envNames)
	nsSame := utils.StringArrayEqual(tenant.Status.Namespaces, namespaces)
	if envSame && nsSame {
		return false
	}
	tenant.Status.Environments = envNames
	tenant.Status.Namespaces = namespaces
	tenant.Status.LastUpdateTime = metav1.Now()
	return true
}

func (r *TenantReconciler) handleTenantResourceQuota(tenant *gemsv1beta1.Tenant, owner *metav1.OwnerReference, ctx context.Context, log logr.Logger) {
	var trq gemsv1beta1.TenantResourceQuota
	trqKey := types.NamespacedName{
		Name: tenant.Spec.TenantName,
	}

	nlabels := map[string]string{
		gemlabels.LabelTenant: tenant.Spec.TenantName,
	}

	if err := r.Get(ctx, trqKey, &trq); err != nil {
		if !errors.IsNotFound(err) {
			log.Info("Failed to get TenantResourceQuota")
			return
		}
		log.Info("NotFound TenantResouceQuota, create one")
		trq.Name = tenant.Name
		controllerutil.SetControllerReference(tenant, &trq, r.Scheme)
		trq.Labels = labels.Merge(trq.Labels, nlabels)
		trq.Spec.Hard = utils.GetDefaultTeantResourceQuota()
		if err := r.Create(ctx, &trq); err != nil {
			r.Recorder.Eventf(tenant, corev1.EventTypeWarning, utils.ReasonFailedCreateSubResource, "Failed to Create TenantTesourceQuota for tenant %s: %v", tenant.Spec.TenantName, err)
			log.Info("Faield to create TenantResourceQuota: " + err.Error())
			return
		}
		r.Recorder.Eventf(tenant, corev1.EventTypeNormal, utils.ReasonCreated, "Successfully create TenantTesourceQuota for tenant %s", tenant.Spec.TenantName)
	}
	var changed bool
	if !utils.ExistOwnerRef(trq.ObjectMeta, *owner) {
		controllerutil.SetControllerReference(tenant, &trq, r.Scheme)
		changed = true
	}
	if utils.LabelChanged(trq.Labels, nlabels) {
		trq.Labels = labels.Merge(trq.Labels, nlabels)
		changed = true
	}
	if changed {
		if err := r.Update(ctx, &trq); err != nil {
			r.Recorder.Eventf(tenant, corev1.EventTypeWarning, utils.ReasonFailedUpdate, "Failed to update TenantTesourceQuota for tenant %s", tenant.Spec.TenantName)
			log.Info("Faield to update TenantResourceQuota")
		}
		r.Recorder.Eventf(tenant, corev1.EventTypeNormal, utils.ReasonUpdated, "Successfully update TenantTesourceQuota for tenant %s", tenant.Spec.TenantName)
	}
}

func (r *TenantReconciler) handleTenantNetworkPolicy(tenant *gemsv1beta1.Tenant, owner *metav1.OwnerReference, ctx context.Context, log logr.Logger) {
	var tnetpol gemsv1beta1.TenantNetworkPolicy
	tnetpolKey := types.NamespacedName{
		Name: tenant.Name,
	}

	nlabels := map[string]string{
		gemlabels.LabelTenant: tenant.Name,
	}

	if err := r.Get(ctx, tnetpolKey, &tnetpol); err != nil {
		if !errors.IsNotFound(err) {
			log.Info("Failed to get TenantNetworkPolicy")
			return
		}
		log.Info("NotFound TenantNetworkPolicy, create one")
		tnetpol.Name = tenant.Name
		controllerutil.SetControllerReference(tenant, &tnetpol, r.Scheme)
		tnetpol.Labels = labels.Merge(tnetpol.Labels, nlabels)
		tnetpol.Spec.Tenant = tenant.Name
		tnetpol.Spec.TenantIsolated = false
		if err := r.Create(ctx, &tnetpol); err != nil {
			r.Recorder.Eventf(tenant, corev1.EventTypeWarning, utils.ReasonFailedCreateSubResource, "Failed to Create TenantNetworkPolicy for tenant %s: %v", tenant.Spec.TenantName, err)
			log.Info("Faield to create TenantNetworkpolicy: " + err.Error())
			return
		}
		r.Recorder.Eventf(tenant, corev1.EventTypeNormal, utils.ReasonCreated, "Successfully create TenantNetworkPolicy for tenant %s", tenant.Spec.TenantName)
	}
	var changed bool
	if !utils.ExistOwnerRef(tnetpol.ObjectMeta, *owner) {
		controllerutil.SetControllerReference(tenant, &tnetpol, r.Scheme)
		changed = true
	}
	if utils.LabelChanged(tnetpol.Labels, nlabels) {
		tnetpol.Labels = labels.Merge(tnetpol.Labels, nlabels)
		changed = true
	}
	if changed {
		if err := r.Update(ctx, &tnetpol); err != nil {
			r.Recorder.Eventf(tenant, corev1.EventTypeWarning, utils.ReasonFailedUpdate, "Failed to update TenantNetworkPolicy for tenant %s", tenant.Spec.TenantName)
			log.Info("Faield to update TenantNetworkPolicy")
		}
		r.Recorder.Eventf(tenant, corev1.EventTypeNormal, utils.ReasonUpdated, "Successfully update TenantNetworkPloicy for tenant %s", tenant.Spec.TenantName)
	}
}
