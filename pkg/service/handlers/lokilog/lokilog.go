package lokilog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/loki"
)

const LokiExportDir = "lokiExport"

// QueryRange 获取loki查询结果
// @Tags         Log
// @Summary      获取loki查询结果
// @Description  获取loki查询结果
// @Accept       json
// @Produce      json
// @Param        cluster_name  path      string                                true   "cluster_name"
// @Param        filters[]     query     array                                 false  "filters[]"
// @Param        query         query     string                                true   "query"
// @Param        level         query     string                                false  "level"
// @Param        step          query     string                                false  "step"
// @Param        pod           query     string                                false  "pod"
// @Param        interval      query     string                                false  "interval"
// @Param        direction     query     string                                false  "direction"
// @Param        limit         query     int                                   false  "limit"
// @Param        start         query     string                                false  "start"
// @Param        end           query     string                                false  "end"
// @Success      200           {object}  handlers.ResponseStruct{Data=string}  "QueryRange"
// @Router       /v1/log/{cluster_name}/queryrange [get]
// @Security     JWT
func (l *LogHandler) QueryRange(c *gin.Context) {
	var query loki.QueryRangeParam
	if err := c.ShouldBindQuery(&query); err != nil {
		handlers.NotOK(c, err)
		return
	}

	level := c.Query("level")
	if level != "" {
		levelExpr := loki.GenerateLevelRegex(level)
		if levelExpr != "" {
			query.Query = fmt.Sprintf("%s %s", query.Query, levelExpr)
		}
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "500"))
	if limit > 50000 {
		handlers.NotOK(c, errors.New("最大支持50000条日志输出"))
		return
	}

	ctx := c.Request.Context()
	queryData, err := l.LokiQueryRange(ctx, c.Param("cluster_name"), query.ToMap())
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	cstZone := time.FixedZone("GMT", 8*3600)
	var queryResults []interface{}
	chartResult := make(map[string]interface{})
	podResults := []interface{}{}
	podSetStr := ""
	resultType := queryData.ResultType
	results := queryData.Result
	if resultType == loki.ResultTypeMatrix {
		chartResult["yAxis-data"] = make(map[string][]string)
		chartResult["table-data"] = []string{}

		if len(results) > 50 {
			results = results[0:50]
			chartResult["long"] = true
		}

		xdata := []int64{}
		for index, result := range results {
			var matrix loki.SampleStream
			matrix = matrix.ToStruct(result.(map[string]interface{}))

			d, _ := json.Marshal(matrix.Metric)
			chartResult["table-data"] = append(chartResult["table-data"].([]string), string(d))
			values := matrix.Values
			ydata := []string{}
			for _, v := range values {
				if index == 0 {
					timestamp := int64(v[0].(float64))
					xdata = append(xdata, timestamp)
				}
				value := v[1].(string)
				ydata = append(ydata, value)

			}
			chartResult["yAxis-data"].(map[string][]string)[fmt.Sprintf("%d", index)] = ydata
		}
		chartResult["xAxis-data"] = xdata
	} else {
		size := 10
		splitDateTimeArray, step := loki.SplitDateTime(query.Start, query.End, size)
		chartResult["xAxis-data"] = splitDateTimeArray
		chartResult["yAxis-data"] = loki.InitSplitDateTime(size)

		for _, result := range results {
			var stream loki.Stream
			stream = stream.ToStruct(result.(map[string]interface{}))
			// pod信息
			podKey := ""
			for key := range stream.Labels {
				if strings.Contains(key, "pod") {
					podKey = key
					break
				}
			}
			if podKey != "" && !strings.Contains(podSetStr, stream.Labels[podKey]) {
				podMap := make(map[string]interface{})
				podMap["text"] = stream.Labels[podKey]
				podMap["selected"] = false
				podResults = append(podResults, podMap)
				podSetStr += fmt.Sprintf("%s,", stream.Labels[podKey])
			}

			values := stream.Entries
			for index, value := range values {
				item := loki.QueryResult{}
				item.Stream = stream.Labels
				timestamp := value[0]
				message := value[1]

				info := loki.Info{}
				info.Timestamp = timestamp
				t, _ := strconv.ParseInt(timestamp, 10, 64)
				info.Timestampstr = time.Unix(0, t*int64(time.Nanosecond)).In(cstZone).Format("2006-01-02 15:04:05.000")
				info.Message = message
				info.Message = loki.ShellHighlightShow(info.Message)
				for _, filter := range c.QueryArray("filters[]") {
					info.Message = loki.RegexHighlightShow(info.Message, filter)
				}

				// 正则匹配出日志类型
				logLevel := loki.LogLevel(message)
				info.Level = logLevel
				info.Animation = ""
				info.Index = fmt.Sprintf("%s-%d", timestamp, index)
				item.Info = info

				// 获取表格数据
				part := loki.TimeInPart(splitDateTimeArray, timestamp, step)
				if part >= 0 && part < size {
					chartResult["yAxis-data"].(map[string][]int)[logLevel][part]++
				}

				queryResults = append(queryResults, item)
			}
		}
	}

	data := make(map[string]interface{})
	data["query"] = queryResults
	data["chart"] = chartResult
	if c.Query("pod") == "" {
		data["pod"] = podResults
	}
	data["resultType"] = resultType

	handlers.OK(c, data)
}

// Labels 获取loki标签
// @Tags         Log
// @Summary      获取loki标签
// @Description  获取loki标签
// @Accept       json
// @Produce      json
// @Param        cluster_name  path      string                                true   "cluster_name"
// @Param        start         query     string                                false  "start"
// @Param        end           query     string                                false  "end"
// @Success      200           {object}  handlers.ResponseStruct{Data=string}  "Labels"
// @Router       /v1/log/{cluster_name}/labels [get]
// @Security     JWT
func (l *LogHandler) Labels(c *gin.Context) {
	var query loki.LabelParam
	if err := c.ShouldBindQuery(&query); err != nil {
		handlers.NotOK(c, err)
		return
	}

	labelData, err := l.LokiLabels(c.Request.Context(), c.Param("cluster_name"), query.ToMap())
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, labelData)
}

// LabelValues 获取loki指定标签值
// @Tags         Log
// @Summary      获取loki指定标签值
// @Description  获取loki指定标签值
// @Accept       json
// @Produce      json
// @Param        cluster_name  path      string                                true   "cluster_name"
// @Param        label         path      string                                true   "label"
// @Param        start         query     string                                false  "start"
// @Param        end           query     string                                false  "end"
// @Success      200           {object}  handlers.ResponseStruct{Data=string}  "LabelValues"
// @Router       /v1/log/{cluster_name}/label/{label}/values [get]
// @Security     JWT
func (l *LogHandler) LabelValues(c *gin.Context) {
	var query loki.LabelParam
	if err := c.ShouldBindQuery(&query); err != nil {
		handlers.NotOK(c, err)
		return
	}

	labelData, err := l.LokiLabelValues(c.Request.Context(), c.Param("cluster_name"), c.Param("label"), query.ToMap())
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, labelData)
}

// Export 导出loki查询结果
// @Tags         Log
// @Summary      导出loki查询结果
// @Description  导出loki查询结果
// @Accept       json
// @Produce      json
// @Param        cluster_name  path      string                                true   "cluster_name"
// @Param        query         query     string                                true   "query"
// @Param        interval      query     string                                false  "interval"
// @Param        direction     query     string                                false  "direction"
// @Param        limit         query     int                                   false  "limit"
// @Param        start         query     string                                false  "start"
// @Param        end           query     string                                false  "end"
// @Success      200           {object}  handlers.ResponseStruct{Data=string}  "Export"
// @Router       /v1/log/{cluster_name}/export [get]
// @Security     JWT
func (l *LogHandler) Export(c *gin.Context) {
	var query loki.QueryRangeParam
	if err := c.ShouldBindQuery(&query); err != nil {
		handlers.NotOK(c, err)
		return
	}

	level := c.Param("level")
	if level != "" {
		levelExpr := loki.GenerateLevelRegex(level)
		if levelExpr != "" {
			query.Query = fmt.Sprintf("%s %s", query.Query, levelExpr)
		}
	}

	err := utils.EnsurePathExists(LokiExportDir)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	filename := fmt.Sprintf("%s.log", time.Now().UTC().Format("20060102150405"))
	targetOutputFile := path.Join(LokiExportDir, filename)
	file, err := os.Create(targetOutputFile)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	defer file.Close()

	_, err = file.WriteString("\xEF\xBB\xBF")
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	res := make(map[string]interface{})
	res["exist"] = true

	ctx := c.Request.Context()

	length := 1
	index := 0
	for {
		if index >= 10 {
			break
		}
		index++
		if length == 0 {
			break
		}

		queryData, err := l.LokiQueryRange(ctx, c.Param("cluster_name"), query.ToMap())
		if err != nil {
			index--
			continue
		}

		resultType := queryData.ResultType
		if resultType == loki.ResultTypeMatrix {
			// 暂不支持matrix
			handlers.NotOK(c, errors.New("不支持matrix类型导出"))
			return
		}

		results := queryData.Result
		messages := loki.LokiMessages{}
		for _, result := range results {
			var stream loki.Stream
			stream = stream.ToStruct(result.(map[string]interface{}))

			values := stream.Entries
			for _, value := range values {
				messages = append(messages, loki.LokiMessage{Timestamp: value[0], Message: value[1]})
			}
		}
		length = len(messages)

		if length > 0 {
			if query.Direction == "backward" {
				sort.Sort(messages)
				query.End = messages[len(messages)-1].Timestamp
			} else {
				sort.Sort(sort.Reverse(messages))
				query.Start = messages[len(messages)-1].Timestamp
			}
			for _, message := range messages {
				_, err = file.WriteString(message.Message + "\r\n")
				if err != nil {
					handlers.NotOK(c, err)
					return
				}
			}
		}
	}

	res["filename"] = filename
	handlers.OK(c, res)
}

// Context 获取loki上下文
// @Tags         Log
// @Summary      获取loki上下文
// @Description  获取loki上下文
// @Accept       json
// @Produce      json
// @Param        cluster_name  path      string                                true   "cluster_name"
// @Param        query         query     string                                true   "query"
// @Param        step          query     string                                false  "step"
// @Param        interval      query     string                                false  "interval"
// @Param        direction     query     string                                false  "direction"
// @Param        limit         query     int                                   false  "limit"
// @Param        start         query     string                                false  "start"
// @Param        end           query     string                                false  "end"
// @Success      200           {object}  handlers.ResponseStruct{Data=string}  "Context"
// @Router       /v1/log/{cluster_name}/context [get]
// @Security     JWT
func (l *LogHandler) Context(c *gin.Context) {
	var query loki.QueryRangeParam
	if err := c.ShouldBindQuery(&query); err != nil {
		handlers.NotOK(c, err)
		return
	}

	queryData, err := l.LokiQueryRange(c.Request.Context(), c.Param("cluster_name"), query.ToMap())
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	queryResults := []interface{}{}
	resultType := queryData.ResultType
	if resultType == "matrix" {
		handlers.NotOK(c, errors.New("暂不支持matrix类型查询"))
		return
	}

	results := queryData.Result
	for _, result := range results {
		var stream loki.Stream
		stream = stream.ToStruct(result.(map[string]interface{}))

		for _, value := range stream.Entries {
			item := make(map[string]interface{})
			// 正则匹配出日志类型
			logLevel := loki.LogLevel(value[1])
			item["timestamp"] = value[0]
			item["level"] = logLevel
			item["message"] = loki.ShellHighlightShow(value[1])
			queryResults = append(queryResults, item)
		}
	}

	handlers.OK(c, queryResults)
}

// QueryLanguage 获取loki查询语句
// @Tags         Log
// @Summary      获取loki查询语句
// @Description  获取loki查询语句
// @Accept       json
// @Produce      json
// @Param        cluster_name  path      string                                true   "cluster_name"
// @Param        filters[]     query     array                                 false  "filters[]"
// @Param        pod           query     string                                false  "pod"
// @Param        start         query     string                                false  "start"
// @Param        end           query     string                                false  "end"
// @Success      200           {object}  handlers.ResponseStruct{Data=string}  "QueryLanguage"
// @Router       /v1/log/{cluster_name}/querylanguage [get]
// @Security     JWT
func (l *LogHandler) QueryLanguage(c *gin.Context) {
	filters := c.QueryArray("filters[]")
	pod := c.DefaultQuery("pod", "")
	start := fmt.Sprintf("%d", time.Now().UTC().Add(time.Hour*-24).UnixNano())
	end := fmt.Sprintf("%d", time.Now().UTC().UnixNano())

	queryExprArray := []string{}
	labelRes, _ := l.LokiLabels(c.Request.Context(), c.Param("cluster_name"), map[string]string{"start": start, "end": end})
	for _, label := range labelRes {
		if c.DefaultQuery(label, "") != "" {
			queryExprArray = append(queryExprArray, loki.GetExpr(label, c.DefaultQuery(label, "")))
		}
	}

	if pod != "" {
		queryExprArray = append(queryExprArray, loki.GetExpr("pod", pod))
	}

	queryExpr := fmt.Sprintf("{%s}", strings.Join(queryExprArray, ","))
	for _, filter := range filters {
		_, err := regexp.Compile(filter)
		if err != nil {
			handlers.OK(c, "")
			return
		}
		queryExpr = fmt.Sprintf("%s |~ `%s`", queryExpr, filter)
	}

	handlers.OK(c, queryExpr)
}

// LabelValues 获取loki Series
// @Tags         Log
// @Summary      获取loki Series
// @Description  获取loki Series
// @Accept       json
// @Produce      json
// @Param        cluster_name  path      string                                true   "cluster_name"
// @Param        match         query     string                                true   "match"
// @Param        start         query     string                                false  "start"
// @Param        end           query     string                                false  "end"
// @Success      200           {object}  handlers.ResponseStruct{Data=string}  "Series data"
// @Router       /v1/log/{cluster_name}/series [get]
// @Security     JWT
func (l *LogHandler) Series(c *gin.Context) {
	var query loki.SeriesForm
	if err := c.ShouldBindQuery(&query); err != nil {
		handlers.NotOK(c, err)
		return
	}

	seriesData, err := l.LokiSeries(c.Request.Context(), c.Param("cluster_name"), query.ToMap())
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, seriesData)
}
