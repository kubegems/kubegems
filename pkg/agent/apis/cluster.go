package apis

import (
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/set"
)

type ClusterHandler struct {
	cluster cluster.Interface
}

var groups = set.NewSet[string]().Append(
	"v1",
	"apps/v1",
	"batch/v1",
	"events.k8s.io/v1",
	"metrics.k8s.io/v1beta1",
	"networking.k8s.io/v1",
	"storage.k8s.io/v1",
	"snapshot.storage.k8s.io/v1",
	"metrics.k8s.io/v1beta1",
)

// @Tags         Agent.V1
// @Summary      获取k8s api-resources
// @Description  获取k8s api-resources
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                  true  "cluster"
// @Success      200      {object}  handlers.ResponseStruct{Data=[]object}  "resp"
// @Router       /v1/proxy/cluster/{cluster}/api-resources [get]
func (h *ClusterHandler) APIResources(c *gin.Context) {
	resources, err := h.cluster.Discovery().ServerPreferredResources()
	if err != nil {
		if discovery.IsGroupDiscoveryFailedError(err) {
			log.Warnf("get api-resources failed: %v", err)
			OK(c, resources)
			return
		} else {
			NotOK(c, err)
			return
		}
	}
	ret := []*metav1.APIResourceList{}
	for _, v := range resources {
		if groups.Has(v.GroupVersion) {
			ret = append(ret, v)
		}
	}
	OK(c, ret)
}
