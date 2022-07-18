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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

/*
	监听所有的环境事件，当环境租户和namespace变更的时候,让对应的租户重新计算状态
*/
var _envHandler *EnvironmentHandler

type EnvironmentHandler struct {
	Client client.Client
	Log    logr.Logger
}

func (h *EnvironmentHandler) Create(e event.CreateEvent, r workqueue.RateLimitingInterface) {
	env, ok := e.Object.(*gemsv1beta1.Environment)
	if !ok {
		return
	}
	r.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: env.Spec.Tenant,
		},
	})
}

func (h *EnvironmentHandler) Update(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
	newobj, okn := e.ObjectNew.(*gemsv1beta1.Environment)
	oldobj, oko := e.ObjectOld.(*gemsv1beta1.Environment)

	if !okn || !oko {
		return
	}
	if newobj.Spec.Tenant == oldobj.Spec.Tenant && newobj.Spec.Namespace == oldobj.Spec.Namespace {
		return
	}
	r.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: newobj.Spec.Tenant,
		},
	})
	r.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: oldobj.Spec.Tenant,
		},
	})
}

func (h *EnvironmentHandler) Delete(e event.DeleteEvent, r workqueue.RateLimitingInterface) {
	env, ok := e.Object.(*gemsv1beta1.Environment)
	if !ok {
		return
	}
	r.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: env.Spec.Tenant,
		},
	})
}

func (h *EnvironmentHandler) Generic(e event.GenericEvent, r workqueue.RateLimitingInterface) {
}

func newEnvHandler(c *client.Client, log *logr.Logger) *EnvironmentHandler {
	if _envHandler != nil {
		return _envHandler
	}
	_envHandler = &EnvironmentHandler{
		Client: *c,
		Log:    *log,
	}
	return _envHandler
}

func NewEnvironmentHandler(c client.Client, log logr.Logger) *EnvironmentHandler {
	return newEnvHandler(&c, &log)
}
