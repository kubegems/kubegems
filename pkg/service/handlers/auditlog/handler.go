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
	"encoding/csv"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
)

var (
	ModelName      = "AuditLog"
	SearchFields   = []string{"username", "module", "name"}
	FilterFields   = []string{"Username", "Tenant", "Action", "Success", "CreatedAt_gte", "CreatedAt_lte"}
	PrimaryKeyName = "auditlog_id"
	OrderFields    = []string{"CreatedAt"}
)

// ListAuditLog 列表 AuditLog
//
//	@Tags			AuditLog
//	@Summary		AuditLog列表
//	@Description	AuditLog列表
//	@Accept			json
//	@Produce		json
//	@Param			Username		query		string																	false	"Username"
//	@Param			Tenant			query		string																	false	"Tenant"
//	@Param			Action			query		string																	false	"Action"
//	@Param			Success			query		string																	false	"Success"
//	@Param			CreatedAt_gte	query		string																	false	"CreatedAt_gte"
//	@Param			CreatedAt_lte	query		string																	false	"CreatedAt_lte"
//	@Param			page			query		int																		false	"page"
//	@Param			size			query		int																		false	"page"
//	@Param			search			query		string																	false	"search in (username,module,name)"
//	@Success		200				{object}	handlers.ResponseStruct{Data=handlers.PageData{List=[]models.AuditLog}}	"AuditLog"
//	@Router			/v1/auditlog [get]
//	@Security		JWT
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
	where, err = h.checkWhereOnTenant(c, where)
	if err != nil {
		handlers.NotOK(c, err)
		return
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

func (h *AuditLogHandler) checkWhereOnTenant(c *gin.Context, where []*handlers.QArgs) ([]*handlers.QArgs, error) {
	tenant := c.Query("Tenant")
	// nolint: nestif
	if len(tenant) > 0 {
		ok, err := h.HasTenantPerm(c, tenant)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("don't have permission to access tenant %s's audit log", tenant)
		}
		where = append(where, handlers.Args("tenant = ?", tenant))
	} else {
		isadmin, err := h.HasSystemAdminPerm(c)
		if err != nil {
			return nil, err
		}
		if !isadmin {
			return nil, fmt.Errorf("don't have permission to access all tenants")
		}
	}
	return where, nil
}

// RetrieveAuditLog AuditLog详情
//
//	@Tags			AuditLog
//	@Summary		AuditLog详情
//	@Description	get AuditLog详情
//	@Accept			json
//	@Produce		json
//	@Param			auditlog_id	path		uint											true	"auditlog_id"
//	@Success		200			{object}	handlers.ResponseStruct{Data=models.AuditLog}	"AuditLog"
//	@Router			/v1/auditlog/{auditlog_id} [get]
//	@Security		JWT
func (h *AuditLogHandler) RetrieveAuditLog(c *gin.Context) {
	var obj models.AuditLog
	if err := h.GetDB().WithContext(c.Request.Context()).First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

func (h *AuditLogHandler) ExportAuditLogExcel(c *gin.Context) {
	where, err := h.checkWhereOnTenant(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	queries := c.Request.URL.Query()
	if from := queries.Get("CreatedAt_gte"); from != "" {
		where = append(where, handlers.Args("created_at > ?", from))
	}
	if to := c.Query("CreatedAt_lte"); to != "" {
		where = append(where, handlers.Args("created_at < ?", to))
	}
	if action := c.Query("Action"); action != "" {
		where = append(where, handlers.Args("action = ?", action))
	}
	if username := c.Query("Username"); len(username) > 0 {
		where = append(where, handlers.Args("username = ?", username))
	}
	if success := c.Query("Success"); success != "" {
		where = append(where, handlers.Args("success = ?", success == "true"))
	}
	var list []models.AuditLog
	if err := h.GetDB().WithContext(c.Request.Context()).Order("id DESC").Where(where).Model(&models.AuditLog{}).Find(&list).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=auditlog.csv")
	c.Header("Content-Type", "text/csv")

	if err := toCSV(list, c.Writer); err != nil {
		handlers.NotOK(c, err)
		return
	}
}

func toCSV(list []models.AuditLog, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := []string{"ID", "Username", "Module", "Name", "Action", "Success", "Tenant", "CreatedAt"}
	toRecord := func(item models.AuditLog) []string {
		return []string{
			fmt.Sprintf("%d", item.ID),
			item.Username,
			item.Module,
			item.Name,
			item.Action,
			fmt.Sprintf("%t", item.Success),
			item.Tenant,
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}
	for _, item := range list {
		if err := csvWriter.Write(toRecord(item)); err != nil {
			return err
		}
	}
	return nil
}
