package metrics

import (
	"context"

	"github.com/gin-gonic/gin"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/pagination"
	"kubegems.io/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ListMetricTarget 采集器列表
// @Tags         Metrics
// @Summary      采集器列表
// @Description  采集器列表
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                                   true  "cluster"
// @Param        namespace  path      string                                                   true  "namespace"
// @Success      200        {object}  handlers.ResponseStruct{Data=[]prometheus.MetricTarget}  "resp"
// @Router       /v1/metrics/cluster/{cluster}/namespaces/{namespace}/targets [get]
// @Security     JWT
func (h *MonitorHandler) ListMetricTarget(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	if namespace == "_all" {
		namespace = corev1.NamespaceAll
	}
	ret := []*prometheus.MetricTarget{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, tc agents.Client) error {
		var err error
		ret, err = tc.Extend().ListMetricTargets(ctx, namespace)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, pagination.NewPageDataFromContextReflect(c, ret))
}

// AddOrUpdateMetricTarget 添加/更新采集器
// @Tags         Metrics
// @Summary      添加/更新采集器
// @Description  添加/更新采集器
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        form       body      prometheus.MetricTarget               true  "采集器内容"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/metrics/cluster/{cluster}/namespaces/{namespace}/targets [post]
// @Security     JWT
func (h *MonitorHandler) AddOrUpdateMetricTarget(c *gin.Context) {
	req := prometheus.MetricTarget{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
	}
	// 以url上的c为准
	req.Cluster = c.Param("cluster")
	req.Namespace = c.Param("namespace")

	if err := h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
		switch req.TargetType {
		case prometheus.MetricTargetService:
			sm := &v1.ServiceMonitor{
				ObjectMeta: req.GetMeta(),
			}
			_, err := controllerutil.CreateOrUpdate(ctx, cli, sm, prometheus.MutateServiceMonitorFunc(&req, sm))
			return err
		case prometheus.MetricTargetDeployment, prometheus.MetricTargetStatefulset, prometheus.MetricTargetDaemonset:
			pm := &v1.PodMonitor{
				ObjectMeta: req.GetMeta(),
			}
			_, err := controllerutil.CreateOrUpdate(ctx, cli, pm, prometheus.MutatePodMonitorFunc(&req, pm))
			return err
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

// DeleteMetricTarget 删除采集器
// @Tags         Metrics
// @Summary      删除采集器
// @Description  删除采集器
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "采集器名"
// @Param        type       query     string                                true  "采集器类型, service/deployment/statefulset/daemonset"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/metrics/cluster/{cluster}/namespaces/{namespace}/targets/{name} [delete]
// @Security     JWT
func (h *MonitorHandler) DeleteMetricTarget(c *gin.Context) {
	req := &prometheus.MetricTarget{
		Cluster:      c.Param("cluster"),
		Namespace:    c.Param("namespace"),
		Name:         c.Param("name"),
		TargetType:   c.Query("type"),
		TargetLabels: make(map[string]string), // 避免空指针
	}

	if err := h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
		obj, err := prometheus.ConvertToServiceOrPodMonitor(req)
		if err != nil {
			return err
		}
		return cli.Delete(ctx, obj)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}
