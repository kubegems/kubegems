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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	gemlabels "kubegems.io/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/utils/statistics"
)

// TenantResourceQuotaReconciler reconciles a TenantResourceQuota object
type TenantResourceQuotaReconciler struct {
	client.Client
}

//+kubebuilder:rbac:groups=gems.kubegems.io,resources=tenantresourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems.kubegems.io,resources=tenantresourcequotas/status,verbs=get;update;patch

func (r *TenantResourceQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		调度逻辑:
		计算总资源，筛选所有关联的ResourceQuota，将资源加起来就是使用的和申请的
	*/
	log := ctrl.LoggerFrom(ctx)

	log.Info("reconciling...")
	defer log.Info("reconcile done")

	var rq gemsv1beta1.TenantResourceQuota
	if err := r.Get(ctx, req.NamespacedName, &rq); err != nil {
		log.Error(err, "get resource quota")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var resourceQuotaList corev1.ResourceQuotaList
	if err := r.List(ctx, &resourceQuotaList,
		client.MatchingLabels{gemlabels.LabelTenant: rq.Name},
		client.InNamespace(metav1.NamespaceAll),
	); err != nil {
		log.Error(err, "list resource quota")
		return ctrl.Result{}, nil
	}

	used, hard := corev1.ResourceList{}, corev1.ResourceList{}
	for _, item := range resourceQuotaList.Items {
		statistics.AddResourceList(used, item.Status.Used)
		statistics.AddResourceList(hard, item.Status.Hard)
	}

	if !equality.Semantic.DeepEqual(rq.Status.Used, used) || !equality.Semantic.DeepEqual(rq.Status.Allocated, hard) {
		log.Info("updateing status")
		rq.Status.LastUpdateTime = metav1.Now()
		rq.Status.Used = used
		rq.Status.Allocated = hard // Hard is the set of enforced hard limits for each named resource.
		rq.Status.Hard = hard      // Hard is the set of enforced hard limits for each named resource.
		if err := r.Status().Update(ctx, &rq); err != nil {
			log.Error(err, "update resource quota status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *TenantResourceQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gemsv1beta1.TenantResourceQuota{}).
		Watches(&source.Kind{Type: &corev1.ResourceQuota{}}, NewResourceQuotaHandler()).
		Complete(r)
}

/*
	监听所有的ResourceQuota事件，当ResourceQuota变更的时候,让对应的TenantResourceQuota重新计算
*/

func NewResourceQuotaHandler() handler.Funcs {
	return handler.Funcs{
		CreateFunc: func(e event.CreateEvent, r workqueue.RateLimitingInterface) {
			requeueTenantResourceQuota(e.Object, r)
		},
		UpdateFunc: func(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
			newRq, okn := e.ObjectNew.(*corev1.ResourceQuota)
			oldRq, oko := e.ObjectOld.(*corev1.ResourceQuota)
			if !okn || !oko {
				return
			}
			// reconcile only resource quota status changed
			if !equality.Semantic.DeepEqual(newRq.Status, oldRq.Status) {
				requeueTenantResourceQuota(e.ObjectNew, r)
			}
		},
		DeleteFunc: func(e event.DeleteEvent, r workqueue.RateLimitingInterface) {
			requeueTenantResourceQuota(e.Object, r)
		},
	}
}

func requeueTenantResourceQuota(obj client.Object, r workqueue.RateLimitingInterface) {
	labels := obj.GetLabels()
	if labels == nil {
		return
	}
	if tenantName := labels[gemlabels.LabelTenant]; tenantName != "" {
		r.Add(ctrl.Request{NamespacedName: types.NamespacedName{Name: tenantName}})
	}
}
