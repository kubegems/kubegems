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

package observability

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

func checkScope(scope string) error {
	if scope != prometheus.AlertTypeMonitor && scope != prometheus.AlertTypeLogging {
		return fmt.Errorf("scope must be one of logging/monitor")
	}
	return nil
}

// ListReceiver （日志/监控）告警接收器列表
// @Tags        Observability
// @Summary     （日志/监控）告警接收器列表
// @Description （日志/监控）告警接收器列表
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                                    true "cluster"
// @Param       namespace path     string                                                    true "namespace"
// @Param       scope     query    string                                                    true "接收器类型(monitor/logging)"
// @Param       search    query    string                                                    true "search"
// @Success     200       {object} handlers.ResponseStruct{Data=[]prometheus.ReceiverConfig} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/receivers [get]
// @Security    JWT
func (h *ObservabilityHandler) ListReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	search := c.Query("search")
	scope := c.Query("scope")
	ret := []prometheus.ReceiverConfig{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		err := checkScope(scope)
		if err != nil {
			return err
		}
		ret, err = cli.Extend().ListReceivers(ctx, namespace, scope, search)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// CreateReceiver 创建（日志/监控）告警接收器
// @Tags        Observability
// @Summary     创建（日志/监控）告警接收器
// @Description 创建（日志/监控）告警接收器
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       form      body     prometheus.ReceiverConfig            true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/receivers [post]
// @Security    JWT
func (h *ObservabilityHandler) CreateReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	req := prometheus.ReceiverConfig{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	req.Namespace = namespace
	h.SetAuditData(c, "创建", "告警接收器", req.Name)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Extend().CreateReceiver(ctx, req)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// UpdateReceiver 更新（日志/监控）告警接收器
// @Tags        Observability
// @Summary     更新（日志/监控）告警接收器
// @Description 更新（日志/监控）告警接收器
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       name      path     string                               true "name"
// @Param       form      body     prometheus.ReceiverConfig            true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/receivers/{name} [put]
// @Security    JWT
func (h *ObservabilityHandler) UpdateReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	req := prometheus.ReceiverConfig{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	req.Namespace = namespace
	h.SetAuditData(c, "更新", "告警接收器", req.Name)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Extend().UpdateReceiver(ctx, req)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteReceiver 删除（日志/监控）告警接收器
// @Tags        Observability
// @Summary     删除（日志/监控）告警接收器
// @Description 删除（日志/监控）告警接收器
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       source    query    string                               true "source"
// @Param       name      path     string                               true "name"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/receivers/{name} [delete]
// @Security    JWT
func (h *ObservabilityHandler) DeleteReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	source := c.Query("source")
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	h.SetAuditData(c, "删除", "监控告警接收器", name)

	h.m.Lock()
	defer h.m.Unlock()
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Extend().DeleteReceiver(ctx, prometheus.ReceiverConfig{
			Name:      name,
			Namespace: namespace,
			Source:    source,
		})
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// TestEmail 发送测试邮件
// @Tags        Observability
// @Summary     发送测试邮件
// @Description 发送测试邮件
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       form      body     prometheus.EmailConfig               true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/receivers/_/actions/test [post]
// @Security    JWT
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
