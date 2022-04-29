package alerthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
)

/*
prometheusrule 和 alertmanagerconfig 的配合使用
prometheusrule 按照名字分组
alertmanagerconfig 则根据labels匹配
*/

type Receiver struct {
	Name     string `json:"name"`
	Interval string `json:"interval"` // 分组间隔
}

type AlertLevel struct {
	Op       string  `json:"op"`
	Value    float64 `json:"value"`
	Severity string  `json:"severity"`
}

// ListAlertRule 获取AlertRule列表
// @Tags         Alert
// @Summary      获取AlertRule列表
// @Description  获取AlertRule列表
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "规则"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert [get]
// @Security     JWT
func (h *AlertsHandler) ListAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		promeAlertRules, err := cli.Extend().GetPromeAlertRules(ctx, "")
		if err != nil {
			return nil, err
		}

		monitoropts := new(prometheus.MonitorOptions)
		h.DynamicConfig.Get(ctx, monitoropts)
		ret := []prometheus.AlertRule{}
		if namespace == allNamespace {
			ret, err = cli.Extend().ListAllAlertRules(ctx, monitoropts)
			if err != nil {
				return nil, err
			}
		} else {
			raw, err := cli.Extend().GetRawAlertResource(ctx, namespace, monitoropts)
			if err != nil {
				return nil, err
			}
			ret, err = raw.ToAlerts(false)
			if err != nil {
				return nil, err
			}
		}

		for i := range ret {
			key := prometheus.RealTimeAlertKey(ret[i].Namespace, ret[i].Name)
			if promRule, ok := promeAlertRules[key]; ok {
				ret[i].State = promRule.State
			} else {
				ret[i].State = "inactive"
			}
		}

		sort.Slice(ret, func(i, j int) bool {
			return strings.ToLower(ret[i].Name) < strings.ToLower(ret[j].Name)
		})
		return ret, nil
	})(c)
}

// GetAlertRule AlertRule详情
// @Tags         Alert
// @Summary      AlertRule详情
// @Description  AlertRule详情
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "name"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "规则"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name} [get]
// @Security     JWT
func (h *AlertsHandler) GetAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")

	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		monitoropts := new(prometheus.MonitorOptions)
		h.DynamicConfig.Get(ctx, monitoropts)
		raw, err := cli.Extend().GetRawAlertResource(ctx, namespace, monitoropts)
		if err != nil {
			return nil, err
		}
		alerts, err := raw.ToAlerts(true)
		if err != nil {
			return nil, err
		}
		// get realtime alert
		promeAlertRules, err := cli.Extend().GetPromeAlertRules(ctx, name)
		if err != nil {
			return nil, err
		}

		realtimeAertRule := promeAlertRules[prometheus.RealTimeAlertKey(namespace, name)]
		sort.Sort(&realtimeAertRule)

		index := -1
		for i := range alerts {
			if alerts[i].Name == name {
				index = i
				break
			}
		}
		if index == -1 {
			return nil, fmt.Errorf("alert %s not found", name)
		}
		alerts[index].State = realtimeAertRule.State
		alerts[index].RealTimeAlerts = realtimeAertRule.Alerts
		return alerts[index], nil
	})(c)
}

// GetAlertRule 禁用AlertRule
// @Tags         Alert
// @Summary      禁用AlertRule
// @Description  禁用AlertRule
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "name"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  ""
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name}/actions/disable [post]
// @Security     JWT
func (h *AlertsHandler) DisableAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	h.SetAuditData(c, "禁用", "告警规则", name)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		if err := createSilenceIfNotExist(ctx, namespace, name, cli); err != nil {
			log.Error(err, "create silence", "alert name", name)
			return nil, err
		}
		return "ok", nil
	})(c)
}

// GetAlertRule 启用AlertRule
// @Tags         Alert
// @Summary      启用AlertRule
// @Description  启用AlertRule
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "name"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  ""
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name}/actions/enable [post]
// @Security     JWT
func (h *AlertsHandler) EnableAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	h.SetAuditData(c, "启用", "告警规则", name)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		if err := deleteSilenceIfExist(ctx, namespace, name, cli); err != nil {
			log.Error(err, "delete silence", "alert name", name)
			return nil, err
		}
		return "ok", nil
	})(c)
}

// CreateAlertRule 创建AlertRule
// @Tags         Alert
// @Summary      创建AlertRule
// @Description  创建AlertRule
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        form       body      string                                true  "body"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "规则"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert [post]
// @Security     JWT
func (h *AlertsHandler) CreateAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		req := prometheus.AlertRule{}
		if err := c.BindJSON(&req); err != nil {
			return nil, err
		}
		req.Namespace = namespace
		h.SetAuditData(c, "创建", "告警规则", req.Name)

		monitoropts := new(prometheus.MonitorOptions)
		h.DynamicConfig.Get(ctx, monitoropts)
		if err := req.CheckAndModify(monitoropts); err != nil {
			return nil, err
		}

		// get、update、commit
		raw, err := cli.Extend().GetRawAlertResource(ctx, namespace, monitoropts)
		if err != nil {
			return nil, err
		}

		if err := raw.ModifyAlertRule(req, prometheus.Add); err != nil {
			return nil, err
		}

		if err := cli.Extend().CommitRawAlertResource(ctx, raw); err != nil {
			return nil, err
		}
		return "ok", nil
	})(c)
}

// ModifyAlertRule 修改AlertRule
// @Tags         Alert
// @Summary      修改AlertRule
// @Description  修改AlertRule
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "name"
// @Param        form       body      string                                true  "body"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "规则"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name} [put]
// @Security     JWT
func (h *AlertsHandler) ModifyAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		req := prometheus.AlertRule{}
		if err := c.BindJSON(&req); err != nil {
			return nil, err
		}
		req.Namespace = namespace
		h.SetAuditData(c, "更新", "告警规则", req.Name)

		monitoropts := new(prometheus.MonitorOptions)
		h.DynamicConfig.Get(ctx, monitoropts)
		if err := req.CheckAndModify(monitoropts); err != nil {
			return nil, err
		}

		// get、update、commit
		raw, err := cli.Extend().GetRawAlertResource(ctx, namespace, monitoropts)
		if err != nil {
			return nil, err
		}

		if err := raw.ModifyAlertRule(req, prometheus.Update); err != nil {
			return nil, err
		}

		if err := cli.Extend().CommitRawAlertResource(ctx, raw); err != nil {
			return nil, err
		}
		return "ok", nil
	})(c)
}

// DeleteAlertRule 删除AlertRule
// @Tags         Alert
// @Summary      修改AlertRule
// @Description  修改AlertRule
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        namespace  path      string                                true  "namespace"
// @Param        name       path      string                                true  "name"
// @Success      200        {object}  handlers.ResponseStruct{Data=string}  "规则"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name} [delete]
// @Security     JWT
func (h *AlertsHandler) DeleteAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	req := prometheus.AlertRule{
		Namespace: namespace,
		Name:      name,
	}

	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		// get、update、commit
		monitoropts := new(prometheus.MonitorOptions)
		h.DynamicConfig.Get(ctx, monitoropts)
		raw, err := cli.Extend().GetRawAlertResource(ctx, namespace, monitoropts)
		if err != nil {
			return nil, err
		}

		if err := raw.ModifyAlertRule(req, prometheus.Delete); err != nil {
			return nil, err
		}

		if err := cli.Extend().CommitRawAlertResource(ctx, raw); err != nil {
			return nil, err
		}

		// 清理silence规则
		if err := deleteSilenceIfExist(ctx, namespace, name, cli); err != nil {
			return nil, err
		}
		return "ok", nil
	})(c)
}

type AlertMessageGroup struct {
	// group by字段
	Fingerprint string
	StartsAt    *time.Time `gorm:"index"` // 告警开始时间

	// 附加字段
	Message        string
	EndsAt         *time.Time // 告警结束时间
	CreatedAt      *time.Time // 上次告警时间
	Status         string     // firing or resolved
	Labels         datatypes.JSON
	SilenceCreator string
	// 计数
	Count int64
}

// AlertHistory 告警历史
// @Tags         Alert
// @Summary      告警历史
// @Description  告警历史
// @Accept       json
// @Produce      json
// @Param        cluster        path      string                                             true   "cluster"
// @Param        namespace      path      string                                             true   "namespace"
// @Param        name           path      string                                             true   "name"
// @Param        status         query     string                                             false  "告警状态(resolved, firing),  为空则是所有状态"
// @Param        CreatedAt_gte  query     string                                             false  "CreatedAt_gte"
// @Param        CreatedAt_lte  query     string                                             false  "CreatedAt_lte"
// @Param        page           query     int                                                false  "page"
// @Param        size           query     int                                                false  "size"
// @Success      200            {object}  handlers.ResponseStruct{Data=[]AlertMessageGroup}  "规则"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name}/history [get]
// @Security     JWT
func (h *AlertsHandler) AlertHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	alertName := c.Param("name")

	var total int64
	var messages []AlertMessageGroup
	// 若同时有resolved和firing。展示resolved
	// select max(status) from alert_messages
	// output: resolved
	tmpQuery := h.GetDB().Table("alert_messages").
		Select(`alert_messages.fingerprint, 
			starts_at,
			max(ends_at) as ends_at, 
			max(value) as value, 
			max(message) as message, 
			max(created_at) as created_at, 
			max(status) as status,
			max(labels) as labels,
			count(created_at) as count`).
		Joins("join alert_infos on alert_messages.fingerprint = alert_infos.fingerprint").
		Where("cluster_name = ?", cluster).
		Where("namespace = ?", namespace).
		Where("name = ?", alertName).
		Group("alert_messages.fingerprint").Group("starts_at")

	start := c.Query("CreatedAt_gte")
	if start != "" {
		tmpQuery.Where("created_at > ?", start)
	}
	end := c.Query("CreatedAt_lte")
	if start != "" {
		tmpQuery.Where("created_at < ?", end)
	}

	// 中间表
	query := h.GetDB().Table("(?) as t", tmpQuery)
	status := c.Query("status")
	if status != "" {
		query.Where("status = ?", status)
	}

	// 总数, 不能直接count，需要count临时表
	if err := query.Count(&total).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	// 分页
	if err := query.Order("created_at desc").Limit(size).Offset((page - 1) * size).Scan(&messages).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, handlers.Page(total, messages, int64(page), int64(size)))
}

// AlertHistory 重复的告警记录
// @Tags         Alert
// @Summary      重复的告警记录
// @Description  重复的告警记录
// @Accept       json
// @Produce      json
// @Param        cluster      path      string                                               true   "cluster"
// @Param        namespace    path      string                                               true   "namespace"
// @Param        name         path      string                                               true   "name"
// @Param        fingerprint  query     string                                               true   "告警指纹"
// @Param        starts_at    query     string                                               true   "告警开始时间"
// @Param        page         query     int                                                  false  "page"
// @Param        size         query     int                                                  false  "size"
// @Success      200          {object}  handlers.ResponseStruct{Data=[]models.AlertMessage}  "规则"
// @Router       /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name}/repeats [get]
// @Security     JWT
func (h *AlertsHandler) AlertRepeats(c *gin.Context) {
	var messages []models.AlertMessage
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model: "AlertMessage",
		Where: []*handlers.QArgs{
			handlers.Args("cluster_name = ?", c.Param("cluster")),
			handlers.Args("namespace = ?", c.Param("namespace")),
			handlers.Args("name = ?", c.Param("name")),
			handlers.Args("alert_messages.fingerprint = ?", c.Query("fingerprint")),
			handlers.Args("starts_at = ?", c.Query("starts_at")),
		},
		Join: handlers.Args("join alert_infos on alert_messages.fingerprint = alert_infos.fingerprint"),
	}
	total, page, size, err := query.PageList(h.GetDB().Order("created_at desc"), cond, &messages)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, handlers.Page(total, messages, int64(page), int64(size)))
}

type AlertCountStatus struct {
	TodayCount     int  `json:"todayCount"`
	YesterdayCount int  `json:"yesterdayCount"`
	IsIncrease     bool `json:"isIncrease"` // 今天相比昨天，是否在增加
	Percent        int  `json:"percent"`    // 增加/减少的百分数，-1 表示无穷大，0 表示不增不减
}

func (a *AlertCountStatus) calculate() {
	if a.TodayCount >= a.YesterdayCount {
		if a.TodayCount == 0 {
			// 两个都是0
			a.IsIncrease = false
			a.Percent = 0
			return
		}

		a.IsIncrease = true
		if a.YesterdayCount == 0 {
			a.Percent = -1 // -1 表示无穷大
		} else {
			a.Percent = ((a.TodayCount - a.YesterdayCount) * 100) / a.YesterdayCount
		}
	} else {
		a.IsIncrease = false
		a.Percent = ((a.YesterdayCount - a.TodayCount) * 100) / a.YesterdayCount
	}
}

type AlertCountRet struct {
	Total    AlertCountStatus `json:"total,omitempty"`
	Firing   AlertCountStatus `json:"firing,omitempty"`
	Resolved AlertCountStatus `json:"resolved,omitempty"`
}

// AlertToday 今日告警数量统计
// @Tags         Alert
// @Summary      今日告警数量统计
// @Description  今日告警数量统计
// @Accept       json
// @Produce      json
// @Param        tenant  query     string                                          true   "租户名"
// @Param        status  query     string                                          false  "状态(firing, resolved)"
// @Success      200     {object}  handlers.ResponseStruct{Data=AlertCountStatus}  "resp"
// @Router       /v1/alerts/today [get]
// @Security     JWT
func (h *AlertsHandler) AlertToday(c *gin.Context) {
	todayBedin := utils.DayStartTime(time.Now())
	yesterdayBegin := todayBedin.Add(-24 * time.Hour)
	todayEnd := todayBedin.Add(24 * time.Hour)

	yesterdaySubQuery := h.GetDB().Table("alert_messages").
		Select("fingerprint, max(created_at) as max_created_at").
		Where("starts_at >= ?", yesterdayBegin).
		Where("starts_at < ?", todayBedin).
		Group("fingerprint")
	yesterdayQuery := h.GetDB().Table("(?) as tmp", yesterdaySubQuery).
		Select("status, count(alert_infos.fingerprint) as count").
		Joins("join alert_messages on tmp.fingerprint = alert_messages.fingerprint and tmp.max_created_at = alert_messages.created_at").
		Joins("join alert_infos on tmp.fingerprint = alert_infos.fingerprint").
		Where("tenant_name = ?", c.Query("tenant")).
		Group("status")

	todaySubQuery := h.GetDB().Table("alert_messages").
		Select("fingerprint, max(created_at) as max_created_at").
		Where("starts_at >= ?", todayBedin).
		Where("starts_at < ?", todayEnd).
		Group("fingerprint")
	todayQuery := h.GetDB().Table("(?) as tmp", todaySubQuery).
		Select("status, count(alert_infos.fingerprint) as count").
		Joins("join alert_messages on tmp.fingerprint = alert_messages.fingerprint and tmp.max_created_at = alert_messages.created_at").
		Joins("join alert_infos on tmp.fingerprint = alert_infos.fingerprint").
		Where("tenant_name = ?", c.Query("tenant")).
		Group("status")

	type result struct {
		Status string
		Count  int
	}
	yesterdayResult := []result{}
	todayResult := []result{}
	if err := yesterdayQuery.Scan(&yesterdayResult).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := todayQuery.Scan(&todayResult).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	ret := AlertCountRet{}
	for _, v := range yesterdayResult {
		if v.Status == "firing" {
			ret.Firing.YesterdayCount = v.Count
		} else {
			ret.Resolved.YesterdayCount = v.Count
		}
	}
	for _, v := range todayResult {
		if v.Status == "firing" {
			ret.Firing.TodayCount = v.Count
		} else {
			ret.Resolved.TodayCount = v.Count
		}
	}
	ret.Total.YesterdayCount = ret.Firing.YesterdayCount + ret.Resolved.YesterdayCount
	ret.Total.TodayCount = ret.Firing.TodayCount + ret.Resolved.TodayCount
	ret.Firing.calculate()
	ret.Resolved.calculate()
	ret.Total.calculate()

	handlers.OK(c, ret)
}

type AlertGraph struct {
	ProjectName string
	Date        string
	Count       int64

	DateTimeStamp int64
}

// AlertGraph 告警趋势图
// @Tags         Alert
// @Summary      告警趋势图
// @Description  告警趋势图
// @Accept       json
// @Produce      json
// @Param        tenant  query     string                                      true   "租户名"
// @Param        start    query     string                                    false  "开始时间，格式 2006-01-02T15:04:05Z07:00"
// @Param        end      query     string                                    false  "结束时间，格式 2006-01-02T15:04:05Z07:00"
// @Success      200     {object}  handlers.ResponseStruct{Data=model.Matrix}  "resp"
// @Router       /v1/alerts/graph [get]
// @Security     JWT
func (h *AlertsHandler) AlertGraph(c *gin.Context) {
	start, err := time.Parse(time.RFC3339, c.Query("start"))
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if start.IsZero() {
		start = utils.DayStartTime(time.Now())
	}
	end, err := time.Parse(time.RFC3339, c.Query("end"))
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if end.IsZero() {
		end = utils.NextDayStartTime(time.Now())
	}

	subQuery := h.GetDB().Table("alert_messages").
		Select("fingerprint, max(created_at) as max_created_at").
		Where("starts_at >= ?", start).
		Where("starts_at < ?", end).
		Group("fingerprint")
	query := h.GetDB().Table("(?) as tmp", subQuery).
		Select(`project_name, DATE_FORMAT(starts_at, "%Y-%m-%d") as date, count(alert_messages.fingerprint) as count`).
		Joins("join alert_messages on tmp.fingerprint = alert_messages.fingerprint and tmp.max_created_at = alert_messages.created_at").
		Joins("join alert_infos on tmp.fingerprint = alert_infos.fingerprint").
		Where("tenant_name = ?", c.Query("tenant")).
		Group("project_name").Group(`DATE_FORMAT(starts_at, "%Y-%m-%d")`)

	result := []AlertGraph{}
	if err := query.Scan(&result).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	tmp := map[string][]AlertGraph{}
	for _, v := range result {
		t, _ := time.Parse("2006-01-02", v.Date)
		v.DateTimeStamp = t.UnixMilli()
		if elems, ok := tmp[v.ProjectName]; ok {
			elems = append(elems, v)
			tmp[v.ProjectName] = elems
		} else {
			tmp[v.ProjectName] = []AlertGraph{v}
		}
	}

	ret := model.Matrix{}
	for k, points := range tmp {
		series := &model.SampleStream{
			Metric: model.Metric{
				"project": model.LabelValue(k),
			},
			Values: newDefaultSamplePair(start, end),
		}
		fillInPoints(points, series.Values)
		ret = append(ret, series)
	}

	handlers.OK(c, ret)
}

type TableRet struct {
	GroupValue string `json:"groupValue,omitempty"`
	Count      int    `json:"count,omitempty"`
}

// AlertByGroup 告警分组统计
// @Tags         Alert
// @Summary      告警分组统计
// @Description  告警分组统计
// @Accept       json
// @Produce      json
// @Param        tenant   query     string                                    true   "租户名"
// @Param        start   query     string                                      false  "开始时间，格式 2006-01-02T15:04:05Z07:00"
// @Param        end     query     string                                      false  "结束时间，格式 2006-01-02T15:04:05Z07:00"
// @Param        groupby  query     string                                    true   "按什么分组(project_name, alert_type)"
// @Success      200      {object}  handlers.ResponseStruct{Data=[]TableRet}  "resp"
// @Router       /v1/alerts/group [get]
// @Security     JWT
func (h *AlertsHandler) AlertByGroup(c *gin.Context) {
	start, err := time.Parse(time.RFC3339, c.Query("start"))
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if start.IsZero() {
		start = utils.DayStartTime(time.Now())
	}
	end, err := time.Parse(time.RFC3339, c.Query("end"))
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if end.IsZero() {
		end = utils.NextDayStartTime(time.Now())
	}

	subQuery := h.GetDB().Table("alert_messages").
		Select("fingerprint, max(created_at) as max_created_at").
		Where("starts_at >= ?", start).
		Where("starts_at < ?", end).
		Group("fingerprint")
	query := h.GetDB().Table("(?) as tmp", subQuery).
		Joins("join alert_messages on tmp.fingerprint = alert_messages.fingerprint and tmp.max_created_at = alert_messages.created_at").
		Joins("join alert_infos on tmp.fingerprint = alert_infos.fingerprint").
		Where("tenant_name = ?", c.Query("tenant"))

	switch c.Query("groupby") {
	case "alert_type":
		query.Select("concat(alert_infos.labels ->> '$.gems_alert_resource', '.', alert_infos.labels ->> '$.gems_alert_rule') as group_value, count(alert_infos.fingerprint) as count").
			Group("concat(alert_infos.labels ->> '$.gems_alert_resource', '.', alert_infos.labels ->> '$.gems_alert_rule')")
	case "project_name":
		query.Select("alert_infos.project_name as group_value, count(alert_infos.fingerprint) as count").
			Group("alert_infos.project_name")
	default:
		handlers.NotOK(c, fmt.Errorf("groupby not valid"))
		return
	}

	ret := []TableRet{}
	// 最多的10条
	if err := query.Order("count desc").Limit(10).Scan(&ret).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// SearchAlert 搜索告警
// @Tags         Alert
// @Summary      搜索告警
// @Description  搜索告警
// @Accept       json
// @Produce      json
// @Param        tenant_id    path      string                                                                       true   "租户ID，所有租户为_all"
// @Param        project      query     string                                                                       false  "项目名，默认所有"
// @Param        environment  query     string                                                                       false  "环境名，默认所有"
// @Param        cluster      query     string                                                                       false  "集群名，默认所有"
// @Param        namespace    query     string                                                                       false  "命名空间，默认所有"
// @Param        alertname    query     string                                                                       false  "告警名，默认所有"
// @Param        resource     query     string                                                                       false  "告警资源，默认所有"
// @Param        rule         query     string                                                                       false  "告警指标，默认所有"
// @Param        labelpairs   query     string                                                                       false  "标签键值对,不支持正则 eg. labelpairs[host]=k8s-master&labelpairs[pod]=pod1"
// @Param        start        query     string                                                                       false  "开始时间"
// @Param        end          query     string                                                                       false  "结束时间"
// @Param        status       query     string                                                                       false  "状态(firing, resolved)"
// @Param        page         query     int                                                                          false  "page"
// @Param        size         query     int                                                                          false  "size"
// @Success      200          {object}  handlers.ResponseStruct{Data=pagination.PageData{List=[]AlertMessageGroup}}  "resp"
// @Router       /v1/alerts/tenant/{tenant_id}/search [get]
// @Security     JWT
func (h *AlertsHandler) SearchAlert(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	tenantID := c.Param("tenant_id")
	project := c.Query("project")
	environment := c.Query("environment")
	cluster := c.Query("cluster")
	namespace := c.Query("namespace")
	alertName := c.Query("alertname")
	resoure := c.Query("resource")
	rule := c.Query("rule")
	labelpairs := c.QueryMap("labelpairs")
	start := c.Query("start")
	end := c.Query("end")
	status := c.Query("status")
	var total int64
	var messages []models.AlertMessage

	query := h.GetDB().Preload("AlertInfo").Joins("AlertInfo")
	if tenantID != "_all" {
		t := models.Tenant{}
		h.GetDB().First(&t, "id = ?", tenantID)
		query.Where("tenant_name = ?", t.TenantName)
	}
	if project != "" {
		query.Where("project_name = ?", project)
	}
	if environment != "" {
		query.Where("environment_name = ?", environment)
	}
	if cluster != "" {
		query.Where("cluster_name = ?", cluster)
	}
	if namespace != "" {
		query.Where("namespace = ?", namespace)
	}
	if alertName != "" {
		query.Where("name = ?", alertName)
	}
	if resoure != "" {
		query.Where(fmt.Sprintf(`labels -> '$.%s' = ?`, prometheus.AlertResourceLabel), resoure)
	}
	if rule != "" {
		query.Where(fmt.Sprintf(`labels -> '$.%s' = ?`, prometheus.AlertRuleLabel), rule)
	}
	for k, v := range labelpairs {
		query.Where(fmt.Sprintf(`labels -> '$.%s' = ?`, k), v)
	}

	if start != "" {
		query.Where("created_at >= ?", start)
	}
	if end != "" {
		query.Where("created_at <= ?", end)
	}
	if status != "" {
		query.Where("status = ?", status)
	}

	// 总数, 不能直接count，需要count临时表
	if err := query.Model(&models.AlertMessage{}).Count(&total).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	// 分页
	if err := query.Order("created_at desc").Limit(size).Offset((page - 1) * size).Find(&messages).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, messages, int64(page), int64(size)))
}

func fillInPoints(points []AlertGraph, samples []model.SamplePair) {
	var i, j int
	for i < len(points) && j < len(samples) {
		if points[i].DateTimeStamp < int64(samples[j].Timestamp) {
			i++
			continue
		}
		if points[i].DateTimeStamp == int64(samples[j].Timestamp) {
			samples[j].Value = model.SampleValue(points[i].Count)
			i++
			j++
			continue
		}
		if points[i].DateTimeStamp > int64(samples[j].Timestamp) {
			j++
			continue
		}
	}
}

// 每天一个sample
func newDefaultSamplePair(start, end time.Time) []model.SamplePair {
	ret := []model.SamplePair{}
	for tmp := start; tmp.Before(end); tmp = tmp.Add(24 * time.Hour) {
		ret = append(ret, model.SamplePair{
			Timestamp: model.Time(tmp.UnixMilli()),
			Value:     0,
		})
	}
	return ret
}

var (
	silenceCommentPrefix = "fingerprint-"

	forever = time.Date(9893, time.December, 26, 0, 0, 0, 0, time.UTC) // 伟人8000年诞辰
)

// AlertHistory 告警黑名单
// @Tags         Alert
// @Summary      告警黑名单
// @Description  告警黑名单
// @Accept       json
// @Produce      json
// @Param        cluster    query     string                                                                      false  "集群, 默认所有"
// @Param        namespace  query     string                                                                      false  "命名空间, 默认所有"
// @Param        page       query     int                                                                         false  "page"
// @Param        size       query     int                                                                         false  "size"
// @Success      200        {object}  handlers.ResponseStruct{Data=pagination.PageData{List=[]models.AlertInfo}}  "resp"
// @Router       /v1/alerts/blacklist [get]
// @Security     JWT
func (h *AlertsHandler) ListBlackList(c *gin.Context) {
	cluster := c.Query("cluster")
	namespace := c.Query("namespace")
	var ret []models.AlertInfo
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model: "AlertInfo",
		Where: []*handlers.QArgs{
			handlers.Args("length(silence_creator) > 0"),
		},
	}
	if cluster != "" {
		cond.Where = append(cond.Where, handlers.Args("cluster_name = ?", cluster))
	}
	if namespace != "" {
		cond.Where = append(cond.Where, handlers.Args("namespace = ?", namespace))
	}

	total, page, size, err := query.PageList(h.GetDB().Order("silence_updated_at desc"), cond, &ret)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	monitoropts := new(prometheus.MonitorOptions)
	h.DynamicConfig.Get(c.Request.Context(), monitoropts)
	for i := range ret {
		if ret[i].SilenceEndsAt.Equal(forever) {
			ret[i].SilenceEndsAt = nil
		}
		ret[i].LabelMap = make(map[string]string)
		_ = json.Unmarshal(ret[i].Labels, &ret[i].LabelMap)
		ret[i].Summary = h.formatBlackListSummary(ret[i].LabelMap, monitoropts)
	}
	handlers.OK(c, handlers.Page(total, ret, int64(page), int64(size)))
}

func (h *AlertsHandler) formatBlackListSummary(labels map[string]string, opts *prometheus.MonitorOptions) string {
	resKey := labels[prometheus.AlertResourceLabel]
	ruleKey := labels[prometheus.AlertRuleLabel]
	res := opts.Resources[resKey]
	rule := res.Rules[ruleKey]
	labelStr := fmt.Sprintf("%s: [集群:%s] ", labels[prometheus.AlertNameLabel], labels[prometheus.AlertClusterKey])
	for _, l := range rule.Labels {
		labelStr += fmt.Sprintf("[%s:%s] ", l, labels[l])
	}
	return fmt.Sprintf("%s%s%s告警", labelStr, res.ShowName, rule.ShowName)
}

// AlertHistory 加入/更新告警黑名单
// @Tags         Alert
// @Summary      加入/更新告警黑名单
// @Description  加入/更新告警黑名单
// @Accept       json
// @Produce      json
// @Param        form  body      models.AlertInfo                      true  "黑名单详情，必传AlertInfo.Fingerprint"
// @Success      200   {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/alerts/blacklist [post]
// @Security     JWT
func (h *AlertsHandler) AddToBlackList(c *gin.Context) {
	if err := h.withBlackListReq(c, func(req models.AlertInfo) error {
		return h.GetDB().Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&req).Error; err != nil {
				return err
			}
			ctx := c.Request.Context()
			cli, err := h.GetAgents().ClientOf(ctx, req.ClusterName)
			if err != nil {
				return err
			}
			return cli.Extend().CreateOrUpdateSilenceIfNotExist(c.Request.Context(), req)
		})
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// AlertHistory 移除告警黑名单
// @Tags         Alert
// @Summary      移除告警黑名单
// @Description  移除告警黑名单
// @Accept       json
// @Produce      json
// @Param        fingerprint  path      string                                true  "告警指纹"
// @Success      200          {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/alerts/blacklist/{fingerprint} [delete]
// @Security     JWT
func (h *AlertsHandler) RemoveInBlackList(c *gin.Context) {
	req := models.AlertInfo{
		Fingerprint: c.Param("fingerprint"),
	}
	h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&req, "fingerprint = ?", req.Fingerprint).Error; err != nil {
			return err
		}
		req.SilenceCreator = ""
		req.SilenceStartsAt = nil
		req.SilenceUpdatedAt = nil
		req.SilenceEndsAt = nil
		if err := tx.Save(&req).Error; err != nil {
			return err
		}
		// 先decode labels
		if err := json.Unmarshal(req.Labels, &req.LabelMap); err != nil {
			return err
		}
		ctx := c.Request.Context()
		cli, err := h.GetAgents().ClientOf(ctx, req.ClusterName)
		if err != nil {
			return err
		}
		return cli.Extend().DeleteSilenceIfExist(ctx, req)
	})
	handlers.OK(c, "ok")
}

func (h *AlertsHandler) withBlackListReq(c *gin.Context, f func(req models.AlertInfo) error) error {
	u, exist := h.GetContextUser(c)
	if !exist {
		return fmt.Errorf("not login")
	}

	req := models.AlertInfo{}
	if err := c.BindJSON(&req); err != nil {
		return err
	}
	if err := h.GetDB().First(&req, "fingerprint = ?", req.Fingerprint).Error; err != nil {
		return err
	}
	req.SilenceCreator = u.GetUsername()
	req.LabelMap = map[string]string{}
	if err := json.Unmarshal(req.Labels, &req.LabelMap); err != nil {
		return err
	}

	now := time.Now()
	req.SilenceUpdatedAt = &now
	if req.SilenceStartsAt == nil {
		req.SilenceStartsAt = &now
	}
	if req.SilenceEndsAt == nil {
		req.SilenceEndsAt = &forever
	}
	return f(req)
}
