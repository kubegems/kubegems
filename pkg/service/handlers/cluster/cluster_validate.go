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

package clusterhandler

import (
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kversion "k8s.io/apimachinery/pkg/version"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/gemsplugin"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ValidateKubeConfigReq struct {
	KubeConfig string `json:"kubeconfig" binding:"json"`
}

type ValidateKubeConfigResp struct {
	ServerInfo     *kversion.Info `json:"serverInfo,omitempty"`
	StorageClasses []string       `json:"storageClasses,omitempty"`

	// 判断是否存在installer，若存在即可加为控制集群
	ExistInstaller        bool   `json:"existInstaller"`
	ExistStorageClassName string `json:"existStorageClassName"`
	ClusterName           string `json:"clusterName"`

	Connectable bool   `json:"connectable,omitempty"`
	Message     string `json:"message,omitempty"`
}

// ValidateKubeConfig 添加cluster前的kubeconfig检测接口，验证kubeconfig，返回集群信息和可用的storageClass列表
// @Tags        Cluster
// @Summary     添加cluster前的kubeconfig检测接口，验证kubeconfig，返回集群信息和可用的storageClass列表
// @Description 添加cluster前的kubeconfig检测接口，验证kubeconfig，返回集群信息和可用的storageClass列表
// @Accept      json
// @Produce     json
// @Param       param body     ValidateKubeConfigReq                                true "表单"
// @Success     200   {object} handlers.ResponseStruct{Data=ValidateKubeConfigResp} "Cluster"
// @Router      /v1/cluster/validate-kubeconfig [post]
// @Security    JWT
func (h *ClusterHandler) ValidateKubeConfig(c *gin.Context) {
	resp := ValidateKubeConfigResp{
		StorageClasses: []string{},
		Connectable:    false,
	}
	ctx := c.Request.Context()
	cfg := &ValidateKubeConfigReq{}
	if err := c.BindJSON(cfg); err != nil {
		resp.Message = i18n.Sprintf(c, "invalid kubeconfig format: %v", err)
		handlers.OK(c, resp)
		return
	}
	restconfig, clientSet, err := kube.GetKubeClient([]byte(cfg.KubeConfig))
	if err != nil {
		resp.Message = i18n.Sprintf(c, "invalid kubeconfig: %v", err)
		handlers.OK(c, resp)
		return
	}
	serverInfo, err := clientSet.ServerVersion()
	if err != nil {
		resp.Message = i18n.Sprintf(c, "failed to get api-server info: %v", err)
		handlers.OK(c, resp)
		return
	}
	resp.ServerInfo = serverInfo
	resp.Connectable = true
	scList, err := clientSet.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		resp.Message = i18n.Sprintf(c, "failed to list StorageClass: %v", err)
		handlers.OK(c, resp)
		return
	}
	for _, sc := range scList.Items {
		resp.StorageClasses = append(resp.StorageClasses, sc.GetName())
	}

	cli, err := client.New(restconfig, client.Options{})
	if err != nil {
		resp.Message = i18n.Sprintf(c, "failed to init k8s client: %v", err)
		handlers.OK(c, resp)
		return
	}
	if vals, err := (&gemsplugin.PluginManager{Client: cli}).GetGlobalValues(ctx); err == nil {
		if name, ok := vals["clusterName"]; ok {
			resp.ClusterName = name
		}
		resp.ExistInstaller = true
	}
	handlers.OK(c, resp)
}
