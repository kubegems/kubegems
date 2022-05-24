package clusterhandler

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kversion "k8s.io/apimachinery/pkg/version"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/kube"
)

type ValidateKubeConfigReq struct {
	KubeConfig string `json:"kubeconfig" binding:"json"`
}

type ValidateKubeConfigResp struct {
	ServerInfo     *kversion.Info `json:"serverInfo,omitempty"`
	StorageClasses []string       `json:"storageClasses,omitempty"`

	// 判断是否存在installer，若存在即可加为控制集群
	ExistInstaller bool   `json:"existInstaller"`
	ClusterName    string `json:"clusterName"`

	Connectable bool   `json:"connectable,omitempty"`
	Message     string `json:"message,omitempty"`
}

const localPluginURL = "/apis/plugins.kubegems.io/v1beta1/namespaces/kubegems-local/plugins/kubegems-local-stack"

// ValidateKubeConfig 添加cluster前的kubeconfig检测接口，验证kubeconfig，返回集群信息和可用的storageClass列表
// @Tags         Cluster
// @Summary      添加cluster前的kubeconfig检测接口，验证kubeconfig，返回集群信息和可用的storageClass列表
// @Description  添加cluster前的kubeconfig检测接口，验证kubeconfig，返回集群信息和可用的storageClass列表
// @Accept       json
// @Produce      json
// @Param        param  body      ValidateKubeConfigReq                                 true  "表单"
// @Success      200    {object}  handlers.ResponseStruct{Data=ValidateKubeConfigResp}  "Cluster"
// @Router       /v1/cluster/validate-kubeconfig [post]
// @Security     JWT
func (h *ClusterHandler) ValidateKubeConfig(c *gin.Context) {
	resp := ValidateKubeConfigResp{
		StorageClasses: []string{},
		Connectable:    false,
	}
	ctx := c.Request.Context()
	cfg := &ValidateKubeConfigReq{}
	if err := c.BindJSON(cfg); err != nil {
		log.Debugf("validate kubeconfig bind error: %s", err.Error())
		resp.Message = err.Error()
		handlers.OK(c, resp)
		return
	}
	_, clientSet, err := kube.GetKubeClient([]byte(cfg.KubeConfig))
	if err != nil {
		log.Debugf("validate kubeconfig get clientset error: %s", err.Error())
		resp.Message = err.Error()
		handlers.OK(c, resp)
		return
	}
	serverInfo, err := clientSet.ServerVersion()
	if err != nil {
		resp.Message = err.Error()
		handlers.OK(c, resp)
		return
	}
	resp.ServerInfo = serverInfo
	resp.Connectable = true
	scList, err := clientSet.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		resp.Message = fmt.Sprintf("get storageclass failed %v", err)
		handlers.OK(c, err)
		return
	}
	for _, sc := range scList.Items {
		resp.StorageClasses = append(resp.StorageClasses, sc.GetName())
	}

	bts, err := clientSet.Discovery().RESTClient().Get().AbsPath(localPluginURL).Do(ctx).Raw()
	if err == nil {
		plugin := pluginsv1beta1.Plugin{}
		if err := json.Unmarshal(bts, &plugin); err != nil {
			handlers.NotOK(c, errors.Wrap(err, "unmarshal plugin"))
			return
		}
		values := map[string]interface{}{}
		json.Unmarshal(plugin.Spec.Values.Raw, &values)
		if global, ok := values["global"]; ok {
			if globalMap, ok := global.(map[string]interface{}); ok {
				if name, ok := globalMap["clusterName"]; ok {
					resp.ClusterName = name.(string)
				}
			}
		}
		resp.ExistInstaller = true
	}
	handlers.OK(c, resp)
}
