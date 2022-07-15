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

package microservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	prototypes "github.com/gogo/protobuf/types"
	"istio.io/api/operator/v1alpha1"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	pkgv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/apis/networking"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	microservice "kubegems.io/kubegems/pkg/service/handlers/microservice/options"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	virtualSpaceKey = networking.AnnotationVirtualSpace
	istioGatewayKey = networking.AnnotationIstioGateway
)

type IstioGatewayInstance struct {
	// TODO add more fields
	Name            string `binding:"required"`
	Enabled         bool
	Gateways        []v1beta1.Gateway        `json:",omitempty"`
	VirtualServices []v1beta1.VirtualService `json:",omitempty"`
	Pods            []corev1.Pod             `json:",omitempty"`

	appsv1.DeploymentStatus `json:"status"`
	Ports                   []corev1.ServicePort `json:"ports"`
}

type IstioGatewayHandler struct {
	MicroserviceOptions *microservice.MicroserviceOptions
	base.BaseHandler
}

// @Tags         Istio
// @Summary      istio网关实例列表
// @Description  istio网关实例列表
// @Accept       json
// @Produce      json
// @Param        virtualspace_id  path      string                                                true  "virtualspace_id"
// @Param        cluster_id       path      string                                                true  "cluster_id"
// @Success      200              {object}  handlers.ResponseStruct{Data=[]IstioGatewayInstance}  "IstioOperator"
// @Router       /v1/virtualspace/{virtualspace_id}/cluster/{cluster_id}/istiogateways [get]
// @Security     JWT
// nolint: funlen
func (h *IstioGatewayHandler) ListGateway(c *gin.Context) {
	istioOperatorNamespace := h.MicroserviceOptions.IstioNamespace
	istioOperatorName := h.MicroserviceOptions.IstioOperatorName
	istioGatewayNamespace := h.MicroserviceOptions.GatewayNamespace

	vs := models.VirtualSpace{}
	if err := h.GetDB().First(&vs, c.Param("virtualspace_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	cluster := models.Cluster{}
	if err := h.GetDB().First(&cluster, c.Param("cluster_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	ctx := c.Request.Context()
	op := pkgv1alpha1.IstioOperator{}
	deps := appsv1.DeploymentList{} // 获取副本数
	svcs := corev1.ServiceList{}    // 获取端口
	if err := h.Execute(ctx, cluster.ClusterName, func(ctx context.Context, tc agents.Client) error {
		if err := tc.Get(ctx, types.NamespacedName{
			Namespace: istioOperatorNamespace,
			Name:      istioOperatorName,
		}, &op); err != nil {
			return err
		}
		if err := tc.List(ctx, &deps,
			client.InNamespace(istioGatewayNamespace),
			client.MatchingLabels(map[string]string{virtualSpaceKey: vs.VirtualSpaceName})); err != nil {
			return err
		}
		if err := tc.List(ctx, &svcs,
			client.InNamespace(istioGatewayNamespace),
			client.MatchingLabels(map[string]string{virtualSpaceKey: vs.VirtualSpaceName})); err != nil {
			return err
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	depMap := map[string]appsv1.Deployment{}
	svcMap := map[string]corev1.Service{}
	for _, v := range deps.Items {
		depMap[v.Name] = v
	}
	for _, v := range svcs.Items {
		svcMap[v.Name] = v
	}

	if op.Spec.Components == nil {
		op.Spec.Components = &v1alpha1.IstioComponentSetSpec{}
	}

	gws := op.Spec.Components.IngressGateways
	ret := []IstioGatewayInstance{}
	selector := labels.SelectorFromSet(labels.Set{virtualSpaceKey: vs.VirtualSpaceName})
	for _, v := range gws {
		if selector.Matches(labels.Set(v.Label)) {
			ret = append(ret, IstioGatewayInstance{
				Name:             v.Name,
				Enabled:          v.Enabled.Value,
				DeploymentStatus: depMap[v.Name].Status,
				Ports:            svcMap[v.Name].Spec.Ports,
			})
		}
	}

	handlers.OK(c, ret)
}

// @Tags         Istio
// @Summary      istio网关实例列表
// @Description  istio网关实例列表
// @Accept       json
// @Produce      json
// @Param        virtualspace_id  path      string                                              true  "virtualspace_id"
// @Param        cluster_id       path      string                                              true  "cluster_id"
// @Param        name             path      string                                              true  "网关名"
// @Success      200              {object}  handlers.ResponseStruct{Data=IstioGatewayInstance}  "IstioOperator"
// @Router       /v1/virtualspace/{virtualspace_id}/cluster/{cluster_id}/istiogateways/{name} [get]
// @Security     JWT
// nolint: funlen,gocognit
func (h *IstioGatewayHandler) GetGateway(c *gin.Context) {
	istioGatewayNamespace := h.MicroserviceOptions.GatewayNamespace
	istioOperatorNamespace := h.MicroserviceOptions.IstioNamespace
	istioOperatorName := h.MicroserviceOptions.IstioOperatorName

	vs := models.VirtualSpace{}
	if err := h.GetDB().First(&vs, c.Param("virtualspace_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	cluster := models.Cluster{}
	if err := h.GetDB().First(&cluster, c.Param("cluster_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	name := c.Param("name")
	ctx := c.Request.Context()
	op := pkgv1alpha1.IstioOperator{}
	gateways := v1beta1.GatewayList{}
	virtualSvcs := v1beta1.VirtualServiceList{}
	dep := appsv1.Deployment{} // 获取副本数
	svc := corev1.Service{}    // 获取端口
	podlist := corev1.PodList{}
	if err := h.Execute(ctx, cluster.ClusterName, func(ctx context.Context, tc agents.Client) error {
		if err := tc.Get(ctx, types.NamespacedName{
			Namespace: istioOperatorNamespace,
			Name:      istioOperatorName,
		}, &op); err != nil {
			return err
		}
		if err := tc.List(ctx, &gateways, client.InNamespace(v1.NamespaceAll)); err != nil {
			return err
		}
		if err := tc.List(ctx, &virtualSvcs, client.InNamespace(v1.NamespaceAll)); err != nil {
			return err
		}
		// 拿不到状态不能影响网关返回
		if err := tc.Get(ctx, types.NamespacedName{
			Namespace: istioGatewayNamespace,
			Name:      name,
		}, &dep); err != nil {
			log.Warnf("get deployment :%s", err.Error())
		}
		if err := tc.Get(ctx, types.NamespacedName{
			Namespace: istioGatewayNamespace,
			Name:      name,
		}, &svc); err != nil {
			log.Warnf("get service :%s", err.Error())
		}
		if err := tc.List(ctx, &podlist,
			client.InNamespace(istioGatewayNamespace),
			client.MatchingLabels(map[string]string{
				virtualSpaceKey: vs.VirtualSpaceName,
				istioGatewayKey: name,
			})); err != nil {
			log.Warnf("list pods :%s", err.Error())
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if op.Spec.Components == nil {
		op.Spec.Components = &v1alpha1.IstioComponentSetSpec{}
	}

	gws := op.Spec.Components.IngressGateways
	var gw *v1alpha1.GatewaySpec
	for i := range gws {
		if gws[i].Name == name {
			gw = gws[i]
			break
		}
	}
	if gw == nil {
		handlers.NotOK(c, fmt.Errorf("未找到网关%s", name))
		return
	}

	ret := IstioGatewayInstance{
		Name:             gw.Name,
		Enabled:          gw.Enabled.Value,
		DeploymentStatus: dep.Status,
		Ports:            svc.Spec.Ports,
		Pods:             podlist.Items,
	}

	for _, v := range gateways.Items {
		selector := labels.SelectorFromSet(v.Spec.Selector)
		if selector.Matches(labels.Set(gw.Label)) {
			ret.Gateways = append(ret.Gateways, v)
		}
	}

	// Gateways in other namespaces may be referred to by <gateway namespace>/<gateway name>;
	// specifying a gateway with no namespace qualifier is the same as specifying the VirtualService’s namespace
	gatewayNamespaceNameMap := map[string]struct{}{}
	for _, gw := range ret.Gateways {
		gatewayNamespaceNameMap[gw.Namespace+"/"+gw.Name] = struct{}{}
	}
	for _, vs := range virtualSvcs.Items {
		for _, namespacedName := range vs.Spec.Gateways {
			// 没指定namespace需要加上，以匹配同namespace的gateway
			if !strings.Contains(namespacedName, "/") {
				namespacedName = vs.Namespace + "/" + namespacedName
			}
			if _, ok := gatewayNamespaceNameMap[namespacedName]; ok {
				ret.VirtualServices = append(ret.VirtualServices, vs)
			}
		}
	}

	handlers.OK(c, ret)
}

// @Tags         Istio
// @Summary      创建istio网关实例`
// @Description  创建istio网关实例
// @Accept       json
// @Produce      json
// @Param        virtualspace_id  path      string                                              true  "virtualspace_id"
// @Param        cluster_id       path      string                                              true  "cluster_id"
// @Param        param            body      IstioGatewayInstance                                true  "网关内容"
// @Success      200              {object}  handlers.ResponseStruct{Data=IstioGatewayInstance}  "网关内容"
// @Router       /v1/virtualspace/{virtualspace_id}/cluster/{cluster_id}/istiogateways [post]
// @Security     JWT
// nolint: funlen
func (h *IstioGatewayHandler) CreateGateway(c *gin.Context) {
	istioOperatorNamespace := h.MicroserviceOptions.IstioNamespace
	istioOperatorName := h.MicroserviceOptions.IstioOperatorName
	istioGatewayNamespace := h.MicroserviceOptions.GatewayNamespace

	vs := models.VirtualSpace{}
	if err := h.GetDB().First(&vs, c.Param("virtualspace_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	cluster := models.Cluster{}
	if err := h.GetDB().First(&cluster, c.Param("cluster_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	gw := IstioGatewayInstance{}
	if err := c.BindJSON(&gw); err != nil {
		handlers.NotOK(c, err)
		return
	}
	gw.Enabled = true // 默认启用

	ctx := c.Request.Context()
	op := pkgv1alpha1.IstioOperator{}
	if err := h.Execute(ctx, cluster.ClusterName, func(ctx context.Context, tc agents.Client) error {
		dep := appsv1.Deployment{}
		err := tc.Get(ctx, types.NamespacedName{
			Namespace: istioGatewayNamespace,
			Name:      gw.Name,
		}, &dep)
		// 避免与租户网关同名
		if err == nil {
			return fmt.Errorf("网关%s已存在", gw.Name)
		}
		if !kerrors.IsNotFound(err) {
			return err
		}

		if err := tc.Get(ctx, types.NamespacedName{
			Namespace: istioOperatorNamespace,
			Name:      istioOperatorName,
		}, &op); err != nil {
			return err
		}

		if op.Spec.Components == nil {
			op.Spec.Components = &v1alpha1.IstioComponentSetSpec{}
		}
		found := false
		for _, v := range op.Spec.Components.IngressGateways {
			if v.Name == gw.Name {
				found = true
				break
			}
		}
		if found {
			return fmt.Errorf("网关%s已存在", gw.Name)
		}

		op.Spec.Components.IngressGateways = append(op.Spec.Components.IngressGateways,
			istioGateway(vs.VirtualSpaceName, istioGatewayNamespace, gw.Name, gw.Enabled))

		return tc.Update(ctx, &op)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, gw)
}

// @Tags         Istio
// @Summary      更新istio网关实例
// @Description  更新istio网关实例
// @Accept       json
// @Produce      json
// @Param        cluster_id       path      string                                              true  "cluster_id"
// @Param        virtualspace_id  path      string                                              true  "virtualspace_id"
// @Param        param            body      IstioGatewayInstance                                true  "网关内容"
// @Success      200              {object}  handlers.ResponseStruct{Data=IstioGatewayInstance}  "网关内容"
// @Router       /v1/virtualspace/{virtualspace_id}/cluster/{cluster_id}/istiogateways [put]
// @Security     JWT
func (h *IstioGatewayHandler) UpdateGateway(c *gin.Context) {
	istioOperatorNamespace := h.MicroserviceOptions.IstioNamespace
	istioOperatorName := h.MicroserviceOptions.IstioOperatorName
	istioGatewayNamespace := h.MicroserviceOptions.GatewayNamespace

	vs := models.VirtualSpace{}
	if err := h.GetDB().First(&vs, c.Param("virtualspace_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	cluster := models.Cluster{}
	if err := h.GetDB().First(&cluster, c.Param("cluster_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	gw := IstioGatewayInstance{}
	if err := c.BindJSON(&gw); err != nil {
		handlers.NotOK(c, err)
		return
	}

	ctx := c.Request.Context()
	op := pkgv1alpha1.IstioOperator{}
	if err := h.Execute(ctx, cluster.ClusterName, func(ctx context.Context, tc agents.Client) error {
		if err := tc.Get(ctx, types.NamespacedName{
			Namespace: istioOperatorNamespace,
			Name:      istioOperatorName,
		}, &op); err != nil {
			return err
		}

		if op.Spec.Components == nil {
			op.Spec.Components = &v1alpha1.IstioComponentSetSpec{}
		}

		found := false
		index := 0
		for i, v := range op.Spec.Components.IngressGateways {
			if v.Name == gw.Name {
				found = true
				index = i
				break
			}
		}
		if !found {
			return fmt.Errorf("网关%s不存在", gw.Name)
		}

		op.Spec.Components.IngressGateways[index] = istioGateway(vs.VirtualSpaceName, istioGatewayNamespace, gw.Name, gw.Enabled)
		return tc.Update(ctx, &op)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, gw)
}

func istioGateway(vsName, ns, name string, enabled bool) *v1alpha1.GatewaySpec {
	return &v1alpha1.GatewaySpec{
		Name:      name,
		Namespace: ns,
		Enabled: &v1alpha1.BoolValueForPB{
			BoolValue: prototypes.BoolValue{Value: enabled},
		},
		Label: map[string]string{
			virtualSpaceKey: vsName,
			istioGatewayKey: name,
		},
		K8S: &v1alpha1.KubernetesResourcesSpec{
			PodAnnotations: map[string]string{
				"proxy.istio.io/config": `proxyStatsMatcher:
  inclusionRegexps:
    - .*downstream_rq.*`,
			},
			Resources: &v1alpha1.Resources{
				Requests: map[string]string{
					"cpu":    "1",
					"memory": "2Gi",
				},
				Limits: map[string]string{
					"cpu":    "1",
					"memory": "2Gi",
				},
			},
		},
	}
}

// @Tags         Istio
// @Summary      删除istio网关实例
// @Description  删除istio网关实例
// @Accept       json
// @Produce      json
// @Param        virtualspace_id  path      string                                true  "virtualspace_id"
// @Param        cluster_id       path      string                                true  "cluster_id"
// @Success      200              {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/virtualspace/{virtualspace_id}/cluster/{cluster_id}/istiogateways/{name} [delete]
// @Security     JWT
func (h *IstioGatewayHandler) DeleteGateway(c *gin.Context) {
	istioOperatorNamespace := h.MicroserviceOptions.IstioNamespace
	istioOperatorName := h.MicroserviceOptions.IstioOperatorName

	vs := models.VirtualSpace{}
	if err := h.GetDB().First(&vs, c.Param("virtualspace_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	cluster := models.Cluster{}
	if err := h.GetDB().First(&cluster, c.Param("cluster_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	gwName := c.Param("name")

	ctx := c.Request.Context()

	op := &pkgv1alpha1.IstioOperator{
		ObjectMeta: v1.ObjectMeta{
			Name:      istioOperatorName,
			Namespace: istioOperatorNamespace,
		},
	}
	err := h.Execute(ctx, cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
		if err := cli.Get(ctx, client.ObjectKeyFromObject(op), op); err != nil {
			return err
		}

		if op.Spec.Components == nil {
			op.Spec.Components = &v1alpha1.IstioComponentSetSpec{}
		}

		found := false
		index := 0
		for i, v := range op.Spec.Components.IngressGateways {
			if v.Name == gwName {
				found = true
				index = i
				break
			}
		}
		if !found {
			return fmt.Errorf("gateway %s not found", gwName)
		}

		op.Spec.Components.IngressGateways = append(op.Spec.Components.IngressGateways[:index],
			op.Spec.Components.IngressGateways[index+1:]...)

		return cli.Update(ctx, op)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// func getOrCreateIstioOperator(cluster string) (*pkgv1alpha1.IstioOperator, error) {
// 	ope, err := kubeclient.GetClient().GetIstioOperator(cluster, istioOperatorNamespace, istioOperatorName, nil)
// 	if err != nil {
// 		if kerrors.IsNotFound(err) {
// 			op, err := kubeclient.GetClient().CreateIstioOperator(cluster, istioOperatorNamespace, istioOperatorName, &pkgv1alpha1.IstioOperator{
// 				ObjectMeta: v1.ObjectMeta{
// 					Name:      istioOperatorName,
// 					Namespace: istioOperatorNamespace,
// 				},
// 				Spec: &v1alpha1.IstioOperatorSpec{
// 					Profile: "empty",
// 					Hub:     istioOperatorImageHub,
// 					Values: map[string]interface{}{
// 						"global": map[string]interface{}{
// 							"meshID": "mesh-default",
// 							"multiCluster": map[string]string{
// 								"clusterName": cluster,
// 							},
// 							"network": "network-" + cluster,
// 						},
// 					},
// 				},
// 			})
// 			if err != nil {
// 				return nil, err
// 			} else {
// 				return op, nil
// 			}
// 		} else {
// 			return nil, err
// 		}
// 	}
// 	return ope, nil
// }

func (h *IstioGatewayHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/virtualspace/:virtualspace_id/cluster/:cluster_id/istiogateways", h.ListGateway)
	rg.GET("/virtualspace/:virtualspace_id/cluster/:cluster_id/istiogateways/:name", h.GetGateway)
	rg.POST("/virtualspace/:virtualspace_id/cluster/:cluster_id/istiogateways", h.CreateGateway)
	rg.PUT("/virtualspace/:virtualspace_id/cluster/:cluster_id/istiogateways", h.UpdateGateway)
	rg.DELETE("/virtualspace/:virtualspace_id/cluster/:cluster_id/istiogateways/:name", h.DeleteGateway)
}
