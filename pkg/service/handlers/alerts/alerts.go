package alerthandler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/prometheus"
)

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
