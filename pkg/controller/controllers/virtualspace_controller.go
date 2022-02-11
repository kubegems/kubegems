package controllers

import (
	"context"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	istionetworkingv1beta1 "istio.io/api/networking/v1beta1"
	istioclinetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"kubegems.io/pkg/apis/networking"
	"kubegems.io/pkg/controller/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VirtuslspaceReconciler 用于解析 ns 上的annotation以在不同的空间中创建sidecar
type VirtuslspaceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.istio.io,resources=sidecars,verbs=*

// virtualservices.networking.istio.io
func (r *VirtuslspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 若未安装istio则不执行操作
	if !PluginStatusInstance.ComponentEnabled(ComponentIstio) {
		return ctrl.Result{}, nil
	}

	ns := &corev1.Namespace{}
	if err := r.Client.Get(ctx, req.NamespacedName, ns); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if ns.DeletionTimestamp != nil {
		// ignore
		return ctrl.Result{}, nil
	} else {
		return r.OnChange(ctx, ns)
	}
}

func (r *VirtuslspaceReconciler) OnChange(ctx context.Context, ns *corev1.Namespace) (ctrl.Result, error) {
	namespaces := &corev1.NamespaceList{}
	if err := r.Client.List(ctx, namespaces); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	errlist := []error{}

	// 统计空间是否为有虚拟空间标记
	isvirtualspacedns := map[string]bool{}

	// 统计虚拟空间下有哪些 ns
	virtualspacenamespaces := map[string][]string{}
	for _, ns := range namespaces.Items {
		if val, ok := ns.Annotations[networking.AnnotationVirtualSpace]; ok {
			isvirtualspacedns[ns.Name] = true
			for _, vs := range strings.Split(val, ",") {
				if vs == "" {
					continue
				}
				virtualspacenamespaces[vs] = append(virtualspacenamespaces[vs], ns.Name)
			}
		} else {
			isvirtualspacedns[ns.Name] = false
		}
	}

	sisternss := func(namespace string) []string {
		hosts := utils.NewSet()
		for _, nss := range virtualspacenamespaces {
			for _, ns := range nss {
				if ns == namespace {
					// 该 vs 下所有 ns 都是
					for _, ns := range nss {
						hosts.Append(ns)
					}
				}
			}
		}
		return hosts.Slice()
	}

	for _, ns := range namespaces.Items {
		// 如果一个空间在多个 vs 下
		// 找到所有 vs
		sisters := sisternss(ns.Name)

		if !isManagerdNamespace(&ns) {
			continue
		}

		if err := r.ensureInjectLabel(ctx, ns.Name, len(sisters) != 0); err != nil {
			errlist = append(errlist, err)
			ctrl.Log.Error(err, "faield update namespace inject label", "namespace", ns.Name)
		}

		if err := r.ensuresidecar(ctx, ns.Name, sisters); err != nil {
			errlist = append(errlist, err)
			ctrl.Log.Error(err, "faield update default sidecar", "namespace", ns.Name)
		}
	}

	if len(errlist) != 0 {
		return ctrl.Result{Requeue: true}, utilerrors.NewAggregate(errlist)
	}
	return ctrl.Result{}, nil
}

// 入参： 当前空间，以及与当前空间处于同一个virtualspace下的空间(包含自身)
func (r *VirtuslspaceReconciler) ensuresidecar(ctx context.Context, namespace string, namespaces []string) error {
	sidecar := &istioclinetworkingv1beta1.Sidecar{
		ObjectMeta: v1.ObjectMeta{
			Name:      "default",
			Namespace: namespace,
		},
		Spec: istionetworkingv1beta1.Sidecar{
			Egress: func() []*istionetworkingv1beta1.IstioEgressListener {
				hosts := []string{"./*"}
				for _, ns := range namespaces {
					if ns == namespace {
						continue
					}
					hosts = append(hosts, ns+"/*")
				}
				return []*istionetworkingv1beta1.IstioEgressListener{{Hosts: hosts}}
			}(),
		},
	}

	// 如果为空则可删除
	if len(namespaces) == 0 {
		if err := r.Client.Delete(ctx, sidecar); err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		r.Log.Info("removed sidecar for namespace", "namespace", namespace)
		return nil
	}

	exist := &istioclinetworkingv1beta1.Sidecar{}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(sidecar), exist); err != nil {
		if errors.IsNotFound(err) {
			// create a new
			r.Log.Info("create sidecar for namespace", "namespace", namespace)
			return r.Client.Create(ctx, sidecar)
		}
		return err
	}

	if reflect.DeepEqual(exist.Spec.Egress, sidecar.Spec.Egress) {
		return nil
	}
	// update
	exist.Spec.Egress = sidecar.Spec.Egress
	r.Log.Info("update sidecar for namespace", "namespace", namespace)
	return r.Client.Update(ctx, exist)
}

const istioInjectionLabel = "istio-injection"

func (r *VirtuslspaceReconciler) ensureInjectLabel(ctx context.Context, namespace string, inject bool) error {
	ns := &corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: namespace}}

	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(ns), ns); err != nil {
		return err
	}

	// https://istio.io/latest/docs/ops/common-problems/injection/#the-result-of-sidecar-injection-was-not-what-i-expected
	// istio namespace selector using  enabled/disabled  for on/off
	val := ns.Labels[istioInjectionLabel]
	if !inject {
		if val == "enabled" || val == "true" {
			ns.Labels[istioInjectionLabel] = "disabled"
			r.Log.Info("uninject istio for namespace", "namespace", ns.Name)
			return r.Client.Update(ctx, ns)
		}
	} else {
		if val != "enabled" && val != "true" {
			ns.Labels[istioInjectionLabel] = "enabled"
			r.Log.Info("inject istio for namespace", "namespace", ns.Name)
			return r.Client.Update(ctx, ns)
		}
	}
	return nil
}

// 被管理的空间上有label
func isManagerdNamespace(ns *corev1.Namespace) bool {
	for k := range ns.Annotations {
		// 仅存在 AnnotationVirtualSpace annotation的 ns 才被"管理”
		if k == networking.AnnotationVirtualSpace {
			return true
		}
	}
	return false
}

func (r *VirtuslspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&corev1.Namespace{}).Complete(r)
}
