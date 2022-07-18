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
	"fmt"
	"strconv"

	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"sigs.k8s.io/yaml"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ListDashboard 监控dashboard列表
// @Tags         Observability
// @Summary      监控dashboard列表
// @Description  监控dashboard列表
// @Accept       json
// @Produce      json
// @Param        environment_id  path      string                                                 true  "环境ID"
// @Success      200             {object}  handlers.ResponseStruct{Data=[]models.MonitorDashboard}  "监控dashboard列表"
// @Router       /v1/observability/environment/{environment_id}/monitor/dashboard [get]
// @Security     JWT
func (h *ObservabilityHandler) ListDashboard(c *gin.Context) {
	ret := []models.MonitorDashboard{}
	if err := h.GetDB().Find(&ret, "environment_id = ?", c.Param("environment_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// DashboardDetail 监控dashboard详情
// @Tags         Observability
// @Summary      监控dashboard详情
// @Description  监控dashboard详情
// @Accept       json
// @Produce      json
// @Param        environment_id  path      string                                                   true  "环境ID"
// @Param        dashboard_id    path      uint                                                   true  "dashboard id"
// @Success      200             {object}  handlers.ResponseStruct{Data=models.MonitorDashboard}  "监控dashboard列表"
// @Router       /v1/observability/environment/{environment_id}/monitor/dashboard/{dashboard_id} [get]
// @Security     JWT
func (h *ObservabilityHandler) DashboardDetail(c *gin.Context) {
	ret := models.MonitorDashboard{}
	if err := h.GetDB().Find(&ret, "id = ?", c.Param("dashboard_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// CreateDashboard 创建监控dashboad
// @Tags         Observability
// @Summary      创建监控dashboad
// @Description  创建监控dashboad
// @Accept       json
// @Produce      json
// @Param        environment_id  path      string                                true  "环境ID"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/environment/{environment_id}/monitor/dashboard [post]
// @Security     JWT
func (h *ObservabilityHandler) CreateDashboard(c *gin.Context) {
	req, err := h.getDashboardReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "创建", "监控面板", req.Name)
	h.SetExtraAuditData(c, models.ResEnvironment, *req.EnvironmentID)

	if err := h.GetDB().Save(req).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// UpdateDashboard 更新监控dashboad
// @Tags         Observability
// @Summary      更新监控dashboad
// @Description  更新监控dashboad
// @Accept       json
// @Produce      json
// @Param        environment_id  path      string                                true  "环境ID"
// @Param        dashboard_id    path      uint                                  true  "dashboard id"
// @Param        from            body      models.MonitorDashboard               true  "dashboad配置"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/environment/{environment_id}/monitor/dashboard/{dashboard_id} [put]
// @Security     JWT
func (h *ObservabilityHandler) UpdateDashboard(c *gin.Context) {
	req, err := h.getDashboardReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "更新", "监控面板", req.Name)
	h.SetExtraAuditData(c, models.ResEnvironment, *req.EnvironmentID)

	if err := h.GetDB().Save(req).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteDashboard 删除监控dashboad
// @Tags         Observability
// @Summary      删除监控dashboad
// @Description  删除监控dashboad
// @Accept       json
// @Produce      json
// @Param        environment_id  path      string                                true  "环境ID"
// @Param        dashboard_id    path      uint                                  true  "dashboard id"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/observability/environment/{environment_id}/monitor/dashboard/{dashboard_id} [delete]
// @Security     JWT
func (h *ObservabilityHandler) DeleteDashboard(c *gin.Context) {
	d := models.MonitorDashboard{}
	if err := h.GetDB().First(&d, "id = ?", c.Param("dashboard_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "删除", "监控面板", d.Name)
	h.SetExtraAuditData(c, models.ResEnvironment, *d.EnvironmentID)

	if err := h.GetDB().Delete(&d).Error; err != nil {
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
	if req.Template != "" {
		tpls := []models.MonitorDashboard{}
		if err := yaml.Unmarshal(alltemplates, &tpls); err != nil {
			return nil, err
		}
		found := false
		for _, v := range tpls {
			if v.Name == req.Template {
				v.Name = req.Name
				req = v
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("template %s not found", req.Template)
		}
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
	if err := h.GetDB().Preload("Cluster").First(&env, "id = ?", req.EnvironmentID).Error; err != nil {
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

	monitoropts := new(prometheus.MonitorOptions)
	h.DynamicConfig.Get(c.Request.Context(), monitoropts)
	// 逐个校验graph
	for i, v := range req.Graphs {
		if v.Name == "" {
			return nil, fmt.Errorf("图表名不能为空")
		}

		if v.PromqlGenerator.IsEmpty() {
			if v.Expr == "" {
				return nil, fmt.Errorf("模板与原生promql不能同时为空")
			}
			if err := prometheus.CheckQueryExprNamespace(v.Expr, env.Namespace); err != nil {
				return nil, err
			}
			if v.Unit != "" {
				if _, err := prometheus.ParseUnit(v.Unit); err != nil {
					return nil, err
				}
			}
		} else {
			rulectx, err := v.PromqlGenerator.FindRuleContext(monitoropts)
			if err != nil {
				return nil, err
			}
			if rulectx.ResourceDetail.Namespaced == false {
				return nil, fmt.Errorf("图表: %s 错误！不能查询集群范围资源", v.Name)
			}
			req.Graphs[i].Unit = rulectx.RuleDetail.Unit
			req.Graphs[i].PromqlGenerator.Unit = rulectx.RuleDetail.Unit
		}
	}
	return &req, nil
}

var alltemplates = []byte(`
- name: 容器基础指标监控
  step: 30s
  refresh: 30s
  start: now-30m
  end: now
  graphs:
    - name: 容器CPU总量
      promqlGenerator:
        resource: container
        rule: cpuTotal
    - name: 容器CPU使用量
      promqlGenerator:
        resource: container
        rule: cpuUsage
    - name: 容器CPU使用率
      promqlGenerator:
        resource: container
        rule: cpuUsagePercent
    - name: 容器内存总量
      promqlGenerator:
        resource: container
        rule: memoryTotal
    - name: 容器内存使用量
      promqlGenerator:
        resource: container
        rule: memoryUsage
    - name: 容器内存使用率
      promqlGenerator:
        resource: container
        rule: memoryUsagePercent
    - name: 容器网络接收速率
      promqlGenerator:
        resource: container
        rule: networkInBPS
    - name: 容器网络发送速率
      promqlGenerator:
        resource: container
        rule: networkOutBPS
- name: 存储卷监控
  step: 30s
  refresh: 30s
  start: now-30m
  end: now
  graphs:
    - name: 存储卷总量
      promqlGenerator:
        resource: pvc
        rule: volumeTotal
    - name: 存储卷总量
      promqlGenerator:
        resource: pvc
        rule: volumeUsage
    - name: 存储卷总量
      promqlGenerator:
        resource: pvc
        rule: volumeUsagePercent
`)
