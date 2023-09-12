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

package alerthandler

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/observe"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/templates"
	_ "kubegems.io/library/rest/response"
)

var silenceCommentPrefix = "fingerprint-"

// AlertHistory 告警黑名单
//
//	@Tags			Alert
//	@Summary		告警黑名单
//	@Description	告警黑名单
//	@Accept			json
//	@Produce		json
//	@Param			cluster		query		string																					false	"集群, 默认所有"
//	@Param			namespace	query		string																					false	"命名空间, 默认所有"
//	@Param			page		query		int																						false	"page"
//	@Param			size		query		int																						false	"size"
//	@Success		200			{object}	handlers.ResponseStruct{Data=response.Page[models.AlertInfo]{List=[]models.AlertInfo}}	"resp"
//	@Router			/v1/alerts/blacklist [get]
//	@Security		JWT
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

	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()).Order("silence_updated_at desc"), cond, &ret)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	// 使用map避免循环查询数据库
	tplGetter := h.GetDataBase().NewPromqlTplMapperFromDB().FindPromqlTpl
	for i := range ret {
		if ret[i].SilenceEndsAt.Equal(utils.TimeForever) {
			ret[i].SilenceEndsAt = nil
		}
		ret[i].LabelMap = make(map[string]string)
		_ = json.Unmarshal(ret[i].Labels, &ret[i].LabelMap)
		ret[i].Summary = formatBlackListSummary(ret[i].LabelMap, tplGetter)
	}
	handlers.OK(c, handlers.Page(total, ret, int64(page), int64(size)))
}

func formatBlackListSummary(labels map[string]string, f templates.TplGetter) string {
	tplname := labels[prometheus.AlertPromqlTpl]
	var subStr string
	tmp := strings.Split(tplname, ".")
	if len(tmp) == 3 {
		tpl, err := f(tmp[0], tmp[1], tmp[2])
		if err == nil {
			for _, l := range tpl.Labels {
				subStr += fmt.Sprintf("[%s:%s] ", l, labels[l])
			}
		} else {
			log.Warnf("tpl: %s not found", tplname)
		}
	}

	header := fmt.Sprintf("%s: [集群:%s] ", labels[prometheus.AlertNameLabel], labels[prometheus.AlertClusterKey])

	return fmt.Sprintf("%s%s告警", header, subStr)
}

// AlertHistory 加入/更新告警黑名单
//
//	@Tags			Alert
//	@Summary		加入/更新告警黑名单
//	@Description	加入/更新告警黑名单
//	@Accept			json
//	@Produce		json
//	@Param			form	body		models.AlertInfo						true	"黑名单详情，必传AlertInfo.Fingerprint"
//	@Success		200		{object}	handlers.ResponseStruct{Data=string}	"resp"
//	@Router			/v1/alerts/blacklist [post]
//	@Security		JWT
func (h *AlertsHandler) AddToBlackList(c *gin.Context) {
	if err := h.withBlackListReq(c, func(req models.AlertInfo) error {
		return h.GetDB().WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&req).Error; err != nil {
				return err
			}
			ctx := c.Request.Context()
			cli, err := h.GetAgents().ClientOf(ctx, req.ClusterName)
			if err != nil {
				return err
			}
			return observe.NewClient(cli, tx).CreateOrUpdateSilenceIfNotExist(c.Request.Context(), req)
		})
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// AlertHistory 移除告警黑名单
//
//	@Tags			Alert
//	@Summary		移除告警黑名单
//	@Description	移除告警黑名单
//	@Accept			json
//	@Produce		json
//	@Param			fingerprint	path		string									true	"告警指纹"
//	@Success		200			{object}	handlers.ResponseStruct{Data=string}	"resp"
//	@Router			/v1/alerts/blacklist/{fingerprint} [delete]
//	@Security		JWT
func (h *AlertsHandler) RemoveInBlackList(c *gin.Context) {
	req := models.AlertInfo{
		Fingerprint: c.Param("fingerprint"),
	}
	if err := h.GetDB().WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
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
		return observe.NewClient(cli, tx).DeleteSilenceIfExist(ctx, req)
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
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
	if err := h.GetDB().WithContext(c.Request.Context()).First(&req, "fingerprint = ?", req.Fingerprint).Error; err != nil {
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
		req.SilenceEndsAt = &utils.TimeForever
	}
	return f(req)
}
