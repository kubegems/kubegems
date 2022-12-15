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
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/observe"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/promql"
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

	Label      string // 要查询的标签值
	TargetName string // panel中多个查询的id
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
	req := h.getMetricQuery(c)
	if err := h.mutateMetricQueryReq(c, req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
		var err error
		ret, err = cli.Extend().PrometheusQueryRange(ctx, req.Expr, req.Start, req.End, req.Step)
		return err
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
	req := h.getMetricQuery(c)
	if err := h.mutateMetricQueryReq(c, req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
		var err error
		matchs := req.Query.GetVectorSelectors()
		ret, err = cli.Extend().GetPrometheusLabelValues(ctx, matchs, req.Label, req.Start, req.End)
		return err
	}); err != nil {
		handlers.NotOK(c, i18n.Errorf(c, "get prometheus label values failed, cluster: %s, promql: %s, %w", req.Cluster, req.Expr, err))
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
	req := h.getMetricQuery(c)
	if err := h.mutateMetricQueryReq(c, req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
		var err error
		matchs := req.Query.GetVectorSelectors()
		ret, err = cli.Extend().GetPrometheusLabelNames(ctx, matchs, req.Start, req.End)
		return err
	}); err != nil {
		handlers.NotOK(c, i18n.Errorf(c, "get prometheus label names failed, cluster: %s, promql: %s, %w", req.Cluster, req.Expr, err))
		return
	}

	handlers.OK(c, ret)
}

// OtelMetricsGraphs OtelMetricsGraphs
// @Tags        Observability
// @Summary     OtelMetricsGraphs
// @Description OtelMetricsGraphs
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true  "集群名"
// @Param       namespace path     string                               true  "命名空间"
// @Param       service   query    string                               false "jaeger service"
// @Param       start     query    string                               false "开始时间，默认现在-30m"
// @Param       end       query    string                               false "结束时间，默认现在"
// @Success     200       {object} handlers.ResponseStruct{Data=object} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/otel/metrics/graphs [get]
// @Security    JWT
func (h *ObservabilityHandler) OtelMetricsGraphs(c *gin.Context) {
	ns := c.Param("namespace")
	svc := c.Query("service")
	start, end, _ := getRangeParams(c.Query("start"), c.Query("end"))

	ret := gin.H{}
	if err := h.Execute(c.Request.Context(), c.Param("cluster"), func(ctx context.Context, cli agents.Client) error {
		queries := map[string]string{
			"latencyP95": fmt.Sprintf(`histogram_quantile(0.95, sum(rate(latency_bucket{namespace="%s", service_name="%s"}[5m])) by (le,namespace,service_name))`, ns, svc),
			"latencyP75": fmt.Sprintf(`histogram_quantile(0.75, sum(rate(latency_bucket{namespace="%s", service_name="%s"}[5m])) by (le,namespace,service_name))`, ns, svc),
			"latencyP50": fmt.Sprintf(`histogram_quantile(0.50, sum(rate(latency_bucket{namespace="%s", service_name="%s"}[5m])) by (le,namespace,service_name))`, ns, svc),
			"errorRate": fmt.Sprintf(`sum(irate(calls_total{namespace="%[1]s", service_name="%[2]s", status_code="STATUS_CODE_ERROR"}[5m]))by(namespace, service_name) /
			sum(irate(calls_total{namespace="%[1]s", service_name="%[2]s"}[5m]))by(namespace, service_name)`, ns, svc),
			"requestRate":          fmt.Sprintf(`sum(irate(calls_total{namespace="%s", service_name="%s"}[5m]))by(namespace, service_name)`, ns, svc),
			"operationlatencyP95":  fmt.Sprintf(`histogram_quantile(0.95, sum(rate(latency_bucket{namespace="%s", service_name="%s"}[5m])) by (le,namespace,service_name,operation))`, ns, svc),
			"operationRequestRate": fmt.Sprintf(`sum(irate(calls_total{namespace="%s", service_name="%s"}[5m]))by(namespace, service_name, operation)`, ns, svc),
			"operationErrorRate": fmt.Sprintf(`sum(irate(calls_total{namespace="%[1]s", service_name="%[2]s", status_code="STATUS_CODE_ERROR"}[5m]))by(namespace, service_name, operation) /
			sum(irate(calls_total{namespace="%[1]s", service_name="%[2]s"}[5m]))by(namespace, service_name, operation)`, ns, svc),
		}
		wg := sync.WaitGroup{}
		lock := sync.Mutex{}
		for key, query := range queries {
			wg.Add(1)
			go func(k, q string) {
				v, err := cli.Extend().PrometheusQueryRange(ctx, q, start, end, "")
				if err != nil {
					log.Error(err, "query failed", "key", k)
				}
				lock.Lock()
				defer lock.Unlock()
				ret[k] = v
				wg.Done()
			}(key, query)
		}
		wg.Wait()
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

func (h *ObservabilityHandler) getMetricQuery(c *gin.Context) *MetricQueryReq {
	return &MetricQueryReq{
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
}

func (h *ObservabilityHandler) mutateMetricQueryReq(ctx context.Context, q *MetricQueryReq) error {
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
			return i18n.Errorf(ctx, "Template and native promql cannot be empty at the same time")
		}
		if err := observe.CheckQueryExprNamespace(q.Expr, q.Namespace); err != nil {
			return err
		}
	} else {
		if err := q.PromqlGenerator.SetTpl(h.GetDataBase().FindPromqlTpl); err != nil {
			return err
		}
		if !q.PromqlGenerator.Tpl.Namespaced && q.Namespace != "" {
			return i18n.Errorf(ctx, "Non namespace resources cannot filter the project environment")
		}
		q.Expr = q.PromqlGenerator.Tpl.Expr
	}

	var err error
	q.Query, err = promql.New(q.Expr)
	if err != nil {
		return err
	}

	// from tpl
	if !q.PromqlGenerator.Notpl() {
		if q.Namespace != "" {
			q.Query.AddLabelMatchers(&labels.Matcher{
				Type:  labels.MatchEqual,
				Name:  prometheus.PromqlNamespaceKey,
				Value: q.Namespace,
			})
		}
		q.Query.Sumby(q.Tpl.Labels...)
	}

	for label, value := range q.LabelPairs {
		q.Query.AddLabelMatchers(&labels.Matcher{
			Type:  labels.MatchRegexp,
			Name:  label,
			Value: value,
		})
	}
	q.Expr = q.Query.String()
	log.Debugf("query cluster: %s, expr: %s", q.Cluster, q.Expr)
	return nil
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
// @Param       search    query    string                                                false "search in (name)"
// @Param       preload   query    string                                                false "choices (Resources)"
// @Success     200         {object} handlers.ResponseStruct{Data=[]models.PromqlTplScope} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/scopes [get]
// @Security    JWT
func (h *ObservabilityHandler) ListScopes(c *gin.Context) {
	list := []*models.PromqlTplScope{}
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "PromqlTplScope",
		SearchFields:  []string{"name"},
		PreloadFields: []string{"Resources", "Resources.Rules"},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// ListResources 获取promql模板二级目录resource
// @Tags        Observability
// @Summary     获取promql模板二级目录resource
// @Description 获取promql模板二级目录resource
// @Accept      json
// @Produce     json
// @Param       tenant_id path     int                                                   true  "租户ID"
// @Param       scope_id  path     int                                                   true  "scope id"
// @Param       preload   query    string                                                false "choices (Scope, Rules)"
// @Param       search    query    string                                                false "search in (name)"
// @Param       page      query    int                                                   false "page"
// @Param       size      query    int                                                   false "size"
// @Success     200       {object} handlers.ResponseStruct{Data=[]models.PromqlTplScope} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/scopes/{scope_id}/resources [get]
// @Security    JWT
func (h *ObservabilityHandler) ListResources(c *gin.Context) {
	list := []*models.PromqlTplResource{}
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "PromqlTplResource",
		SearchFields:  []string{"name"},
		PreloadFields: []string{"Scope", "Rules"},
		Where:         []*handlers.QArgs{handlers.Args("scope_id = ?", c.Param("scope_id"))},
	}
	total, page, size, err := query.PageList(h.GetDB().Order("name"), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))

}

// ListRules 获取promql模板三级目录rule
// @Tags        Observability
// @Summary     获取promql模板三级目录rule
// @Description 获取promql模板三级目录rule
// @Accept      json
// @Produce     json
// @Param       tenant_id   path     string                                                true  "租户ID"
// @Param       resource_id path     string                                                true  "resource id"
// @Param       preload     query    string                                                false "choices (Resource, Resource.Scope)"
// @Param       search      query    string                                                false "search in (name, show_name)"
// @Param       page      query    int                                                   false "page"
// @Param       size      query    int                                                   false "size"
// @Success     200       {object} handlers.ResponseStruct{Data=[]models.PromqlTplScope} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/resources{resource_id}/rules [get]
// @Security    JWT
func (h *ObservabilityHandler) ListRules(c *gin.Context) {
	list := []*models.PromqlTplRule{}
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "PromqlTplRule",
		SearchFields:  []string{"name", "show_name"},
		PreloadFields: []string{"Resource", "Resource.Scope"},
	}
	tenantID := c.Param("tenant_id")
	if tenantID != "_all" {
		cond.Where = append(cond.Where, handlers.Args("tenant_id is null or tenant_id = ?", tenantID))
	}
	resourceID := c.Param("resource_id")
	if resourceID != "_all" {
		cond.Where = append(cond.Where, handlers.Args("resource_id = ?", resourceID))
	}
	total, page, size, err := query.PageList(h.GetDB().Order("name"), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// GetRule 获取promql模板三级目录rule
// @Tags        Observability
// @Summary     获取promql模板三级目录rule
// @Description 获取promql模板三级目录rule
// @Accept      json
// @Produce     json
// @Param       tenant_id path     string                               true "租户ID"
// @Param       rule_id   path     string                                              true  "rule ID"
// @Param       preload   query    string                                              false "Resource, Resource.Scope"
// @Success     200       {object} handlers.ResponseStruct{Data=models.PromqlTplScope} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/rules/{rule_id} [get]
// @Security    JWT
func (h *ObservabilityHandler) GetRule(c *gin.Context) {
	rule := models.PromqlTplRule{}
	tenantID := c.Param("tenant_id")
	preload := c.Query("preload")

	query := h.GetDB().Model(&models.PromqlTplRule{})
	if preload == "Resource" || preload == "Resource.Scope" {
		query.Preload(preload)
	}
	if tenantID != "_all" {
		query.Where("tenant_id is null or tenant_id = ?", tenantID)
	}
	if err := query.First(&rule, c.Param("rule_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, rule)
}

// SearchTpl 由scope,resource,rule name获取tpl
// @Tags        Observability
// @Summary     由scope,resource,rule name获取tpl
// @Description 由scope,resource,rule name获取tpl
// @Accept      json
// @Produce     json
// @Param       tenant_id path     string                                              true  "租户ID"
// @Param       scope     query    string                               true "scope"
// @Param       resource  query    string                               true "scope"
// @Param       rule      query    string                               true "scope"
// @Success     200       {object} handlers.ResponseStruct{Data=object} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/template/search [get]
// @Security    JWT
func (h *ObservabilityHandler) SearchTpl(c *gin.Context) {
	tpl, err := h.GetDataBase().FindPromqlTpl(c.Query("scope"), c.Query("resource"), c.Query("rule"))
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, tpl)
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

	var count int64
	if err := h.GetDB().Model(&models.PromqlTplRule{}).
		Where("resource_id = ? and name = ?", req.ResourceID, req.Name).
		Count(&count).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if count > 0 {
		handlers.NotOK(c, i18n.Errorf(c, "rule %s already exist", req.Name))
		return
	}
	action := i18n.Sprintf(context.TODO(), "create")
	module := i18n.Sprintf(context.TODO(), "monitoring query template")
	h.SetAuditData(c, action, module, req.Name)
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
	action := i18n.Sprintf(context.TODO(), "update")
	module := i18n.Sprintf(context.TODO(), "monitoring query template")
	h.SetAuditData(c, action, module, req.Name)
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
	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "monitoring query template")
	h.SetAuditData(c, action, module, rule.Name)
	if rule.TenantID == nil {
		if c.Param("tenant_id") != "_all" {
			handlers.NotOK(c, i18n.Errorf(c, "prefabricated template cannot be deleted"))
			return
		}
	} else {
		h.SetExtraAuditData(c, models.ResTenant, *rule.TenantID)
	}

	if err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		dashborads := []models.MonitorDashboard{}
		if err := tx.Preload("Environment").Find(&dashborads).Error; err != nil {
			return err
		}
		for _, dash := range dashborads {
			for _, graph := range dash.Graphs {
				for _, target := range graph.Targets {
					if target.PromqlGenerator.Scope == rule.Resource.Scope.Name &&
						target.PromqlGenerator.Resource == rule.Resource.Name &&
						target.PromqlGenerator.Rule == rule.Name {
						return fmt.Errorf("此模板正在被环境: %s 中的监控大盘: %s 使用", dash.Environment.EnvironmentName, dash.Name)
					}
				}
			}
		}

		tpls := []models.MonitorDashboardTpl{}
		if err := tx.Find(&tpls).Error; err != nil {
			return err
		}
		for _, tpl := range tpls {
			for _, graph := range tpl.Graphs {
				for _, target := range graph.Targets {
					if target.PromqlGenerator.Scope == rule.Resource.Scope.Name &&
						target.PromqlGenerator.Resource == rule.Resource.Name &&
						target.PromqlGenerator.Rule == rule.Name {
						return fmt.Errorf("此模板正在被监控大盘模板: %s 使用", tpl.Name)
					}
				}
			}
		}
		return tx.Delete(rule).Error
	}); err != nil {
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
		return nil, errors.Wrap(err, "promql syntax error")
	}
	tenantID := c.Param("tenant_id")
	if tenantID != "_all" {
		t, _ := strconv.Atoi(tenantID)
		if t == 0 {
			return nil, i18n.Errorf(c, "tenant id not valid")
		}
		tmp := uint(t)
		h.SetExtraAuditData(c, models.ResTenant, tmp)
		req.TenantID = &tmp
	}

	return &req, nil
}

func getRangeParams(startStr, endStr string) (string, string, string) {
	// 由于agent解析时没有管时区，所以这里需设置为UTC
	start, end := prometheus.ParseRangeTime(startStr, endStr, time.UTC)
	return start.Format(time.RFC3339), end.Format(time.RFC3339), end.Sub(start).String()
}

type OtelView struct {
	LabelName  string                           `json:"labelname"`
	LabelValue string                           `json:"labelvalue"`
	ValueMap   map[string]prommodel.SampleValue `json:"valueMap"`
}

type OtelViews map[string]*OtelView

func newOtelViews() OtelViews {
	return OtelViews{}
}

func (views OtelViews) addVectors(vectors map[string]prommodel.Vector, labelname string) OtelViews {
	for queryname, vector := range vectors {
		for _, v := range vector {
			if labelvalue, ok := v.Metric[prommodel.LabelName(labelname)]; ok {
				if otelSvc, ok := views[string(labelvalue)]; ok {
					otelSvc.ValueMap[queryname] = v.Value
				} else {
					otelSvc := &OtelView{
						LabelName:  labelname,
						LabelValue: string(labelvalue),
						ValueMap: map[string]prommodel.SampleValue{
							queryname: v.Value,
						},
					}
					views[string(labelvalue)] = otelSvc
				}
			}
		}
	}
	return views
}

func (views OtelViews) slice(sortby string) []*OtelView {
	ret := make([]*OtelView, len(views))
	index := 0
	for _, view := range views {
		ret[index] = view
		index++
	}
	sortasc := true
	sortfield := ""
	if strings.HasSuffix(sortby, "Desc") {
		sortasc = false
		sortfield = strings.TrimSuffix(sortby, "Desc")
	} else {
		sortfield = strings.TrimSuffix(sortby, "Asc")
	}

	if sortfield == "" {
		sort.Slice(ret, func(i, j int) bool {
			if sortasc {
				return ret[i].LabelValue < ret[j].LabelValue
			} else {
				return ret[i].LabelValue > ret[j].LabelValue
			}
		})
	} else {
		sort.Slice(ret, func(i, j int) bool {
			if sortasc {
				return lessThan(ret[i].ValueMap[sortfield], ret[j].ValueMap[sortfield])
			} else {
				return lessThan(ret[j].ValueMap[sortfield], ret[i].ValueMap[sortfield])
			}
		})
	}
	return ret
}

func lessThan(a, b prommodel.SampleValue) bool {
	if math.IsNaN(float64(a)) {
		a = 0
	}
	if math.IsNaN(float64(b)) {
		b = 0
	}
	return float64(a) < float64(b)
}

// OtelServices 应用性能监控服务
// @Tags        Observability
// @Summary     应用性能监控服务
// @Description 应用性能监控服务
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                                           true  "集群名"
// @Param       namespace path     string                                                           true  "命名空间，所有namespace为_all"
// @Param       start     query    string                                                           false "开始时间，默认现在-30m"
// @Param       end       query    string                                                           false "结束时间，默认现在"
// @Param       sortby    query    string                                                           false "通过valueMap的哪个字段排序，默认根据labelvalue排序"
// @Param       page      query    int                                                              false "page"
// @Param       size      query    int                                                              false "size"
// @Success     200       {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]OtelView}} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/otel/appmonitor/services [get]
// @Security    JWT
func (h *ObservabilityHandler) OtelServices(c *gin.Context) {
	ns := c.Param("namespace")
	_, _, dur := getRangeParams(c.Query("start"), c.Query("end"))
	vectors := map[string]prommodel.Vector{}
	if err := h.Execute(c.Request.Context(), c.Param("cluster"), func(ctx context.Context, cli agents.Client) error {
		// query prometheus
		// 由于span metrics采集的数据，operation之间会有重合，这里计算出的qps、响应时间会不准确(偏大)
		vectors = batchVector(map[string]string{
			"avgRequestQPS":              fmt.Sprintf(`sum(rate(calls_total{namespace="%s"}[%s]))by(service_name)`, ns, dur),
			"avgResponseDurationSeconds": fmt.Sprintf(`sum(increase(latency_sum{namespace="%[1]s"}[%[2]s]))by(service_name) / sum(increase(latency_count{namespace="%[1]s"}[%[2]s]))by(service_name) / 1000`, ns, dur),
			"p75ResponseDurationSeconds": fmt.Sprintf(`histogram_quantile(0.75, sum(increase(latency_bucket{namespace="%s"}[%s]))by(service_name, le)) / 1000`, ns, dur),
			"p90ResponseDurationSeconds": fmt.Sprintf(`histogram_quantile(0.90, sum(increase(latency_bucket{namespace="%s"}[%s]))by(service_name, le)) / 1000`, ns, dur),
			"errorCount":                 fmt.Sprintf(`sum(increase(calls_total{status_code="STATUS_CODE_ERROR", namespace="%s"}[%s]))by(service_name)`, ns, dur),
			"errorRate":                  fmt.Sprintf(`sum(increase(calls_total{status_code="STATUS_CODE_ERROR", namespace="%[1]s"}[%[2]s]))by(service_name) / sum(increase(calls_total{namespace="%[1]s"}[%[2]s]))by(service_name)`, ns, dur),
		}, ctx, cli)
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	ret := newOtelViews().addVectors(vectors, "service_name").slice(c.Query("sortby"))
	handlers.OK(c, handlers.NewPageDataFromContext(c, ret, nil, nil))
}

func batchVector(queries map[string]string, ctx context.Context, cli agents.Client) map[string]prommodel.Vector {
	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	vectors := map[string]prommodel.Vector{}
	for key, query := range queries {
		wg.Add(1)
		go func(k, q string) {
			v, err := cli.Extend().PrometheusVector(ctx, q)
			if err != nil {
				log.Warnf("query: %s failed: %v", q, err)
			}
			lock.Lock()
			defer lock.Unlock()
			vectors[k] = v
			wg.Done()
		}(key, query)
	}
	wg.Wait()
	return vectors
}

type KV struct {
	LabelName  string                `json:"labelname"`
	LabelValue string                `json:"labelvalue"`
	Value      prommodel.SampleValue `json:"value"`
}

type OtelOverViewResp struct {
	P90ServiceDurationSeconds   []KV `json:"p90ServiceDurationSeconds"`   // p90最耗时服务
	P90OperationDurationSeconds []KV `json:"p90OperationDurationSeconds"` // p90最耗时操作
	ServiceErrorCount           []KV `json:"serviceErrorCount"`           // 服务错误数
	DBOperationCount            []KV `json:"dbOperationCount"`            // 数据库操作数
}

// OtelOverview 应用性能监控概览
// @Tags        Observability
// @Summary     应用性能监控概览
// @Description 应用性能监控概览
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                         true  "集群名"
// @Param       namespace path     string                                         true  "命名空间"
// @Param       start     query    string                                         false "开始时间，默认现在-30m"
// @Param       end       query    string                                         false "结束时间，默认现在"
// @Param       pick      query    string                                         false "选择什么值(max/min/avg), default max"
// @Success     200       {object} handlers.ResponseStruct{Data=OtelOverViewResp} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/otel/appmonitor/overview [get]
// @Security    JWT
func (h *ObservabilityHandler) OtelOverview(c *gin.Context) {
	ns := c.Param("namespace")
	pick := c.DefaultQuery("pick", "max")
	start, end, dur := getRangeParams(c.Query("start"), c.Query("end"))
	overView := OtelOverViewResp{}
	if err := h.Execute(c.Request.Context(), c.Param("cluster"), func(ctx context.Context, cli agents.Client) error {
		p90ServiceDurationSeconds, err := cli.Extend().PrometheusQueryRange(ctx, fmt.Sprintf(`histogram_quantile(0.9, sum(rate(latency_bucket{namespace="%s"}[5m]))by(service_name, le)) / 1000`, ns), start, end, "")
		if err != nil {
			return err
		}
		overView.P90ServiceDurationSeconds = pickMatrixValue(p90ServiceDurationSeconds, "service_name", pick)
		p90OperationDurationSeconds, err := cli.Extend().PrometheusQueryRange(ctx, fmt.Sprintf(`histogram_quantile(0.9, sum(rate(latency_bucket{namespace="%s"}[5m]))by(operation, le)) / 1000`, ns), start, end, "")
		if err != nil {
			return err
		}
		overView.P90OperationDurationSeconds = pickMatrixValue(p90OperationDurationSeconds, "operation", pick)

		serviceErrorCount, err := cli.Extend().PrometheusVector(ctx, fmt.Sprintf(`sum(increase(calls_total{status_code="STATUS_CODE_ERROR", namespace="%s"}[%s]))by(service_name)`, ns, dur))
		if err != nil {
			return err
		}
		overView.ServiceErrorCount = pickVectorValue(serviceErrorCount, "service_name")
		dbOperationCount, err := cli.Extend().PrometheusVector(ctx, fmt.Sprintf(`sum(increase(calls_total{namespace="%s", operation=~"SELECT.*|UPDATE.*|INSERT.*|DELETE.*"}[%s]))by(operation)`, ns, dur))
		if err != nil {
			return err
		}
		overView.DBOperationCount = pickVectorValue(dbOperationCount, "operation")
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, overView)
}

func pickMatrixValue(matrix prommodel.Matrix, field string, pick string) []KV {
	// 在matrix中取最大/平均/最小的值
	getValue := func(pairs []prommodel.SamplePair, pick string) prommodel.SampleValue {
		if len(pairs) == 0 {
			return 0
		}
		// pick max/min/avg value
		switch pick {
		case "min":
			min := pairs[0].Value
			for _, v := range pairs {
				if v.Value < min {
					min = v.Value
				}
			}
			return min
		case "avg":
			var sum prommodel.SampleValue
			for _, v := range pairs {
				sum += v.Value
			}
			return sum / prommodel.SampleValue(len(pairs))
		default:
			max := pairs[0].Value
			for _, v := range pairs {
				if v.Value > max {
					max = v.Value
				}
			}
			return max
		}
	}
	ret := []KV{}
	for _, v := range matrix {
		if labelvalue, ok := v.Metric[prommodel.LabelName(field)]; ok {
			value := getValue(v.Values, pick)
			if !math.IsNaN(float64(value)) {
				ret = append(ret, KV{
					LabelName:  field,
					LabelValue: string(labelvalue),
					Value:      value,
				})
			}
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return lessThan(ret[j].Value, ret[i].Value)
	})
	return ret
}

func pickVectorValue(vector prommodel.Vector, field string) []KV {
	ret := []KV{}
	for _, v := range vector {
		if labelvalue, ok := v.Metric[prommodel.LabelName(field)]; ok {
			if !math.IsNaN(float64(v.Value)) {
				ret = append(ret, KV{
					LabelName:  field,
					LabelValue: string(labelvalue),
					Value:      v.Value,
				})
			}
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return lessThan(ret[j].Value, ret[i].Value)
	})
	return ret
}

// OtelServiceRequests 应用请求
// @Tags        Observability
// @Summary     应用请求
// @Description 应用请求
// @Accept      json
// @Produce     json
// @Param       cluster      path     string                              true  "集群名"
// @Param       namespace    path     string                              true  "命名空间"
// @Param       service_name path     string                              true  "应用"
// @Param       start        query    string                              false "开始时间，默认现在-30m"
// @Param       end          query    string                              false "结束时间，默认现在"
// @Success     200          {object} handlers.ResponseStruct{Data=gin.H} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/otel/appmonitor/services/{service_name}/requests [get]
// @Security    JWT
func (h *ObservabilityHandler) OtelServiceRequests(c *gin.Context) {
	ns := c.Param("namespace")
	svc := c.Param("service_name")
	start, end, _ := getRangeParams(c.Query("start"), c.Query("end"))
	ret := gin.H{}
	if err := h.Execute(c.Request.Context(), c.Param("cluster"), func(ctx context.Context, cli agents.Client) error {
		queries := map[string]string{
			"requestRate":        fmt.Sprintf(`sum(irate(calls_total{namespace="%s", service_name="%s"}[5m]))`, ns, svc),
			"errorRate":          fmt.Sprintf(`sum(irate(calls_total{namespace="%[1]s", service_name="%[2]s", status_code="STATUS_CODE_ERROR"}[5m])) / sum(irate(calls_total{namespace="%[1]s", service_name="%[2]s"}[5m]))`, ns, svc),
			"p75DurationSeconds": fmt.Sprintf(`histogram_quantile(0.75, sum(rate(latency_bucket{namespace="%s", service_name="%s"}[5m]))by(le)) / 1000`, ns, svc),
			"p90DurationSeconds": fmt.Sprintf(`histogram_quantile(0.90, sum(rate(latency_bucket{namespace="%s", service_name="%s"}[5m]))by(le)) / 1000`, ns, svc),
		}
		wg := sync.WaitGroup{}
		lock := sync.Mutex{}
		for key, query := range queries {
			wg.Add(1)
			go func(k, q string) {
				v, err := cli.Extend().PrometheusQueryRange(ctx, q, start, end, "")
				if err != nil {
					log.Error(err, "query failed", "key", k)
				}
				lock.Lock()
				defer lock.Unlock()
				ret[k] = v
				wg.Done()
			}(key, query)
		}
		wg.Wait()
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// OtelServiceOperations 应用操作
// @Tags        Observability
// @Summary     应用操作
// @Description 应用操作
// @Accept      json
// @Produce     json
// @Param       cluster      path     string                                                                true  "集群名"
// @Param       namespace    path     string                                                                true  "命名空间"
// @Param       service_name path     string                                                                true  "应用"
// @Param       start        query    string                                                                false "开始时间，默认现在-30m"
// @Param       end          query    string                                                                false "结束时间，默认现在"
// @Param       sortby       query    string                                                           false "通过valueMap的哪个字段排序，默认根据labelvalue排序"
// @Param       page         query    int                                                              false "page"
// @Param       size         query    int                                                              false "size"
// @Success     200          {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]OtelView}} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/otel/appmonitor/services/{service_name}/operations [get]
// @Security    JWT
func (h *ObservabilityHandler) OtelServiceOperations(c *gin.Context) {
	ns := c.Param("namespace")
	svc := c.Param("service_name")
	_, _, dur := getRangeParams(c.Query("start"), c.Query("end"))
	vectors := map[string]prommodel.Vector{}
	if err := h.Execute(c.Request.Context(), c.Param("cluster"), func(ctx context.Context, cli agents.Client) error {
		// query prometheus
		// 由于span metrics采集的数据，operation之间会有重合，这里计算出的qps、响应时间会不准确(偏大)
		vectors = batchVector(map[string]string{
			"requestRate":        fmt.Sprintf(`sum(rate(calls_total{namespace="%s", service_name="%s"}[%s]))by(operation)`, ns, svc, dur),
			"p90DurationSeconds": fmt.Sprintf(`histogram_quantile(0.90, sum(rate(latency_bucket{namespace="%s", service_name="%s"}[%s]))by(le, operation)) / 1000`, ns, svc, dur),
			"p50DurationSeconds": fmt.Sprintf(`histogram_quantile(0.50, sum(rate(latency_bucket{namespace="%s", service_name="%s"}[%s]))by(le, operation)) / 1000`, ns, svc, dur),
			"errorRate":          fmt.Sprintf(`sum(rate(calls_total{namespace="%[1]s", service_name="%[2]s", status_code="STATUS_CODE_ERROR"}[%[3]s]))by(operation) / sum(irate(calls_total{namespace="%[1]s", service_name="%[2]s"}[%[3]s]))by(operation)`, ns, svc, dur),
		}, ctx, cli)
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	ret := newOtelViews().addVectors(vectors, "operation").slice(c.Query("sortby"))
	handlers.OK(c, handlers.NewPageDataFromContext(c, ret, nil, nil))
}

// OtelServiceTraces 应用traces
// @Tags        Observability
// @Summary     应用traces
// @Description 应用traces
// @Accept      json
// @Produce     json
// @Param       cluster      path     string                                                           true  "集群名"
// @Param       namespace    path     string                                                           true  "命名空间"
// @Param       service_name path     string                                                           true  "应用"
// @Param       start        query    string                                                           false "开始时间，默认现在-30m"
// @Param       end          query    string                                                           false "结束时间，默认现在"
// @Param       maxDuration  query    string                                                                true  "trace的maxDuration"
// @Param       minDuration  query    string                                                                true  "trace的minDuration"
// @Param       limit        query    int                                                                   true  "limit"
// @Param       page         query    int                                                                   false "page"
// @Param       size         query    int                                                                   false "size"
// @Success     200          {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]observe.Trace}} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/otel/appmonitor/services/{service_name}/traces [get]
// @Security    JWT
func (h *ObservabilityHandler) OtelServiceTraces(c *gin.Context) {
	// 前端传来的是UTC时间
	start, end := prometheus.ParseRangeTime(c.Query("start"), c.Query("end"), time.UTC)
	var traces []observe.Trace
	if err := h.Execute(c.Request.Context(), c.Param("cluster"), func(ctx context.Context, cli agents.Client) error {
		var err error
		observecli := observe.NewClient(cli, h.GetDB())
		traces, err = observecli.SearchTrace(ctx,
			c.Param("service_name"),
			start, end,
			c.Query("maxDuration"), c.Query("minDuration"),
			c.Query("limit"),
		)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.NewPageDataFromContext(c, traces, nil, nil))
}
