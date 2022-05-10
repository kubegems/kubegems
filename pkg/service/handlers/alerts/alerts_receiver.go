package alerthandler

import (
	"context"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
)

// @Tags         Alert
// @Summary      在namespace下获取receiver列表
// @Description  在namespace下获取receiver列表
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                                     true  "cluster"
// @Param        namespace  path      string                                                     true  "namespace"
// @Param        search     query     string                                                     true  "search"
// @Success      200        {object}  handlers.ResponseStruct{Data=[]prometheus.ReceiverConfig}  "resp"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver [get]
// @Security     JWT
func (h *AlertmanagerConfigHandler) ListReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	search := c.Query("search")

	ret := []prometheus.ReceiverConfig{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		var err error
		ret, err = cli.Extend().ListReceivers(ctx, namespace, prometheus.MonitorAlertmanagerConfigName, search)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// @Tags         Alert
// @Summary      在namespace下创建receiver
// @Description  在namespace下创建receiver
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        form       body      prometheus.ReceiverConfig             true  "body"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver [post]
// @Security     JWT
func (h *AlertmanagerConfigHandler) CreateReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	req := prometheus.ReceiverConfig{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "创建", "监控告警接收器", req.Name)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Extend().CreateReceiver(ctx, namespace, prometheus.MonitorAlertmanagerConfigName, req)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// @Tags         Alert
// @Summary      在namespace下修改receiver
// @Description  在namespace下修改receiver
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "name"
// @Param        form       body      prometheus.ReceiverConfig             true  "body"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver/{name} [put]
// @Security     JWT
func (h *AlertmanagerConfigHandler) UpdateReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	req := prometheus.ReceiverConfig{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "修改", "告警接收器", req.Name)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Extend().UpdateReceiver(ctx, namespace, prometheus.MonitorAlertmanagerConfigName, req)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// @Tags         Alert
// @Summary      在namespace下删除receiver
// @Description  在namespace下创建receiver
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "name"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver/{name} [delete]
// @Security     JWT
func (h *AlertmanagerConfigHandler) DeleteReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	h.SetAuditData(c, "删除", "监控告警接收器", name)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Extend().DeleteReceiver(ctx, namespace, name, prometheus.MonitorAlertmanagerConfigName)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// @Tags         Alert
// @Summary      发送测试邮件
// @Description  发送测试邮件
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "name"
// @Param        form       body      prometheus.EmailConfig                true  "body"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver/{name}/actions/test [post]
// @Security     JWT
func (h *AlertmanagerConfigHandler) TestEmail(c *gin.Context) {
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
