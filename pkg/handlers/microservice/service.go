package microservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
	kialimodels "github.com/kiali/kiali/models"
	"github.com/kubegems/gems/pkg/handlers"
	"github.com/kubegems/gems/pkg/kubeclient"
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/utils/agents"
	"github.com/kubegems/gems/pkg/utils/istio"
	"github.com/kubegems/gems/pkg/utils/pagination"
	"golang.org/x/sync/errgroup"
	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	networkingpkgv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ pagination.SortAndSearchAble = ServiceDetail{}

type ServiceDetail struct {
	Service          v1.Service                              `json:"service"`
	DestinationRules []networkingpkgv1alpha3.DestinationRule `json:"destinationRules"`
	VirtualServices  []networkingpkgv1alpha3.VirtualService  `json:"virtualServices"`
	Workloads        kialimodels.WorkloadOverviews           `json:"workloads"`
}

func (s ServiceDetail) GetName() string {
	return s.Service.GetName()
}

func (s ServiceDetail) GetCreationTimestamp() metav1.Time {
	return s.Service.GetCreationTimestamp()
}

// ListServices service列表
// @Tags VirtualSpace
// @Summary service列表
// @Description service列表
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param search query string true "service名称"
// @Success 200 {object} handlers.ResponseStruct{Data=pagination.PageData{List=[]string}} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/service [get]
// @Security JWT
func (h *VirtualSpaceHandler) ListServices(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		svcList := v1.ServiceList{}
		destionationRuleList := networkingpkgv1alpha3.DestinationRuleList{}
		virtualServiceList := networkingpkgv1alpha3.VirtualServiceList{}
		gatewayList := networkingpkgv1alpha3.GatewayList{}
		podList := v1.PodList{}
		deploymentsList := appsv1.DeploymentList{}
		if err := kubeclient.Execute(ctx, env.Cluster.ClusterName, func(tc *agents.TypedClient) error {
			g := new(errgroup.Group)
			g.Go(func() error {
				return tc.List(ctx, &svcList, client.InNamespace(env.Namespace))
			})
			g.Go(func() error {
				return tc.List(ctx, &destionationRuleList, client.InNamespace(env.Namespace))
			})
			g.Go(func() error {
				return tc.List(ctx, &virtualServiceList, client.InNamespace(env.Namespace))
			})
			g.Go(func() error {
				return tc.List(ctx, &gatewayList, client.InNamespace(env.Namespace))
			})
			g.Go(func() error {
				return tc.List(ctx, &podList, client.InNamespace(env.Namespace))
			})
			g.Go(func() error {
				return tc.List(ctx, &deploymentsList, client.InNamespace(env.Namespace))
			})
			return g.Wait()
		}); err != nil {
			return nil, err
		}
		istioConfigList := kialimodels.IstioConfigList{
			DestinationRules: destionationRuleList.Items,
			VirtualServices:  virtualServiceList.Items,
			Gateways:         gatewayList.Items,
		}
		sl := istio.BuildKubernetesServiceList(kialimodels.Namespace{Name: env.Namespace}, svcList.Items, podList.Items, deploymentsList.Items, istioConfigList)
		sl.Validations = istio.GetServiceValidations(svcList.Items, deploymentsList.Items, podList.Items)
		ret := handlers.NewPageDataFromContext(c, sl.Services, func(i int) bool {
			return strings.Contains(strings.ToLower(sl.Services[i].Name), strings.ToLower(c.Query("search")))
		}, func(i, j int) bool {
			return strings.ToLower(sl.Services[i].Name) < strings.ToLower(sl.Services[j].Name)
		})
		return gin.H{
			"pagedata":    ret,
			"validations": sl.Validations,
		}, nil
	})
}

// GetService service详情
// @Tags VirtualSpace
// @Summary service详情
// @Description service详情
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param service_name path string true "service_name"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/service/{service_name} [get]
// @Security JWT
func (h *VirtualSpaceHandler) GetService(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		return getServiceDetails(c.Request.Context(), env.Cluster.ClusterName, env.Namespace, c.Param("service_name"))
	})
}

const (
	// 兼容kiali
	kiali_wizard_request_routing      = "request_routing"
	kiali_wizard_fault_injection      = "fault_injection"
	kiali_wizard_traffic_shifting     = "traffic_shifting"
	kiali_wizard_tcp_traffic_shifting = "tcp_traffic_shifting"
	kiali_wizard_request_timeouts     = "request_timeouts"
)

// 给前端看，实际不用
type HTTPRoute struct {
	Match   []*networkingv1alpha3.HTTPMatchRequest     `protobuf:"bytes,1,rep,name=match,proto3" json:"match,omitempty"`
	Route   []*networkingv1alpha3.HTTPRouteDestination `protobuf:"bytes,2,rep,name=route,proto3" json:"route,omitempty"`
	Timeout *ptypes.Duration                           `protobuf:"bytes,6,opt,name=timeout,proto3" json:"timeout,omitempty"`
	Retries *networkingv1alpha3.HTTPRetry              `protobuf:"bytes,7,opt,name=retries,proto3" json:"retries,omitempty"`
	Fault   *networkingv1alpha3.HTTPFaultInjection     `protobuf:"bytes,8,opt,name=fault,proto3" json:"fault,omitempty"`
}

// ServiceRequestRouting service请求路由
// @Tags VirtualSpace
// @Summary service请求路由
// @Description service请求路由
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param service_name path string true "service_name"
// @Param param body []HTTPRoute true "请求路由"
// @Success 200 {object} handlers.ResponseStruct{Data=ServiceDetail} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/service/{service_name}/request_routing [post]
// @Security JWT
func (h *VirtualSpaceHandler) ServiceRequestRouting(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		svcDetails, err := getServiceDetails(c.Request.Context(), env.Cluster.ClusterName, env.Namespace, c.Param("service_name"))
		if err != nil {
			return nil, err
		}
		req := []*networkingv1alpha3.HTTPRoute{}
		if err := c.BindJSON(&req); err != nil {
			return nil, err
		}

		return "ok", kubeclient.Execute(ctx, env.Cluster.ClusterName, func(tc *agents.TypedClient) error {
			return updateOrCreateDrAndVs(svcDetails,
				kiali_wizard_request_routing,
				ctx,
				tc,
				func(vs *networkingpkgv1alpha3.VirtualService) {
					vs.Spec.Http = req
				},
			)
		})
	})
}

// ServiceFaultInjection service故障注入
// @Tags VirtualSpace
// @Summary service故障注入
// @Description service故障注入
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param service_name path string true "service_name"
// @Param param body HTTPRoute true "故障注入"
// @Success 200 {object} handlers.ResponseStruct{Data=ServiceDetail} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/service/{service_name}/fault_injection [post]
// @Security JWT
func (h *VirtualSpaceHandler) ServiceFaultInjection(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		svcDetails, err := getServiceDetails(c.Request.Context(), env.Cluster.ClusterName, env.Namespace, c.Param("service_name"))
		if err != nil {
			return nil, err
		}
		req := networkingv1alpha3.HTTPRoute{}
		if err := c.BindJSON(&req); err != nil {
			return nil, err
		}
		if req.Route == nil {
			req.Route = []*networkingv1alpha3.HTTPRouteDestination{
				{
					Destination: &networkingv1alpha3.Destination{
						Host: constructHostFQDN(svcDetails.Service.Namespace.Name, svcDetails.Service.Name),
					},
					Weight: 100,
				},
			}
		}
		return "ok", kubeclient.Execute(ctx, env.Cluster.ClusterName, func(tc *agents.TypedClient) error {
			return updateOrCreateDrAndVs(svcDetails,
				kiali_wizard_fault_injection,
				ctx,
				tc,
				func(vs *networkingpkgv1alpha3.VirtualService) {
					vs.Spec.Http = append(vs.Spec.Http, &req)
				},
			)
		})
	})
}

// ServiceTrafficShifting service流量切换
// @Tags VirtualSpace
// @Summary service流量切换
// @Description service流量切换
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param service_name path string true "service_name"
// @Param param body HTTPRoute true "流量切换"
// @Success 200 {object} handlers.ResponseStruct{Data=ServiceDetail} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/service/{service_name}/traffic_shifting  [post]
// @Security JWT
func (h *VirtualSpaceHandler) ServiceTrafficShifting(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		svcDetails, err := getServiceDetails(c.Request.Context(), env.Cluster.ClusterName, env.Namespace, c.Param("service_name"))
		if err != nil {
			return nil, err
		}
		req := networkingv1alpha3.HTTPRoute{}
		if err := c.BindJSON(&req); err != nil {
			return nil, err
		}

		return "ok", kubeclient.Execute(ctx, env.Cluster.ClusterName, func(tc *agents.TypedClient) error {
			return updateOrCreateDrAndVs(svcDetails,
				kiali_wizard_traffic_shifting,
				ctx,
				tc,
				func(vs *networkingpkgv1alpha3.VirtualService) {
					vs.Spec.Http = append(vs.Spec.Http, &req)
				},
			)
		})
	})
}

type TCPRoute struct {
	// Match conditions to be satisfied for the rule to be
	// activated. All conditions inside a single match block have AND
	// semantics, while the list of match blocks have OR semantics. The rule
	// is matched if any one of the match blocks succeed.
	Match []*networkingv1alpha3.L4MatchAttributes `protobuf:"bytes,1,rep,name=match,proto3" json:"match,omitempty"`
	// The destination to which the connection should be forwarded to.
	Route []*networkingv1alpha3.RouteDestination `protobuf:"bytes,2,rep,name=route,proto3" json:"route,omitempty"`
}

// ServiceTCPTrafficShifting service tcp流量切换
// @Tags VirtualSpace
// @Summary service tcp流量切换
// @Description service tcp流量切换
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param service_name path string true "service_name"
// @Param param body TCPRoute true "tcp流量切换"
// @Success 200 {object} handlers.ResponseStruct{Data=ServiceDetail} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/service/{service_name}/tcp_traffic_shifting  [post]
// @Security JWT
func (h *VirtualSpaceHandler) ServiceTCPTrafficShifting(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		svcDetails, err := getServiceDetails(c.Request.Context(), env.Cluster.ClusterName, env.Namespace, c.Param("service_name"))
		if err != nil {
			return nil, err
		}
		req := networkingv1alpha3.TCPRoute{}
		if err := c.BindJSON(&req); err != nil {
			return nil, err
		}

		return "ok", kubeclient.Execute(ctx, env.Cluster.ClusterName, func(tc *agents.TypedClient) error {
			return updateOrCreateDrAndVs(svcDetails,
				kiali_wizard_tcp_traffic_shifting,
				ctx,
				tc,
				func(vs *networkingpkgv1alpha3.VirtualService) {
					vs.Spec.Tcp = append(vs.Spec.Tcp, &req)
				},
			)
		})
	})
}

// ServiceRequestTimeout service超时配置
// @Tags VirtualSpace
// @Summary service超时配置
// @Description service超时配置
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param service_name path string true "service_name"
// @Param param body HTTPRoute true "超时设置"
// @Success 200 {object} handlers.ResponseStruct{Data=ServiceDetail} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/service/{service_name}/request_timeouts  [post]
// @Security JWT
func (h *VirtualSpaceHandler) ServiceRequestTimeout(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		svcDetails, err := getServiceDetails(c.Request.Context(), env.Cluster.ClusterName, env.Namespace, c.Param("service_name"))
		if err != nil {
			return nil, err
		}
		req := networkingv1alpha3.HTTPRoute{}
		if err := c.BindJSON(&req); err != nil {
			return nil, err
		}
		if req.Route == nil {
			req.Route = []*networkingv1alpha3.HTTPRouteDestination{
				{
					Destination: &networkingv1alpha3.Destination{
						Host: constructHostFQDN(svcDetails.Service.Namespace.Name, svcDetails.Service.Name),
					},
					Weight: 100,
				},
			}
		}
		return "ok", kubeclient.Execute(ctx, env.Cluster.ClusterName, func(tc *agents.TypedClient) error {
			return updateOrCreateDrAndVs(svcDetails,
				kiali_wizard_request_timeouts,
				ctx,
				tc,
				func(vs *networkingpkgv1alpha3.VirtualService) {
					vs.Spec.Http = append(vs.Spec.Http, &req)
				},
			)
		})
	})
}

// ServicetReset service重置
// @Tags VirtualSpace
// @Summary service重置
// @Description service重置
// @Accept json
// @Produce json
// @Param virtualspace_id path uint true "virtualspace_id"
// @Param environment_id path uint true "environment_id"
// @Param service_name path string true "service_name"
// @Success 200 {object} handlers.ResponseStruct{Data=ServiceDetail} "resp"
// @Router /v1/virtualspace/{virtualspace_id}/environment/environment_id/service/{service_name}/reset [post]
// @Security JWT
func (h *VirtualSpaceHandler) ServicetReset(c *gin.Context) {
	h.environmentProcess(c, nil, func(ctx context.Context, env models.Environment) (interface{}, error) {
		svcDetails, err := getServiceDetails(c.Request.Context(), env.Cluster.ClusterName, env.Namespace, c.Param("service_name"))
		if err != nil {
			return nil, err
		}
		return "ok", kubeclient.Execute(ctx, env.Cluster.ClusterName, func(tc *agents.TypedClient) error {
			for _, v := range svcDetails.VirtualServices {
				if err := tc.Delete(ctx, &v); err != nil {
					return err
				}
			}
			for _, v := range svcDetails.DestinationRules {
				if err := tc.Delete(ctx, &v); err != nil {
					return err
				}
			}
			return nil
		})
	})
}

func updateOrCreateDrAndVs(
	svcDetail kialimodels.ServiceDetails,
	kiali_wizard_value string,
	ctx context.Context,
	tc *agents.TypedClient,
	mutateVirtualService func(vs *networkingpkgv1alpha3.VirtualService),
) error {
	dr := &networkingpkgv1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcDetail.Service.Name,
			Namespace: svcDetail.Service.Namespace.Name,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, tc, dr, func() error {
		dr.ObjectMeta.Labels = map[string]string{
			"kiali_wizard": kiali_wizard_value,
		}
		dr.Spec = networkingv1alpha3.DestinationRule{
			Host: constructHostFQDN(svcDetail.Service.Namespace.Name, svcDetail.Service.Name),
		}
		conf := config.Get()
		for _, w := range svcDetail.Workloads {
			if w.IstioSidecar && w.AppLabel && w.VersionLabel {
				dr.Spec.Subsets = append(dr.Spec.Subsets, &networkingv1alpha3.Subset{
					Name: w.Labels[conf.IstioLabels.VersionLabelName],
					Labels: map[string]string{
						conf.IstioLabels.VersionLabelName: w.Labels[conf.IstioLabels.VersionLabelName],
					},
				})
			}
		}
		return nil
	}); err != nil {
		return err
	}

	vs := &networkingpkgv1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcDetail.Service.Name,
			Namespace: svcDetail.Service.Namespace.Name,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, tc, vs, func() error {
		vs.ObjectMeta.Labels = map[string]string{
			"kiali_wizard": kiali_wizard_value,
		}
		vs.Spec = networkingv1alpha3.VirtualService{
			Hosts: []string{
				constructHostFQDN(svcDetail.Service.Namespace.Name, svcDetail.Service.Name),
			},
		}
		mutateVirtualService(vs)
		for _, v := range vs.Spec.Http {
			for _, r := range v.Route {
				if err := checkFQDN(r.Destination.GetHost()); err != nil {
					return err
				}
			}
		}
		for _, v := range vs.Spec.Tcp {
			for _, r := range v.Route {
				if err := checkFQDN(r.Destination.GetHost()); err != nil {
					return err
				}
			}
		}
		return nil
	})

	return err
}

func checkFQDN(host string) error {
	conf := config.Get()
	if !strings.HasSuffix(host, conf.ExternalServices.Istio.IstioIdentityDomain) {
		return fmt.Errorf("host: %s is not vaild FQDN, should end with: %s", host, conf.ExternalServices.Istio.IstioIdentityDomain)
	}
	return nil
}

func constructHostFQDN(namespace, name string) string {
	// FQDN. <service>.<namespace>.svc.<zone>
	return fmt.Sprintf("%s.%s.%s", name, namespace, config.Get().ExternalServices.Istio.IstioIdentityDomain)
}

func getServiceDetails(ctx context.Context, cluster, namespace, name string) (kialimodels.ServiceDetails, error) {
	svc := v1.Service{}
	ep := v1.Endpoints{}
	destionationRuleList := networkingpkgv1alpha3.DestinationRuleList{}
	virtualServiceList := networkingpkgv1alpha3.VirtualServiceList{}
	rsList := appsv1.ReplicaSetList{}
	depList := appsv1.DeploymentList{}
	stsList := appsv1.StatefulSetList{}
	dsList := appsv1.DaemonSetList{}
	podList := v1.PodList{}
	kialisvc := kialimodels.ServiceDetails{}
	if err := kubeclient.Execute(ctx, cluster, func(tc *agents.TypedClient) error {
		g := new(errgroup.Group)
		g.Go(func() error {
			return tc.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &svc)
		})
		g.Go(func() error {
			return tc.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &ep)
		})
		g.Go(func() error {
			return tc.List(ctx, &destionationRuleList, client.InNamespace(namespace))
		})
		g.Go(func() error {
			return tc.List(ctx, &virtualServiceList, client.InNamespace(namespace))
		})
		g.Go(func() error {
			return tc.List(ctx, &rsList, client.InNamespace(namespace))
		})
		g.Go(func() error {
			return tc.List(ctx, &depList, client.InNamespace(namespace))
		})
		g.Go(func() error {
			return tc.List(ctx, &stsList, client.InNamespace(namespace))
		})
		g.Go(func() error {
			return tc.List(ctx, &dsList, client.InNamespace(namespace))
		})
		if err := g.Wait(); err != nil {
			return err
		}

		return tc.List(ctx, &podList, client.InNamespace(namespace), client.MatchingLabels(svc.Spec.Selector))
	}); err != nil {
		return kialisvc, err
	}

	ws, err := istio.FetchWorkloads(podList.Items, rsList.Items, depList.Items, stsList.Items, dsList.Items, svc.Namespace, labels.SelectorFromSet(svc.Spec.Selector).String())
	if err != nil {
		return kialisvc, fmt.Errorf("FetchWorkloads: %w", err)
	}
	wo := kialimodels.WorkloadOverviews{}
	for _, w := range ws {
		wo = append(wo, istio.ParseWorkload(w))
	}

	s := kialimodels.ServiceDetails{
		Service:          kialisvc.Service,
		Workloads:        wo,
		VirtualServices:  kubernetes.FilterVirtualServices(virtualServiceList.Items, namespace, svc.Name),
		DestinationRules: kubernetes.FilterDestinationRules(destionationRuleList.Items, namespace, svc.Name),
	}
	s.SetService(&svc)
	s.SetPods(kubernetes.FilterPodsForEndpoints(&ep, podList.Items))
	s.SetIstioSidecar(wo)
	s.SetEndpoints(&ep)
	return s, nil
}
