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

//	@Tags			Agent.V1
//	@Summary		获取k8s api-resources
//	@Description	获取k8s api-resources
//	@Accept			json
//	@Produce		json
//	@Param			cluster	path		string									true	"cluster"
//	@Success		200		{object}	handlers.ResponseStruct{Data=[]object}	"resp"
//	@Router			/v1/proxy/cluster/{cluster}/api-resources [get]
func (h *ClusterHandler) APIResources(c *gin.Context) {
	resources, err := h.cluster.Discovery().ServerPreferredResources()
	if err != nil {
		if discovery.IsGroupDiscoveryFailedError(err) {
			log.Warnf("get api-resources failed: %v", err)
			OK(c, resources)
			return
		} else {
			OK(c, err)
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
