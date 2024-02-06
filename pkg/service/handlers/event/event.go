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
	"strconv"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/loki"
)

// QueryRange 获取事件
//	@Tags			Event
//	@Summary		获取事件
//	@Description	获取事件
//	@Accept			json
//	@Produce		json
//	@Param			cluster	path		string									true	"cluster_name"
//	@Param			query	query		string									true	"query"
//	@Param			limit	query		int										false	"limit"
//	@Param			start	query		string									false	"start"
//	@Param			end		query		string									false	"end"
//	@Success		200		{object}	handlers.ResponseStruct{Data=string}	"QueryRange"
//	@Router			/v1/event/{cluster} [get]
//	@Security		JWT
func (l *EventHandler) Event(c *gin.Context) {
	clustername := c.Param("cluster")
	var query loki.QueryRangeParam
	if err := c.ShouldBindQuery(&query); err != nil {
		handlers.NotOK(c, err)
		return
	}

	query.Direction = "backward"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5000"))
	if limit > 5000 {
		handlers.NotOK(c, i18n.Errorf(c, "max limitation is 5000"))
		return
	}

	queryData, err := l.LokiQueryRange(c.Request.Context(), clustername, query.ToMap())
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
