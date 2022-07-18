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

package handler

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

/*
	Node 变更的时候，networkploicy 中间的cidr列表需要更新
*/

var _nodeHandler *NodeHandler

type NodeHandler struct {
	Client client.Client
	Log    logr.Logger
}

func (h *NodeHandler) Create(e event.CreateEvent, r workqueue.RateLimitingInterface) {
	dep, ok := e.Object.(*corev1.Node)
	if !ok {
		return
	}
	h.requeueTNetPol(dep.OwnerReferences, r)
}

func (h *NodeHandler) Update(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
}

func (h *NodeHandler) Delete(e event.DeleteEvent, r workqueue.RateLimitingInterface) {
	dep, ok := e.Object.(*corev1.Node)
	if !ok {
		return
	}
	h.requeueTNetPol(dep.OwnerReferences, r)
}

func (h *NodeHandler) Generic(e event.GenericEvent, r workqueue.RateLimitingInterface) {
}

func newNodeHandler(c client.Client, log logr.Logger) *NodeHandler {
	if _nodeHandler != nil {
		return _nodeHandler
	}
	_nodeHandler = &NodeHandler{
		Client: c,
		Log:    log,
	}
	return _nodeHandler
}

func NewNodeHandler(c client.Client, log logr.Logger) *NodeHandler {
	return newNodeHandler(c, log)
}

func (h *NodeHandler) requeueTNetPol(owners []metav1.OwnerReference, r workqueue.RateLimitingInterface) {
	tnetpol := v1beta1.TenantNetworkPolicyList{}
	if err := h.Client.List(context.Background(), &tnetpol); err != nil {
		h.Log.Error(err, "faield to list tenant network policiesj")
		return
	}
	for _, tp := range tnetpol.Items {
		r.Add(ctrl.Request{
			NamespacedName: client.ObjectKeyFromObject(&tp),
		})
	}
}
