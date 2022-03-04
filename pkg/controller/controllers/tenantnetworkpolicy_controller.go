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
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	gemlabels "kubegems.io/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/controller/utils"
)

const (
	isoKindTenant      = "tenant"
	isoKindProject     = "project"
	isoKindEnvironment = "environment"
)

type NetworkPolicyAction struct {
	TenantISO      bool
	ProjectISO     bool
	EnvironmentISO bool

	Tenant      string
	Project     string
	Environment string

	Origin *netv1.NetworkPolicy
	Modify *netv1.NetworkPolicy

	Labels map[string]string

	action string
}

// TenantNetworkPolicyReconciler reconciles a TenantNetworkPolicy object
type TenantNetworkPolicyReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=gems.kubegems.io,resources=tenantnetworkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems.kubegems.io,resources=tenantnetworkpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

func (r *TenantNetworkPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		租户网关逻辑:
		1. 根据情况创建或者修改
	*/
	log := r.Log.WithValues("tenantnetworkpolicy", req.NamespacedName)

	var netpol gemsv1beta1.TenantNetworkPolicy
	if err := r.Get(ctx, req.NamespacedName, &netpol); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "failed to get tenantnetworkpolicy")
		}
	}

	if !netpol.ObjectMeta.DeletionTimestamp.IsZero() {
		tenantname, exist := netpol.Labels[gemlabels.LabelTenant]
		if !exist {
			log.Error(
				fmt.Errorf("failed to delete tenantnetworkpolicy [%s] related networkpolicies, tenant label not exist", netpol.Name),
				"",
			)
			return ctrl.Result{}, nil
		}

		nplist := netv1.NetworkPolicyList{}
		r.List(ctx, &nplist, &client.ListOptions{
			Namespace: corev1.NamespaceAll,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				gemlabels.LabelTenant: tenantname,
			}),
		})
		for _, np := range nplist.Items {
			r.Delete(ctx, &np)
		}
		netpol.SetOwnerReferences(nil)
		netpol.SetFinalizers(nil)
		r.Update(ctx, &netpol)
		return ctrl.Result{}, nil
	}

	// 这儿需要先把所有关联NS下的np拿出来，根据各级别的开关情况，修改后保存到MAP中
	// 最后统一判断差异后进行patch or create
	statusMap := map[string]NetworkPolicyAction{}

	r.handleStatusMap(ctx, statusMap, &netpol)

	for _, action := range statusMap {
		switch action.action {
		case "create":
			action.Modify.Labels = labels.Merge(action.Modify.Labels, action.Labels)
			if err := r.Create(ctx, action.Modify); err != nil {
				log.Info("Error create networkpolicy " + err.Error())
			}
		case "delete":
			if err := r.Delete(ctx, action.Modify); err != nil {
				log.Info("Error delete networkpolicy " + err.Error())
			}
		case "update":
			action.Modify.Labels = labels.Merge(action.Modify.Labels, action.Labels)
			if err := r.Update(ctx, action.Modify); err != nil {
				log.Info("Error update networkpolicy " + err.Error())
			}
		default:
			continue
		}
	}

	if !controllerutil.ContainsFinalizer(&netpol, gemlabels.FinalizerNetworkPolicy) {
		controllerutil.AddFinalizer(&netpol, gemlabels.FinalizerNetworkPolicy)
		r.Update(ctx, &netpol)
	}
	return ctrl.Result{}, nil
}

func (r *TenantNetworkPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gemsv1beta1.TenantNetworkPolicy{}).
		Complete(r)
}

func (r *TenantNetworkPolicyReconciler) handleStatusMap(ctx context.Context, st map[string]NetworkPolicyAction, netpol *gemsv1beta1.TenantNetworkPolicy) {
	cidrs, err := utils.GetCIDRs(r.Client)
	if err != nil {
		panic(err)
	}
	// 如果开启租户隔离，那么租户关联的所有环境对应的namespace下，都将存在np,且允许带租户label的ns访问
	labelSel := labels.SelectorFromSet(map[string]string{gemlabels.LabelTenant: netpol.Spec.Tenant})
	r.handleRelatedObjectList(ctx, labelSel, st, isoKindTenant, netpol.Spec.TenantIsolated)

	// 如果开启项目隔离，那么项目关联的所有namespace下，都将存在np,且允许带项目label的ns访问
	for _, proj := range netpol.Spec.ProjectNetworkPolicies {
		labelSel := labels.SelectorFromSet(map[string]string{gemlabels.LabelProject: proj.Name})
		r.handleRelatedObjectList(ctx, labelSel, st, isoKindProject, true)
	}

	// 如果开启环境隔离，那么项目关联的所有namespace下，都将存在np,且允许带环境label的ns访问
	for _, env := range netpol.Spec.EnvironmentNetworkPolicies {
		labelSel := labels.SelectorFromSet(map[string]string{gemlabels.LabelEnvironment: env.Name})
		r.handleRelatedObjectList(ctx, labelSel, st, isoKindEnvironment, true)
	}

	for ns, action := range st {
		defaultnetpol := utils.DefaultNetworkPolicy(ns, "default", cidrs)
		action.Modify = &defaultnetpol
		if action.TenantISO {
			utils.AddNamespaceSelector(action.Modify, gemlabels.LabelTenant, action.Tenant)
		}
		if action.ProjectISO {
			utils.AddNamespaceSelector(action.Modify, gemlabels.LabelProject, action.Project)
		}
		if action.EnvironmentISO {
			utils.AddNamespaceSelector(action.Modify, gemlabels.LabelEnvironment, action.Environment)
		}
		if action.Origin == nil {
			if !action.EnvironmentISO && !action.ProjectISO && !action.TenantISO {
				action.action = ""
			} else {
				action.action = "create"
			}
		} else {
			if !action.EnvironmentISO && !action.ProjectISO && !action.TenantISO {
				action.action = "delete"
			} else if !equality.Semantic.DeepDerivative(action.Origin, action.Modify) {
				action.action = "update"
			} else {
				action.action = ""
			}
		}
		st[ns] = action
	}
}

func (r *TenantNetworkPolicyReconciler) handleRelatedObjectList(ctx context.Context, sel labels.Selector, st map[string]NetworkPolicyAction, kind string, isolated bool) {
	nslist := &corev1.NamespaceList{}
	nplist := &netv1.NetworkPolicyList{}
	r.List(ctx, nslist, &client.ListOptions{
		LabelSelector: sel,
		Namespace:     corev1.NamespaceAll,
	})
	r.List(ctx, nplist, &client.ListOptions{
		LabelSelector: sel,
		Namespace:     corev1.NamespaceAll,
	})
	npmap := map[string]netv1.NetworkPolicy{}
	for _, np := range nplist.Items {
		npmap[np.Namespace] = np
	}
	for _, ns := range nslist.Items {
		var tmpaction NetworkPolicyAction
		if action, exist := st[ns.Name]; exist {
			tmpaction = action
		} else {
			tmpaction = NetworkPolicyAction{}
			tmpnp, exist := npmap[ns.Name]
			if exist {
				tmpaction.Origin = &tmpnp
			}
		}
		tmpaction.Tenant = ns.Labels[gemlabels.LabelTenant]
		tmpaction.Project = ns.Labels[gemlabels.LabelProject]
		tmpaction.Environment = ns.Labels[gemlabels.LabelEnvironment]
		switch kind {
		case isoKindTenant:
			tmpaction.TenantISO = isolated
		case isoKindProject:
			tmpaction.ProjectISO = isolated
		case isoKindEnvironment:
			tmpaction.EnvironmentISO = isolated
		}
		tmpaction.Labels = utils.GetLabels(ns.Labels, gemlabels.CommonLabels)
		st[ns.Name] = tmpaction
	}
}
