package controllers

import (
	"context"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	istionetworkingv1beta1 "istio.io/api/networking/v1beta1"
	istioclientworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"istio.io/istio/pkg/config/kube"
	"istio.io/istio/pkg/config/protocol"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"kubegems.io/pkg/apis/networking"
	"kubegems.io/pkg/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ServiceentryReconciler 用于为对开启 虚拟域名的 namespace 中service创建与虚拟域名相同的 serviceentry
// 功能：
// 1. 观察 namespace 是否具有虚拟空间标志 annotation "kubegems.io/virtualdomain={virtualdomain name}"
// 2. 若有，则为该namespace下的service创建一个virtual service，并设置其hosts 为 {servicename}.{virtualservicename}
// 处理流程：
// 1. 若 service 变化，则判断该 namespace 是否具有 annotation "kubegems.io/virtualdomain={virtualdomain name}"
// 2. 判断 service 是否具有annotation "kubegems.io/virtualdomain={virtualdomain name}"
// 3. 确定service同名的 serviceentry 是否存在并 uptodate
//
type ServiceentryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.istio.io,resources=serviceentries,verbs=*
//+kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=*

// virtualservices.networking.istio.io
func (r *ServiceentryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 若未安装istio则不执行操作
	if !PluginStatusInstance.ComponentEnabled(ComponentIstio) {
		return ctrl.Result{}, nil
	}

	if strings.HasPrefix(req.Namespace, "kube") {
		return ctrl.Result{}, nil
	}

	svc := &corev1.Service{}
	if err := r.Client.Get(ctx, req.NamespacedName, svc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if svc.DeletionTimestamp != nil {
		return r.OnRemove(ctx, svc)
	} else {
		return r.OnChange(ctx, svc)
	}
}

func (r *ServiceentryReconciler) OnRemove(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	// 删除同名 serviceentry
	se := &istioclientworkingv1beta1.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
		},
	}
	if err := r.Client.Delete(ctx, se); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// 即使出错也不重试
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ServiceentryReconciler) OnChange(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	//  skip no service ip sevice
	if svc.Spec.ClusterIP == "" || svc.Spec.ClusterIP == corev1.ClusterIPNone {
		return ctrl.Result{}, nil
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: svc.Namespace}}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(ns), ns); err != nil {
		return ctrl.Result{}, nil
	}

	if !isManagerdNamespace(ns) {
		return ctrl.Result{}, nil
	}

	hosts := set.NewSet[string]()

	addtohosts := func(val string) {
		for _, domain := range strings.Split(val, ",") {
			if domain == "" {
				continue
			}
			hosts.Append(svc.Name + "." + domain)
		}
	}
	// namespace 指定 domain
	if val, ok := ns.Annotations[networking.AnnotationVirtualDomain]; ok {
		addtohosts(val)
	}
	// service 指定 domain
	if val, ok := svc.Annotations[networking.AnnotationVirtualDomain]; ok {
		addtohosts(val)
	}

	// 如果有同名的 virtualservice 也为vs增加上域名
	if err := r.updateVirtualServiceHosts(ctx, svc, hosts.Slice()); err != nil {
		r.Log.Error(err, "update virtual service hosts failed")
	}

	// skip if no domain specified
	if hosts.Len() == 0 {
		// 有两种情况：
		// 1. namespace 上增加了注解又删除了
		// 2. svc 上增加了注解又删除了
		return r.OnRemove(ctx, svc)
	}

	// serviceentry
	se := &istioclientworkingv1beta1.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
		},
		Spec: istionetworkingv1beta1.ServiceEntry{
			Hosts:     hosts.Slice(),
			Addresses: []string{svc.Spec.ClusterIP},
			Ports: func(svc *corev1.Service) []*istionetworkingv1beta1.Port {
				ports := make([]*istionetworkingv1beta1.Port, 0, len(svc.Spec.Ports))
				for _, port := range svc.Spec.Ports {
					proto := kube.ConvertProtocol(port.Port, port.Name, port.Protocol, port.AppProtocol)
					if proto.IsUnsupported() {
						proto = protocol.TCP
					}
					ports = append(ports, &istionetworkingv1beta1.Port{
						Name:     port.Name,
						Number:   uint32(port.Port),
						Protocol: string(proto),
					})
				}
				return ports
			}(svc),
			// https://istio.io/latest/docs/reference/config/networking/service-entry/#ServiceEntry
			Endpoints:  []*istionetworkingv1beta1.WorkloadEntry{{Address: svc.Name + "." + svc.Namespace}},
			Location:   istionetworkingv1beta1.ServiceEntry_MESH_INTERNAL,
			Resolution: istionetworkingv1beta1.ServiceEntry_DNS,
		},
	}

	// ensure
	exist := &istioclientworkingv1beta1.ServiceEntry{}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(se), exist); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Client.Create(ctx, se); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// update
	if reflect.DeepEqual(exist.Spec, se.Spec) {
		return ctrl.Result{}, nil
	}
	exist.Spec = se.Spec
	if err := r.Client.Update(ctx, exist); err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	return ctrl.Result{}, nil
}

func (r *ServiceentryReconciler) updateVirtualServiceHosts(ctx context.Context, svc *corev1.Service, hosts []string) error {
	vs := &istioclientworkingv1beta1.VirtualService{}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(svc), vs); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	sets := set.NewSet[string]()
	sets.Append(svc.Name)
	// sets.Append(vs.Spec.Hosts...)
	sets.Append(hosts...)

	desiredhosts := sets.Slice()
	if reflect.DeepEqual(vs.Spec.Hosts, desiredhosts) {
		return nil
	}
	r.Log.Info("update virtualservice hosts", "virtualservice", vs, "hosts", desiredhosts)

	vs.Spec.Hosts = desiredhosts
	return r.Client.Update(ctx, vs)
}

func OnNamespceChangeFunc(cli client.Client) handler.MapFunc {
	return func(obj client.Object) []reconcile.Request {
		switch data := obj.(type) {
		case *corev1.Namespace:
			ctx := context.Background()

			svcs := &corev1.ServiceList{}
			if err := cli.List(ctx, svcs, client.InNamespace(data.Name)); err != nil {
				return []reconcile.Request{}
			}
			reqs := make([]reconcile.Request, len(svcs.Items))
			for i, svc := range svcs.Items {
				reqs[i] = reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&svc)}
			}
			return reqs
		default:
			return []reconcile.Request{}
		}
	}
}

func (r *ServiceentryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		// 当ns发生变化时，enqueue 所有该空间下的service
		Watches(&source.Kind{Type: &corev1.Namespace{}}, handler.EnqueueRequestsFromMapFunc(OnNamespceChangeFunc(mgr.GetClient()))).
		Complete(r)
}
