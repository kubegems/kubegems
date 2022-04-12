package metrics

import (
	"fmt"

	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/prometheus"

	"github.com/gin-gonic/gin"
)

// Config 监控dashboard列表
// @Tags         Metrics
// @Summary      监控dashboard列表
// @Description  监控dashboard列表
// @Accept       json
// @Produce      json
// @Success      200  {object}  handlers.ResponseStruct{Data=[]models.MetricDashborad}  "监控dashboard列表"
// @Router       /v1/metrics/dashboard [get]
// @Security     JWT
func (h *MonitorHandler) ListDashborad(c *gin.Context) {
	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.NotOK(c, fmt.Errorf("not login"))
		return
	}

	ret := []models.MetricDashborad{}
	if err := h.GetDB().Find(&ret, "creator = ?", u.GetUsername()).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// Config 创建/更新dashboad
// @Tags         Metrics
// @Summary      创建/更新dashboad
// @Description  创建/更新dashboad
// @Accept       json
// @Produce      json
// @Param        from  body      models.MetricDashborad                true  "dashboad配置"
// @Success      200   {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/metrics/dashboard [post]
// @Security     JWT
func (h *MonitorHandler) CreateOrUpdateDashborad(c *gin.Context) {
	req, err := h.getDashboardReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := h.GetDB().Save(req).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// Config 删除dashboad
// @Tags         Metrics
// @Summary      删除dashboad
// @Description  删除dashboad
// @Accept       json
// @Produce      json
// @Param        dashboard_id  path      uint                                  true  "dashboard id"
// @Success      200           {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/metrics/dashboard/{dashboard_id} [delete]
// @Security     JWT
func (h *MonitorHandler) DeleteDashborad(c *gin.Context) {
	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.NotOK(c, fmt.Errorf("not login"))
		return
	}

	d := models.MetricDashborad{}
	if err := h.GetDB().Delete(&d, "id = ? and creator = ?", c.Param("dashboard_id"), u.GetUsername()).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

func (h *MonitorHandler) getDashboardReq(c *gin.Context) (*models.MetricDashborad, error) {
	u, exist := h.GetContextUser(c)
	if !exist {
		return nil, fmt.Errorf("not login")
	}
	req := models.MetricDashborad{}
	if err := c.BindJSON(&req); err != nil {
		return nil, err
	}
	req.Creator = u.GetUsername()

	monitoropts := new(prometheus.MonitorOptions)
	h.DynamicConfig.Get(c.Request.Context(), monitoropts)
	// 逐个校验graph
	for _, v := range req.Graphs {
		if v.Name == "" {
			return nil, fmt.Errorf("图表名不能为空")
		}
		_, err := v.FindRuleContext(monitoropts)
		if err != nil {
			return nil, err
		}
	}
	return &req, nil
}
