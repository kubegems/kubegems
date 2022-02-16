package metrics

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	prommodel "github.com/prometheus/common/model"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/handlers/base"
	"kubegems.io/pkg/service/kubeclient"
	"kubegems.io/pkg/utils/prometheus"
	"kubegems.io/pkg/utils/prometheus/promql"
)

// Config 获取metric配置
// @Tags Metrics
// @Summary 获取metric配置
// @Description 不同用户，获取的配置有所不同
// @Accept json
// @Produce json
// @Param isAdminView query bool false "是否在admin视图"
// @Success 200 {object} handlers.ResponseStruct{Data=prometheus.GemsMetricConfig} "Metrics配置"
// @Router /v1/metrics/config [get]
// @Security JWT
func (h *MonitorHandler) Config(c *gin.Context) {
	isAdminView, _ := strconv.ParseBool(c.Query("isAdminView"))

	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, "not login")
		return
	}
	handlers.OK(c, prometheus.GetGemsMetricConfig(u.SystemRoleID == 1 && isAdminView))
}

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
		values := url.Values{}
		values.Add("query", req.Promql)
		values.Add("start", req.Start)
		values.Add("end", req.End)
		values.Add("step", req.Step)
		err := kubeclient.DoRequest(http.MethodGet, req.Cluster,
			fmt.Sprintf("/custom/prometheus/v1/matrix?%s", values.Encode()), nil, &ret)
		if err != nil {
			return fmt.Errorf("prometheus query range failed, cluster: %s, promql: %s, %w", req.Cluster, req.Promql, err)
		}
		return nil
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
	ret := map[string]interface{}{}
	if err := h.withQueryParam(c, func(req *MetricQueryReq) error {
		values := url.Values{}
		values.Add("match", req.SeriesSelector)
		values.Add("start", req.Start)
		values.Add("end", req.End)
		values.Add("label", req.Label)
		err := kubeclient.DoRequest(http.MethodGet, req.Cluster,
			fmt.Sprintf("/custom/prometheus/v1/labelvalues?%s", values.Encode()), nil, &ret)
		if err != nil {
			return fmt.Errorf("prometheus label values failed, cluster: %s, promql: %s, %w", req.Cluster, req.Promql, err)
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret["labels"])
}

func (h *MonitorHandler) withQueryParam(c *gin.Context, f func(req *MetricQueryReq) error) error {
	u, exist := h.GetContextUser(c)
	if !exist {
		return fmt.Errorf("not login")
	}

	cfg := prometheus.GetGemsMetricConfig(u.SystemRoleID == 1)

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

	ruleCtx, err := q.FindRuleContext(cfg)
	if err != nil {
		return err
	}
	if u.SystemRoleID != 1 && q.Namespace == "" {
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

type MonitorHandler struct {
	base.BaseHandler
}

func (h *MonitorHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/metrics/config", h.Config)
	rg.GET("/metrics/queryrange", h.QueryRange)
	rg.GET("/metrics/labelvalues", h.LabelValues)

	rg.GET("/metrics/cluster/:cluster/namespaces/:namespace/targets", h.CheckByClusterNamespace, h.ListMetricTarget)
	rg.POST("/metrics/cluster/:cluster/namespaces/:namespace/targets", h.CheckByClusterNamespace, h.AddOrUpdateMetricTarget)
	rg.DELETE("/metrics/cluster/:cluster/namespaces/:namespace/targets/:name", h.CheckByClusterNamespace, h.DeleteMetricTarget)

	rg.GET("/metrics/dashboard", h.ListDashborad)
	rg.POST("/metrics/dashboard", h.CreateOrUpdateDashborad)
	rg.DELETE("/metrics/dashboard/:dashboard_id", h.DeleteDashborad)
}
