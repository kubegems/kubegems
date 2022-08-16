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
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
	v1 "k8s.io/api/core/v1"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/promql"
	"sigs.k8s.io/yaml"
)

type MetricQueryReq struct {
	// 查询范围
	Cluster   string
	Namespace string
	// EnvironmentID string `json:"environment_id"` // 可获取Cluster、namespace信息

	// 查询目标
	*prometheus.PromqlGenerator
	Expr string // 不传则自动生成

	Query *promql.Query

	// 时间
	Start string // 开始时间
	End   string // 结束时间
	Step  string // 样本间隔, 单位秒

	Label string // 要查询的标签值
}

// Query 监控指标查询
// @Tags        Observability
// @Summary     监控指标查询
// @Description 监控指标查询
// @Accept      json
// @Produce     json
// @Param       cluster    path     string                               true  "集群名"
// @Param       namespace  path     string                               true  "命名空间，所有namespace为_all"
// @Param       resource   query    string                               false "查询资源"
// @Param       rule       query    string                               false "查询规则"
// @Param       unit       query    string                               false "单位"
// @Param       labelpairs query    string                               false "标签键值对(value为空或者_all表示所有，支持正则),  eg.  labelpairs[host]=k8s-master&labelpairs[pod]=_all"
// @Param       expr       query    string                               false "promql表达式"
// @Param       start      query    string                               false "开始时间，默认现在-30m"
// @Param       end        query    string                               false "结束时间，默认现在"
// @Param       step       query    int                                  false "step, 单位秒，默认0"
// @Success     200        {object} handlers.ResponseStruct{Data=object} "Metrics配置"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/metrics/queryrange [get]
// @Security    JWT
func (h *ObservabilityHandler) QueryRange(c *gin.Context) {
	ret := prommodel.Matrix{}
	if err := h.withQueryParam(c, func(req *MetricQueryReq) error {
		return h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
			var err error
			ret, err = cli.Extend().PrometheusQueryRange(ctx, req.Expr, req.Start, req.End, req.Step)
			return err
		})
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// Query 监控标签值
// @Tags        Observability
// @Summary     监控标签值
// @Description 查询label对应的标签值
// @Accept      json
// @Produce     json
// @Param       label      query    string                                 true  "要查询的标签"
// @Param       cluster    path     string                                 true  "集群名"
// @Param       namespace  path     string                                 true  "命名空间，所有namespace为_all"
// @Param       resource   query    string                                 false "查询资源"
// @Param       rule       query    string                                 false "查询规则"
// @Param       unit       query    string                                 false "单位"
// @Param       labelpairs query    string                                 false "标签键值对(value为空或者_all表示所有，支持正则),  eg.  labelpairs[host]=k8s-master&labelpairs[pod]=_all"
// @Param       expr       query    string                                 false "promql表达式"
// @Param       start      query    string                                 false "开始时间，默认现在-30m"
// @Param       end        query    string                                 false "结束时间，默认现在"
// @Param       step       query    int                                    false "step, 单位秒，默认0"
// @Success     200        {object} handlers.ResponseStruct{Data=[]string} "Metrics配置"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/metrics/labelvalues [get]
// @Security    JWT
func (h *ObservabilityHandler) LabelValues(c *gin.Context) {
	ret := []string{}
	if err := h.withQueryParam(c, func(req *MetricQueryReq) error {
		if err := h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
			var err error
			matchs := req.Query.GetVectorSelectors()
			ret, err = cli.Extend().GetPrometheusLabelValues(ctx, matchs, req.Label, req.Start, req.End)
			return err
		}); err != nil {
			return fmt.Errorf("prometheus label values failed, cluster: %s, promql: %s, %w", req.Cluster, req.Expr, err)
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// LabelNames 查群prometheus label names
// @Tags        Observability
// @Summary     查群prometheus label names
// @Description 查群prometheus label names
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                 true  "集群名"
// @Param       namespace path     string                                 true  "命名空间，所有namespace为_all"
// @Param       resource  query    string                                 false "查询资源"
// @Param       rule      query    string                                 false "查询规则"
// @Param       start     query    string                                 false "开始时间，默认现在-30m"
// @Param       end       query    string                                 false "结束时间，默认现在"
// @Param       expr      query    string                                 true  "promql表达式"
// @Success     200       {object} handlers.ResponseStruct{Data=[]string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/metrics/labelnames [get]
// @Security    JWT
func (h *ObservabilityHandler) LabelNames(c *gin.Context) {
	ret := []string{}
	if err := h.withQueryParam(c, func(req *MetricQueryReq) error {
		if err := h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
			var err error
			matchs := req.Query.GetVectorSelectors()
			ret, err = cli.Extend().GetPrometheusLabelNames(ctx, matchs, req.Start, req.End)
			return err
		}); err != nil {
			return fmt.Errorf("prometheus label names failed, cluster: %s, promql: %s, %w", req.Cluster, req.Expr, err)
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

func (h *ObservabilityHandler) withQueryParam(c *gin.Context, f func(req *MetricQueryReq) error) error {
	q := &MetricQueryReq{
		Cluster:   c.Param("cluster"),
		Namespace: c.Param("namespace"),
		Start:     c.Query("start"),
		End:       c.Query("end"),
		Step:      c.Query("step"),
		Expr:      c.Query("expr"),
		Label:     c.Query("label"),

		PromqlGenerator: &prometheus.PromqlGenerator{
			Scope:      c.Query("scope"),
			Resource:   c.Query("resource"),
			Rule:       c.Query("rule"),
			LabelPairs: c.QueryMap("labelpairs"),
		},
	}
	if q.Namespace == "_all" {
		q.Namespace = ""
	}

	now := time.Now().UTC()
	// 默认查近30m
	if q.Start == "" || q.End == "" {
		q.Start = now.Add(-30 * time.Minute).Format(time.RFC3339)
		q.End = now.Format(time.RFC3339)
	}

	// 优先选用原生promql
	if q.PromqlGenerator.Notpl() {
		if q.Expr == "" {
			return fmt.Errorf("模板与原生promql不能同时为空")
		}
		if err := prometheus.CheckQueryExprNamespace(q.Expr, q.Namespace); err != nil {
			return err
		}
	} else {
		if err := q.PromqlGenerator.SetTpl(h.GetDataBase().FindPromqlTpl); err != nil {
			return err
		}
		if !q.PromqlGenerator.Tpl.Namespaced && q.Namespace != "" {
			return fmt.Errorf("非namespace资源不能过滤项目环境")
		}
		q.Expr = q.PromqlGenerator.Tpl.Expr
	}

	var err error
	q.Query, err = promql.New(q.Expr)
	if err != nil {
		return err
	}
	if q.PromqlGenerator != nil {
		if q.Namespace != "" {
			q.Query.AddLabelMatchers(&labels.Matcher{
				Type:  labels.MatchEqual,
				Name:  prometheus.PromqlNamespaceKey,
				Value: q.Namespace,
			})
		}
		for label, value := range q.LabelPairs {
			q.Query.AddLabelMatchers(&labels.Matcher{
				Type:  labels.MatchRegexp,
				Name:  label,
				Value: value,
			})
		}
		q.Query.Sumby(q.Tpl.Labels...)
	}
	q.Expr = q.Query.String()

	return f(q)
}

// ListScopes 获取promql模板一级目录scope
// @Tags        Observability
// @Summary     获取promql模板一级目录scope
// @Description 获取promql模板一级目录scope
// @Accept      json
// @Produce     json
// @Param       tenant_id path     string                                                true "租户ID"
// @Param       page        query    int                                                   false "page"
// @Param       size        query    int                                                   false "size"
// @Success     200         {object} handlers.ResponseStruct{Data=[]models.PromqlTplScope} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/scopes [get]
// @Security    JWT
func (h *ObservabilityHandler) ListScopes(c *gin.Context) {
	scopes := []models.PromqlTplScope{}
	if err := h.GetDB().Order("name").Find(&scopes).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.NewPageDataFromContext(c, scopes, nil, nil))
}

// ListResources 获取promql模板二级目录resource
// @Tags        Observability
// @Summary     获取promql模板二级目录resource
// @Description 获取promql模板二级目录resource
// @Accept      json
// @Produce     json
// @Param       tenant_id path     int                                                   true  "租户ID"
// @Param       scope_id  path     int                                                   true  "scope id"
// @Param       page      query    int                                                   false "page"
// @Param       size      query    int                                                   false "size"
// @Success     200       {object} handlers.ResponseStruct{Data=[]models.PromqlTplScope} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/scopes/{scope_id}/resources [get]
// @Security    JWT
func (h *ObservabilityHandler) ListResources(c *gin.Context) {
	resources := []models.PromqlTplResource{}
	if err := h.GetDB().Where("scope_id = ?", c.Param("scope_id")).Order("name").Find(&resources).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.NewPageDataFromContext(c, resources, nil, nil))
}

// ListRules 获取promql模板三级目录rule
// @Tags        Observability
// @Summary     获取promql模板三级目录rule
// @Description 获取promql模板三级目录rule
// @Accept      json
// @Produce     json
// @Param       tenant_id   path     string                                                true  "租户ID"
// @Param       resource_id path     string                                                true  "resource id, 可以是_all"
// @Param       preload     query    string                                                false "Resource, Resource.Scope"
// @Param       search      query    string                                                false "search string"
// @Param       page      query    int                                                   false "page"
// @Param       size      query    int                                                   false "size"
// @Success     200       {object} handlers.ResponseStruct{Data=[]models.PromqlTplScope} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/resources{resource_id}/rules [get]
// @Security    JWT
func (h *ObservabilityHandler) ListRules(c *gin.Context) {
	rules := []models.PromqlTplRule{}
	tenantID := c.Param("tenant_id")
	resourceID := c.Param("resource_id")
	preload := c.Query("preload")
	search := c.Query("search")

	query := h.GetDB().Model(&models.PromqlTplRule{})
	if resourceID != "_all" {
		query.Where("resource_id = ?", resourceID)
	}
	if preload == "Resource" || preload == "Resource.Scope" {
		query.Preload(preload)
	}
	if search != "" {
		query.Where("name like ? or show_name like ?", fmt.Sprintf("%%%s%%", search), fmt.Sprintf("%%%s%%", search))
	}
	if tenantID != "_all" {
		query.Where("tenant_id is null or tenant_id = ?", tenantID)
	}
	if err := query.Order("name").Find(&rules).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.NewPageDataFromContext(c, rules, nil, nil))
}

// AddRules 添加promql模板三级目录rule
// @Tags        Observability
// @Summary     添加promql模板三级目录rule
// @Description 添加promql模板三级目录rule
// @Accept      json
// @Produce     json
// @Param       tenant_id path     string                                                true "租户ID"
// @Param       param     body     models.PromqlTplRule                                  true "rule"
// @Success     200       {object} handlers.ResponseStruct{Data=[]models.PromqlTplScope} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/rules [post]
// @Security    JWT
func (h *ObservabilityHandler) AddRules(c *gin.Context) {
	req, err := h.getRuleReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Create(&req).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// UpdateRules 更新promql模板三级目录rule
// @Tags        Observability
// @Summary     更新promql模板三级目录rule
// @Description 更新promql模板三级目录rule
// @Accept      json
// @Produce     json
// @Param       tenant_id path     string                                                true "租户ID"
// @Param       rule_id   path     string                                                true "rule ID"
// @Param       param     body     models.PromqlTplRule                                  true "rule"
// @Success     200       {object} handlers.ResponseStruct{Data=[]models.PromqlTplScope} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/rules/{rule_id} [put]
// @Security    JWT
func (h *ObservabilityHandler) UpdateRules(c *gin.Context) {
	req, err := h.getRuleReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Save(&req).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteRules 删除promql模板三级目录rule
// @Tags        Observability
// @Summary     删除promql模板三级目录rule
// @Description 删除promql模板三级目录rule
// @Accept      json
// @Produce     json
// @Param       tenant_id path     string                                                true  "租户ID"
// @Param       rule_id   path     string                                                true "rule ID"
// @Success     200       {object} handlers.ResponseStruct{Data=[]models.PromqlTplScope} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/rules/{rule_id} [delete]
// @Security    JWT
func (h *ObservabilityHandler) DeleteRules(c *gin.Context) {
	rule := &models.PromqlTplRule{}
	if err := h.GetDB().Preload("Resource.Scope").First(rule, "id = ?", c.Param("rule_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	allalerts := []prometheus.MonitorAlertRule{}
	if err := h.GetAgents().ExecuteInEachCluster(c.Request.Context(), func(ctx context.Context, cli agents.Client) error {
		alerts, err := cli.Extend().ListMonitorAlertRules(ctx, v1.NamespaceAll, false)
		if err != nil {
			return errors.Wrapf(err, "list alert in cluster %s failed: %v", cli.Name())
		}
		allalerts = append(allalerts, alerts...)
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	for _, v := range allalerts {
		if !v.PromqlGenerator.Notpl() &&
			v.PromqlGenerator.Scope == rule.Resource.Scope.Name &&
			v.PromqlGenerator.Resource == rule.Resource.Name &&
			v.PromqlGenerator.Rule == rule.Name {
			handlers.NotOK(c, fmt.Errorf("prometheus 模板 %s.%s.%s 正在被告警规则%s使用", v.PromqlGenerator.Scope, v.PromqlGenerator.Resource, v.PromqlGenerator.Rule, v.Name))
			return
		}
	}
	if err := h.GetDB().Delete(rule).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

func (h *ObservabilityHandler) getRuleReq(c *gin.Context) (*models.PromqlTplRule, error) {
	req := models.PromqlTplRule{}
	if err := c.BindJSON(&req); err != nil {
		return nil, err
	}
	if _, err := parser.ParseExpr(req.Expr); err != nil {
		return nil, errors.Wrap(err, "promql语法错误")
	}
	tenantID := c.Param("tenant_id")
	if tenantID != "_all" {
		t, _ := strconv.Atoi(tenantID)
		if t == 0 {
			return nil, fmt.Errorf("tenant id not valid")
		}
		tmp := uint(t)
		req.TenantID = &tmp
	}

	return &req, nil
}

// ListDashboardTemplates 监控面板模板列表
// @Tags        Observability
// @Summary     监控面板模板列表
// @Description 监控面板模板列表
// @Accept      json
// @Produce     json
// @Success     200 {object} handlers.ResponseStruct{Data=[]models.MonitorDashboard} "resp"
// @Router      /v1/observability/template/dashboard [delete]
// @Security    JWT
func (h *ObservabilityHandler) ListDashboardTemplates(c *gin.Context) {
	tpls := []models.MonitorDashboard{}
	if err := yaml.Unmarshal(alltemplates, &tpls); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, tpls)
}
