package clusterhandler

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

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/loki"
)

const LokiExportDir = "lokiExport"

func (l *Handler) QueryRange(req *restful.Request, resp *restful.Response) {
	cluster := req.PathParameter("cluster")
	var query loki.QueryRangeParam
	if err := handlers.BindQuery(req, &query); err != nil {
		handlers.BadRequest(resp, err)
	}

	level := req.QueryParameter("level")
	if level != "" {
		levelExpr := loki.GenerateLevelRegex(level)
		if levelExpr != "" {
			query.Query = fmt.Sprintf("%s %s", query.Query, levelExpr)
		}
	}

	lmtStr := query.Limit
	if query.Limit == "" {
		lmtStr = "500"
	} else {
	}
	limit, _ := strconv.Atoi(lmtStr)
	if limit > 50000 {
		handlers.BadRequest(resp, errors.New("最大支持50000条日志输出"))
		return
	}

	ctx := req.Request.Context()
	queryData, err := l.LokiQueryRange(ctx, cluster, query.ToMap())
	if err != nil {
		handlers.BadRequest(resp, err)
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
				for _, filter := range req.QueryParameters("filters") {
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
	if req.QueryParameter("pod") == "" {
		data["pod"] = podResults
	}
	data["resultType"] = resultType

	handlers.OK(resp, data)
}

func (l *Handler) Labels(req *restful.Request, resp *restful.Response) {
	var query loki.LabelParam
	if err := handlers.BindQuery(req, &query); err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	labelData, err := l.LokiLabels(req.Request.Context(), req.PathParameter("cluster"), query.ToMap())
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	handlers.OK(resp, labelData)
}

func (l *Handler) LabelValues(req *restful.Request, resp *restful.Response) {
	var query loki.LabelParam
	if err := handlers.BindQuery(req, &query); err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	label := req.QueryParameter("label")
	labelData, err := l.LokiLabelValues(req.Request.Context(), req.PathParameter("cluster"), label, query.ToMap())
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, labelData)
}

func (l *Handler) Export(req *restful.Request, resp *restful.Response) {
	var query loki.QueryRangeParam
	if err := handlers.BindQuery(req, &query); err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	level := req.QueryParameter("level")
	if level != "" {
		levelExpr := loki.GenerateLevelRegex(level)
		if levelExpr != "" {
			query.Query = fmt.Sprintf("%s %s", query.Query, levelExpr)
		}
	}

	err := utils.EnsurePathExists(LokiExportDir)
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	filename := fmt.Sprintf("%s.log", time.Now().UTC().Format("20060102150405"))
	targetOutputFile := path.Join(LokiExportDir, filename)
	file, err := os.Create(targetOutputFile)
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	defer file.Close()

	_, err = file.WriteString("\xEF\xBB\xBF")
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	res := make(map[string]interface{})
	res["exist"] = true

	ctx := req.Request.Context()

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

		queryData, err := l.LokiQueryRange(ctx, req.PathParameter("cluster"), query.ToMap())
		if err != nil {
			index--
			continue
		}

		resultType := queryData.ResultType
		if resultType == loki.ResultTypeMatrix {
			// 暂不支持matrix
			handlers.BadRequest(resp, errors.New("不支持matrix类型导出"))
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
					handlers.BadRequest(resp, err)
					return
				}
			}
		}
	}

	res["filename"] = filename
	handlers.OK(resp, res)
}

func (l *Handler) Context(req *restful.Request, resp *restful.Response) {
	var query loki.QueryRangeParam
	if err := handlers.BindData(req, &query); err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	queryData, err := l.LokiQueryRange(req.Request.Context(), req.PathParameter("cluster"), query.ToMap())
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	queryResults := []interface{}{}
	resultType := queryData.ResultType
	if resultType == "matrix" {
		handlers.BadRequest(resp, errors.New("暂不支持matrix类型查询"))
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

	handlers.OK(resp, queryResults)
}

func (l *Handler) QueryLanguage(req *restful.Request, resp *restful.Response) {
	filters := req.QueryParameters("filters")
	pod := req.QueryParameter("pod")
	start := fmt.Sprintf("%d", time.Now().UTC().Add(time.Hour*-24).UnixNano())
	end := fmt.Sprintf("%d", time.Now().UTC().UnixNano())

	queryExprArray := []string{}
	labelRes, _ := l.LokiLabels(req.Request.Context(), req.PathParameter("cluster"), map[string]string{"start": start, "end": end})
	for _, label := range labelRes {
		if req.QueryParameter(label) != "" {
			queryExprArray = append(queryExprArray, loki.GetExpr(label, req.QueryParameter(label)))
		}
	}

	if pod != "" {
		queryExprArray = append(queryExprArray, loki.GetExpr("pod", pod))
	}

	queryExpr := fmt.Sprintf("{%s}", strings.Join(queryExprArray, ","))
	for _, filter := range filters {
		_, err := regexp.Compile(filter)
		if err != nil {
			handlers.OK(resp, "")
			return
		}
		queryExpr = fmt.Sprintf("%s |~ `%s`", queryExpr, filter)
	}

	handlers.OK(resp, queryExpr)
}

func (l *Handler) Series(req *restful.Request, resp *restful.Response) {
	var query loki.SeriesForm
	if err := handlers.BindQuery(req, &query); err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	seriesData, err := l.LokiSeries(req.Request.Context(), req.PathParameter("cluster"), query.ToMap())
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, seriesData)
}
