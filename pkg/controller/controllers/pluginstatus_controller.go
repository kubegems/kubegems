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

package controllers

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	nginxv1alpha1 "github.com/nginxinc/nginx-ingress-operator/api/v1alpha1"
	istiooperatorv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var PluginStatusInstance = &PluginStatus{init: make(chan struct{})}

type ComponentName string

const (
	ComponentNginx ComponentName = "nginx"
	ComponentIstio ComponentName = "istio"
)

type PluginStatus struct {
	init                          chan struct{} // 是否完成
	istioOperatorEnabled          bool
	nginxIngressControllerEnabled bool
}

func (p *PluginStatus) ComponentEnabled(name ComponentName) bool {
	// 如果未初始化完成，则阻塞这里
	// 因为还不知道这些组件是否正常安装了
	<-p.init
	switch name {
	case ComponentNginx:
		return p.nginxIngressControllerEnabled
	case ComponentIstio:
		return p.istioOperatorEnabled
	default:
		return false
	}
}

//  PluginStatusController 通过crd是否存在以判断对应组件是否被正常安装
// 用于解决当集群中未安装对应crd时，controller 执行产生错误
// 举例： 当istio未安装时，查询istio serviceentry 会产生错误
// 使用方式：
//    ⬆️ 这里有一个 单例 ，可以从里面判断是否安装

//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

type PluginStatusController struct {
	client.Client
	Log logr.Logger
}

func (r *PluginStatusController) Init(ctx context.Context) error {
	crds := &apiextensionsv1.CustomResourceDefinitionList{}
	if err := r.Client.List(ctx, crds); err != nil {
		return err
	}
	for _, crd := range crds.Items {
		r.OnChange(ctx, &crd, true)
	}
	// 初始化完成
	close(PluginStatusInstance.init)
	return nil
}

func (r *PluginStatusController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	exist := &apiextensionsv1.CustomResourceDefinition{}
	if err := r.Client.Get(ctx, req.NamespacedName, exist); err != nil {
		return ctrl.Result{}, err
	}

	if exist.DeletionTimestamp != nil {
		r.OnChange(ctx, exist, false)
	}
	return r.OnChange(ctx, exist, true)
}

func (r *PluginStatusController) OnChange(ctx context.Context, crd *apiextensionsv1.CustomResourceDefinition, exist bool) (ctrl.Result, error) {
	switch crd.Spec.Group {
	// 判断nginxingress operator是否被安装 nginxingresscontrollers.k8s.nginx.org
	case nginxv1alpha1.GroupVersion.Group:
		PluginStatusInstance.nginxIngressControllerEnabled = exist
	// 判断istio operator是否被安装 istiooperators.install.istio.io
	case istiooperatorv1alpha1.SchemeGroupVersion.Group:
		PluginStatusInstance.istioOperatorEnabled = exist
		r.Log.Info("istio plugin status", "enabled", exist)
	}
	return ctrl.Result{}, nil
}

func (r *PluginStatusController) SetupWithManager(mgr ctrl.Manager) error {
	go func() {
		<-mgr.Elected()
		// 等待 mgr 准备就绪，也是等待cache完成
		if err := r.Init(context.TODO()); err != nil {
			r.Log.Error(err, "failed init plugin status")
			os.Exit(1)
		}
	}()
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiextensionsv1.CustomResourceDefinition{}).
		Complete(r)
}
