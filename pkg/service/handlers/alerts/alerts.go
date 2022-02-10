package alerthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus/alertmanager/pkg/labels"
	alertmanagertypes "github.com/prometheus/alertmanager/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/kubeclient"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
// @Tags Alert
// @Summary  获取AlertRule列表
// @Description 获取AlertRule列表
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Success 200 {object} handlers.ResponseStruct{Data=[]prometheus.AlertRule} "规则"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert [get]
// @Security JWT
func (h *AlertsHandler) ListAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	ctx := c.Request.Context()

	promeAlertRules := map[string]prometheus.RealTimeAlertRule{}
	done := make(chan struct{})
	go func() {
		kubeclient.DoRequest(http.MethodGet, cluster, "/custom/prometheus/v1/alertrule", nil, &promeAlertRules)
		close(done)
	}()

	ret := prometheus.AlertRuleList{}
	if namespace == allNamespace {
		ruleList := monitoringv1.PrometheusRuleList{}
		configList := v1alpha1.AlertmanagerConfigList{}

		configNamespaceMap := map[string]*v1alpha1.AlertmanagerConfig{}
		silenceNamespaceMap := map[string][]*alertmanagertypes.Silence{}
		if err := kubeclient.Execute(ctx, cluster, func(tc *agents.TypedClient) error {
			if err := tc.List(ctx, &ruleList, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(prometheus.PrometheusRuleSelector)); err != nil {
				return err
			}
			if err := tc.List(ctx, &configList, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(prometheus.AlertmanagerConfigSelector)); err != nil {
				return err
			}

			// 按照namespace分组
			for _, v := range configList.Items {
				if v.Name == prometheus.AlertmanagerConfigName {
					configNamespaceMap[v.Namespace] = v
				}
			}

			// 按照namespace分组
			allSilences, err := kubeclient.GetClient().ListSilences(cluster, "")
			if err != nil {
				return err
			}
			for _, silence := range allSilences {
				for _, m := range silence.Matchers {
					if m.Name == prometheus.AlertNamespaceLabel {
						silenceNamespaceMap[m.Value] = append(silenceNamespaceMap[m.Value], silence)
					}
				}
			}

			for _, rule := range ruleList.Items {
				if rule.Name == prometheus.PrometheusRuleName {
					amconfig, ok := configNamespaceMap[rule.Namespace]
					if !ok {
						log.Warnf("alertmanager config: %s not found", rule.Name)
						continue
					}
					raw := &prometheus.RawAlertResource{
						PrometheusRule:     rule,
						AlertmanagerConfig: amconfig,
						Silences:           silenceNamespaceMap[rule.Namespace],
					}

					alerts, err := raw.ToAlerts(false)
					if err != nil {
						return err
					}
					ret = append(ret, alerts...)
				}
			}
			return nil
		}); err != nil {
			handlers.NotOK(c, err)
			return
		}
	} else {
		raw, err := getRawAlertResource(ctx, cluster, namespace)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		ret, err = raw.ToAlerts(false)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
	}

	<-done
	for i := range ret {
		key := prometheus.RealTimeAlertKey(ret[i].Namespace, ret[i].Name)
		if promRule, ok := promeAlertRules[key]; ok {
			ret[i].State = promRule.State
		} else {
			ret[i].State = "inactive"
		}
	}

	sort.Sort(ret)
	handlers.OK(c, ret)
}

// GetAlertRule AlertRule详情
// @Tags Alert
// @Summary  AlertRule详情
// @Description AlertRule详情
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=prometheus.AlertRule} "规则"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name} [get]
// @Security JWT
func (h *AlertsHandler) GetAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")

	ctx := c.Request.Context()
	raw, err := getRawAlertResource(ctx, cluster, namespace)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	alerts, err := raw.ToAlerts(true)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	// get realtime alert
	u := fmt.Sprintf("/custom/prometheus/v1/alertrule?name=%s", name)
	promeAlertRules := map[string]prometheus.RealTimeAlertRule{}
	kubeclient.DoRequest(http.MethodGet, cluster, u, nil, &promeAlertRules)
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
		handlers.NotOK(c, fmt.Errorf("alert %s not found", name))
		return
	}
	alerts[index].State = realtimeAertRule.State
	alerts[index].RealTimeAlerts = realtimeAertRule.Alerts
	handlers.OK(c, alerts[index])
}

// GetAlertRule 禁用AlertRule
// @Tags Alert
// @Summary  禁用AlertRule
// @Description 禁用AlertRule
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=string} ""
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name}/actions/disable [post]
// @Security JWT
func (h *AlertsHandler) DisableAlertRule(c *gin.Context) {
	h.SetAuditData(c, "禁用", "告警规则", c.Param("name"))
	h.SetExtraAuditDataByClusterNamespace(c, c.Param("cluster"), c.Param("namespace"))
	if err := kubeclient.GetClient().CreateSilenceIfNotExist(c.Param("cluster"), c.Param("namespace"), c.Param("name")); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "")
}

// GetAlertRule 启用AlertRule
// @Tags Alert
// @Summary  启用AlertRule
// @Description 启用AlertRule
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=string} ""
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name}/actions/enable [post]
// @Security JWT
func (h *AlertsHandler) EnableAlertRule(c *gin.Context) {
	h.SetAuditData(c, "启用", "告警规则", c.Param("name"))
	h.SetExtraAuditDataByClusterNamespace(c, c.Param("cluster"), c.Param("namespace"))
	if err := kubeclient.GetClient().DeleteSilenceIfExist(c.Param("cluster"), c.Param("namespace"), c.Param("name")); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "")
}

// CreateAlertRule 创建AlertRule
// @Tags Alert
// @Summary  创建AlertRule
// @Description 创建AlertRule
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param form body prometheus.AlertRuleList true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "规则"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert [post]
// @Security JWT
func (h *AlertsHandler) CreateAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	req := prometheus.AlertRule{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	req.Namespace = namespace
	h.SetAuditData(c, "创建", "告警规则", req.Name)
	if err := req.CheckAndModify(); err != nil {
		handlers.NotOK(c, err)
		return
	}

	// get、update、commit
	ctx := c.Request.Context()
	raw, err := getRawAlertResource(ctx, cluster, namespace)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := raw.ModifyAlertRule(req, prometheus.Add); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := commitToK8s(ctx, cluster, raw); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

// ModifyAlertRule 修改AlertRule
// @Tags Alert
// @Summary  修改AlertRule
// @Description 修改AlertRule
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Param form body prometheus.AlertRuleList true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "规则"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name} [put]
// @Security JWT
func (h *AlertsHandler) ModifyAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	req := prometheus.AlertRule{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	req.Namespace = namespace
	h.SetAuditData(c, "更新", "告警规则", req.Name)
	if err := req.CheckAndModify(); err != nil {
		handlers.NotOK(c, err)
		return
	}

	// get、update、commit
	ctx := c.Request.Context()
	raw, err := getRawAlertResource(ctx, cluster, namespace)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := raw.ModifyAlertRule(req, prometheus.Update); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := commitToK8s(ctx, cluster, raw); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

// DeleteAlertRule 删除AlertRule
// @Tags Alert
// @Summary  修改AlertRule
// @Description 修改AlertRule
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "规则"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name} [delete]
// @Security JWT
func (h *AlertsHandler) DeleteAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	req := prometheus.AlertRule{
		Namespace: namespace,
		Name:      name,
	}

	// get、update、commit
	ctx := c.Request.Context()
	raw, err := getRawAlertResource(ctx, cluster, namespace)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := raw.ModifyAlertRule(req, prometheus.Delete); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := commitToK8s(ctx, cluster, raw); err != nil {
		handlers.NotOK(c, err)
		return
	}

	// 清理silence规则
	if err := kubeclient.GetClient().DeleteSilenceIfExist(cluster, namespace, name); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
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
// @Tags Alert
// @Summary  告警历史
// @Description 告警历史
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Param status query string false "告警状态(resolved, firing), 为空则是所有状态"
// @Param CreatedAt_gte query string false "CreatedAt_gte"
// @Param CreatedAt_lte query string false "CreatedAt_lte"
// @Param page query int false "page"
// @Param size query int false "size"
// @Success 200 {object} handlers.ResponseStruct{Data=[]AlertMessageGroup} "规则"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name}/history [get]
// @Security JWT
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
// @Tags Alert
// @Summary  重复的告警记录
// @Description 重复的告警记录
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Param fingerprint query string true "告警指纹"
// @Param starts_at query string true "告警开始时间"
// @Param page query int false "page"
// @Param size query int false "size"
// @Success 200 {object} handlers.ResponseStruct{Data=[]models.AlertMessage} "规则"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/alert/{name}/repeats [get]
// @Security JWT
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

// SearchAlert 搜索告警
// @Tags Alert
// @Summary  搜索告警
// @Description 搜索告警
// @Accept json
// @Produce json
// @Param cluster query string false "集群,默认所有"
// @Param namespace query string false "告警命名空间，默认所有"
// @Param alertname query string false "告警规则名，默认所有"
// @Param resource query string false "告警资源，默认所有"
// @Param rule query string false "告警指标，默认所有"
// @Param labelpairs query string false "标签键值对,不支持正则 eg. labelpairs[host]=k8s-master&labelpairs[pod]=pod1"
// @Param start query string false "开始时间"
// @Param end query string false "结束时间"
// @Param status query string false "状态(firing, resolved)"
// @Param page query int false "page"
// @Param size query int false "size"
// @Success 200 {object} handlers.ResponseStruct{Data=pagination.PageData{List=[]AlertMessageGroup}} "resp"
// @Router /v1/alerts/search [get]
// @Security JWT
func (h *AlertsHandler) SearchAlert(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	cluster := c.Query("cluster")
	namespace := c.Query("namespace")
	alertName := c.Query("alertname")
	resoure := c.Query("resource")
	rule := c.Query("rule")
	labelpairs := c.QueryMap("labelpairs")
	start := c.Query("start")
	end := c.Query("end")
	var total int64
	var messages []AlertMessageGroup
	// 若同时有resolved和firing。展示resolved
	// select max(status) from alert_messages
	// output: resolved
	tmpQuery := h.GetDB().Table("alert_messages").
		Select(`alert_messages.fingerprint, 
			max(starts_at) as starts_at,
			max(ends_at) as ends_at, 
			max(value) as value, 
			max(message) as message, 
			max(created_at) as created_at, 
			max(status) as status,
			max(labels) as labels,
			max(silence_creator) as silence_creator,
			count(created_at) as count`).
		Joins("join alert_infos on alert_messages.fingerprint = alert_infos.fingerprint").
		Group("alert_messages.fingerprint")

	if cluster != "" {
		tmpQuery.Where("cluster_name = ?", cluster)
	}
	if namespace != "" {
		tmpQuery.Where("namespace = ?", namespace)
	}
	if alertName != "" {
		tmpQuery.Where("name = ?", alertName)
	}
	if resoure != "" {
		tmpQuery.Where(fmt.Sprintf(`labels -> '$.%s' = ?`, prometheus.AlertResourceLabel), resoure)
	}
	if rule != "" {
		tmpQuery.Where(fmt.Sprintf(`labels -> '$.%s' = ?`, prometheus.AlertRuleLabel), rule)
	}
	for k, v := range labelpairs {
		tmpQuery.Where(fmt.Sprintf(`labels -> '$.%s' = ?`, k), v)
	}
	if start != "" {
		tmpQuery.Where("created_at >= ?", start)
	}
	if end != "" {
		tmpQuery.Where("created_at <= ?", end)
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

var (
	alertProxyHeader = map[string]string{
		"namespace": "gemcloud-monitoring-system",
		"service":   "alertmanager",
		"port":      "9093",
	}
	silenceCommentPrefix = "fingerprint-"

	forever = time.Date(9893, time.December, 26, 0, 0, 0, 0, time.UTC) // 伟人8000年诞辰
)

// AlertHistory 告警黑名单
// @Tags Alert
// @Summary  告警黑名单
// @Description 告警黑名单
// @Accept json
// @Produce json
// @Param cluster query string false "集群, 默认所有"
// @Param namespace query string false "命名空间, 默认所有"
// @Param page query int false "page"
// @Param size query int false "size"
// @Success 200 {object} handlers.ResponseStruct{Data=pagination.PageData{List=[]models.AlertInfo}} "resp"
// @Router /v1/alerts/blacklist [get]
// @Security JWT
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

	for i := range ret {
		if ret[i].SilenceEndsAt.Equal(forever) {
			ret[i].SilenceEndsAt = nil
		}
		ret[i].LabelMap = make(map[string]string)
		_ = json.Unmarshal(ret[i].Labels, &ret[i].LabelMap)
		ret[i].Summary = formatBlackListSummary(ret[i].LabelMap)
	}
	handlers.OK(c, handlers.Page(total, ret, int64(page), int64(size)))
}

func formatBlackListSummary(labels map[string]string) string {
	resKey := labels[prometheus.AlertResourceLabel]
	ruleKey := labels[prometheus.AlertRuleLabel]
	res := prometheus.GetGemsMetricConfig(true).Resources[resKey]
	rule := res.Rules[ruleKey]
	labelStr := fmt.Sprintf("%s: [集群:%s] ", labels[prometheus.AlertNameLabel], labels[prometheus.AlertClusterKey])
	for _, l := range rule.Labels {
		labelStr += fmt.Sprintf("[%s:%s] ", l, labels[l])
	}
	return fmt.Sprintf("%s%s%s告警", labelStr, res.ShowName, rule.ShowName)
}

// AlertHistory 加入/更新告警黑名单
// @Tags Alert
// @Summary  加入/更新告警黑名单
// @Description 加入/更新告警黑名单
// @Accept json
// @Produce json
// @Param form body models.AlertInfo true "黑名单详情，必传AlertInfo.Fingerprint"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "resp"
// @Router /v1/alerts/blacklist [post]
// @Security JWT
func (h *AlertsHandler) AddToBlackList(c *gin.Context) {
	if err := h.withBlackListReq(c, func(req models.AlertInfo) error {
		return h.GetDB().Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&req).Error; err != nil {
				return err
			}
			return h.createOrUpdateSilenceIfNotExist(c.Request.Context(), req.ClusterName, req)
		})
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// AlertHistory 移除告警黑名单
// @Tags Alert
// @Summary  移除告警黑名单
// @Description 移除告警黑名单
// @Accept json
// @Produce json
// @Param fingerprint path string true "告警指纹"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "resp"
// @Router /v1/alerts/blacklist/{fingerprint} [delete]
// @Security JWT
func (h *AlertsHandler) RemoveInBlackList(c *gin.Context) {
	req := models.AlertInfo{
		Fingerprint: c.Param("fingerprint"),
	}
	h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := h.GetDB().First(&req, "fingerprint = ?", req.Fingerprint).Error; err != nil {
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
		return h.deleteSilenceIfExist(c.Request.Context(), req.ClusterName, req)
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
	req.SilenceCreator = u.Username
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

func (h *AlertsHandler) listSilences(ctx context.Context, cluster string, labels map[string]string) ([]alertmanagertypes.Silence, error) {
	agentcli, err := h.ClientOf(ctx, cluster)
	if err != nil {
		return nil, err
	}
	allSilences := []alertmanagertypes.Silence{}
	cli := resty.NewWithClient(agentcli.HttpClient.Client).R().
		SetHeaders(alertProxyHeader)
	values := url.Values{}
	for k, v := range labels {
		values.Add("filter", fmt.Sprintf(`%s="%s"`, k, v))
	}
	// 由于filter是数组，不能直接SetQueryParam
	cli.SetQueryParamsFromValues(values)

	_, err = cli.SetResult(&allSilences).
		Get(fmt.Sprintf("%s/v1/service-proxy/api/v2/silences", agentcli.BaseAddr))
	if err != nil {
		return nil, fmt.Errorf("list silence by %v, %w", labels, err)
	}
	// 只返回活跃的
	ret := []alertmanagertypes.Silence{}
	for _, v := range allSilences {
		if v.Status.State == alertmanagertypes.SilenceStateActive &&
			strings.HasPrefix(v.Comment, silenceCommentPrefix) {
			ret = append(ret, v)
		}
	}
	return ret, nil
}

func (h *AlertsHandler) createOrUpdateSilenceIfNotExist(ctx context.Context, cluster string, req models.AlertInfo) error {
	silenceList, err := h.listSilences(ctx, cluster, req.LabelMap)
	if err != nil {
		return err
	}
	agentcli, err := h.ClientOf(ctx, cluster)
	if err != nil {
		return err
	}
	silence := convertBlackListToSilence(req)
	cli := resty.NewWithClient(agentcli.HttpClient.Client).R().
		SetHeaders(alertProxyHeader)
	switch len(silenceList) {
	case 0:
		break
	case 1:
		silence.ID = silenceList[0].ID
	default:
		return fmt.Errorf("too many silences for alert: %v", req)
	}
	resp, err := cli.SetBody(silence).
		Post(fmt.Sprintf("%s/v1/service-proxy/api/v2/silences", agentcli.BaseAddr))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("failed to create silence, code %d, body %s", resp.StatusCode(), string(resp.Body()))
	}
	return nil
}

func (h *AlertsHandler) deleteSilenceIfExist(ctx context.Context, cluster string, req models.AlertInfo) error {
	silenceList, err := h.listSilences(ctx, cluster, req.LabelMap)
	if err != nil {
		return err
	}
	switch len(silenceList) {
	case 0:
		return nil
	case 1:
		agentcli, err := h.ClientOf(ctx, cluster)
		if err != nil {
			return err
		}
		_, err = resty.NewWithClient(agentcli.HttpClient.Client).R().
			SetHeaders(alertProxyHeader).
			Delete(fmt.Sprintf("%s/v1/service-proxy/api/v2/silences/%s", agentcli.BaseAddr, silenceList[0].ID))
		return err
	default:
		return fmt.Errorf("too many silences for alert: %v", req)
	}
}

func convertBlackListToSilence(info models.AlertInfo) alertmanagertypes.Silence {
	ret := alertmanagertypes.Silence{
		StartsAt:  *info.SilenceStartsAt,
		EndsAt:    *info.SilenceEndsAt,
		UpdatedAt: *info.SilenceUpdatedAt,
		CreatedBy: info.SilenceCreator,
		Comment:   fmt.Sprintf("%s%s", silenceCommentPrefix, info.Fingerprint), // comment存指纹，以便取出时做匹配
		Matchers:  make(labels.Matchers, len(info.LabelMap)),
	}
	index := 0
	for k, v := range info.LabelMap {
		ret.Matchers[index] = &labels.Matcher{
			Type:  labels.MatchEqual,
			Name:  k,
			Value: v,
		}
		index++
	}
	return ret
}
