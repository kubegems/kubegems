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
	"net/http"
	"strconv"

	prommodel "github.com/prometheus/common/model"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ListDashboard 监控dashboard列表
// @Tags        Observability
// @Summary     监控dashboard列表
// @Description 监控dashboard列表
// @Accept      json
// @Produce     json
// @Param       environment_id path     string                                                  true "环境ID"
// @Success     200            {object} handlers.ResponseStruct{Data=[]models.MonitorDashboard} "监控dashboard列表"
// @Router      /v1/observability/environment/{environment_id}/monitor/dashboard [get]
// @Security    JWT
func (h *ObservabilityHandler) ListDashboard(c *gin.Context) {
	ret := []models.MonitorDashboard{}
	if err := h.GetDB().Find(&ret, "environment_id = ?", c.Param("environment_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// DashboardDetail 监控dashboard详情
// @Tags        Observability
// @Summary     监控dashboard详情
// @Description 监控dashboard详情
// @Accept      json
// @Produce     json
// @Param       environment_id path     string                                                    true  "环境ID"
// @Param       dashboard_id   path     uint                                                      true  "dashboard id"
// @Success     200            {object} handlers.ResponseStruct{Data=models.MonitorDashboard} "监控dashboard列表"
// @Router      /v1/observability/environment/{environment_id}/monitor/dashboard/{dashboard_id} [get]
// @Security    JWT
func (h *ObservabilityHandler) DashboardDetail(c *gin.Context) {
	ret := models.MonitorDashboard{}
	if err := h.GetDB().Find(&ret, "id = ?", c.Param("dashboard_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// DashboardDetail 监控dashboard panne指标查询
// @Tags        Observability
// @Summary     监控dashboard panne指标查询
// @Description 监控dashboard panne指标查询
// @Accept      json
// @Produce     json
// @Param       environment_id path     string                                                true "环境ID"
// @Param       dashboard_id   path     uint                                                  true "dashboard id"
// @Param       panel_id       query    uint                                                      true  "panel id"
// @Param       labelpairs     query    string                                                    false "标签键值对(value为空或者_all表示所有，支持正则),  eg.  labelpairs[host]=k8s-master&labelpairs[pod]=_all"
// @Param       start          query    string                                                    false "开始时间，默认现在-30m"
// @Param       end            query    string                                                    false "结束时间，默认现在"
// @Param       step           query    int                                                       false "step, 单位秒，默认0"
// @Success     200            {object} handlers.ResponseStruct{Data=map[string]prommodel.Matrix} "pannel 查询结果"
// @Router      /v1/observability/environment/{environment_id}/monitor/dashboard/{dashboard_id}/query [get]
// @Security    JWT
func (h *ObservabilityHandler) DashboardQuery(c *gin.Context) {
	querys, err := h.getMetricQuerysByDashboard(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	ret := map[string]prommodel.Matrix{}
	for _, query := range querys {
		if err := h.Execute(c.Request.Context(), query.Cluster, func(ctx context.Context, cli agents.Client) error {
			if err := h.mutateMetricQueryReq(ctx, query); err != nil {
				return err
			}
			result, err := cli.Extend().PrometheusQueryRange(ctx, query.Expr, query.Start, query.End, query.Step)
			if err != nil {
				return err
			}
			ret[query.TargetName] = result
			return nil
		}); err != nil {
			handlers.NotOK(c, errors.Wrap(err, "query dashboard failed"))
			return
		}
	}
	handlers.OK(c, ret)
}

func (h *ObservabilityHandler) getMetricQuerysByDashboard(c *gin.Context) ([]*MetricQueryReq, error) {
	dash := models.MonitorDashboard{}
	queries := []*MetricQueryReq{}
	if err := h.GetDB().Preload("Environment.Cluster").Find(&dash, "id = ?", c.Param("dashboard_id")).Error; err != nil {
		return nil, err
	}
	panelID, err := strconv.Atoi(c.Query("panel_id"))
	if err != nil {
		return nil, errors.Wrap(err, "panel id not valid")
	}
	if panelID >= len(dash.Graphs) || panelID < 0 {
		return nil, errors.Errorf("panel id %d out of range", panelID)
	}

	for _, target := range dash.Graphs[panelID].Targets {
		queries = append(queries, &MetricQueryReq{
			Cluster:         dash.Environment.Cluster.ClusterName,
			Namespace:       dash.Environment.Namespace,
			Start:           c.Query("start"),
			End:             c.Query("end"),
			Step:            c.Query("step"),
			Expr:            target.Expr,
			PromqlGenerator: target.PromqlGenerator,
			TargetName:      target.TargetName,
		})
	}
	globalLabelPairs := c.QueryMap("labelpairs")
	if len(globalLabelPairs) > 0 {
		for _, query := range queries {
			for k, v := range globalLabelPairs {
				if query.PromqlGenerator.LabelPairs == nil {
					query.PromqlGenerator.LabelPairs = make(map[string]string)
				}
				query.PromqlGenerator.LabelPairs[k] = v
			}
		}
	}
	return queries, nil
}

// CreateDashboard 创建监控dashboad
// @Tags        Observability
// @Summary     创建监控dashboad
// @Description 创建监控dashboad
// @Accept      json
// @Produce     json
// @Param       environment_id path     string                               true "环境ID"
// @Success     200            {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/environment/{environment_id}/monitor/dashboard [post]
// @Security    JWT
func (h *ObservabilityHandler) CreateDashboard(c *gin.Context) {
	req, err := h.getDashboardReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	action := i18n.Sprintf(context.TODO(), "create")
	module := i18n.Sprintf(context.TODO(), "monitor dashboard")
	h.SetAuditData(c, action, module, req.Name)
	h.SetExtraAuditData(c, models.ResEnvironment, *req.EnvironmentID)

	if err := h.GetDB().Save(req).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// UpdateDashboard 更新监控dashboad
// @Tags        Observability
// @Summary     更新监控dashboad
// @Description 更新监控dashboad
// @Accept      json
// @Produce     json
// @Param       environment_id path     string                               true "环境ID"
// @Param       dashboard_id   path     uint                                 true "dashboard id"
// @Param       from           body     models.MonitorDashboard              true "dashboad配置"
// @Success     200            {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/environment/{environment_id}/monitor/dashboard/{dashboard_id} [put]
// @Security    JWT
func (h *ObservabilityHandler) UpdateDashboard(c *gin.Context) {
	req, err := h.getDashboardReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	action := i18n.Sprintf(context.TODO(), "update")
	module := i18n.Sprintf(context.TODO(), "monitor dashboard")
	h.SetAuditData(c, action, module, req.Name)
	h.SetExtraAuditData(c, models.ResEnvironment, *req.EnvironmentID)

	if err := h.GetDB().Save(req).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteDashboard 删除监控dashboad
// @Tags        Observability
// @Summary     删除监控dashboad
// @Description 删除监控dashboad
// @Accept      json
// @Produce     json
// @Param       environment_id path     string                               true "环境ID"
// @Param       dashboard_id   path     uint                                 true "dashboard id"
// @Success     200            {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/environment/{environment_id}/monitor/dashboard/{dashboard_id} [delete]
// @Security    JWT
func (h *ObservabilityHandler) DeleteDashboard(c *gin.Context) {
	d := models.MonitorDashboard{}
	if err := h.GetDB().First(&d, "id = ?", c.Param("dashboard_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "monitor dashboard")
	h.SetAuditData(c, action, module, d.Name)
	h.SetExtraAuditData(c, models.ResEnvironment, *d.EnvironmentID)

	if err := h.GetDB().Delete(&d).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// ListDashboardTemplates 监控面板模板列表
// @Tags        Observability
// @Summary     监控面板模板列表
// @Description 监控面板模板列表
// @Accept      json
// @Produce     json
// @Param       page query    int                                                        false "page"
// @Param       size query    int                                                        false "size"
// @Success     200  {object} handlers.ResponseStruct{Data=[]models.MonitorDashboardTpl} "resp"
// @Router      /v1/observability/template/dashboard [get]
// @Security    JWT
func (h *ObservabilityHandler) ListDashboardTemplates(c *gin.Context) {
	tpls := []models.MonitorDashboardTpl{}
	if err := h.GetDB().Find(&tpls).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.NewPageDataFromContext(c, tpls, nil, nil))
}

// GetDashboardTemplate 监控面板模板详情
// @Tags        Observability
// @Summary     监控面板模板详情
// @Description 监控面板模板详情
// @Accept      json
// @Produce     json
// @Param       name path     string                                                   true "模板名"
// @Success     200  {object} handlers.ResponseStruct{Data=models.MonitorDashboardTpl} "resp"
// @Router      /v1/observability/template/dashboard/{name} [get]
// @Security    JWT
func (h *ObservabilityHandler) GetDashboardTemplate(c *gin.Context) {
	tpl := models.MonitorDashboardTpl{Name: c.Param("name")}
	if err := h.GetDB().First(&tpl).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, tpl)
}

// AddDashboardTemplates 导入监控面板模板
// @Tags        Observability
// @Summary     导入监控面板模板
// @Description 导入监控面板模板
// @Accept      json
// @Produce     json
// @Param       form body     models.MonitorDashboardTpl           true "模板内容"
// @Success     200  {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/template/dashboard [post]
// @Security    JWT
func (h *ObservabilityHandler) AddDashboardTemplates(c *gin.Context) {
	tpl := models.MonitorDashboardTpl{}
	if err := c.BindJSON(&tpl); err != nil {
		handlers.NotOK(c, err)
		return
	}
	action := i18n.Sprintf(context.TODO(), "import")
	module := i18n.Sprintf(context.TODO(), "monitor dashboard template")
	h.SetAuditData(c, action, module, tpl.Name)
	tplGetter := h.GetDataBase().NewPromqlTplMapperFromDB().FindPromqlTpl
	if err := models.CheckGraphs(tpl.Graphs, "", tplGetter); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Create(&tpl).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// UpdateDashboardTemplates 更新监控面板模板
// @Tags        Observability
// @Summary     更新监控面板模板
// @Description 更新监控面板模板
// @Accept      json
// @Produce     json
// @Param       form body     models.MonitorDashboardTpl           true "模板内容"
// @Success     200  {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/observability/template/dashboard/{name} [put]
// @Security    JWT
func (h *ObservabilityHandler) UpdateDashboardTemplates(c *gin.Context) {
	tpl := models.MonitorDashboardTpl{}
	if err := c.BindJSON(&tpl); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "更新", "监控面板模板", tpl.Name)
	tplGetter := h.GetDataBase().NewPromqlTplMapperFromDB().FindPromqlTpl
	if err := models.CheckGraphs(tpl.Graphs, "", tplGetter); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Save(&tpl).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteDashboardTemplate 删除监控面板模板
// @Tags        Observability
// @Summary     删除监控面板模板
// @Description 删除监控面板模板
// @Accept      json
// @Produce     json
// @Param       name path     string                                                   true "模板名"
// @Success     200  {object} handlers.ResponseStruct{Data=models.MonitorDashboardTpl} "resp"
// @Router      /v1/observability/template/dashboard/{name} [delete]
// @Security    JWT
func (h *ObservabilityHandler) DeleteDashboardTemplate(c *gin.Context) {
	tpl := models.MonitorDashboardTpl{Name: c.Param("name")}
	if err := h.GetDB().Delete(&tpl).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

func (h *ObservabilityHandler) getDashboardReq(c *gin.Context) (*models.MonitorDashboard, error) {
	req := models.MonitorDashboard{}
	if err := c.BindJSON(&req); err != nil {
		return nil, err
	}
	if c.Request.Method == http.MethodPost && req.Template != "" {
		tpl := models.MonitorDashboardTpl{Name: req.Template}
		if err := h.GetDB().First(&tpl).Error; err != nil {
			return nil, errors.Wrapf(err, "get template: %s failed", req.Template)
		}
		req.Start = tpl.Start
		req.End = tpl.End
		req.Refresh = tpl.Refresh
		req.Graphs = tpl.Graphs
	}

	envid, err := strconv.Atoi(c.Param("environment_id"))
	if err != nil {
		return nil, errors.Wrap(err, "environment_id")
	}
	uintid := uint(envid)
	req.EnvironmentID = &uintid
	u, exist := h.GetContextUser(c)
	if !exist {
		return nil, fmt.Errorf("not login")
	}
	req.Creator = u.GetUsername()

	env := models.Environment{}
	if err := h.GetDB().First(&env, "id = ?", req.EnvironmentID).Error; err != nil {
		return nil, err
	}

	// 默认查近30m
	if req.Start == "" || req.End == "" {
		req.Start = "now-30m"
		req.End = "now"
	}
	if req.Refresh == "" {
		req.Refresh = "30s"
	}
	if req.Step == "" {
		req.Step = "30s"
	}

	tplGetter := h.GetDataBase().NewPromqlTplMapperFromDB().FindPromqlTpl
	if err := models.CheckGraphs(req.Graphs, env.Namespace, tplGetter); err != nil {
		return nil, err
	}
	return &req, nil
}
