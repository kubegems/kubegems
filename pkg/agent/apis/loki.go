package apis

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/grafana/loki/pkg/logcli/client"
	"github.com/grafana/loki/pkg/loghttp"
	"kubegems.io/pkg/agent/ws"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/loki"
)

var cstZone = time.FixedZone("GMT", 8*3600)

func (h *LokiHandler) _http(path string, method string, params map[string]string, data interface{}) ([]byte, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			ReadBufferSize:  4 << 20,
		},
	}

	paramStr := _query(params)
	requestData, _ := json.Marshal(data)
	url := fmt.Sprintf("%s%s?%s", h.Server, path, paramStr)
	log.WithField("h", "loki").Infof("http request to: %v", url)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, fmt.Errorf("初始化 requests 错误 %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求loki错误 %v", err)
	}

	body, _ := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if response.StatusCode >= 200 || response.StatusCode < 300 {
		return body, nil
	} else {
		return body, fmt.Errorf("请求loki异常 code=%v, body=%v ", response.StatusCode, string(body))
	}
}

type LokiHandler struct {
	Server string
}

// @Tags Agent.V1
// @Summary Loki QueryRange
// @Description Loki QueryRange
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param start query string true "The start time for the query as a nanosecond Unix epoch"
// @Param end query string true "The end time for the query as a nanosecond Unix epoch"
// @Param direction query string true "The order to all results"
// @Param limit query string false "The max number of entries to return"
// @Param query query string true "loki query language"
// @Success 200 {object} handlers.ResponseStruct{Data=object} ""
// @Router /v1/proxy/cluster/{cluster}/custom/loki/v1/queryrange [get]
// @Security JWT
func (h *LokiHandler) QueryRange(c *gin.Context) {
	var data loki.QueryRangeParam
	if err := c.ShouldBindQuery(&data); err != nil {
		NotOK(c, err)
		return
	}

	data.Query, _ = url.QueryUnescape(data.Query)
	body, err := h._http("/loki/api/v1/query_range", "GET", data.ToMap(), nil)
	if err != nil {
		NotOK(c, fmt.Errorf("请求错误 %v", err))
		return
	}
	res := loki.QueryResponse{}
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		NotOK(c, fmt.Errorf("解析loki数据错误 err=%v,data=%v", err.Error(), string(body[:20])))
		return
	}
	OK(c, res.Data)
}

// @Tags Agent.V1
// @Summary Loki Labels
// @Description Loki Labels
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param start query string true "The start time for the query as a nanosecond Unix epoch"
// @Param end query string true "The end time for the query as a nanosecond Unix epoch"
// @Success 200 {object} handlers.ResponseStruct{Data=object} ""
// @Router /v1/proxy/cluster/{cluster}/custom/loki/v1/labels [get]
// @Security JWT
func (h *LokiHandler) Labels(c *gin.Context) {
	var data loki.LabelParam
	if err := c.ShouldBindQuery(&data); err != nil {
		NotOK(c, err)
		return
	}

	body, err := h._http("/loki/api/v1/labels", "GET", data.ToMap(), nil)
	if err != nil {
		NotOK(c, err)
		return
	}

	res := loki.LabelResponse{}
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		NotOK(c, fmt.Errorf("解析loki数据错误 err=%v,data=%v", err.Error(), string(body[:20])))
		return
	}
	OK(c, res.Data)
}

// @Tags Agent.V1
// @Summary Loki LabelValues
// @Description Loki LabelValues
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param start query string true "The start time for the query as a nanosecond Unix epoch"
// @Param end query string true "The end time for the query as a nanosecond Unix epoch"
// @Param label query string true "label"
// @Success 200 {object} handlers.ResponseStruct{Data=object} ""
// @Router /v1/proxy/cluster/{cluster}/custom/loki/v1/labelvalues [get]
// @Security JWT
func (h *LokiHandler) LabelValues(c *gin.Context) {
	var data loki.LabelParam
	if err := c.ShouldBindQuery(&data); err != nil {
		NotOK(c, err)
		return
	}

	body, err := h._http(fmt.Sprintf("/loki/api/v1/label/%s/values", data.Label), "GET", data.ToMap(), nil)
	if err != nil {
		NotOK(c, err)
		return
	}

	res := loki.LabelResponse{}
	if err := json.Unmarshal(body, &res); err != nil {
		NotOK(c, fmt.Errorf("解析loki数据错误 err=%v,data=%v", err.Error(), string(body[:20])))
		return
	}
	OK(c, res.Data)
}

func _query(params map[string]string) string {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return q.Encode()
}

// @Tags Agent.V1
// @Summary Loki Series
// @Description Loki Series
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param start query string true "The start time for the query as a nanosecond Unix epoch"
// @Param end query string true "The end time for the query as a nanosecond Unix epoch"
// @Param match query string true "match"
// @Success 200 {object} handlers.ResponseStruct{Data=object} ""
// @Router /v1/proxy/cluster/{cluster}/custom/loki/v1/series [get]
// @Security JWT
func (h *LokiHandler) Series(c *gin.Context) {
	var data loki.SeriesForm
	if err := c.ShouldBindQuery(&data); err != nil {
		NotOK(c, err)
		return
	}
	body, err := h._http("/loki/api/v1/series", "GET", data.ToMap(), nil)
	if err != nil {
		NotOK(c, err)
		return
	}

	res := loki.SeriesResponse{}
	if err := json.Unmarshal(body, &res); err != nil {
		NotOK(c, fmt.Errorf("解析loki数据错误 err=%v,data=%v", err.Error(), string(body[:20])))
		return
	}
	OK(c, res.Data)
}

// @Tags Agent.V1
// @Summary Loki LabelValues
// @Description Loki LabelValues
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param start query string true "The start time for the query as a nanosecond Unix epoch"
// @Param limit query string false "The max number of entries to return"
// @Param query query string true "loki query language"
// @Param delay_for query string true "The number of seconds to delay retrieving logs to let slow loggers catch up. Defaults to 0 and cannot be larger than 5."
// @Param stream query string true "must be true"
// @Success 200 {object} handlers.ResponseStruct{Data=object} ""
// @Router /v1/proxy/cluster/{cluster}/custom/loki/v1/tail [get]
// @Security JWT
func (h *LokiHandler) Tail(c *gin.Context) {
	var queryParam loki.TailParam

	wsServer, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.WithField("h", "tail").Errorf("upgrade websocket failed: %v", err)
		NotOK(c, err)
		return
	}
	defer wsServer.Close()

	lokiurl, err := url.Parse(h.Server)
	if err != nil {
		log.WithField("h", "tail").Errorf("parse server failed: %v", err)
		NotOK(c, err)
		return
	}

	if err := c.BindHeader(&queryParam); err != nil {
		log.WithField("h", "tail").Errorf("query failed: %v", err)
		NotOK(c, err)
		return
	}
	if err := c.Bind(&queryParam); err != nil {
		log.WithField("h", "tail").Errorf("query failed: %v", err)
		NotOK(c, err)
		return
	}

	queryParam.Query, _ = url.QueryUnescape(queryParam.Query)
	filterArgs := strings.Split(queryParam.Filter, ",")
	if queryParam.Level != "" {
		levelExpr := loki.GenerateLevelRegex(queryParam.Level)
		if levelExpr != "" {
			queryParam.Query = fmt.Sprintf("%s %s", queryParam.Query, levelExpr)
		}
	}

	lokiTailURL := url.URL{
		Scheme:   "ws",
		Host:     lokiurl.Host,
		Path:     "/loki/api/v1/tail",
		RawQuery: _query(queryParam.ToMap()),
	}

	log.WithField("h", "tail").Info(lokiTailURL.String())
	wsClient, resp, err := websocket.DefaultDialer.Dial(lokiTailURL.String(), nil)
	if err != nil {
		NotOK(c, err)
		return
	}
	defer resp.Body.Close()
	defer wsClient.Close()

	sHandler := newLogStreamHandler(wsServer, wsClient, filterArgs)
	sHandler.handle()
	log.WithField("h", "tail").Info("end with handle")
}

type logStreamHandler struct {
	serverConn *websocket.Conn
	clientConn *websocket.Conn
	filterArgs []string
}

func newLogStreamHandler(serverConn, clientConn *websocket.Conn, filterArgs []string) *logStreamHandler {
	return &logStreamHandler{
		serverConn: serverConn,
		clientConn: clientConn,
		filterArgs: filterArgs,
	}
}

func (l *logStreamHandler) handle() {
	go func() {
		for {
			_, _, e := l.serverConn.ReadMessage()
			if e != nil {
				log.WithField("h", "tail").Infof("exit handle due %v", e.Error())
				l.serverConn.Close()
				l.clientConn.Close()
				return
			}
		}
	}()
	defer func() {
		l.serverConn.Close()
		l.clientConn.Close()
	}()
	for {
		msgType, msgContent, err := l.clientConn.ReadMessage()
		if err != nil {
			log.WithField("h", "tail").Infof("exit read handle due %v", err.Error())
			return
		}
		convertedContent, err := l.convertMessage(msgContent)
		if err != nil {
			log.WithField("h", "tail").Infof("exit convert handle due %v", err.Error())
			return
		}
		if len(convertedContent) == 0 {
			continue
		}
		if err := l.serverConn.WriteMessage(msgType, convertedContent); err != nil {
			log.WithField("h", "tail").Infof("exit send handle due %v", err.Error())
			return
		}
	}
}

func (l *logStreamHandler) convertMessage(content []byte) ([]byte, error) {
	var (
		queryResults []interface{}
		m            map[string]interface{}
		stream       loki.Stream
	)
	if len(content) == 0 {
		return nil, fmt.Errorf("empty content")
	}
	if err := json.Unmarshal(content, &m); err != nil {
		return nil, err
	}
	tmpStream, exist := m["streams"]
	if !exist {
		return []byte{}, nil
	}
	results := tmpStream.([]interface{})
	for _, result := range results {
		stream = stream.ToStruct(result.(map[string]interface{}))
		values := stream.Entries
		for index, value := range values {
			item := loki.QueryResult{
				Stream: stream.Labels,
			}
			t, _ := strconv.ParseInt(value[0], 10, 64)
			info := loki.Info{
				Timestamp:    value[0],
				Timestampstr: time.Unix(0, t*int64(time.Nanosecond)).In(cstZone).Format("2006-01-02 15:04:05.000"),
				Message:      loki.ShellHighlightShow(value[1]),
				Level:        loki.LogLevel(value[1]),
				Animation:    "background-color: yellow;transition: background-color 2s;",
				Index:        fmt.Sprintf("%s-%d", value[0], index),
			}

			for _, filter := range l.filterArgs {
				info.Message = loki.RegexHighlightShow(info.Message, filter)
			}

			item.Info = info
			queryResults = append(queryResults, item)
		}
	}
	return json.Marshal(queryResults)
}

// TODO: 官方LOKI SDK; 由于SERVICE端还要改造，留着一起处理
type LokiCliHandler struct {
	Server string
}

func (h *LokiCliHandler) cli() client.Client {
	return &client.DefaultClient{
		Address: h.Server,
	}
}

func prepareRequest(c *gin.Context) *http.Request {
	r := c.Copy().Request
	r.ParseForm()
	q := r.Form.Get("query")
	if len(q) > 0 {
		return r
	}
	escaped, _ := url.QueryUnescape(q)
	r.Form.Set("query", escaped)

	queries := r.URL.Query()
	for k, v := range r.Header {
		if len(v) > 0 {
			queries.Set(k, v[0])
		}
	}
	r.URL.RawQuery = queries.Encode()
	return r
}

func (h *LokiCliHandler) QueryRange(c *gin.Context) {
	cpReq := prepareRequest(c)
	req, err := loghttp.ParseRangeQuery(cpReq)
	if err != nil {
		NotOK(c, fmt.Errorf("参数错误: %v", err))
		return
	}
	resp, err := h.cli().QueryRange(req.Query, int(req.Limit), req.Start, req.End, req.Direction, req.Step, req.Interval, false)
	if err != nil {
		NotOK(c, fmt.Errorf("查询错误 %v", err))
		return
	}
	OK(c, resp.Data)
}

func (h *LokiCliHandler) LabelNames(c *gin.Context) {
	cpReq := prepareRequest(c)
	req, err := loghttp.ParseLabelQuery(cpReq)
	if err != nil {
		NotOK(c, fmt.Errorf("参数错误: %v", err))
		return
	}
	resp, err := h.cli().ListLabelNames(false, *req.Start, *req.End)
	if err != nil {
		NotOK(c, fmt.Errorf("查询错误 %v", err))
		return
	}
	OK(c, resp.Data)
}

func (h *LokiCliHandler) LabelValues(c *gin.Context) {
	cpReq := prepareRequest(c)
	req, err := loghttp.ParseLabelQuery(cpReq)
	if err != nil {
		NotOK(c, fmt.Errorf("参数错误: %v", err))
		return
	}
	resp, err := h.cli().ListLabelValues(req.Name, false, *req.Start, *req.End)
	if err != nil {
		NotOK(c, fmt.Errorf("查询错误 %v", err))
		return
	}
	OK(c, resp.Data)
}

func (h *LokiCliHandler) Series(c *gin.Context) {
	cpReq := prepareRequest(c)
	req, err := loghttp.ParseSeriesQuery(cpReq)
	if err != nil {
		NotOK(c, fmt.Errorf("参数错误: %v", err))
		return
	}
	resp, err := h.cli().Series(req.Groups, req.Start, req.End, false)
	if err != nil {
		NotOK(c, fmt.Errorf("查询错误 %v", err))
		return
	}
	OK(c, resp.Data)
}

func (h *LokiCliHandler) Tail(c *gin.Context) {
	cpReq := prepareRequest(c)
	req, err := loghttp.ParseTailQuery(cpReq)
	if err != nil {
		NotOK(c, fmt.Errorf("参数错误: %v", err))
		return
	}
	wsconn, err := h.cli().LiveTailQueryConn(req.Query, time.Duration(req.DelayFor), int(req.Limit), req.GetStart(), false)
	if err != nil {
		NotOK(c, fmt.Errorf("查询错误 %v", err))
		return
	}
	defer wsconn.Close()

	wsServer, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.WithField("h", "tail").Errorf("upgrade websocket failed: %v", err)
		NotOK(c, err)
		return
	}
	defer wsServer.Close()

	level := c.Query("level")
	filterArgs := strings.Split(req.Query, ",")

	if level != "" {
		levelExpr := loki.GenerateLevelRegex(level)
		if levelExpr != "" {
			req.Query = fmt.Sprintf("%s %s", req.Query, levelExpr)
		}
	}

	sHandler := newLogStreamHandler(wsServer, wsconn, filterArgs)
	sHandler.handle()
	log.WithField("h", "tail").Info("end with handle")
}
