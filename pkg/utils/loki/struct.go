package loki

import (
	"encoding/json"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type LabelResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data,omitempty"`
}

const (
	ResultTypeVector = "vector"
	ResultTypeMatrix = "matrix"
)

// QueryResponse represents the http json response to a Loki range and instant query
type QueryResponse struct {
	Status string            `json:"status"`
	Data   QueryResponseData `json:"data"`
}

type QueryResponseData struct {
	ResultType string        `json:"resultType"`
	Result     []interface{} `json:"result"`
}

type Matrix []SampleStream

type SampleStream struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}

func (q *SampleStream) ToStruct(m map[string]interface{}) SampleStream {
	b, _ := json.Marshal(m)
	var s SampleStream
	_ = json.Unmarshal(b, &s)
	return s
}

type Streams []Stream

type Stream struct {
	Labels  map[string]string `json:"stream"`
	Entries [][]string        `json:"values"`
}

func (q *Stream) ToStruct(m map[string]interface{}) Stream {
	b, _ := json.Marshal(m)
	var s Stream
	_ = json.Unmarshal(b, &s)
	return s
}

type LabelParam struct {
	Start string `form:"start" json:"start,omitempty"`
	End   string `form:"end" json:"end,omitempty"`
	Label string `form:"label" json:"label,omitempty"`
}

func (l *LabelParam) ToMap() map[string]string {
	b, _ := json.Marshal(&l)
	var m map[string]string
	_ = json.Unmarshal(b, &m)
	return m
}

type LokiPromeRuleResp struct {
	Status    string         `json:"status"`
	Data      v1.RulesResult `json:"data"`
	ErrorType v1.ErrorType   `json:"errorType"`
	Error     string         `json:"error"`
	Warnings  []string       `json:"warnings,omitempty"`
}

type QueryRangeParam struct {
	Start     string `form:"start" json:"start,omitempty"`
	End       string `form:"end" json:"end,omitempty"`
	Step      string `form:"step" json:"step,omitempty"`
	Interval  string `form:"interval" json:"interval,omitempty"`
	Query     string `form:"query" json:"query,omitempty"`
	Direction string `form:"direction" json:"direction,omitempty"`
	Limit     string `form:"limit" json:"limit,omitempty"`
}

func (q *QueryRangeParam) ToMap() map[string]string {
	b, _ := json.Marshal(&q)
	var m map[string]string
	_ = json.Unmarshal(b, &m)
	return m
}

type TailParam struct {
	Start     string `form:"start" json:"start,omitempty" header:"start"`
	Query     string `form:"query" json:"query,omitempty" header:"query"`
	Limit     string `form:"limit" json:"limit,omitempty" header:"limit"`
	Delay_For string `form:"delay_for"  json:"delay_for,omitempty" header:"delay_for"`
	Level     string `form:"level" json:"level,omitempty" header:"level"`
	Filter    string `form:"filter" json:"filter,omitempty" header:"filter"`
}

func (q *TailParam) ToMap() map[string]string {
	b, _ := json.Marshal(&q)
	var m map[string]string
	_ = json.Unmarshal(b, &m)
	return m
}

type LokiMessage struct {
	Timestamp string
	Message   string
}

type LokiMessages []LokiMessage

func (msg LokiMessages) Len() int { return len(msg) }

func (msg LokiMessages) Less(i, j int) bool {
	return msg[i].Timestamp > msg[j].Timestamp
}

func (msg LokiMessages) Swap(i, j int) { msg[i], msg[j] = msg[j], msg[i] }

type QueryResult struct {
	Info   Info              `json:"info"`
	Stream map[string]string `json:"stream"`
}

type Info struct {
	Animation    string `json:"animation"`
	Level        string `json:"level"`
	Message      string `json:"message"`
	Timestamp    string `json:"timestamp"`
	Timestampstr string `json:"timestampstr"`
	Index        string `json:"index"`
}

type SeriesForm struct {
	Match string `form:"match" json:"match,omitempty"`
	End   string `form:"end" json:"end,omitempty"`
	Start string `form:"start" json:"start,omitempty"`
	Label string `form:"label" json:"label,omitempty"`
}

func (l *SeriesForm) ToMap() map[string]string {
	b, _ := json.Marshal(&l)
	var m map[string]string
	_ = json.Unmarshal(b, &m)
	return m
}

type SeriesResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
}
