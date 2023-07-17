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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	resourcehelper "k8s.io/kubectl/pkg/util/resource"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NodeHandler struct {
	C client.Client
}

type metaForm struct {
	Labels      map[string]string
	Annotations map[string]string
}

//	@Tags			Agent.V1
//	@Summary		修改node的元数据,label和annotations
//	@Description	修改node的元数据,label和annotations
//	@Accept			json
//	@Produce		json
//	@Param			param	body		metaForm								true	"表单"`
//	@Param			name	path		string									true	"name"
//	@Param			cluster	path		string									true	"cluster"
//	@Success		200		{object}	handlers.ResponseStruct{Data=object}	"Node"
//	@Router			/v1/proxy/cluster/{cluster}/custom/core/v1/nodes/{name}/actions/metadata [patch]
//	@Security		JWT
func (h *NodeHandler) PatchNodeLabelOrAnnotations(c *gin.Context) {
	name := c.Param("name")
	formdata := metaForm{}
	if err := c.BindJSON(&formdata); err != nil {
		NotOK(c, err)
		return
	}

	data := &corev1.Node{}
	if err := h.C.Get(c.Request.Context(),
		types.NamespacedName{Name: name}, data); err != nil {
		NotOK(c, err)
		return
	}
	node := data.DeepCopy()

	node.SetLabels(formdata.Labels)
	node.SetAnnotations(formdata.Annotations)

	if err := h.C.Update(c, node); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, node)
}

type taintForm struct {
	Taints []corev1.Taint
}

//	@Tags			Agent.V1
//	@Summary		修改节点污点
//	@Description	修改节点污点
//	@Accept			json
//	@Produce		json
//	@Param			param	body		taintForm								true	"表单"`
//	@Param			name	path		string									true	"name"
//	@Param			cluster	path		string									true	"cluster"
//	@Success		200		{object}	handlers.ResponseStruct{Data=object}	"Node"
//	@Router			/v1/proxy/cluster/{cluster}/custom/core/v1/nodes/{name}/actions/taint [patch]
//	@Security		JWT
func (h *NodeHandler) PatchNodeTaint(c *gin.Context) {
	name := c.Param("name")
	formdata := taintForm{}
	if err := c.BindJSON(&formdata); err != nil {
		NotOK(c, err)
		return
	}

	data := &corev1.Node{}
	if err := h.C.Get(c.Request.Context(),
		types.NamespacedName{Name: name}, data); err != nil {
		NotOK(c, err)
		return
	}
	node := data.DeepCopy()

	node.Spec.Taints = formdata.Taints

	if err := h.C.Update(c, node); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, node)
}

type cordonForm struct {
	Unschedulable bool
}

//	@Tags			Agent.V1
//	@Summary		修改节点调度
//	@Description	修改节点调度
//	@Accept			json
//	@Produce		json
//	@Param			param	body		cordonForm								true	"表单"`
//	@Param			name	path		string									true	"name"
//	@Param			cluster	path		string									true	"cluster"
//	@Success		200		{object}	handlers.ResponseStruct{Data=object}	"Node"
//	@Router			/v1/proxy/cluster/{cluster}/custom/core/v1/nodes/{name}/actions/cordon [patch]
//	@Security		JWT
func (h *NodeHandler) PatchNodeCordon(c *gin.Context) {
	name := c.Param("name")

	formdata := cordonForm{}
	if err := c.BindJSON(&formdata); err != nil {
		NotOK(c, err)
		return
	}

	data := &corev1.Node{}
	if err := h.C.Get(c.Request.Context(),
		types.NamespacedName{Name: name}, data); err != nil {
		NotOK(c, err)
		return
	}
	node := data.DeepCopy()

	if node.Spec.Unschedulable == formdata.Unschedulable {
		OK(c, "ok")
		return
	}

	node.Spec.Unschedulable = formdata.Unschedulable

	if err := h.C.Update(c, node); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, node)
}

type CustomNode struct {
	Node          *corev1.Node
	TotalRequests map[corev1.ResourceName]resource.Quantity
	TotalLimits   map[corev1.ResourceName]resource.Quantity
}

//	@Tags			Agent.V1
//	@Summary		自定义的NODE详情接口,(可以获取资源分配情况)
//	@Description	自定义的NODE详情接口,(可以获取资源分配情况)
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string										true	"name"
//	@Param			cluster	path		string										true	"cluster"
//	@Success		200		{object}	handlers.ResponseStruct{Data=CustomNode}	"Node"
//	@Router			/v1/proxy/cluster/{cluster}/custom/core/v1/nodes/{name} [get]
//	@Security		JWT
func (h *NodeHandler) Get(c *gin.Context) {
	name := c.Param("name")
	node := &corev1.Node{}
	if err := h.C.Get(c.Request.Context(),
		types.NamespacedName{Name: name}, node); err != nil {
		NotOK(c, err)
		return
	}

	pods := &corev1.PodList{}
	fs := fields.SelectorFromSet(map[string]string{"nodename": name})
	opts := client.ListOptions{
		FieldSelector: fs,
	}
	h.C.List(c.Request.Context(), pods, &opts)
	totalReq, totalLmt := NodeTotalRequestsAndLimits(node, pods)
	cnode := CustomNode{
		Node:          node,
		TotalRequests: totalReq,
		TotalLimits:   totalLmt,
	}
	OK(c, cnode)
}

func NodeTotalRequestsAndLimits(node *corev1.Node, podList *corev1.PodList) (reqs, limits corev1.ResourceList) {
	reqs, limits = map[corev1.ResourceName]resource.Quantity{}, map[corev1.ResourceName]resource.Quantity{}
	for resourceName := range node.Status.Capacity {
		reqs[resourceName], limits[resourceName] = resource.Quantity{}, resource.Quantity{}
	}

	for _, pod := range podList.Items {
		podReqs, podLimits := resourcehelper.PodRequestsAndLimits(&pod)
		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = podReqValue.DeepCopy()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}
		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				limits[podLimitName] = podLimitValue.DeepCopy()
			} else {
				value.Add(podLimitValue)
				limits[podLimitName] = value
			}
		}
	}
	return
}
