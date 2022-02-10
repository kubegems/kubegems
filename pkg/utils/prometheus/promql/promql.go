package promql

import (
	"fmt"
	"sort"
	"strings"
)

const (
	labelValueAll = "_all"
)

type Query struct {
	metric    string
	selectors []string
	sumBy     []string // sum(metric) by (...)

	// eg. > 100
	compare      ComparisonOperator
	compareValue string

	op      BinaryArithmeticOperators
	opValue string

	round float64 // 保留小数
	topk  int     // 前多少个
}

func New(metric string) *Query {
	return &Query{
		metric: metric,
	}
}

func (q *Query) AddSelector(key string, op LabelOperator, value string) *Query {
	if value != "" && value != labelValueAll {
		q.selectors = append(q.selectors, fmt.Sprintf(`%s%s"%s"`, key, op, value))
	}
	sort.Strings(q.selectors)
	return q
}

func (q *Query) SumBy(labels ...string) *Query {
	q.sumBy = labels
	sort.Strings(q.sumBy)
	return q
}

func (q *Query) Compare(compare ComparisonOperator, value string) *Query {
	q.compare = compare
	q.compareValue = value
	return q
}

func (q *Query) Arithmetic(op BinaryArithmeticOperators, value string) *Query {
	q.op = op
	q.opValue = value
	return q
}

func (q *Query) Round(round float64) *Query {
	q.round = round
	return q
}

func (q *Query) Topk(topk int) *Query {
	q.topk = topk
	return q
}

func (q *Query) ToPromql() string {
	ret := q.metric
	if len(q.selectors) > 0 {
		ret = fmt.Sprintf("%s{%s}", ret, strings.Join(q.selectors, ", "))
	}

	if len(q.sumBy) > 0 {
		ret = fmt.Sprintf("sum(%s)by(%s)", ret, strings.Join(q.sumBy, ", "))
	}

	if q.op != "" && q.opValue != "" && q.opValue != "1" {
		ret = fmt.Sprintf("%s %s %s", ret, q.op, q.opValue)
	}

	// 保留小数
	if q.round > 0 && q.round < 1 {
		ret = fmt.Sprintf("round(%s, %v)", ret, q.round)
	}

	// 前几位
	if q.topk > 0 {
		ret = fmt.Sprintf("topk(%d, %s)", q.topk, ret)
	}

	if q.compare != "" {
		ret = fmt.Sprintf("%s %s %s", ret, q.compare, q.compareValue)
	}
	return ret
}
