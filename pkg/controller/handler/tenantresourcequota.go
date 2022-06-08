package handler

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

/*
	TenantResourceQuota 变更的时候，对应的租户同步
*/

var _tenantResourceHandler *TenantResourceQuotaHandler

type TenantResourceQuotaHandler struct {
	Client client.Client
	Log    logr.Logger
}

func (h *TenantResourceQuotaHandler) Create(e event.CreateEvent, r workqueue.RateLimitingInterface) {
	rq, ok := e.Object.(*gemsv1beta1.TenantResourceQuota)
	if !ok {
		return
	}
	h.requeueTenantResourceQuota(rq.Labels, r)
}

func (h *TenantResourceQuotaHandler) Update(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
	newRq, okn := e.ObjectNew.(*gemsv1beta1.TenantResourceQuota)
	oldRq, oko := e.ObjectOld.(*gemsv1beta1.TenantResourceQuota)
	if !okn || !oko {
		return
	}
	if !equality.Semantic.DeepEqual(newRq.Status, oldRq.Status) {
		h.requeueTenantResourceQuota(newRq.Labels, r)
	}
}

func (h *TenantResourceQuotaHandler) Delete(e event.DeleteEvent, r workqueue.RateLimitingInterface) {
	rq, ok := e.Object.(*gemsv1beta1.TenantResourceQuota)
	if !ok {
		return
	}
	h.requeueTenantResourceQuota(rq.Labels, r)
}

func (h *TenantResourceQuotaHandler) Generic(e event.GenericEvent, r workqueue.RateLimitingInterface) {
}

func newTenantResourceQuotaHandler(c *client.Client, log *logr.Logger) *TenantResourceQuotaHandler {
	if _tenantResourceHandler != nil {
		return _tenantResourceHandler
	}
	_tenantResourceHandler = &TenantResourceQuotaHandler{
		Client: *c,
		Log:    *log,
	}
	return _tenantResourceHandler
}

func NewTenantResourceQuotaHandler(c client.Client, log logr.Logger) *TenantResourceQuotaHandler {
	return newTenantResourceQuotaHandler(&c, &log)
}

func (h *TenantResourceQuotaHandler) requeueTenantResourceQuota(labels map[string]string, r workqueue.RateLimitingInterface) {
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
