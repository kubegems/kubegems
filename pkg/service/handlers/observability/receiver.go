package observability

import (
	"context"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
)

func getAMConfigName(scope string) (ret string) {
	if scope == "logging" {
		ret = prometheus.LoggingAlertmanagerConfigName
	} else if scope == "monitor" {
		ret = prometheus.MonitorAlertmanagerConfigName
	}
	return
}

// ListReceiver （日志/监控）告警接收器列表
// @Tags         Observability
// @Summary      （日志/监控）告警接收器列表
// @Description  （日志/监控）告警接收器列表
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                                     true  "cluster"
// @Param        namespace  path      string                                                     true  "namespace"
// @Param        scope      query     string                                                     true  "接收器类型(monitior/logging)"
// @Param        search     query     string                                                     true  "search"
// @Success      200        {object}  handlers.ResponseStruct{Data=[]prometheus.ReceiverConfig}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/receivers [get]
// @Security     JWT
func (h *ObservabilityHandler) ListReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	search := c.Query("search")

	ret := []prometheus.ReceiverConfig{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		var err error
		ret, err = cli.Extend().ListReceivers(ctx, namespace, getAMConfigName(c.Query("scope")), search)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// CreateReceiver 创建（日志/监控）告警接收器
// @Tags         Observability
// @Summary      创建（日志/监控）告警接收器
// @Description  创建（日志/监控）告警接收器
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        scope      query     string                                true  "接收器类型(monitior/logging)"
// @Param        form       body      prometheus.ReceiverConfig             true  "body"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/receivers [post]
// @Security     JWT
func (h *ObservabilityHandler) CreateReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	req := prometheus.ReceiverConfig{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "创建", "告警接收器", req.Name)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Extend().CreateReceiver(ctx, namespace, getAMConfigName(c.Query("scope")), req)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// UpdateReceiver 更新（日志/监控）告警接收器
// @Tags         Observability
// @Summary      更新（日志/监控）告警接收器
// @Description  更新（日志/监控）告警接收器
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        scope      query     string                                true  "接收器类型(monitior/logging)"
// @Param        name       path      string                                true  "name"
// @Param        form       body      prometheus.ReceiverConfig             true  "body"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/receivers/{name} [put]
// @Security     JWT
func (h *ObservabilityHandler) UpdateReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	req := prometheus.ReceiverConfig{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "更新", "告警接收器", req.Name)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Extend().UpdateReceiver(ctx, namespace, getAMConfigName(c.Query("scope")), req)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteReceiver 删除（日志/监控）告警接收器
// @Tags         Observability
// @Summary      删除（日志/监控）告警接收器
// @Description  删除（日志/监控）告警接收器
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        scope      query     string                                true  "接收器类型(monitior/logging)"
// @Param        name       path      string                                true  "name"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/receivers/{name} [delete]
// @Security     JWT
func (h *ObservabilityHandler) DeleteReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	h.SetAuditData(c, "删除", "监控告警接收器", name)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Extend().DeleteReceiver(ctx, namespace, name, getAMConfigName(c.Query("scope")))
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// TestEmail 发送测试邮件
// @Tags         Observability
// @Summary      发送测试邮件
// @Description  发送测试邮件
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        form       body      prometheus.EmailConfig                true  "body"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/receivers/actions/test [post]
// @Security     JWT
func (h *ObservabilityHandler) TestEmail(c *gin.Context) {
	req := prometheus.EmailConfig{}
	c.BindJSON(&req)
	h.SetExtraAuditDataByClusterNamespace(c, c.Param("cluster"), c.Param("namespace"))
	h.SetAuditData(c, "测试", "告警接收器", "")

	if err := prometheus.TestEmail(req, c.Param("cluster"), c.Param("namespace")); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "邮件发送成功！")
}
