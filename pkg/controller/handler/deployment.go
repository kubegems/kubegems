package handler

import (
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
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

var _depoymentHandler *DepoymentHandler

type DepoymentHandler struct {
	Client client.Client
	Log    logr.Logger
}

func (h *DepoymentHandler) Create(e event.CreateEvent, r workqueue.RateLimitingInterface) {
	dep, ok := e.Object.(*appsv1.Deployment)
	if !ok {
		return
	}
	h.requeueTenantGateway(dep.OwnerReferences, r)
}

func (h *DepoymentHandler) Update(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
	newDep, okn := e.ObjectNew.(*appsv1.Deployment)
	oldDep, oko := e.ObjectOld.(*appsv1.Deployment)
	if !okn || !oko {
		return
	}

	if newDep.Status.AvailableReplicas != oldDep.Status.AvailableReplicas {
		h.requeueTenantGateway(oldDep.OwnerReferences, r)
	}
}

func (h *DepoymentHandler) Delete(e event.DeleteEvent, r workqueue.RateLimitingInterface) {
}

func (h *DepoymentHandler) Generic(e event.GenericEvent, r workqueue.RateLimitingInterface) {
}

func newDepoymentHandler(c *client.Client, log *logr.Logger) *DepoymentHandler {
	if _depoymentHandler != nil {
		return _depoymentHandler
	}
	_depoymentHandler = &DepoymentHandler{
		Client: *c,
		Log:    *log,
	}
	return _depoymentHandler
}

func NewDepoymentHandler(c client.Client, log logr.Logger) *DepoymentHandler {
	return newDepoymentHandler(&c, &log)
}

func (h *DepoymentHandler) requeueTenantGateway(owners []metav1.OwnerReference, r workqueue.RateLimitingInterface) {
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
