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

package eventhandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/loki"
	"kubegems.io/kubegems/pkg/utils/set"
)

// QueryRange 获取事件
//
//	@Tags			Event
//	@Summary		获取事件
//	@Description	获取事件
//	@Accept			json
//	@Produce		json
//	@Param			cluster	path		string									true	"cluster_name"
//	@Param			query	query		string									true	"query"
//	@Param			condition	query		string									false	"condition"
//	@Param			tenant 	query		string									false	"tenant"
//	@Param			limit	query		int										false	"limit"
//	@Param			start	query		string									false	"start"
//	@Param			end		query		string									false	"end"
//	@Success		200		{object}	handlers.ResponseStruct{Data=string}	"QueryRange"
//	@Router			/v1/event/{cluster} [get]
//	@Security		JWT
func (l *EventHandler) Event(c *gin.Context) {
	options := GetEventOptions{
		Tenant:    c.Query("tenant"),
		Cluster:   c.Param("cluster"),
		Start:     c.Query("start"),
		End:       c.Query("end"),
		Condition: c.Query("condition"),
	}
	options.Limit, _ = strconv.Atoi(c.Query("limit"))

	// nolint: nestif
	if options.Tenant != "" {
		ok, err := l.HasTenantPerm(c, options.Tenant)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		if !ok {
			handlers.Forbidden(c, fmt.Errorf("no permission to access tenant %s", options.Tenant))
			return
		}
		// get all namespaces of the tenant
		namespaces, err := ListAllTenantNamespaces(c.Request.Context(), l.GetDataBase(), options.Tenant)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		options.Namespaces = namespaces
	} else {
		ok, err := l.HasSystemAdminPerm(c)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		if !ok {
			handlers.Forbidden(c, fmt.Errorf("no permission to access cluster events"))
			return
		}
	}
	queryData, err := l.GetEvent(c, options)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	var queryResults []interface{}
	if queryData != nil {
		queryResults = queryData.Result
	}

	handlers.OK(c, queryResults)
}

func ListAllTenantNamespaces(ctx context.Context, db *database.Database, tenantname string) ([]string, error) {
	var list []models.Environment
	// TODO: improve performance by using a single query
	if err := db.
		DB().
		WithContext(ctx).
		Preload("Project.Tenant").
		Find(&list).Error; err != nil {
		return nil, err
	}
	namespaces := set.NewSet[string]()
	for _, env := range list {
		if env.Project == nil || env.Project.Tenant == nil || env.Project.Tenant.TenantName != tenantname {
			continue
		}
		namespaces.Append(env.Namespace)
	}
	return namespaces.Slice(), nil
}

type GetEventOptions struct {
	Tenant     string   `json:"tenant"`
	Condition  string   `json:"condition"`
	Cluster    string   `json:"cluster"`
	Namespaces []string `json:"namespaces"`
	Start      string   `json:"start,omitempty"`
	End        string   `json:"end,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

func (l *EventHandler) GetEvent(c *gin.Context, options GetEventOptions) (*loki.QueryResponseData, error) {
	query := `{container="event-exporter", stream="stdout"} | json | __error__=""`
	if len(options.Namespaces) != 0 {
		query += fmt.Sprintf(` | metadata_namespace =~ "%s"`, strings.Join(options.Namespaces, "|"))
	}
	if options.Condition != "" {
		query += fmt.Sprintf(` | %s`, options.Condition)
	}
	lokiq := loki.QueryRangeParam{
		Query:     query,
		Start:     options.Start,
		End:       options.End,
		Limit:     options.Limit,
		Direction: "backward",
	}
	queryData, err := l.LokiQueryRange(c.Request.Context(), options.Cluster, lokiq.ToMap())
	if err != nil {
		return nil, err
	}
	return queryData, nil
}
