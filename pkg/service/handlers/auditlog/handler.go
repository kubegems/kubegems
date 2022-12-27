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

package auditloghandler

import (
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"

	"github.com/gin-gonic/gin"
)

var (
	ModelName      = "AuditLog"
	SearchFields   = []string{"username", "module", "name"}
	FilterFields   = []string{"Username", "Tenant", "Action", "Success", "CreatedAt_gte", "CreatedAt_lte"}
	PrimaryKeyName = "auditlog_id"
	OrderFields    = []string{"CreatedAt"}
)

// ListAuditLog 列表 AuditLog
// @Tags        AuditLog
// @Summary     AuditLog列表
// @Description AuditLog列表
// @Accept      json
// @Produce     json
// @Param       Username      query    string                                                                  false "Username"
// @Param       Tenant        query    string                                                                  false "Tenant"
// @Param       Action        query    string                                                                  false "Action"
// @Param       Success       query    string                                                                  false "Success"
// @Param       CreatedAt_gte query    string                                                                  false "CreatedAt_gte"
// @Param       CreatedAt_lte query    string                                                                  false "CreatedAt_lte"
// @Param       page          query    int                                                                     false "page"
// @Param       size          query    int                                                                     false "page"
// @Param       search        query    string                                                                  false "search in (username,module,name)"
// @Success     200           {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.AuditLog}} "AuditLog"
// @Router      /v1/auditlog [get]
// @Security    JWT
func (h *AuditLogHandler) ListAuditLog(c *gin.Context) {
	var list []models.AuditLog
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	where := []*handlers.QArgs{}
	start := c.Query("CreatedAt_gte")
	if len(start) > 0 {
		where = append(where, handlers.Args("created_at > ?", start))
	}
	end := c.Query("CreatedAt_lte")
	if len(end) > 0 {
		where = append(where, handlers.Args("created_at < ?", end))
	}
	tenant := c.Query("Tenant")
	if len(tenant) > 0 {
		where = append(where, handlers.Args("tenant = ?", tenant))
	}
	action := c.Query("Action")
	if len(action) > 0 {
		where = append(where, handlers.Args("action = ?", action))
	}
	username := c.Query("Username")
	if len(username) > 0 {
		where = append(where, handlers.Args("username = ?", username))
	}
	success := c.Query("Success")
	if len(success) > 0 {
		where = append(where, handlers.Args("success = ?", success == "true"))
	}
	cond := &handlers.PageQueryCond{
		Model:        ModelName,
		Where:        where,
		SearchFields: []string{"name"},
	}
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()).Order("id DESC"), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// RetrieveAuditLog AuditLog详情
// @Tags        AuditLog
// @Summary     AuditLog详情
// @Description get AuditLog详情
// @Accept      json
// @Produce     json
// @Param       auditlog_id path     uint                                          true "auditlog_id"
// @Success     200         {object} handlers.ResponseStruct{Data=models.AuditLog} "AuditLog"
// @Router      /v1/auditlog/{auditlog_id} [get]
// @Security    JWT
func (h *AuditLogHandler) RetrieveAuditLog(c *gin.Context) {
	var obj models.AuditLog
	if err := h.GetDB().WithContext(c.Request.Context()).First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}
