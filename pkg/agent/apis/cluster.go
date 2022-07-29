package apis

import (
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/discovery"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/log"
)

type ClusterHandler struct {
	cluster cluster.Interface
}

// @Tags         Agent.V1
// @Summary      获取k8s api-resources
// @Description  获取k8s api-resources
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                  true  "cluster"
// @Success      200      {object}  handlers.ResponseStruct{Data=[]object}  "resp"
// @Router       /v1/proxy/cluster/{cluster}/api-resources [get]
func (h *ClusterHandler) APIResources(c *gin.Context) {
	ret, err := h.cluster.Discovery().ServerPreferredResources()
	if err != nil {
		if discovery.IsGroupDiscoveryFailedError(err) {
			log.Warnf("get api-resources failed: %v", err)
			OK(c, ret)
			return
		} else {
			NotOK(c, err)
			return
		}
	}
	OK(c, ret)
}
