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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"kubegems.io/kubegems/pkg/apis/gems"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/utils/slice"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceHandler struct {
	C client.Client
}

var forbiddenBindNamespaces = []string{
	"kube-system",
	"istio-system",
	gems.NamespaceSystem,
	gems.NamespaceLocal,
	gems.NamespaceInstaller,
	gems.NamespaceMonitor,
	gems.NamespaceLogging,
	gems.NamespaceGateway,
}

// @Tags        Agent.V1
// @Summary     获取可以绑定的环境的namespace列表数据
// @Description 获取可以绑定的环境的namespace列表数据
// @Accept      json
// @Produce     json
// @Param       order   query    string                                                           false "page"
// @Param       search  query    string                                                           false "search"
// @Param       page    query    int                                                              false "page"
// @Param       size    query    int                                                              false "page"
// @Param       cluster path     string                                                           true  "cluster"
// @Success     200     {object} handlers.ResponseStruct{Data=pagination.PageData{List=[]object}} "Namespace"
// @Router      /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces [get]
// @Security    JWT
func (h *NamespaceHandler) List(c *gin.Context) {
	nsList := &corev1.NamespaceList{}
	sel := labels.NewSelector()
	req, _ := labels.NewRequirement(gemlabels.LabelEnvironment, selection.DoesNotExist, []string{})
	listOptions := &client.ListOptions{
		LabelSelector: sel.Add(*req),
	}
	if err := h.C.List(c.Request.Context(), nsList, listOptions); err != nil {
		NotOK(c, err)
		return
	}

	objects := []corev1.Namespace{}
	for _, obj := range nsList.Items {
		if !slice.ContainStr(forbiddenBindNamespaces, obj.Name) {
			objects = append(objects, obj)
		}
	}
	pageData := NewPageDataFromContext(c, func(i int) SortAndSearchAble {
		return &objects[i]
	}, len(objects), objects)
	OK(c, pageData)
}
