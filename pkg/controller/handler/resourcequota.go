package handler

import (
	"github.com/go-logr/logr"
	gemlabels"github.com/kubegems/gems/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

/*
	监听所有的ResourceQuota事件，当ResourceQuota变更的时候,让对应的TenantResourceQuota重新计算
*/

var _resourceQuotaHandler *ResourceQuotaHandler

type ResourceQuotaHandler struct {
	Client client.Client
	Log    logr.Logger
}

func (h *ResourceQuotaHandler) Create(e event.CreateEvent, r workqueue.RateLimitingInterface) {
	rq, ok := e.Object.(*corev1.ResourceQuota)
	if !ok {
		return
	}
	h.requeueTenantResourceQuota(rq.Labels, r)
}

func (h *ResourceQuotaHandler) Update(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
	newRq, okn := e.ObjectNew.(*corev1.ResourceQuota)
	oldRq, oko := e.ObjectOld.(*corev1.ResourceQuota)
	if !okn || !oko {
		return
	}
	if !equality.Semantic.DeepEqual(newRq.Status, oldRq.Status) {
		h.requeueTenantResourceQuota(newRq.Labels, r)
	}
}

func (h *ResourceQuotaHandler) Delete(e event.DeleteEvent, r workqueue.RateLimitingInterface) {
	rq, ok := e.Object.(*corev1.ResourceQuota)
	if !ok {
		return
	}
	h.requeueTenantResourceQuota(rq.Labels, r)
}

func (h *ResourceQuotaHandler) Generic(e event.GenericEvent, r workqueue.RateLimitingInterface) {
}

func newResourceQuotaHandler(c *client.Client, log *logr.Logger) *ResourceQuotaHandler {
	if _resourceQuotaHandler != nil {
		return _resourceQuotaHandler
	}
	_resourceQuotaHandler = &ResourceQuotaHandler{
		Client: *c,
		Log:    *log,
	}
	return _resourceQuotaHandler
}

func NewResourceQuotaHandler(c client.Client, log logr.Logger) *ResourceQuotaHandler {
	return newResourceQuotaHandler(&c, &log)
}

func (h *ResourceQuotaHandler) requeueTenantResourceQuota(labels map[string]string, r workqueue.RateLimitingInterface) {
	tenantName, exist := labels[gemlabels.LabelTenant]
	if !exist {
		return
	}
	r.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: tenantName,
		},
	})
}
