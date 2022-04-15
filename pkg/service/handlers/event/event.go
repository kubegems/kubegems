package eventhandler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/loki"
)

// QueryRange 获取事件
// @Tags         Event
// @Summary      获取事件
// @Description  获取事件
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                true   "cluster_name"
// @Param        query    query     string                                true   "query"
// @Param        limit    query     int                                   false  "limit"
// @Param        start    query     string                                false  "start"
// @Param        end      query     string                                false  "end"
// @Success      200      {object}  handlers.ResponseStruct{Data=string}  "QueryRange"
// @Router       /v1/event/{cluster} [get]
// @Security     JWT
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
		handlers.NotOK(c, errors.New("超过5000条限制"))
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
