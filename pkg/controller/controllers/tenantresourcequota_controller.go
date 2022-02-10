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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"

	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/controller/handler"
	"kubegems.io/pkg/controller/utils"
	gemlabels "kubegems.io/pkg/labels"
)

// TenantResourceQuotaReconciler reconciles a TenantResourceQuota object
type TenantResourceQuotaReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=kubegems.io,resources=tenantresourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubegems.io,resources=tenantresourcequotas/status,verbs=get;update;patch

func (r *TenantResourceQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		调度逻辑:
		计算总资源，筛选所有关联的ResourceQuota，将资源加起来就是使用的和申请的
	*/
	log := r.Log.WithValues("TenantResourceQuota", req.NamespacedName)

	var rq gemsv1beta1.TenantResourceQuota
	if err := r.Get(ctx, req.NamespacedName, &rq); err != nil {
		log.Info("Faild to get TenantResourceQuota")
		return ctrl.Result{}, nil
	}

	var nrqList corev1.ResourceQuotaList
	sel := labels.SelectorFromSet(map[string]string{
		gemlabels.LabelTenant: rq.Name,
	})
	if err := r.List(ctx, &nrqList, &client.ListOptions{
		LabelSelector: sel,
		Namespace:     metav1.NamespaceAll,
	}); err != nil {
		log.Info("Faild to list ResourceQuota")
		return ctrl.Result{}, nil
	}
	oldused := rq.Status.Used.DeepCopy()
	oldAllocated := rq.Status.Allocated.DeepCopy()
	used := utils.EmptyTenantResourceQuota()
	allocated := utils.EmptyTenantResourceQuota()
	for _, nrq := range nrqList.Items {
		for _, resource := range utils.TenantLimitResources {
			tmp, exist := nrq.Status.Used[resource]
			if exist {
				u := used[resource]
				u.Add(tmp)
				used[resource] = u
			}

			tmpu, uexist := nrq.Status.Hard[resource]
			if uexist {
				al := allocated[resource]
				al.Add(tmpu)
				allocated[resource] = al
			}
		}
	}
	rq.Status.Used = used
	rq.Status.Allocated = allocated
	if !equality.Semantic.DeepEqual(oldused, used) || !equality.Semantic.DeepEqual(oldAllocated, allocated) {
		rq.Status.LastCountTime = metav1.Now()
		rq.Status.LastUpdateTime = metav1.Now()
		if err := r.Status().Update(ctx, &rq); err != nil {
			log.Info("Failed to update TenantResouceQuota, reque now; err: " + err.Error())
			return ctrl.Result{}, nil
		}
	}
	return ctrl.Result{}, nil
}

func (r *TenantResourceQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gemsv1beta1.TenantResourceQuota{}).
		Watches(&source.Kind{Type: &corev1.ResourceQuota{}}, handler.NewResourceQuotaHandler(r.Client, r.Log)).
		Complete(r)
}
