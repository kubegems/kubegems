package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	prommodel "github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
	"kubegems.io/pkg/utils/prometheus/promql"
)

type MetricQueryReq struct {
	// 查询范围
	Cluster   string
	Namespace string
	// EnvironmentID string `json:"environment_id"` // 可获取Cluster、namespace信息

	// 查询目标
	*prometheus.PromqlGenerator
	Expr string // 不传则自动生成，目前不支持前端传

	// 时间
	Start string // 开始时间
	End   string // 结束时间
	Step  string // 样本间隔, 单位秒

	SeriesSelector string // 用于查标签值: ref. https://prometheus.io/docs/prometheus/latest/querying/basics/#time-series-selectors

	Label string // 要查询的标签值
}

// Query 监控指标查询
// @Tags         Observability
// @Summary      监控指标查询
// @Description  监控指标查询
// @Accept       json
// @Produce      json
// @Param        cluster     path      string                                true   "集群名"
// @Param        namespace   path      string                                true   "命名空间，所有namespace为_all"
// @Param        resource    query     string                                false  "查询资源"
// @Param        rule        query     string                                false  "查询规则"
// @Param        unit        query     string                                false  "单位"
// @Param        labelpairs  query     string                                false  "标签键值对(value为空或者_all表示所有，支持正则),  eg.  labelpairs[host]=k8s-master&labelpairs[pod]=_all"
// @Param        expr        query     string                                false  "promql表达式"
// @Param        start       query     string                                false  "开始时间，默认现在-30m"
// @Param        end         query     string                                false  "结束时间，默认现在"
// @Param        step        query     int                                   false  "step, 单位秒，默认0"
// @Success      200         {object}  handlers.ResponseStruct{Data=object}  "Metrics配置"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/metrics/queryrange [get]
// @Security     JWT
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
// @Tags         Observability
// @Summary      监控标签值
// @Description  查询label对应的标签值
// @Accept       json
// @Produce      json
// @Param        label       query     string                                  true   "要查询的标签"
// @Param        cluster     path      string                                  true   "集群名"
// @Param        namespace   path      string                                  true   "命名空间，所有namespace为_all"
// @Param        resource    query     string                                  false  "查询资源"
// @Param        rule        query     string                                  false  "查询规则"
// @Param        unit        query     string                                  false  "单位"
// @Param        labelpairs  query     string                                  false  "标签键值对(value为空或者_all表示所有，支持正则),  eg.  labelpairs[host]=k8s-master&labelpairs[pod]=_all"
// @Param        expr        query     string                                  false  "promql表达式"
// @Param        start      query     string                                  false  "开始时间，默认现在-30m"
// @Param        end        query     string                                  false  "结束时间，默认现在"
// @Param        step        query     int                                     false  "step, 单位秒，默认0"
// @Success      200         {object}  handlers.ResponseStruct{Data=[]string}  "Metrics配置"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/metrics/labelvalues [get]
// @Security     JWT
func (h *ObservabilityHandler) LabelValues(c *gin.Context) {
	ret := []string{}
	if err := h.withQueryParam(c, func(req *MetricQueryReq) error {
		if err := h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
			var err error
			ret, err = cli.Extend().GetPrometheusLabelValues(ctx, req.SeriesSelector, req.Label, req.Start, req.End)
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
// @Tags         Observability
// @Summary      查群prometheus label names
// @Description  查群prometheus label names
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                  true   "集群名"
// @Param        namespace  path      string                                  true   "命名空间，所有namespace为_all"
// @Param        start       query     string                                  false  "开始时间，默认现在-30m"
// @Param        end         query     string                                  false  "结束时间，默认现在"
// @Param        expr       query     string                                  true   "promql表达式"
// @Success      200        {object}  handlers.ResponseStruct{Data=[]string}  "resp"
// @Router       /v1/observability/cluster/{cluster}/namespaces/{namespace}/monitor/metrics/labelnames [get]
// @Security     JWT
func (h *ObservabilityHandler) LabelNames(c *gin.Context) {
	ret := []string{}
	if err := h.withQueryParam(c, func(req *MetricQueryReq) error {
		if err := h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
			var err error
			ret, err = cli.Extend().GetPrometheusLabelNames(ctx, req.SeriesSelector, req.Start, req.End)
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
			BaseQueryParams: prometheus.BaseQueryParams{
				Resource:   c.Query("resource"),
				Rule:       c.Query("rule"),
				Unit:       c.Query("unit"),
				LabelPairs: c.QueryMap("labelpairs"),
			},
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

	monitoropts := new(prometheus.MonitorOptions)
	h.DynamicConfig.Get(c.Request.Context(), monitoropts)
	// 优先选用原生promql
	if q.PromqlGenerator.IsEmpty() {
		if q.Expr == "" {
			return fmt.Errorf("模板与原生promql不能同时为空")
		}
		if err := prometheus.CheckQueryExprNamespace(q.Expr, q.Namespace); err != nil {
			return err
		}

		q.SeriesSelector = q.Expr
		unit := q.PromqlGenerator.GetUnit()
		if unit != "" {
			_, ok := monitoropts.Units[unit]
			if !ok {
				return fmt.Errorf("unit %s not valid", unit)
			}
			q.Expr = promql.New(q.Expr).Arithmetic(promql.
				BinaryArithmeticOperators(prometheus.UnitValueMap[unit].Op), prometheus.UnitValueMap[unit].Value).
				ToPromql()
		}
	} else {
		if q.Expr != "" {
			return fmt.Errorf("模板与原生promql只能指定一种")
		}
		q.PromqlGenerator = &prometheus.PromqlGenerator{
			BaseQueryParams: prometheus.BaseQueryParams{
				Resource:   c.Query("resource"),
				Rule:       c.Query("rule"),
				Unit:       c.Query("unit"),
				LabelPairs: c.QueryMap("labelpairs"),
			},
		}
		ruleCtx, err := q.PromqlGenerator.BaseQueryParams.FindRuleContext(monitoropts)
		if err != nil {
			return err
		}
		if !ruleCtx.ResourceDetail.Namespaced && q.Namespace != "" {
			return fmt.Errorf("非namespace资源不能过滤项目环境")
		}

		query := promql.New(ruleCtx.RuleDetail.Expr)
		if q.Namespace != "" {
			query.AddSelector(prometheus.PromqlNamespaceKey, promql.LabelEqual, q.Namespace)
		}
		for label, value := range q.LabelPairs {
			query.AddSelector(label, promql.LabelRegex, value)
		}

		q.SeriesSelector = query.ToPromql() // SeriesSelector 不能有运算符
		q.Expr = query.
			Arithmetic(promql.BinaryArithmeticOperators(prometheus.UnitValueMap[q.Unit].Op), prometheus.UnitValueMap[q.Unit].Value).
			Round(0.001). // 默认保留三位小数
			ToPromql()
		log.Infof("promql: %s", q.Expr)
	}

	return f(q)
}

// GetMetricTemplate 获取prometheu查询模板
// @Tags         Observability
// @Summary      获取prometheu查询模板
// @Description  获取prometheu查询模板
// @Accept       json
// @Produce      json
// @Param        resource_name  path      string                                               true  "resource"
// @Param        rule_name      path      string                                               true  "rule"
// @Success      200            {object}  handlers.ResponseStruct{Data=prometheus.RuleDetail}  "resp"
// @Router       /v1/observability/template/resources/{resource_name}/rules/{rule_name} [get]
// @Security     JWT
func (h *ObservabilityHandler) GetMetricTemplate(c *gin.Context) {
	resName := c.Param("resource_name")
	ruleName := c.Param("rule_name")

	monitoropts := new(prometheus.MonitorOptions)
	h.DynamicConfig.Get(c.Request.Context(), monitoropts)
	resDetail, ok := monitoropts.Resources[resName]
	if !ok {
		handlers.NotOK(c, fmt.Errorf("resource %s not found", resName))
		return
	}
	ruleDetail, ok := resDetail.Rules[ruleName]
	if !ok {
		handlers.NotOK(c, fmt.Errorf("rule %s not found", ruleName))
		return
	}

	handlers.OK(c, ruleDetail)
}

// AddOrUpdateMetricTemplate 添加/更新prometheu查询模板
// @Tags         Observability
// @Summary      添加prometheu查询模板
// @Description  添加prometheu查询模板
// @Accept       json
// @Produce      json
// @Param        resource_name  path      string                                true  "resource"
// @Param        rule_name      path      string                                true  "rule"
// @Param        from           body      prometheus.RuleDetail                 true  "查询模板配置"
// @Success      200            {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/template/resources/{resource_name}/rules/{rule_name} [post]
// @Security     JWT
func (h *ObservabilityHandler) AddOrUpdateMetricTemplate(c *gin.Context) {
	resName := c.Param("resource_name")
	ruleName := c.Param("rule_name")
	rule := prometheus.RuleDetail{}
	if err := c.BindJSON(&rule); err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "Prometheus模板", resName+"."+ruleName)

	monitoropts := new(prometheus.MonitorOptions)
	h.DynamicConfig.Get(c.Request.Context(), monitoropts)
	for _, unit := range rule.Units {
		if _, ok := monitoropts.Units[unit]; !ok {
			handlers.NotOK(c, fmt.Errorf("unit %s is not valid", unit))
			return
		}
	}

	resDetail, ok := monitoropts.Resources[resName]
	if !ok {
		handlers.NotOK(c, fmt.Errorf("resource %s not found", resName))
		return
	}

	resDetail.Rules[ruleName] = rule
	if err := h.DynamicConfig.Set(c.Request.Context(), monitoropts); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteMetricTemplate 删除prometheu查询模板
// @Tags         Observability
// @Summary      删除prometheu查询模板
// @Description  删除prometheu查询模板
// @Accept       json
// @Produce      json
// @Param        resource_name  path      string                                true  "resource"
// @Param        rule_name      path      string                                true  "rule"
// @Success      200            {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/template/resources/{resource_name}/rules/{rule_name} [delete]
// @Security     JWT
func (h *ObservabilityHandler) DeleteMetricTemplate(c *gin.Context) {
	resName := c.Param("resource_name")
	ruleName := c.Param("rule_name")

	h.SetAuditData(c, "删除", "Prometheus模板", resName+"."+ruleName)

	monitoropts := new(prometheus.MonitorOptions)
	h.DynamicConfig.Get(c.Request.Context(), monitoropts)
	resDetail, ok := monitoropts.Resources[resName]
	if !ok {
		handlers.NotOK(c, fmt.Errorf("resource %s not found", resName))
		return
	}
	_, ok = resDetail.Rules[ruleName]
	if !ok {
		handlers.NotOK(c, fmt.Errorf("rule %s not found", ruleName))
		return
	}

	allalerts := []prometheus.MonitorAlertRule{}
	if err := h.GetAgents().ExecuteInEachCluster(c.Request.Context(), func(ctx context.Context, cli agents.Client) error {
		alerts, err := cli.Extend().ListMonitorAlertRules(ctx, v1.NamespaceAll, monitoropts, false)
		if err != nil {
			return fmt.Errorf("list alert in cluster %s failed: %v", cli.Name(), err)
		}
		allalerts = append(allalerts, alerts...)
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	for _, v := range allalerts {
		if !v.PromqlGenerator.IsEmpty() && v.PromqlGenerator.Resource == resName && v.PromqlGenerator.Rule == ruleName {
			handlers.NotOK(c, fmt.Errorf("prometheus 模板 %s.%s 正在被告警规则%s使用", resName, ruleName, v.Name))
			return
		}
	}

	delete(resDetail.Rules, ruleName)
	if err := h.DynamicConfig.Set(c.Request.Context(), monitoropts); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}
