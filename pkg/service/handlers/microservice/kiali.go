package microservice

import (
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/agents"
)

type KialiAPIRequest struct {
	Path           string
	VirtualspaceId string
}

// @Tags VirtualSpace
// @Summary kiali代理
// @Description kiali api 代理
// @Accept json
// @Produce json
// @Param virtualspace_id 	path	uint 	true	"virtualspace_id"
// @Param environment_id 	path 	uint 	true	"environment_id（通过环境寻找目标集群）"
// @Param path				path 	string	true	"访问 kiali service 的路径"
// @Success 200 {object} 	object "kiali 原始响应"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/kiali/{kiaklipath} [get]
// @Security JWT
func (h *VirtualSpaceHandler) KialiAPI(c *gin.Context) {
	options := h.ServerInterface.GetOptions().Microservice
	kialisvc, kialinamespace := options.KialiName, options.KialiNamespace
	// get and check env
	env := models.Environment{}
	if err := h.GetDB().Preload("Cluster").First(&env, c.Param("environment_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	cluster, namespace, kialipath := env.Cluster.ClusterName, env.Namespace, c.Param("path")
	ctx := c.Request.Context()
	_ = namespace

	process := func() error {
		cli, err := h.clientOf(ctx, cluster)
		if err != nil {
			return err
		}
		// kiali svc 自带kiali前缀，我们的api也带kiali前缀，重复了
		c.Request.URL.Path = "/kiali" + kialipath
		if c.Request.URL.Scheme == "" {
			c.Request.URL.Scheme = "https"
		}

		kialisvc := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: kialisvc, Namespace: kialinamespace}}
		// 不直接使用 httputil 以方便对请求和响应进行改写，这里应该还可以改进
		rewirteresponsfunc := func(src io.Reader, dst io.Writer) error {
			var data interface{}
			if err := json.NewDecoder(src).Decode(&data); err != nil {
				return err
			}
			return json.NewEncoder(dst).Encode(handlers.ResponseStruct{Data: data})
		}
		if err := cli.Proxy(ctx, kialisvc, 20001, c.Request, c.Writer, agents.ResponseBodyRewriter(rewirteresponsfunc)); err != nil {
			return err
		}
		return nil
	}

	if err := process(); err != nil {
		handlers.NotOK(c, err)
		return
	}
}
