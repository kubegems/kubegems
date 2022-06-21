package observability

import (
	"fmt"
	"strconv"
	"time"

	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/prometheus"

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
	u, exist := h.GetContextUser(c)
	if !exist {
		return nil, fmt.Errorf("not login")
	}
	req := models.MonitorDashboard{}
	if err := c.BindJSON(&req); err != nil {
		return nil, err
	}

	envid, err := strconv.Atoi(c.Param("environment_id"))
	if err != nil {
		return nil, errors.Wrap(err, "environment_id")
	}
	uintid := uint(envid)
	req.EnvironmentID = &uintid

	req.Creator = u.GetUsername()

	env := models.Environment{}
	if err := h.GetDB().Preload("Cluster").First(&env, "id = ?", req.EnvironmentID).Error; err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	// 默认查近30m
	if req.Start == "" || req.End == "" {
		req.Start = now.Add(-30 * time.Minute).Format(time.RFC3339)
		req.End = now.Format(time.RFC3339)
	}

	monitoropts := new(prometheus.MonitorOptions)
	h.DynamicConfig.Get(c.Request.Context(), monitoropts)
	// 逐个校验graph
	for _, v := range req.Graphs {
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
		}
	}
	return &req, nil
}
