package observability

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type MonitorCollector struct {
	Service string `json:"service"` // 服务名
	Port    string `json:"port"`    // 端口名
	Path    string `json:"path"`    // 采集路径
}

// GetMonitorCollector 监控采集器详情
// @Tags         Observability
// @Summary      监控采集器详情
// @Description  监控采集器详情
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                          true  "cluster"
// @Param        namespace  path      string                                          true  "namespace"
// @Param        service    query     string                                          true  "服务名"
// @Success      200        {object}  handlers.ResponseStruct{Data=MonitorCollector}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor [get]
// @Security     JWT
func (h *ObservabilityHandler) GetMonitorCollector(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	svcname := c.Query("service")

	ret := MonitorCollector{
		Service: svcname,
	}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		sm := v1.ServiceMonitor{}
		if err := cli.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      svcname,
		}, &sm); err != nil {
			return err
		}
		svc := corev1.Service{}
		if err := cli.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      svcname,
		}, &svc); err != nil {
			return err
		}

		if !labels.SelectorFromSet(sm.Spec.Selector.MatchLabels).Matches(labels.Set(svc.Labels)) {
			log.Warnf("selector on servicemonitor %s doesn't match service labels", svcname)
			return nil
		}

		if len(sm.Spec.Endpoints) == 0 {
			log.Warnf("endpoint on servicemonitor %s is null", svcname)
			return nil
		}
		ret.Port = sm.Spec.Endpoints[0].Port
		ret.Path = sm.Spec.Endpoints[0].Path
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// AddOrUpdateMonitorCollector 添加/更新监控采集器
// @Tags         Observability
// @Summary      添加/更新监控采集器
// @Description  添加/更新监控采集器
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                                     true  "cluster"
// @Param        namespace  path      string                                                     true  "namespace"
// @Param        form       body      MonitorCollector                      true  "采集器内容"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor [post]
// @Security     JWT
func (h *ObservabilityHandler) AddOrUpdateMonitorCollector(c *gin.Context) {
	req := MonitorCollector{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	// 以url上的c为准
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	h.SetAuditData(c, "创建", "监控采集器", req.Service)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		svc := corev1.Service{}
		if err := cli.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      req.Service,
		}, &svc); err != nil {
			return err
		}

		found := false
		for _, v := range svc.Spec.Ports {
			if v.Name == req.Port {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("port %s not found in service %s", req.Port, svc.Name)
		}

		sm := v1.ServiceMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      req.Service,
			},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, cli, &sm, func() error {
			sm.Spec = v1.ServiceMonitorSpec{
				Selector: *metav1.SetAsLabelSelector(svc.Labels),
				NamespaceSelector: v1.NamespaceSelector{
					Any:        false,
					MatchNames: []string{namespace},
				},
				Endpoints: []v1.Endpoint{{
					Port:        req.Port,
					HonorLabels: true,
					Interval:    "30s",
					Path:        req.Path,
				}},
			}
			return nil
		})
		if err != nil {
			return err
		}

		if svc.Labels == nil {
			svc.Labels = make(map[string]string)
		}
		svc.Labels[gems.LabelMonitorCollector] = gems.StatusEnabled
		return cli.Update(ctx, &svc)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

// DeleteMonitorCollector 删除监控采集器
// @Tags         Observability
// @Summary      删除监控采集器
// @Description  删除监控采集器
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        service    query     string                                true  "服务名"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor [delete]
// @Security     JWT
func (h *ObservabilityHandler) DeleteMonitorCollector(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	svcname := c.Query("service")

	h.SetAuditData(c, "删除", "监控采集器", svcname)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		svc := corev1.Service{}
		if err := cli.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      svcname,
		}, &svc); err != nil {
			return err
		}

		if err := cli.Delete(ctx, &v1.ServiceMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      svcname,
			},
		}); err != nil {
			return err
		}
		delete(svc.Labels, gems.LabelMonitorCollector)
		return cli.Update(ctx, &svc)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

// ListMonitorAlertRule 监控告警规则列表
// @Tags         Observability
// @Summary      监控告警规则列表
// @Description  监控告警规则列表
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                                       true  "cluster"
// @Param        namespace  path      string                                                       true  "namespace"
// @Success      200        {object}  handlers.ResponseStruct{Data=[]prometheus.MonitorAlertRule}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts [get]
// @Security     JWT
func (h *ObservabilityHandler) ListMonitorAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	ret := []prometheus.MonitorAlertRule{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		monitoropts := new(prometheus.MonitorOptions)
		h.DynamicConfig.Get(ctx, monitoropts)
		var err error
		ret, err = cli.Extend().ListMonitorAlertRules(ctx, namespace, monitoropts, false)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// GetMonitorAlertRule 监控告警规则详情
// @Tags         Observability
// @Summary      监控告警规则详情
// @Description  监控告警规则详情
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                                     true  "name"
// @Success      200        {object}  handlers.ResponseStruct{Data=prometheus.MonitorAlertRule}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts/{name} [get]
// @Security     JWT
func (h *ObservabilityHandler) GetMonitorAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")

	var alerts []prometheus.MonitorAlertRule
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		monitoropts := new(prometheus.MonitorOptions)
		h.DynamicConfig.Get(ctx, monitoropts)
		var err error
		alerts, err = cli.Extend().ListMonitorAlertRules(ctx, namespace, monitoropts, true)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	index := -1
	for i := range alerts {
		if alerts[i].Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		handlers.NotOK(c, fmt.Errorf("alert %s not found", name))
	}
	handlers.OK(c, alerts[index])
}

// CreateMonitorAlertRule 创建监控告警规则
// @Tags         Observability
// @Summary      创建监控告警规则
// @Description  创建监控告警规则
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        form       body      prometheus.MonitorAlertRule           true  "body"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts [post]
// @Security     JWT
func (h *ObservabilityHandler) CreateMonitorAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	req := prometheus.MonitorAlertRule{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	req.Namespace = namespace
	h.SetAuditData(c, "创建", "监控告警规则", req.Name)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		monitoropts := new(prometheus.MonitorOptions)
		h.DynamicConfig.Get(ctx, monitoropts)
		if err := req.CheckAndModify(monitoropts); err != nil {
			return err
		}

		// get、update、commit
		raw, err := cli.Extend().GetRawMonitorAlertResource(ctx, namespace, monitoropts)
		if err != nil {
			return err
		}

		// check name duplicated in log alert
		logAMConfig, err := cli.Extend().GetOrCreateAlertmanagerConfig(ctx, namespace, prometheus.LoggingAlertmanagerConfigName)
		if err != nil {
			return err
		}
		if err := prometheus.CheckAlertNameInAMConfig(req.Name, logAMConfig, "日志"); err != nil {
			return err
		}

		if err := raw.ModifyAlertRule(req, prometheus.Add); err != nil {
			return err
		}

		return cli.Extend().CommitRawMonitorAlertResource(ctx, raw)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// UpdateMonitorAlertRule 修改监控告警规则
// @Tags         Observability
// @Summary      修改监控告警规则
// @Description  修改监控告警规则
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "name"
// @Param        form       body      prometheus.MonitorAlertRule           true  "body"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts/{name} [put]
// @Security     JWT
func (h *ObservabilityHandler) UpdateMonitorAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	req := prometheus.MonitorAlertRule{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	req.Namespace = namespace
	h.SetAuditData(c, "更新", "告警规则", req.Name)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		monitoropts := new(prometheus.MonitorOptions)
		h.DynamicConfig.Get(ctx, monitoropts)
		if err := req.CheckAndModify(monitoropts); err != nil {
			return err
		}

		// get、update、commit
		raw, err := cli.Extend().GetRawMonitorAlertResource(ctx, namespace, monitoropts)
		if err != nil {
			return err
		}

		if err := raw.ModifyAlertRule(req, prometheus.Update); err != nil {
			return err
		}

		return cli.Extend().CommitRawMonitorAlertResource(ctx, raw)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteMonitorAlertRule 删除AlertRule
// @Tags         Observability
// @Summary      修改监控告警规则
// @Description  修改监控告警规则
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "name"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/alerts/{name} [delete]
// @Security     JWT
func (h *ObservabilityHandler) DeleteMonitorAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	req := prometheus.MonitorAlertRule{
		BaseAlertRule: &prometheus.BaseAlertRule{
			Namespace: namespace,
			Name:      name,
		},
	}

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		// get、update、commit
		monitoropts := new(prometheus.MonitorOptions)
		h.DynamicConfig.Get(ctx, monitoropts)
		raw, err := cli.Extend().GetRawMonitorAlertResource(ctx, namespace, monitoropts)
		if err != nil {
			return err
		}

		if err := raw.ModifyAlertRule(req, prometheus.Delete); err != nil {
			return err
		}

		if err := cli.Extend().CommitRawMonitorAlertResource(ctx, raw); err != nil {
			return err
		}

		// 清理silence规则
		return deleteSilenceIfExist(ctx, namespace, name, cli)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}
