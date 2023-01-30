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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus/alertmanager/pkg/labels"
	alerttypes "github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	promlabels "github.com/prometheus/prometheus/pkg/labels"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/channels"
	"kubegems.io/kubegems/pkg/utils/prometheus/promql"
	"kubegems.io/kubegems/pkg/utils/set"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// DisableAlertRule 禁用告警规则
// @Tags        Observability
// @Summary     禁用告警规则
// @Description 禁用告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       name      path     string                               true "name"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/alerts/{name}/actions/disable [post]
// @Security    JWT
func (h *ObservabilityHandler) DisableAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	action := i18n.Sprintf(context.TODO(), "disable")
	module := i18n.Sprintf(context.TODO(), "alert rule")
	h.SetAuditData(c, action, module, name)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			alertrule := &models.AlertRule{}
			if err := tx.First(&alertrule, "cluster = ? and namespace = ? and name = ?", cluster, namespace, name).Error; err != nil {
				return err
			}
			if err := tx.Model(alertrule).Update("is_open", false).Error; err != nil {
				return err
			}
			return createSilenceIfNotExist(ctx, namespace, name, cli)
		})
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DisableAlertRule 启用告警规则
// @Tags        Observability
// @Summary     启用告警规则
// @Description 启用告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       name      path     string                               true "name"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/alerts/{name}/actions/enable [post]
// @Security    JWT
func (h *ObservabilityHandler) EnableAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	action := i18n.Sprintf(context.TODO(), "enable")
	module := i18n.Sprintf(context.TODO(), "alert rule")
	h.SetAuditData(c, action, module, name)
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			alertrule := &models.AlertRule{}
			if err := tx.First(&alertrule, "cluster = ? and namespace = ? and name = ?", cluster, namespace, name).Error; err != nil {
				return err
			}
			if err := tx.Model(alertrule).Update("is_open", true).Error; err != nil {
				return err
			}
			return deleteSilenceIfExist(ctx, namespace, name, cli)
		})
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// GenerateAlertMessage 生成告警规则消息
// @Tags        Observability
// @Summary     生成告警规则消息
// @Description 生成告警规则消息
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       name      path     string                               true "name"
// @Param       form      body     models.AlertRule                     true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/alerts/{name}/actions/message [post]
// @Security    JWT
func (h *ObservabilityHandler) GenerateAlertMessage(c *gin.Context) {
	var msg string
	if err := h.Process(func() error {
		req := &models.AlertRule{}
		err := c.BindJSON(req)
		if err != nil {
			return err
		}
		req.Cluster = c.Param("cluster")
		req.Namespace = c.Param("namespace")
		if req.Namespace == "" {
			return errors.Errorf("namespace can't be empty")
		}
		// set tpl
		if req.PromqlGenerator != nil {
			tpl, err := h.GetDataBase().FindPromqlTpl(req.PromqlGenerator.Scope, req.PromqlGenerator.Resource, req.PromqlGenerator.Rule)
			if err != nil {
				return err
			}
			req.PromqlGenerator.Tpl = tpl
		}
		msg, err = genarateMessage(req)
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, msg)
}

// SyncAlertRule 同步告警规则
// @Tags        Observability
// @Summary     同步告警规则
// @Description 同步告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                          true "cluster, 支持_all"
// @Param       namespace path     string                                          true "namespace, 支持_all"
// @Param       name      path     string                                          true "name, 支持_all"
// @Success     200       {object} handlers.ResponseStruct{Data=map[string]string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/alerts/{name}/actions/sync [post]
// @Security    JWT
func (h *ObservabilityHandler) SyncAlertRule(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	statusMap := map[string]string{}
	if err := h.Process(func() error {
		if cluster == "_all" || namespace == "_all" {
			user, _ := h.GetContextUser(c)
			if user.GetSystemRoleID() != 1 {
				return errors.Errorf("only system admin can sync all cluster or namespace")
			}
		}
		alertrules := []*models.AlertRule{}
		ctx := c.Request.Context()
		query := h.GetDB().WithContext(ctx).Preload("Receivers.AlertChannel")

		setWhere := func(fieldname, fieldvalue string) {
			if fieldvalue != "_all" {
				query.Where("%s = ?", fieldvalue)
			}
		}
		setWhere("cluster", cluster)
		setWhere("namespace", namespace)
		setWhere("name", name)
		if err := query.Find(&alertrules).Error; err != nil {
			return err
		}

		for _, v := range alertrules {
			syncAlertRule := func(alertrule *models.AlertRule) string {
				cli, err := h.GetAgents().ClientOf(ctx, alertrule.Cluster)
				if err != nil {
					return fmt.Sprintf("client of: %s failed, %v", cli.Name(), err)
				}
				p := NewAlertRuleProcessor(cli, h.GetDataBase())
				if err := p.SyncAlertRule(ctx, v); err != nil {
					return err.Error()
				}
				return "success"
			}
			status := syncAlertRule(v)
			log.Info("sync alertrule", "name", v.FullName(), "status", status)
			statusMap[v.FullName()] = status
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, statusMap)
}

// ImportAlertRules 导入告警规则
// @Tags        Observability
// @Summary     导入告警规则
// @Description 导入告警规则
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       namespace path     string                               true "namespace"
// @Param       form      body     []models.AlertRule                   true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/alerts-import [post]
// @Security    JWT
func (h *ObservabilityHandler) ImportAlertRules(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	if err := h.withAlertRuleProcessor(c.Request.Context(), cluster, func(ctx context.Context, p *AlertRuleProcessor) error {
		alertrules := []*models.AlertRule{}
		bts, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(bts, &alertrules); err != nil {
			return err
		}

		for _, v := range alertrules {
			v.Cluster = cluster
			v.Namespace = namespace
			if v.AlertType == "" {
				return fmt.Errorf("alertrule %s type can't be null", v.FullName())
			}
			// replace __namespace__
			v.Expr = strings.ReplaceAll(v.Expr, "__namespace__", v.Namespace)
			if err := p.MutateAlertRule(ctx, v); err != nil {
				return errors.Wrapf(err, "mutate alertrule: %s", v.FullName())
			}
			if err := p.CreateAlertRule(ctx, v); err != nil {
				return errors.Wrapf(err, "create alertrule: %s", v.FullName())
			}
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// Deprecated. use cli.Extend().ListSilences instead
func listSilences(ctx context.Context, namespace string, cli agents.Client) ([]*alerttypes.Silence, error) {
	silences := []*alerttypes.Silence{}
	url := "/custom/alertmanager/v1/silence"
	if namespace != "" {
		url = fmt.Sprintf(`%s?filter=%s="%s"`, url, prometheus.AlertNamespaceLabel, namespace)
	}
	if err := cli.DoRequest(ctx, agents.Request{
		Path: url,
		Into: agents.WrappedResponse(&silences),
	}); err != nil {
		return nil, err
	}

	// 只返回活跃的
	var ret []*alerttypes.Silence
	for _, v := range silences {
		if v.Status.State == alerttypes.SilenceStateActive && strings.HasPrefix(v.Comment, prometheus.SilenceCommentForAlertrulePrefix) {
			ret = append(ret, v)
		}
	}
	return ret, nil
}

func getSilence(ctx context.Context, namespace, alertName string, cli agents.Client) (*alerttypes.Silence, error) {
	silences, err := listSilences(ctx, namespace, cli)
	if err != nil {
		return nil, err
	}

	// 只返回活跃的
	actives := []*alerttypes.Silence{}
	for _, silence := range silences {
		if silence.Status.State == alerttypes.SilenceStateActive &&
			silence.Matchers.Matches(model.LabelSet{
				prometheus.AlertNamespaceLabel: model.LabelValue(namespace),
				prometheus.AlertNameLabel:      model.LabelValue(alertName),
			}) { // 名称匹配
			actives = append(actives, silence)
		}
	}
	if len(actives) == 0 {
		return nil, nil
	}
	if len(actives) > 1 {
		return nil, i18n.Errorf(ctx, "some error happend, more than one silence rule founded, please contact admin")
	}

	return actives[0], nil
}

func createSilenceIfNotExist(ctx context.Context, namespace, alertName string, cli agents.Client) error {
	silence, err := getSilence(ctx, namespace, alertName, cli)
	if err != nil {
		return err
	}
	// 不存在，创建
	if silence == nil {
		silence = &alerttypes.Silence{
			Comment:   fmt.Sprintf("silence for %s", alertName),
			CreatedBy: alertName,
			Matchers: labels.Matchers{
				&labels.Matcher{
					Type:  labels.MatchEqual,
					Name:  prometheus.AlertNamespaceLabel,
					Value: namespace,
				},
				&labels.Matcher{
					Type:  labels.MatchEqual,
					Name:  prometheus.AlertNameLabel,
					Value: alertName,
				},
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().AddDate(1000, 0, 0), // 100年
		}

		// create
		return cli.DoRequest(ctx, agents.Request{
			Method: http.MethodPost,
			Path:   "/custom/alertmanager/v1/silence/_/actions/create",
			Body:   silence,
		})
	}
	return nil
}

func deleteSilenceIfExist(ctx context.Context, namespace, alertName string, cli agents.Client) error {
	silence, err := getSilence(ctx, namespace, alertName, cli)
	if err != nil {
		return err
	}
	// 存在，删除
	if silence != nil {
		values := url.Values{}
		values.Add("id", silence.ID)

		return cli.DoRequest(ctx, agents.Request{
			Method: http.MethodDelete,
			Path:   "/custom/alertmanager/v1/silence/_/actions/delete",
			Query:  values,
		})
	}
	return nil
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
// @Tags        Observability
// @Summary     告警历史
// @Description 告警历史
// @Accept      json
// @Produce     json
// @Param       cluster       path     string                                            true  "cluster"
// @Param       namespace     path     string                                            true  "namespace"
// @Param       name          path     string                                            true  "name"
// @Param       status        query    string                                            false "告警状态(resolved, firing),  为空则是所有状态"
// @Param       CreatedAt_gte query    string                                            false "CreatedAt_gte"
// @Param       CreatedAt_lte query    string                                            false "CreatedAt_lte"
// @Param       page          query    int                                               false "page"
// @Param       size          query    int                                               false "size"
// @Success     200           {object} handlers.ResponseStruct{Data=[]AlertMessageGroup} "规则"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/alerts/{name}/history [get]
// @Security    JWT
func (h *ObservabilityHandler) AlertHistory(c *gin.Context) {
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
	ctx := c.Request.Context()
	tmpQuery := h.GetDB().WithContext(ctx).Table("alert_messages").
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
	query := h.GetDB().WithContext(ctx).Table("(?) as t", tmpQuery)
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
// @Tags        Observability
// @Summary     重复的告警记录
// @Description 重复的告警记录
// @Accept      json
// @Produce     json
// @Param       cluster     path     string                                              true  "cluster"
// @Param       namespace   path     string                                              true  "namespace"
// @Param       name        path     string                                              true  "name"
// @Param       fingerprint query    string                                              true  "告警指纹"
// @Param       starts_at   query    string                                              true  "告警开始时间"
// @Param       page        query    int                                                 false "page"
// @Param       size        query    int                                                 false "size"
// @Success     200         {object} handlers.ResponseStruct{Data=[]models.AlertMessage} "规则"
// @Router      /v1/observability/cluster/{cluster}/namespaces/{namespace}/alerts/{name}/repeats [get]
// @Security    JWT
func (h *ObservabilityHandler) AlertRepeats(c *gin.Context) {
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
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()).Order("created_at desc"), cond, &messages)
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
// @Tags        Observability
// @Summary     今日告警数量统计
// @Description 今日告警数量统计
// @Accept      json
// @Produce     json
// @Param       tenant_id path     string                                         true  "租户ID"
// @Param       status    query    string                                         false "状态(firing, resolved)"
// @Success     200       {object} handlers.ResponseStruct{Data=AlertCountStatus} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/alerts/today [get]
// @Security    JWT
func (h *ObservabilityHandler) AlertToday(c *gin.Context) {
	todayBedin := utils.DayStartTime(time.Now())
	yesterdayBegin := todayBedin.Add(-24 * time.Hour)
	todayEnd := todayBedin.Add(24 * time.Hour)
	tenantID := c.Param("tenant_id")
	ctx := c.Request.Context()
	yesterdayQuery := h.GetDB().WithContext(ctx).Table("alert_messages").
		Select("status, count(alert_infos.fingerprint) as count").
		Joins("join alert_infos on alert_messages.fingerprint = alert_infos.fingerprint").
		Where("created_at >= ?", yesterdayBegin).
		Where("created_at < ?", todayBedin).
		Group("status")
	todayQuery := h.GetDB().WithContext(ctx).Table("alert_messages").
		Select("status, count(alert_infos.fingerprint) as count").
		Joins("join alert_infos on alert_messages.fingerprint = alert_infos.fingerprint").
		Where("created_at >= ?", todayBedin).
		Where("created_at < ?", todayEnd).
		Group("status")

	if tenantID != "_all" {
		t := models.Tenant{}
		h.GetDB().WithContext(ctx).First(&t, "id = ?", tenantID)
		yesterdayQuery.Where("tenant_name = ?", t.TenantName)
		todayQuery.Where("tenant_name = ?", t.TenantName)
	}

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
// @Tags        Observability
// @Summary     告警趋势图
// @Description 告警趋势图
// @Accept      json
// @Produce     json
// @Param       tenant_id query    string                                     true  "租户ID"
// @Param       start     query    string                                     false "开始时间，格式 2006-01-02T15:04:05Z07:00"
// @Param       end       query    string                                     false "结束时间，格式 2006-01-02T15:04:05Z07:00"
// @Success     200       {object} handlers.ResponseStruct{Data=model.Matrix} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/alerts/graph [get]
// @Security    JWT
func (h *ObservabilityHandler) AlertGraph(c *gin.Context) {
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

	ctx := c.Request.Context()
	query := h.GetDB().WithContext(ctx).Table("alert_messages").
		Select(`project_name, DATE_FORMAT(created_at, "%Y-%m-%d") as date, count(alert_messages.fingerprint) as count`).
		Joins("join alert_infos on alert_messages.fingerprint = alert_infos.fingerprint").
		Where("created_at >= ?", start).
		Where("created_at < ?", end).
		Group("project_name").Group(`DATE_FORMAT(created_at, "%Y-%m-%d")`)

	tenantID := c.Param("tenant_id")
	if c.Param("tenant_id") != "_all" {
		t := models.Tenant{}
		h.GetDB().WithContext(ctx).First(&t, "id = ?", tenantID)
		query.Where("tenant_name = ?", t.TenantName)
	}

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
// @Tags        Observability
// @Summary     告警分组统计
// @Description 告警分组统计
// @Accept      json
// @Produce     json
// @Param       tenant_id query    string                                   true  "租户ID"
// @Param       start     query    string                                   false "开始时间，格式 2006-01-02T15:04:05Z07:00"
// @Param       end       query    string                                   false "结束时间，格式 2006-01-02T15:04:05Z07:00"
// @Param       groupby   query    string                                   true  "按什么分组(project_name, alert_type)"
// @Success     200       {object} handlers.ResponseStruct{Data=[]TableRet} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/alerts/group [get]
// @Security    JWT
func (h *ObservabilityHandler) AlertByGroup(c *gin.Context) {
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

	ctx := c.Request.Context()
	query := h.GetDB().WithContext(ctx).Table("alert_messages").
		Joins("join alert_infos on alert_messages.fingerprint = alert_infos.fingerprint").
		Where("created_at >= ?", start).
		Where("created_at < ?", end)

	tenantID := c.Param("tenant_id")
	if c.Param("tenant_id") != "_all" {
		t := models.Tenant{}
		h.GetDB().WithContext(ctx).First(&t, "id = ?", tenantID)
		query.Where("tenant_name = ?", t.TenantName)
	}

	switch c.Query("groupby") {
	case "alert_type":
		// logging, raw promql and from template
		query.Select(`alert_infos.labels ->> '$.gems_alertname' as group_value, count(alert_infos.fingerprint) as count`).
			Group(`alert_infos.labels ->> '$.gems_alertname'`)
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
// @Tags        Observability
// @Summary     搜索告警
// @Description 搜索告警
// @Accept      json
// @Produce     json
// @Param       tenant_id   path     string                                                                      true  "租户ID，所有租户为_all"
// @Param       project     query    string                                                                      false "项目名，默认所有"
// @Param       environment query    string                                                                      false "环境名，默认所有"
// @Param       cluster     query    string                                                                      false "集群名，默认所有"
// @Param       namespace   query    string                                                                      false "命名空间，默认所有"
// @Param       alertname   query    string                                                                      false "告警名，默认所有"
// @Param       search      query    string                                                                      false "告警消息内容和标签，中间以空格隔开，eg. pod=mypod container=mycontainer alertcontent"
// @Param       tpl         query    string                                                                      false "告警模板，默认所有, scope.resource.rule"
// @Param       labelpairs  query    string                                                                      false "标签键值对,不支持正则 eg. labelpairs[host]=k8s-master&labelpairs[pod]=pod1"
// @Param       start       query    string                                                                      false "开始时间"
// @Param       end         query    string                                                                      false "结束时间"
// @Param       status      query    string                                                                      false "状态(firing, resolved)"
// @Param       page        query    int                                                                         false "page"
// @Param       size        query    int                                                                         false "size"
// @Success     200         {object} handlers.ResponseStruct{Data=pagination.PageData{List=[]AlertMessageGroup}} "resp"
// @Router      /v1/observability/tenant/{tenant_id}/alerts/search [get]
// @Security    JWT
func (h *ObservabilityHandler) SearchAlert(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	tenantID := c.Param("tenant_id")
	project := c.Query("project")
	environment := c.Query("environment")
	cluster := c.Query("cluster")
	namespace := c.Query("namespace")
	search := c.Query("search")
	alertName := c.Query("alertname")
	tpl := c.Query("tpl")
	labelpairs := c.QueryMap("labelpairs")
	start := c.Query("start")
	end := c.Query("end")
	status := c.Query("status")
	var total int64
	var messages []models.AlertMessage

	ctx := c.Request.Context()
	query := h.GetDB().WithContext(ctx).Preload("AlertInfo").Joins("AlertInfo")
	if tenantID != "_all" {
		t := models.Tenant{}
		h.GetDB().WithContext(ctx).First(&t, "id = ?", tenantID)
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
		query.Where("name like ?", fmt.Sprintf("%%%s%%", alertName))
	}

	// search message and labels
	if search != "" {
		filters := strings.Split(search, " ")
		for _, filter := range filters {
			kvs := strings.Split(filter, "=")
			switch len(kvs) {
			case 1:
				query.Where("message like ?", fmt.Sprintf("%%%s%%", kvs[0]))
			case 2:
				query.Where(fmt.Sprintf(`labels -> '$.%s' = ?`, kvs[0]), kvs[1])
			}
		}
	}
	if tpl != "" {
		query.Where(fmt.Sprintf(`labels -> '$."%s"' = ?`, prometheus.AlertPromqlTpl), tpl)
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
	start = utils.DayStartTime(start)
	end = utils.NextDayStartTime(end)

	ret := []model.SamplePair{}
	for tmp := start; tmp.Before(end); tmp = tmp.Add(24 * time.Hour) {
		ret = append(ret, model.SamplePair{
			Timestamp: model.Time(tmp.UnixMilli()),
			Value:     0,
		})
	}
	return ret
}

type PromeAlertCount struct {
	Inactive int `json:"inactive"`
	Pending  int `json:"pending"`
	Firing   int `json:"firing"`
}

func (h *ObservabilityHandler) listAlertRulesStatus(c *gin.Context, alerttype string) (PromeAlertCount, error) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	ret := PromeAlertCount{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		dbAlertRules := []*models.AlertRule{}
		err := h.GetDB().WithContext(ctx).Find(&dbAlertRules,
			"alert_type = ? and cluster = ? and namespace = ?", alerttype, cluster, namespace).Error
		if err != nil {
			return err
		}
		var realTimeAlertRules map[string]prometheus.RealTimeAlertRule
		if alerttype == prometheus.AlertTypeMonitor {
			realTimeAlertRules, err = cli.Extend().GetPromeAlertRules(ctx, "")
		} else {
			realTimeAlertRules, err = cli.Extend().GetLokiAlertRules(ctx)
		}
		if err != nil {
			return errors.Wrap(err, "get alert rules status")
		}
		for _, v := range dbAlertRules {
			if promalert, ok := realTimeAlertRules[prometheus.RealTimeAlertKey(v.Namespace, v.Name)]; ok {
				v.State = promalert.State
			} else {
				v.State = "inactive"
			}
			switch v.State {
			case "inactive":
				ret.Inactive++
			case "pending":
				ret.Pending++
			case "firing":
				ret.Firing++
			}
		}
		return nil
	}); err != nil {
		return ret, err
	}
	return ret, nil
}

func (h *ObservabilityHandler) listAlertRules(c *gin.Context, alerttype string) (*handlers.PageData, error) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	// update all alert rules state in this namespace
	thisNSAlerts := []*models.AlertRule{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		err := h.GetDB().WithContext(ctx).Find(&thisNSAlerts,
			"alert_type = ? and cluster = ? and namespace = ?", alerttype, cluster, namespace).Error
		if err != nil {
			return err
		}
		var realTimeAlertRules map[string]prometheus.RealTimeAlertRule
		if alerttype == prometheus.AlertTypeMonitor {
			realTimeAlertRules, err = cli.Extend().GetPromeAlertRules(ctx, "")
		} else {
			realTimeAlertRules, err = cli.Extend().GetLokiAlertRules(ctx)
		}
		if err != nil {
			return errors.Wrap(err, "get prometheus alerts")
		}

		for _, v := range thisNSAlerts {
			var newState string
			if promalert, ok := realTimeAlertRules[prometheus.RealTimeAlertKey(v.Namespace, v.Name)]; ok {
				newState = promalert.State
			} else {
				newState = "inactive"
			}
			if v.State != newState {
				if err := h.GetDB().WithContext(ctx).Model(v).Update("state", newState).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "update alert rules state")
	}

	list := []*models.AlertRule{}
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return nil, err
	}
	cond := &handlers.PageQueryCond{
		Model:         "AlertRule",
		SearchFields:  []string{"name", "expr"},
		PreloadFields: []string{"Receivers", "Receivers.AlertChannel"},
		Where: []*handlers.QArgs{
			handlers.Args("cluster = ? and namespace = ?", cluster, namespace),
			handlers.Args("alert_type = ?", alerttype),
		},
	}
	if state := c.Query("state"); state != "" {
		cond.Where = append(cond.Where, handlers.Args("state in (?)", strings.Split(state, ",")))
	}

	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()).Order("name"), cond, &list)
	if err != nil {
		return nil, err
	}
	return handlers.Page(total, list, page, size), nil
}

func (h *ObservabilityHandler) getAlertRule(c *gin.Context, alerttype string) (*models.AlertRule, error) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")

	ret := models.AlertRule{}
	if err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		err := h.GetDB().WithContext(ctx).Preload("Receivers.AlertChannel").First(&ret,
			"cluster = ? and namespace = ? and name = ?", cluster, namespace, name).Error
		if err != nil {
			return err
		}
		var realTimeAlertRules map[string]prometheus.RealTimeAlertRule
		if alerttype == prometheus.AlertTypeMonitor {
			realTimeAlertRules, err = cli.Extend().GetPromeAlertRules(ctx, "")
		} else {
			realTimeAlertRules, err = cli.Extend().GetLokiAlertRules(ctx)
		}
		if err != nil {
			return errors.Wrap(err, "get prometheus alerts")
		}

		if promalert, ok := realTimeAlertRules[prometheus.RealTimeAlertKey(namespace, name)]; ok {
			ret.State = promalert.State
			sort.Slice(promalert.Alerts, func(i, j int) bool {
				return promalert.Alerts[i].ActiveAt.After(promalert.Alerts[j].ActiveAt)
			})
			ret.RealTimeAlerts = promalert.Alerts
		} else {
			ret.State = "inactive"
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}

type AlertRuleProcessor struct {
	cli agents.Client
	db  *database.Database
}

func NewAlertRuleProcessor(cli agents.Client, db *database.Database) *AlertRuleProcessor {
	return &AlertRuleProcessor{cli: cli, db: db}
}

func (p *AlertRuleProcessor) DBWithCtx(ctx context.Context) *gorm.DB {
	return p.db.DB().WithContext(ctx)
}

func (h *ObservabilityHandler) withAlertRuleProcessor(ctx context.Context, cluster string, f func(ctx context.Context, p *AlertRuleProcessor) error) error {
	cli, err := h.GetAgents().ClientOf(ctx, cluster)
	if err != nil {
		return err
	}
	processor := NewAlertRuleProcessor(cli, h.GetDataBase())
	return f(ctx, processor)
}

func (p *AlertRuleProcessor) MutateAlertRule(ctx context.Context, alertrule *models.AlertRule) error {
	if alertrule.Namespace == "" {
		return errors.Errorf("namespace can't be empty")
	}

	// set tpl
	if alertrule.PromqlGenerator != nil {
		tpl, err := p.db.FindPromqlTpl(alertrule.PromqlGenerator.Scope, alertrule.PromqlGenerator.Resource, alertrule.PromqlGenerator.Rule)
		if err != nil {
			return err
		}
		alertrule.PromqlGenerator.Tpl = tpl
	}

	if alertrule.Message == "" {
		msg, err := genarateMessage(alertrule)
		if err != nil {
			return err
		}
		alertrule.Message = msg
	}

	// set generatedExpr
	generatedExpr, err := GenerateExpr(alertrule)
	if err != nil {
		return err
	}
	alertrule.Expr = generatedExpr

	if err := SetReceivers(alertrule, p.db.DB().WithContext(ctx)); err != nil {
		return err
	}
	return checkAlertLevels(alertrule)
}

func (p *AlertRuleProcessor) getAlertRuleReq(c *gin.Context) (*models.AlertRule, error) {
	req := &models.AlertRule{}
	err := c.BindJSON(&req)
	if err != nil {
		return nil, err
	}
	req.Cluster = c.Param("cluster")
	req.Namespace = c.Param("namespace")

	// set alert type
	if req.AlertType == "" {
		if strings.Contains(c.FullPath(), "monitor/alerts") {
			req.AlertType = prometheus.AlertTypeMonitor
		} else {
			req.AlertType = prometheus.AlertTypeLogging
		}
	}
	if err := p.MutateAlertRule(c.Request.Context(), req); err != nil {
		return nil, err
	}
	return req, nil
}

func genarateMessage(alertrule *models.AlertRule) (string, error) {
	var ret string
	switch alertrule.AlertType {
	case prometheus.AlertTypeMonitor:
		if alertrule.PromqlGenerator != nil {
			ret = fmt.Sprintf("%s: [cluster:{{ $externalLabels.%s }}] ", alertrule.Name, prometheus.AlertClusterKey)
			for _, label := range alertrule.PromqlGenerator.Tpl.Labels {
				ret += fmt.Sprintf("[%s:{{ $labels.%s }}] ", label, label)
			}
			unitValue, err := prometheus.ParseUnit(alertrule.PromqlGenerator.Unit)
			if err != nil {
				return "", err
			}
			ret += fmt.Sprintf("%s trigger alert, value: %s%s", alertrule.PromqlGenerator.Tpl.RuleShowName, prometheus.ValueAnnotationExpr, unitValue.Show)
		} else {
			ret = fmt.Sprintf("%s: [cluster:{{ $externalLabels.%s }}] trigger alert, value: %s", alertrule.Name, prometheus.AlertClusterKey, prometheus.ValueAnnotationExpr)
		}
	case prometheus.AlertTypeLogging:
		if alertrule.LogqlGenerator != nil {
			ret = fmt.Sprintf("%s: [集群:{{ $labels.%s }}] [namespace: {{ $labels.namespace }}] ", alertrule.Name, prometheus.AlertClusterKey)
			for _, m := range alertrule.LogqlGenerator.LabelMatchers {
				alertrule.Message += fmt.Sprintf("[%s:{{ $labels.%s }}] ", m.Name, m.Name)
			}
			ret += fmt.Sprintf("日志中过去 %s 出现字符串 [%s] 次数触发告警, 当前值: %s", alertrule.LogqlGenerator.Duration, alertrule.LogqlGenerator.Match, prometheus.ValueAnnotationExpr)
		} else {
			ret = fmt.Sprintf("%s: [cluster:{{ $labels.%s }}] trigger alert, value: %s", alertrule.Name, prometheus.AlertClusterKey, prometheus.ValueAnnotationExpr)
		}
	default:
		return "", errors.Errorf("unknown alert type: %s", alertrule.AlertType)
	}
	return ret, nil
}

func GenerateExpr(alertrule *models.AlertRule) (string, error) {
	var generatedExpr string
	switch alertrule.AlertType {
	case prometheus.AlertTypeMonitor:
		if alertrule.PromqlGenerator != nil {
			q, err := promql.New(alertrule.PromqlGenerator.Tpl.Expr)
			if err != nil {
				return "", err
			}
			if alertrule.Namespace != prometheus.GlobalAlertNamespace {
				q.AddLabelMatchers(&promlabels.Matcher{
					Type:  promlabels.MatchEqual,
					Name:  "namespace",
					Value: alertrule.Namespace,
				})
			}
			for _, m := range alertrule.PromqlGenerator.LabelMatchers {
				q.AddLabelMatchers(m.ToPromqlLabelMatcher())
			}
			generatedExpr = q.String()
		}
	case prometheus.AlertTypeLogging:
		if alertrule.LogqlGenerator != nil {
			dur, err := model.ParseDuration(alertrule.LogqlGenerator.Duration)
			if err != nil {
				return "", errors.Wrapf(err, "duration %s not valid", alertrule.LogqlGenerator.Duration)
			}
			if time.Duration(dur).Minutes() > 10 {
				return "", errors.New("日志模板时长不能超过10m")
			}
			if _, err := regexp.Compile(alertrule.LogqlGenerator.Match); err != nil {
				return "", errors.Wrapf(err, "match %s not valid", alertrule.LogqlGenerator.Match)
			}
			if len(alertrule.LogqlGenerator.LabelMatchers) == 0 {
				return "", fmt.Errorf("labelMatchers can't be null")
			}

			labelvalues := []string{}
			for _, v := range alertrule.LogqlGenerator.LabelMatchers {
				labelvalues = append(labelvalues, v.String())
			}
			sort.Strings(labelvalues)
			labelvalues = append(labelvalues, fmt.Sprintf(`namespace="%s"`, alertrule.Namespace))
			generatedExpr = fmt.Sprintf(
				"sum(count_over_time({%s} |~ `%s` [%s]))without(fluentd_thread)",
				strings.Join(labelvalues, ", "),
				alertrule.LogqlGenerator.Match,
				alertrule.LogqlGenerator.Duration,
			)
		}
	}

	if generatedExpr == "" {
		generatedExpr = alertrule.Expr
	}
	if generatedExpr == "" {
		return "", errors.New("empty expr")
	}
	_, _, _, hasOp := prometheus.SplitQueryExpr(generatedExpr)
	if hasOp {
		return "", fmt.Errorf("查询表达式不能包含比较运算符(<|<=|==|!=|>|>=)")
	}
	if alertrule.Namespace != prometheus.GlobalAlertNamespace {
		if !strings.Contains(generatedExpr, fmt.Sprintf(`namespace="%s"`, alertrule.Namespace)) {
			return "", fmt.Errorf(`query expr %[1]s must contains namespace %[2]s, eg: {namespace="%[2]s"}`, generatedExpr, alertrule.Namespace)
		}
	}

	return generatedExpr, nil
}

func SetReceivers(alertrule *models.AlertRule, db *gorm.DB) error {
	if len(alertrule.Receivers) == 0 {
		return fmt.Errorf("告警接收器不能为空")
	}
	channelSet := set.NewSet[uint]()
	for _, rec := range alertrule.Receivers {
		if channelSet.Has(rec.AlertChannelID) {
			return fmt.Errorf("告警渠道: %d重复", rec.AlertChannelID)
		}
		channelSet.Append(rec.AlertChannelID)
	}
	if !channelSet.Has(models.DefaultChannel.ID) {
		alertrule.Receivers = append(alertrule.Receivers, &models.AlertReceiver{
			AlertRuleID:    alertrule.ID,
			AlertChannelID: models.DefaultChannel.ID,
			Interval:       alertrule.Receivers[0].Interval,
		})
	}
	for _, v := range alertrule.Receivers {
		v.AlertRuleID = alertrule.ID
		v.AlertChannel = &models.AlertChannel{ID: v.AlertChannelID}
		if err := db.First(v.AlertChannel, "id = ?", v.AlertChannelID).Error; err != nil {
			return errors.Wrapf(err, "alert channel: %d not found", v.AlertChannelID)
		}
	}
	return nil
}

func checkAlertLevels(alertrule *models.AlertRule) error {
	if len(alertrule.AlertLevels) == 0 {
		return fmt.Errorf("告警级别不能为空")
	}
	severitySet := set.NewSet[string]()
	for _, v := range alertrule.AlertLevels {
		if severitySet.Has(v.Severity) {
			return fmt.Errorf("有重复的告警级别")
		}
		severitySet.Append(v.Severity)
	}

	if len(alertrule.AlertLevels) > 1 && len(alertrule.InhibitLabels) == 0 {
		return fmt.Errorf("有多个告警级别时，告警抑制标签不能为空!")
	}
	return nil
}

// create/update/delete receiver by it's status
func updateReceiversInDB(alertrule *models.AlertRule, db *gorm.DB) error {
	oldRecs := []*models.AlertReceiver{}
	if err := db.Find(&oldRecs, "alert_rule_id = ?", alertrule.ID).Error; err != nil {
		return err
	}
	// channelID id the key
	oldRecMap := map[uint]*models.AlertReceiver{}
	for _, v := range oldRecs {
		oldRecMap[v.AlertChannelID] = v
	}
	for _, newRec := range alertrule.Receivers {
		if oldRec, ok := oldRecMap[newRec.AlertChannelID]; ok {
			newRec.ID = oldRec.ID
			if err := db.Select("interval").Updates(newRec).Error; err != nil {
				return err
			}
		} else {
			if err := db.Create(newRec).Error; err != nil {
				return err
			}
		}
		delete(oldRecMap, newRec.AlertChannelID)
	}
	for _, v := range oldRecMap {
		if err := db.Delete(v).Error; err != nil {
			return err
		}
	}
	return nil
}

func (p *AlertRuleProcessor) syncAlertmanagerConfig(ctx context.Context, alertrule *models.AlertRule) error {
	// alertmanagerconfig
	amcfg := &v1alpha1.AlertmanagerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: alertrule.Namespace,
			Name:      alertrule.Name,
			Labels: map[string]string{
				gems.LabelAlertmanagerConfigType: prometheus.AlertTypeMonitor,
				gems.LabelAlertmanagerConfigName: alertrule.Name,
			},
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, p.cli, amcfg, func() error {
		amcfg.Spec = GenerateAmcfgSpec(alertrule)
		return nil
	})
	return err
}

func (p *AlertRuleProcessor) syncEmailSecret(ctx context.Context, alertrule *models.AlertRule) error {
	emails := map[string]*channels.Email{}
	for _, rec := range alertrule.Receivers {
		switch v := rec.AlertChannel.ChannelConfig.ChannelIf.(type) {
		case *channels.Email:
			emails[rec.AlertChannel.ReceiverName()] = v
		}
	}
	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      channels.EmailSecretName,
			Namespace: alertrule.Namespace,
			Labels:    channels.EmailSecretLabel,
		},
		Type: v1.SecretTypeOpaque,
	}
	_, err := controllerutil.CreateOrUpdate(ctx, p.cli, sec, func() error {
		if sec.Data == nil {
			sec.Data = make(map[string][]byte)
		}
		for recName, v := range emails {
			sec.Data[channels.EmailSecretKey(recName, v.From)] = []byte(v.AuthPassword) // 不需要encode
		}
		return nil
	})
	return err
}

func GenerateRuleGroup(alertrule *models.AlertRule) monitoringv1.RuleGroup {
	rg := monitoringv1.RuleGroup{Name: alertrule.Name}
	for _, level := range alertrule.AlertLevels {
		rule := monitoringv1.Rule{
			Alert: alertrule.Name,
			Expr:  intstr.FromString(fmt.Sprintf("%s%s%s", alertrule.Expr, level.CompareOp, level.CompareValue)),
			For:   alertrule.For,
			Labels: map[string]string{
				prometheus.AlertNamespaceLabel: alertrule.Namespace,
				prometheus.AlertNameLabel:      alertrule.Name,
				prometheus.SeverityLabel:       level.Severity,
			},
			Annotations: map[string]string{
				prometheus.MessageAnnotationsKey: alertrule.Message,
				prometheus.ValueAnnotationKey:    prometheus.ValueAnnotationExpr,
			},
		}
		rg.Rules = append(rg.Rules, rule)
	}
	return rg
}

func GenerateAmcfgSpec(alertrule *models.AlertRule) v1alpha1.AlertmanagerConfigSpec {
	ret := v1alpha1.AlertmanagerConfigSpec{
		Route: &v1alpha1.Route{
			Receiver:      prometheus.NullReceiverName,
			GroupBy:       []string{prometheus.AlertNamespaceLabel, prometheus.AlertNameLabel},
			GroupWait:     "30s",
			GroupInterval: "30s",
			Routes:        []apiextensionsv1.JSON{},
		},
		Receivers: []v1alpha1.Receiver{
			prometheus.NullReceiver,
		},
		InhibitRules: []v1alpha1.InhibitRule{},
	}
	if alertrule.Namespace != prometheus.GlobalAlertNamespace {
		// force add namespace matcher
		ret.Route.Matchers = append(ret.Route.Matchers, v1alpha1.Matcher{
			Name:  "namespace",
			Value: alertrule.Namespace,
		})
	}
	for _, rec := range alertrule.Receivers {
		route := v1alpha1.Route{
			Receiver:       rec.AlertChannel.ReceiverName(),
			RepeatInterval: rec.Interval,
			Continue:       true,
			Matchers: []v1alpha1.Matcher{
				{
					Name:  prometheus.AlertNamespaceLabel,
					Value: alertrule.Namespace,
				},
				{
					Name:  prometheus.AlertNameLabel,
					Value: alertrule.Name,
				},
			},
		}
		rawRouteData, _ := json.Marshal(route)
		// receiver
		ret.Receivers = append(ret.Receivers, rec.AlertChannel.ToAlertmanagerReceiver())
		// route
		ret.Route.Routes = append(ret.Route.Routes, apiextensionsv1.JSON{Raw: rawRouteData})
	}
	// inhibit label
	if len(alertrule.InhibitLabels) > 0 {
		inhibitrule := v1alpha1.InhibitRule{
			SourceMatch: []v1alpha1.Matcher{
				{
					Name:  prometheus.AlertNamespaceLabel,
					Value: alertrule.Namespace,
				},
				{
					Name:  prometheus.AlertNameLabel,
					Value: alertrule.Name,
				},
				{
					Name:  prometheus.SeverityLabel,
					Value: prometheus.SeverityCritical,
				},
			},
			TargetMatch: []v1alpha1.Matcher{
				{
					Name:  prometheus.AlertNamespaceLabel,
					Value: alertrule.Namespace,
				},
				{
					Name:  prometheus.AlertNameLabel,
					Value: alertrule.Name,
				},
				{
					Name:  prometheus.SeverityLabel,
					Value: prometheus.SeverityError,
				},
			},
			Equal: append(alertrule.InhibitLabels, prometheus.AlertNamespaceLabel, prometheus.AlertNameLabel),
		}
		if alertrule.Namespace != prometheus.GlobalAlertNamespace {
			inhibitrule.SourceMatch = append(inhibitrule.SourceMatch, v1alpha1.Matcher{
				Name:  "namespace",
				Value: alertrule.Namespace,
			})
			inhibitrule.TargetMatch = append(inhibitrule.TargetMatch, v1alpha1.Matcher{
				Name:  "namespace",
				Value: alertrule.Namespace,
			})
			inhibitrule.Equal = append(inhibitrule.Equal, "namespace")
		}
		ret.InhibitRules = append(ret.InhibitRules, inhibitrule)
	}
	return ret
}

func (p *AlertRuleProcessor) syncPrometheusRule(ctx context.Context, alertrule *models.AlertRule) error {
	prule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: alertrule.Namespace,
			Name:      alertrule.Name,
			Labels: map[string]string{
				gems.LabelPrometheusRuleType: prometheus.AlertTypeMonitor,
				gems.LabelPrometheusRuleName: alertrule.Name,
			},
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, p.cli, prule, func() error {
		prule.Spec.Groups = []monitoringv1.RuleGroup{GenerateRuleGroup(alertrule)}
		return nil
	})
	return err
}

func (p *AlertRuleProcessor) syncMonitorAlertRule(ctx context.Context, alertrule *models.AlertRule) error {
	if err := p.syncEmailSecret(ctx, alertrule); err != nil {
		return errors.Wrap(err, "sync secret failed")
	}
	if err := p.syncPrometheusRule(ctx, alertrule); err != nil {
		return errors.Wrap(err, "sync prometheusrule failed")
	}
	if err := p.syncAlertmanagerConfig(ctx, alertrule); err != nil {
		return errors.Wrap(err, "sync alertmanagerconfig failed")
	}
	return nil
}

func (p *AlertRuleProcessor) deleteMonitorAlertRule(ctx context.Context, alertrule *models.AlertRule) error {
	if err := p.cli.Delete(ctx, &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: alertrule.Namespace,
			Name:      alertrule.Name,
		},
	}); err != nil && !kerrors.IsNotFound(err) {
		return err
	}
	if err := p.cli.Delete(ctx, &v1alpha1.AlertmanagerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: alertrule.Namespace,
			Name:      alertrule.Name,
		},
	}); err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	return deleteSilenceIfExist(ctx, alertrule.Namespace, alertrule.Name, p.cli)
}

func (p *AlertRuleProcessor) syncLoggingAlertRule(ctx context.Context, alertrule *models.AlertRule) error {
	if err := p.syncEmailSecret(ctx, alertrule); err != nil {
		return errors.Wrap(err, "sync secret failed")
	}
	if err := p.syncLokiRules(ctx, alertrule); err != nil {
		return errors.Wrap(err, "sync loki rules failed")
	}
	if err := p.syncAlertmanagerConfig(ctx, alertrule); err != nil {
		return errors.Wrap(err, "sync alertmanagerconfig failed")
	}
	return nil
}

func (p *AlertRuleProcessor) CreateAlertRule(ctx context.Context, req *models.AlertRule) error {
	return p.DBWithCtx(ctx).Transaction(func(tx *gorm.DB) error {
		allRules := []models.AlertRule{}
		if err := tx.Find(&allRules, "cluster = ? and namespace = ? and name = ?", req.Cluster, req.Namespace, req.Name).Error; err != nil {
			return err
		}
		if len(allRules) > 0 {
			return errors.Errorf("alert rule %s is already exist", req.Name)
		}
		for _, rec := range req.Receivers {
			if rec.ID > 0 {
				return errors.Errorf("receiver's id should be null when create")
			}
		}
		if err := tx.Create(req).Error; err != nil {
			return err
		}
		return p.SyncAlertRule(ctx, req)
	})
}

func (p *AlertRuleProcessor) UpdateAlertRule(ctx context.Context, req *models.AlertRule) error {
	return p.DBWithCtx(ctx).Transaction(func(tx *gorm.DB) error {
		if err := updateReceiversInDB(req, tx); err != nil {
			return errors.Wrap(err, "update receivers")
		}
		if err := tx.Select("expr", "for", "message", "inhibit_labels", "alert_levels", "promql_generator", "logql_generator").
			Updates(req).Error; err != nil {
			return err
		}
		return p.SyncAlertRule(ctx, req)
	})
}

func (p *AlertRuleProcessor) SyncAlertRule(ctx context.Context, alertrule *models.AlertRule) error {
	switch alertrule.AlertType {
	case prometheus.AlertTypeMonitor:
		if err := p.syncMonitorAlertRule(ctx, alertrule); err != nil {
			return errors.Wrapf(err, "sync monitor alertrule: %s", alertrule.FullName())
		}
	case prometheus.AlertTypeLogging:
		if err := p.syncLoggingAlertRule(ctx, alertrule); err != nil {
			return errors.Wrapf(err, "sync logging alertrule: %s", alertrule.FullName())
		}
	default:
		return errors.Errorf("unknown alerttype: %v", alertrule)
	}
	if alertrule.IsOpen {
		return deleteSilenceIfExist(ctx, alertrule.Namespace, alertrule.Name, p.cli)
	} else {
		return createSilenceIfNotExist(ctx, alertrule.Namespace, alertrule.Name, p.cli)
	}
}

const (
	LoggingAlertRuleCMName = "kubegems-loki-rules"
	LokiRecordingRulesKey  = "kubegems-loki-recording-rules.yaml"
)

func mutateLokiRuleGroups(
	lokiRuleData map[string]string,
	namespace string,
	mutate func(spec *monitoringv1.PrometheusRuleSpec),
) error {
	// get from cm
	allgroups := monitoringv1.PrometheusRuleSpec{}
	if groupstr, ok := lokiRuleData[namespace]; ok {
		if err := yaml.Unmarshal([]byte(groupstr), &allgroups); err != nil {
			return errors.Wrapf(err, "decode log rulegroups:\n%s", groupstr)
		}
	}

	mutate(&allgroups)

	// set to cm
	bts, err := yaml.Marshal(allgroups)
	if err != nil {
		errors.Wrap(err, "encode log rulegroups")
	}
	if lokiRuleData == nil {
		lokiRuleData = make(map[string]string)
	}
	lokiRuleData[namespace] = string(bts)
	return nil
}

func (p *AlertRuleProcessor) syncLokiRules(ctx context.Context, alertrule *models.AlertRule) error {
	lokiRuleCM := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gems.NamespaceLogging,
			Name:      LoggingAlertRuleCMName,
		},
	}

	rg := GenerateRuleGroup(alertrule)
	_, err := controllerutil.CreateOrUpdate(ctx, p.cli, lokiRuleCM, func() error {
		return mutateLokiRuleGroups(lokiRuleCM.Data, alertrule.Namespace, func(spec *monitoringv1.PrometheusRuleSpec) {
			// create or update
			index := -1
			for i, v := range spec.Groups {
				if v.Name == rg.Name {
					index = i
				}
			}
			if index == -1 {
				// create
				spec.Groups = append(spec.Groups, rg)
			} else {
				// update
				spec.Groups[index] = rg
			}
		})
	})
	return err
}

func (p *AlertRuleProcessor) deleteLoggingAlertRule(ctx context.Context, alertrule *models.AlertRule) error {
	lokiRuleCM := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gems.NamespaceLogging,
			Name:      LoggingAlertRuleCMName,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, p.cli, lokiRuleCM, func() error {
		return mutateLokiRuleGroups(lokiRuleCM.Data, alertrule.Namespace, func(spec *monitoringv1.PrometheusRuleSpec) {
			// delete
			found := false
			newGroups := []monitoringv1.RuleGroup{}
			for _, v := range spec.Groups {
				if v.Name == alertrule.Name {
					found = true
				} else {
					newGroups = append(newGroups, v)
				}
			}
			if !found {
				log.Warnf("log alert rule %s not found in loki rules", alertrule.Name)
			}
			spec.Groups = newGroups
		})
	}); err != nil {
		return errors.Wrap(err, "delete from loki rules")
	}

	if err := p.cli.Delete(ctx, &v1alpha1.AlertmanagerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: alertrule.Namespace,
			Name:      alertrule.Name,
		},
	}); err != nil && !kerrors.IsNotFound(err) {
		return errors.Wrap(err, "delete from alertmanager config")
	}

	return deleteSilenceIfExist(ctx, alertrule.Namespace, alertrule.Name, p.cli)
}

type K8sAlertCfg struct {
	monitoringv1.RuleGroup          `json:"ruleGroup"`
	v1alpha1.AlertmanagerConfigSpec `json:"alertmanagerConfigSpec"`
}

// clustername/namespace/name-alertrule map
func (p *AlertRuleProcessor) GetK8sAlertCfg(ctx context.Context) (map[string]K8sAlertCfg, error) {
	// amcfg
	amcfgList := v1alpha1.AlertmanagerConfigList{}
	if err := p.cli.List(ctx, &amcfgList, client.InNamespace(v1.NamespaceAll), client.HasLabels([]string{
		gems.LabelAlertmanagerConfigType, gems.LabelAlertmanagerConfigName,
	})); err != nil {
		return nil, err
	}

	ruleGroupMap := map[string]monitoringv1.RuleGroup{}

	// rulegroup by prometheus
	promruleList := monitoringv1.PrometheusRuleList{}
	if err := p.cli.List(ctx, &promruleList, client.InNamespace(v1.NamespaceAll), client.HasLabels([]string{
		gems.LabelPrometheusRuleType, gems.LabelPrometheusRuleName,
	})); err != nil {
		log.Error(err, "list prometheusrule")
	}
	for _, rule := range promruleList.Items {
		if rule.Name == prometheus.DefaultAlertCRDName {
			continue
		}
		for _, group := range rule.Spec.Groups {
			key := models.AlertRuleKey(p.cli.Name(), rule.Namespace, group.Name)
			if _, ok := ruleGroupMap[key]; ok {
				log.Warnf("duplicated alert rule: %s", key)
			} else {
				ruleGroupMap[key] = group
			}
		}
	}

	// rulegroup by loki
	lokiruleCm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gems.NamespaceLogging,
			Name:      LoggingAlertRuleCMName,
		},
	}
	if err := p.cli.Get(ctx, client.ObjectKeyFromObject(&lokiruleCm), &lokiruleCm); err != nil {
		log.Warnf("get loki configmap from cluster: %s failed, %v", p.cli.Name(), err)
	}
	for k, v := range lokiruleCm.Data {
		// skip recording rule
		if k == LokiRecordingRulesKey {
			continue
		}
		groups := monitoringv1.PrometheusRuleSpec{}
		if err := yaml.Unmarshal([]byte(v), &groups); err != nil {
			return nil, errors.Wrapf(err, "decode log rulegroups: \n%s", v)
		}
		for _, group := range groups.Groups {
			key := models.AlertRuleKey(p.cli.Name(), k, group.Name)
			if _, ok := ruleGroupMap[key]; ok {
				log.Warnf("duplicated alert rule: %s")
			} else {
				ruleGroupMap[key] = group
			}
		}
	}

	ret := map[string]K8sAlertCfg{}
	for _, amcfg := range amcfgList.Items {
		key := models.AlertRuleKey(p.cli.Name(), amcfg.Namespace, amcfg.Name)
		if group, ok := ruleGroupMap[key]; ok {
			ret[key] = K8sAlertCfg{RuleGroup: group, AlertmanagerConfigSpec: amcfg.Spec}
		} else {
			log.Warnf("rule group for alertrule %s not found", key)
		}
	}

	return ret, nil
}

func (h *ObservabilityHandler) syncAlertRulesWithTimeout(ctx context.Context, alertrules []*models.AlertRule, timeout time.Duration) (status map[string]bool, isTimeout bool) {
	// sync alert rules
	status = map[string]bool{}
	wg := &sync.WaitGroup{}
	for _, v := range alertrules {
		status[v.FullName()] = false
		wg.Add(1)
		go func(alertrule *models.AlertRule) {
			defer wg.Done()
			if err := h.Execute(ctx, alertrule.Cluster, func(ctx context.Context, cli agents.Client) error {
				return NewAlertRuleProcessor(cli, h.GetDataBase()).SyncAlertRule(ctx, alertrule)
			}); err != nil {
				log.Warnf("%s alert rule: %s sync failed", alertrule.AlertType, alertrule.FullName())
				return
			}
			status[alertrule.FullName()] = true
		}(v)
	}
	isTimeout = utils.WaitGroupWithTimeout(wg, timeout)
	if isTimeout {
		log.Warnf("Timed out waiting for wait group")
	}
	return
}
