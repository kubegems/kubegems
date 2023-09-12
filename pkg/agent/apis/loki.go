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

package apis

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"kubegems.io/kubegems/pkg/agent/ws"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/loki"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/library/rest/response"
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
	log.Debugf("http request to: %v", url)
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
	proxy  *LokiHTTPProxy
}

func NewLokiHandler(server string) (*LokiHandler, error) {
	if server == "" {
		return &LokiHandler{}, nil
	}
	p, err := NewLokiHTTPProxy(server)
	if err != nil {
		return nil, err
	}
	return &LokiHandler{Server: server, proxy: p}, nil
}

type LokiHTTPProxy struct {
	rp *httputil.ReverseProxy
}

func NewLokiHTTPProxy(server string) (*LokiHTTPProxy, error) {
	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}
	return &LokiHTTPProxy{
		rp: &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(serverURL)
			},
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}, nil
}

func (h *LokiHTTPProxy) to(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queries := r.URL.Query()
		logql := queries.Get("query")
		// back compatible, some one query escaped logql in query param
		if unescapedLogQL, _ := url.QueryUnescape(logql); unescapedLogQL != logql {
			queries.Set("query", unescapedLogQL)
		}
		r.URL.RawQuery = queries.Encode()

		r.URL.Path = path
		r.URL.RawPath = "" // reset raw path
		h.rp.ServeHTTP(w, r)
	}
}

func (h *LokiHandler) ProxyToPath(path string) http.HandlerFunc {
	if h.proxy == nil {
		msg := fmt.Sprintf("loki proxy not init")
		return func(w http.ResponseWriter, r *http.Request) {
			response.Raw(w, http.StatusNotImplemented, msg, nil)
		}
	}
	return h.proxy.to(path)
}

// @Tags			Agent.V1
// @Summary		Loki Query
// @Description	Loki Query
// @Accept			json
// @Produce		json
// @Param			cluster		path		string									true	"cluster"
// @Param			limit		query		string									false	"The max number of entries to return"
// @Param			query		query		string									true	"loki query language"
// @Param			time		query		int										false	"The evaluation time for the query as a nanosecond Unix epoch or another supported format. Defaults to now"
// @Param			direction	query		string									true	"The order to all results"
// @Success		200			{object}	handlers.ResponseStruct{Data=object}	""
// @Router			/v1/proxy/cluster/{cluster}/custom/loki/v1/query [get]
// @Security		JWT
func (h *LokiHandler) Query(c *gin.Context) {
	h.ProxyToPath("/loki/api/v1/query").ServeHTTP(c.Writer, c.Request)
}

// @Tags			Agent.V1
// @Summary		Loki QueryRange
// @Description	Loki QueryRange
// @Accept			json
// @Produce		json
// @Param			cluster		path		string									true	"cluster"
// @Param			start		query		string									true	"The start time for the query as a nanosecond Unix epoch"
// @Param			end			query		string									true	"The end time for the query as a nanosecond Unix epoch"
// @Param			direction	query		string									true	"The order to all results"
// @Param			limit		query		string									false	"The max number of entries to return"
// @Param			query		query		string									true	"loki query language"
// @Success		200			{object}	handlers.ResponseStruct{Data=object}	""
// @Router			/v1/proxy/cluster/{cluster}/custom/loki/v1/queryrange [get]
// @Security		JWT
func (h *LokiHandler) QueryRange(c *gin.Context) {
	h.ProxyToPath("/loki/api/v1/query_range").ServeHTTP(c.Writer, c.Request)
}

// @Tags			Agent.V1
// @Summary		Loki Labels
// @Description	Loki Labels
// @Accept			json
// @Produce		json
// @Param			cluster	path		string									true	"cluster"
// @Param			start	query		string									true	"The start time for the query as a nanosecond Unix epoch"
// @Param			end		query		string									true	"The end time for the query as a nanosecond Unix epoch"
// @Success		200		{object}	handlers.ResponseStruct{Data=object}	""
// @Router			/v1/proxy/cluster/{cluster}/custom/loki/v1/labels [get]
// @Security		JWT
func (h *LokiHandler) Labels(c *gin.Context) {
	h.ProxyToPath("/loki/api/v1/labels").ServeHTTP(c.Writer, c.Request)
}

// @Tags			Agent.V1
// @Summary		Loki LabelValues
// @Description	Loki LabelValues
// @Accept			json
// @Produce		json
// @Param			cluster	path		string									true	"cluster"
// @Param			start	query		string									true	"The start time for the query as a nanosecond Unix epoch"
// @Param			end		query		string									true	"The end time for the query as a nanosecond Unix epoch"
// @Param			label	query		string									true	"label"
// @Success		200		{object}	handlers.ResponseStruct{Data=object}	""
// @Router			/v1/proxy/cluster/{cluster}/custom/loki/v1/labelvalues [get]
// @Security		JWT
func (h *LokiHandler) LabelValues(c *gin.Context) {
	h.ProxyToPath("/loki/api/v1/label/:label/values").ServeHTTP(c.Writer, c.Request)
}

func _query(params map[string]string) string {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return q.Encode()
}

// @Tags			Agent.V1
// @Summary		Loki Series
// @Description	Loki Series
// @Accept			json
// @Produce		json
// @Param			cluster	path		string									true	"cluster"
// @Param			start	query		string									true	"The start time for the query as a nanosecond Unix epoch"
// @Param			end		query		string									true	"The end time for the query as a nanosecond Unix epoch"
// @Param			match	query		string									true	"match"
// @Success		200		{object}	handlers.ResponseStruct{Data=object}	""
// @Router			/v1/proxy/cluster/{cluster}/custom/loki/v1/series [get]
// @Security		JWT
func (h *LokiHandler) Series(c *gin.Context) {
	h.ProxyToPath("/loki/api/v1/series").ServeHTTP(c.Writer, c.Request)
}

// @Tags			Agent.V1
// @Summary		Loki LabelValues
// @Description	Loki LabelValues
// @Accept			json
// @Produce		json
// @Param			cluster		path		string									true	"cluster"
// @Param			start		query		string									true	"The start time for the query as a nanosecond Unix epoch"
// @Param			limit		query		string									false	"The max number of entries to return"
// @Param			query		query		string									true	"loki query language"
// @Param			delay_for	query		string									true	"The number of seconds to delay retrieving logs to let slow loggers catch up. Defaults to 0 and cannot be larger than 5."
// @Param			stream		query		string									true	"must be true"
// @Success		200			{object}	handlers.ResponseStruct{Data=object}	""
// @Router			/v1/proxy/cluster/{cluster}/custom/loki/v1/tail [get]
// @Security		JWT
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

// @Tags			Agent.V1
// @Summary		Loki Alert Rule
// @Description	Loki Alert Rule
// @Accept			json
// @Produce		json
// @Param			cluster	path		string																	true	"cluster"
// @Success		200		{object}	handlers.ResponseStruct{Data=map[string]prometheus.RealTimeAlertRule}	""
// @Router			/v1/proxy/cluster/{cluster}/custom/loki/v1/alertrule [get]
// @Security		JWT
func (h *LokiHandler) AlertRule(c *gin.Context) {
	body, err := h._http("/prometheus/api/v1/rules", "GET", nil, nil)
	if err != nil {
		NotOK(c, err)
		return
	}
	resp := loki.LokiPromeRuleResp{}
	if err := json.Unmarshal(body, &resp); err != nil {
		NotOK(c, errors.Wrap(err, "get loki prome rule"))
		return
	}
	// gems-namespace-name 作为key
	ret := make(map[string]prometheus.RealTimeAlertRule)
	for _, g := range resp.Data.Groups {
		for _, r := range g.Rules {
			switch v := r.(type) {
			case v1.AlertingRule:
				namespace := v.Labels[prometheus.AlertNamespaceLabel]
				name := v.Labels[prometheus.AlertNameLabel]
				if namespace != "" && name != "" {
					key := prometheus.RealTimeAlertKey(string(namespace), string(name))
					if v.Name == g.Name {
						alert, ok := ret[key]
						if ok {
							alert.Alerts = append(alert.Alerts, v.Alerts...)
							alert.State = getState(alert.State, v.State)
						} else {
							alert = prometheus.RealTimeAlertRule{
								Alerts: v.Alerts,
								Name:   v.Name,
								State:  getState("", v.State),
							}
						}
						ret[key] = alert
					}
				}
			}
		}
	}
	OK(c, ret)
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
