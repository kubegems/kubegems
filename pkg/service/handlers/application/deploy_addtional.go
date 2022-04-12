package application

import (
	"context"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/pointer"
	"kubegems.io/pkg/utils/agents"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RelatedService struct {
	ServiceIP    string               `json:"serviceIP"`
	Ports        []corev1.ServicePort `json:"ports"`
	IngressPorts []IngressPorts       `json:"ingresses"`
}

type IngressPorts struct {
	IngressClassName *string `json:"ingressClassName"`
	Host             string  `json:"host"`
	IngressPort      int     `json:"ingressPort"`
	ServicePort      int     `json:"servicePort"`
}

// @Tags         Application
// @Summary      获取对应的服务信息
// @Description  获取对应的服务信息
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                   true  "tenaut id"
// @Param        project_id      path      int                                   true  "project id"
// @param        environment_id  path      int                                           true  "environment id"
// @Param        name            path      string                                        true  "applicationname"
// @Success      200             {object}  handlers.ResponseStruct{Data=RelatedService}  "-"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/services [get]
// @Security     JWT
func (h *ApplicationHandler) ListRelatedService(c *gin.Context) {
	h.LocalAndRemoteCliFunc(c, nil,
		func(ctx context.Context, local GitStore, remote agents.Client, namespace string, _ PathRef) (interface{}, error) {
			// 从local拿到 service
			localsvcs := &corev1.ServiceList{}
			_ = local.List(ctx, localsvcs)

			if len(localsvcs.Items) == 0 {
				return RelatedService{}, nil
			}
			sort.Slice(localsvcs.Items, func(i, j int) bool {
				return len(localsvcs.Items[i].Name) < len(localsvcs.Items[j].Name)
			})

			// 选择一个svc: 选择名称最短的那个
			service := &localsvcs.Items[0]
			// 从 remote 填充这个 svc
			service.Namespace = namespace
			_ = remote.Get(ctx, client.ObjectKeyFromObject(service), service)

			// list ingress 选择backend为这个 svc 的 ingress
			ingressList := &extensionsv1beta1.IngressList{}
			_ = remote.List(ctx, ingressList, client.InNamespace(namespace))

			var ingresses []IngressPorts
			for _, ingress := range ingressList.Items {
				for _, rule := range ingress.Spec.Rules {
					for _, path := range rule.HTTP.Paths {
						if path.Backend.ServiceName == service.Name {
							ingresses = append(ingresses, IngressPorts{
								Host:             rule.Host,
								ServicePort:      path.Backend.ServicePort.IntValue(),
								IngressClassName: ingress.Spec.IngressClassName,
							})
						}
					}
				}
			}

			// convert
			ret := RelatedService{
				IngressPorts: ingresses,
				ServiceIP:    service.Spec.ClusterIP,
				Ports:        service.Spec.Ports,
			}
			return ret, nil
		}, "")
}

// @Tags         Application
// @Summary      应用编排中副本数scale(包含运行时调整)
// @Description  应用编排中副本数scale
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                           true  "tenaut id"
// @Param        project_id      path      int                                           true  "project id"
// @Param        environment_id  path      int                                   true  "environment_id"
// @Param        name            path      string                                true  "application name"
// @Param        name            query     string                                true  "workload name"
// @Param        body            body      AppReplicas                           true  "scale replicas,body优先"
// @Param        replicas        query     string                                true  "scale replicas，如果body不存在值则使用该值"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/replicas [post]
// @Security     JWT
func (h *ApplicationHandler) SetReplicas(c *gin.Context) {
	body := &AppReplicas{}
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		if body.Replicas == nil {
			replicas, _ := strconv.Atoi(c.Query("replicas"))
			body.Replicas = pointer.Int32(int32(replicas))
		}
		// scale
		if err := h.ApplicationProcessor.SetReplicas(ctx, ref, body.Replicas); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

type AppReplicas struct {
	Replicas *int32 `json:"replicas"`
}

// @Tags         Application
// @Summary      应用编排中副本数scale(包含运行时调整)
// @Description  应用编排中副本数scale
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                        true  "tenaut id"
// @Param        project_id      path      int                                        true  "project id"
// @Param        environment_id  path      int                                        true  "environment_id"
// @Param        name            path      string                                     true  "application name"
// @Param        name            query     string                                     true  "workload name"
// @Param        replicas        query     string                                     true  "scale replicas"
// @Success      200             {object}  handlers.ResponseStruct{Data=AppReplicas}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/replicas [get]
// @Security     JWT
func (h *ApplicationHandler) GetReplicas(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		replicas, err := h.ApplicationProcessor.GetReplicas(ctx, ref)
		if err != nil {
			return nil, err
		}
		return AppReplicas{Replicas: replicas}, nil
	})
}

type HPAMetrics struct {
	MinReplicas *int32 `json:"min_replicas" binding:"required,gte=1"`
	MaxReplicas int32  `json:"max_replicas" binding:"required"`
	Cpu         int32  `json:"cpu" binding:"lte=100"`
	Memory      int32  `json:"memory" binding:"lte=100"`
}

// @Tags         Application
// @Summary      应用编排中HPA
// @Description  应用编排中HPA
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                   true  "tenaut id"
// @Param        project_id      path      int                                   true  "project id"
// @Param        environment_id  path      int                                   true  "environment_id"
// @Param        name            path      string                                true  "application name"
// @Param        body            body      HPAMetrics                            true  "hpa metrics"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/hpa [post]
// @Security     JWT
func (h *ApplicationHandler) SetHPA(c *gin.Context) {
	body := &HPAMetrics{}
	h.NamedRefFunc(c, body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// hpa
		if err := h.ApplicationProcessor.SetHorizontalPodAutoscaler(ctx, ref, *body); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

// @Tags         Application
// @Summary      应用编排中HPA
// @Description  应用编排中HPA
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                       true  "tenaut id"
// @Param        project_id      path      int                                       true  "project id"
// @Param        environment_id  path      int                                       true  "environment_id"
// @Param        name            path      string                                    true  "application name"
// @Success      200             {object}  handlers.ResponseStruct{Data=HPAMetrics}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/hpa [get]
// @Security     JWT
func (h *ApplicationHandler) GetHPA(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// hpa
		sc, err := h.ApplicationProcessor.GetHorizontalPodAutoscaler(ctx, ref)
		if err != nil {
			if errors.IsNotFound(err) {
				return HPAMetrics{}, nil
			}
			return nil, err
		}
		ret := HPAMetrics{
			MinReplicas: sc.Spec.MinReplicas,
			MaxReplicas: sc.Spec.MaxReplicas,
		}
		for _, metrics := range sc.Spec.Metrics {
			if metrics.Resource.Name == v1.ResourceCPU {
				ret.Cpu = *metrics.Resource.TargetAverageUtilization
			}
			if metrics.Resource.Name == v1.ResourceMemory {
				ret.Memory = *metrics.Resource.TargetAverageUtilization
			}
		}
		return ret, nil
	})
}

// @Tags         Application
// @Summary      应用编排HPA
// @Description  应用编排中HPA
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                       true  "tenaut id"
// @Param        project_id      path      int                                       true  "project id"
// @Param        environment_id  path      int                                       true  "environment_id"
// @Param        name            path      string                                    true  "application name"
// @Success      200             {object}  handlers.ResponseStruct{Data=HPAMetrics}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/hpa [delete]
// @Security     JWT
func (h *ApplicationHandler) DeleteHPA(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// hpa
		if err := h.ApplicationProcessor.DeleteHorizontalPodAutoscaler(ctx, ref); err != nil {
			return nil, err
		}
		// sync
		if err := h.ApplicationProcessor.Sync(ctx, ref); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}
