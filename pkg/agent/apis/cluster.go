package apis

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/agent/cluster"
)

type ClusterHandler struct {
	cluster cluster.Interface
}

// @Tags         Agent.V1
// @Summary      获取k8s api-resources
// @Description  获取k8s api-resources
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                                          true  "cluster"
// @Success      200      {object}  handlers.ResponseStruct{Data=handlers.PageData{List=[]object}}  "resp"
// @Router       /v1/proxy/cluster/{cluster}/api-resources [get]
func (h *ClusterHandler) APIResources(c *gin.Context) {
	ret, err := h.cluster.Discovery().ServerPreferredResources()
	if err != nil {
		NotOK(c, err)
	}
	OK(c, ret)
}
