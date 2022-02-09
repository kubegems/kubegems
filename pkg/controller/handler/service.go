package handler

import (
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

/*
	Service 变更的时候，对应的网关同步status
*/

var _serviceHandler *ServiceHandler

type ServiceHandler struct {
	Client client.Client
	Log    logr.Logger
}

func (h *ServiceHandler) Create(e event.CreateEvent, r workqueue.RateLimitingInterface) {
	svc, ok := e.Object.(*v1.Service)
	if !ok {
		return
	}

	h.requeueTenantGateway(svc.OwnerReferences, r)
}

func (h *ServiceHandler) Update(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
	newSvc, okn := e.ObjectNew.(*v1.Service)
	oldSvc, oko := e.ObjectOld.(*v1.Service)
	if !okn || !oko {
		return
	}
	if !equality.Semantic.DeepEqual(newSvc.Spec.Ports, oldSvc.Spec.Ports) {
		h.requeueTenantGateway(oldSvc.OwnerReferences, r)
	}
}

func (h *ServiceHandler) Delete(e event.DeleteEvent, r workqueue.RateLimitingInterface) {
}

func (h *ServiceHandler) Generic(e event.GenericEvent, r workqueue.RateLimitingInterface) {
}

func newServiceHandler(c *client.Client, log *logr.Logger) *ServiceHandler {
	if _serviceHandler != nil {
		return _serviceHandler
	}
	_serviceHandler = &ServiceHandler{
		Client: *c,
		Log:    *log,
	}
	return _serviceHandler
}

func NewServiceHandler(c client.Client, log logr.Logger) *ServiceHandler {
	return newServiceHandler(&c, &log)
}
func (h *ServiceHandler) requeueTenantGateway(owners []metav1.OwnerReference, r workqueue.RateLimitingInterface) {
	if len(owners) == 0 || owners[0].Kind != "NginxIngressController" {
		return
	}

	r.Add(ctrl.Request{
		// gateway 与 NginxIngressController同名
		NamespacedName: types.NamespacedName{
			Name: owners[0].Name,
		},
	})
}
