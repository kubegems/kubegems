// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apis

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/agent/cluster"
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
		NotOK(c, err)
	}
	OK(c, ret)
}
