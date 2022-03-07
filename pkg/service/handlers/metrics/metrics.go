package metrics

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	prommodel "github.com/prometheus/common/model"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/handlers/base"
	"kubegems.io/pkg/service/online"
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
	prometheus.BaseQueryParams

	// 时间
	Start string // 开始时间
	End   string // 结束时间
	Step  string // step，单位秒

	Topk int // 前多少个

	Promql         string // 不传则自动生成，目前不支持前端传
	SeriesSelector string // 用于查标签值: ref. https://prometheus.io/docs/prometheus/latest/querying/basics/#time-series-selectors

	Label string // 要查询的标签值
}

// Query 监控指标查询
// @Tags Metrics
// @Summary 监控指标查询
// @Description 监控指标查询
// @Accept json
// @Produce json
// @Param cluster query string true "集群名"
// @Param namespace query string false "命名空间， 非管理员必传"
// @Param resource query string true "查询资源"
// @Param rule query string true "查询规则"
// @Param unit query string true "单位"
// @Param labelpairs query string false "标签键值对(value为空或者_all表示所有，支持正则), eg. labelpairs[host]=k8s-master&labelpairs[pod]=_all"
// @Param start query string false "开始时间，默认现在-30m"
// @Param end query string false "结束时间，默认现在"
// @Param step query int false "step, 单位秒，默认0"
// @Param topk query int false "限制返回前多少条指标，默认20"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "Metrics配置"
// @Router /v1/metrics/queryrange [get]
// @Security JWT
func (h *MonitorHandler) QueryRange(c *gin.Context) {
	ret := prommodel.Matrix{}
	if err := h.withQueryParam(c, func(req *MetricQueryReq) error {
		return h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
			var err error
			ret, err = cli.Extend().PrometheusQueryRange(ctx, req.Promql, req.Start, req.End, req.Step)
			return err
		})
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// Query 监控标签值
// @Tags Metrics
// @Summary 监控标签值
// @Description 查询label对应的标签值
// @Accept json
// @Produce json
// @Param label query string true "要查询的标签"
// @Param cluster query string true "集群名"
// @Param namespace query string false "命名空间， 非管理员必传"
// @Param resource query string true "查询资源"
// @Param rule query string true "查询规则"
// @Param unit query string true "单位"
// @Param labelpairs query string false "标签键值对(value为空或者_all表示所有，支持正则), eg. labelpairs[host]=k8s-master&labelpairs[pod]=_all"
// @Param start query string false "开始时间，默认现在-30m"
// @Param end query string false "结束时间，默认现在"
// @Param step query int false "step, 单位秒，默认0"
// @Success 200 {object} handlers.ResponseStruct{Data=[]string} "Metrics配置"
// @Router /v1/metrics/labelvalues [get]
// @Security JWT
func (h *MonitorHandler) LabelValues(c *gin.Context) {
	ret := []string{}
	if err := h.withQueryParam(c, func(req *MetricQueryReq) error {
		if err := h.Execute(c.Request.Context(), req.Cluster, func(ctx context.Context, cli agents.Client) error {
			var err error
			ret, err = cli.Extend().GetPrometheusLabelValues(ctx, req.SeriesSelector, req.Label, req.Start, req.End)
			return err
		}); err != nil {
			return fmt.Errorf("prometheus label values failed, cluster: %s, promql: %s, %w", req.Cluster, req.Promql, err)
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

func (h *MonitorHandler) withQueryParam(c *gin.Context, f func(req *MetricQueryReq) error) error {
	u, exist := h.GetContextUser(c)
	if !exist {
		return fmt.Errorf("not login")
	}

	q := &MetricQueryReq{
		Cluster:   c.Query("cluster"),
		Namespace: c.Query("namespace"),
		BaseQueryParams: prometheus.BaseQueryParams{
			Resource:   c.Query("resource"),
			Rule:       c.Query("rule"),
			Unit:       c.Query("unit"),
			LabelPairs: c.QueryMap("labelpairs"),
		},
		Start:  c.Query("start"),
		End:    c.Query("end"),
		Step:   c.Query("step"),
		Promql: c.Query("promql"),
		Label:  c.Query("label"),
	}

	q.Topk, _ = strconv.Atoi(c.DefaultQuery("topk", "20"))

	if q.Cluster == "" {
		return fmt.Errorf("请指定查询集群")
	}

	monitoropts := new(prometheus.MonitorOptions)
	online.LoadOptions(monitoropts, h.GetDB())
	ruleCtx, err := q.FindRuleContext(monitoropts)
	if err != nil {
		return err
	}
	if u.GetSystemRoleID() != 1 && q.Namespace == "" {
		return fmt.Errorf("非管理员必须指定namespace")
	}
	if !ruleCtx.ResourceDetail.Namespaced && q.Namespace != "" {
		return fmt.Errorf("非namespace资源不能过滤项目环境")
	}

	now := time.Now().UTC()
	// 默认查近30m
	if q.Start == "" || q.End == "" {
		q.Start = now.Add(-30 * time.Minute).Format(time.RFC3339)
		q.End = now.Format(time.RFC3339)
	}

	if q.Promql == "" {
		query := promql.New(ruleCtx.RuleDetail.Expr)
		if q.Namespace != "" {
			query.AddSelector(prometheus.PromqlNamespaceKey, promql.LabelEqual, q.Namespace)
		}
		for label, value := range q.LabelPairs {
			query.AddSelector(label, promql.LabelRegex, value)
		}

		q.SeriesSelector = query.ToPromql() // SeriesSelector 不能有运算符

		q.Promql = query.
			Arithmetic(promql.BinaryArithmeticOperators(prometheus.UnitValueMap[q.Unit].Op), prometheus.UnitValueMap[q.Unit].Value).
			Round(0.001). // 默认保留三位小数
			Topk(q.Topk). // 默认最多20条
			ToPromql()
		log.Infof("promql: %s", q.Promql)
	}

	return f(q)
}

// GetMetricTemplate 获取prometheu查询模板
// @Tags Metrics
// @Summary 获取prometheu查询模板
// @Description 获取prometheu查询模板
// @Accept json
// @Produce json
// @Param resource_name path string true "resource"
// @Param rule_name path string true "rule"
// @Success 200 {object} handlers.ResponseStruct{Data=prometheus.RuleDetail} "resp"
// @Router /v1/metrics/template/resources/{resource_name}/rules/{rule_name} [get]
// @Security JWT
func (h *MonitorHandler) GetMetricTemplate(c *gin.Context) {
	resName := c.Param("resource_name")
	ruleName := c.Param("rule_name")

	monitoropts := new(prometheus.MonitorOptions)
	online.LoadOptions(monitoropts, h.GetDB())
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
// @Tags Metrics
// @Summary 添加prometheu查询模板
// @Description 添加prometheu查询模板
// @Accept json
// @Produce json
// @Param resource_name path string true "resource"
// @Param rule_name path string true "rule"
// @Param from body prometheus.RuleDetail true "查询模板配置"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "resp"
// @Router /v1/metrics/template/resources/{resource_name}/rules/{rule_name} [post]
// @Security JWT
func (h *MonitorHandler) AddOrUpdateMetricTemplate(c *gin.Context) {
	resName := c.Param("resource_name")
	ruleName := c.Param("rule_name")
	rule := prometheus.RuleDetail{}
	if err := c.BindJSON(&rule); err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "Prometheus模板", resName+"."+ruleName)

	monitoropts := new(prometheus.MonitorOptions)
	online.LoadOptions(monitoropts, h.GetDB())
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
	if err := online.SaveOptions(monitoropts, h.GetDB()); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteMetricTemplate 删除prometheu查询模板
// @Tags Metrics
// @Summary 删除prometheu查询模板
// @Description 删除prometheu查询模板
// @Accept json
// @Produce json
// @Param resource_name path string true "resource"
// @Param rule_name path string true "rule"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "resp"
// @Router /v1/metrics/template/resources/{resource_name}/rules/{rule_name} [delete]
// @Security JWT
func (h *MonitorHandler) DeleteMetricTemplate(c *gin.Context) {
	resName := c.Param("resource_name")
	ruleName := c.Param("rule_name")

	h.SetAuditData(c, "删除", "Prometheus模板", resName+"."+ruleName)

	monitoropts := new(prometheus.MonitorOptions)
	online.LoadOptions(monitoropts, h.GetDB())
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

	allalerts := []prometheus.AlertRule{}
	if err := h.GetAgents().ExecuteInEachCluster(c.Request.Context(), func(ctx context.Context, cli agents.Client) error {
		alerts, err := cli.Extend().ListAllAlertRules(ctx, monitoropts)
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
		if v.Resource == resName && v.Rule == ruleName {
			handlers.NotOK(c, fmt.Errorf("prometheus 模板 %s.%s 正在被告警规则%s使用", resName, ruleName, v.Name))
			return
		}
	}

	delete(resDetail.Rules, ruleName)
	if err := online.SaveOptions(monitoropts, h.GetDB()); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

type MonitorHandler struct {
	base.BaseHandler
}

func (h *MonitorHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/metrics/queryrange", h.QueryRange)
	rg.GET("/metrics/labelvalues", h.LabelValues)

	rg.GET("/metrics/cluster/:cluster/namespaces/:namespace/targets", h.CheckByClusterNamespace, h.ListMetricTarget)
	rg.POST("/metrics/cluster/:cluster/namespaces/:namespace/targets", h.CheckByClusterNamespace, h.AddOrUpdateMetricTarget)
	rg.DELETE("/metrics/cluster/:cluster/namespaces/:namespace/targets/:name", h.CheckByClusterNamespace, h.DeleteMetricTarget)

	rg.GET("/metrics/dashboard", h.ListDashborad)
	rg.POST("/metrics/dashboard", h.CreateOrUpdateDashborad)
	rg.DELETE("/metrics/dashboard/:dashboard_id", h.DeleteDashborad)

	rg.GET("/metrics/template/resources/:resource_name/rules/:rule_name", h.CheckIsSysADMIN, h.GetMetricTemplate)
	rg.POST("/metrics/template/resources/:resource_name/rules/:rule_name", h.CheckIsSysADMIN, h.AddOrUpdateMetricTemplate)
	rg.DELETE("/metrics/template/resources/:resource_name/rules/:rule_name", h.CheckIsSysADMIN, h.DeleteMetricTemplate)
}
