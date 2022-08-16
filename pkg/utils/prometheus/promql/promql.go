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

package promql

import (
	"fmt"
	"strings"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/set"
)

const (
	labelValueAll = "_all"
)

type Query struct {
	parser.Expr

	sumby []string
}

func New(promql string) (*Query, error) {
	expr, err := parser.ParseExpr(promql)
	if err != nil {
		return nil, err
	}
	return &Query{
		Expr: expr,
	}, nil
}

func (q *Query) AddLabelMatchers(matchers ...*labels.Matcher) *Query {
	parser.Inspect(q.Expr, func(node parser.Node, _ []parser.Node) error {
		vs, ok := node.(*parser.VectorSelector)
		if ok {
			for _, dest := range matchers {
				var found *labels.Matcher
				for _, src := range vs.LabelMatchers {
					if src.Name == dest.Name {
						found = src
					}
				}
				// overwrite if has same label name
				if found == nil {
					vs.LabelMatchers = append(vs.LabelMatchers, dest)
				} else {
					found.Type = dest.Type
					found.Value = dest.Value
				}
			}
		}
		return nil
	})
	return q
}

func (q *Query) Sumby(labels ...string) *Query {
	q.sumby = append(q.sumby, labels...)
	return q
}

func (q *Query) GetVectorSelectors() []string {
	ret := set.NewSet[string]()
	parser.Inspect(q.Expr, func(node parser.Node, _ []parser.Node) error {
		vs, ok := node.(*parser.VectorSelector)
		if ok {
			ret.Append(vs.String())
		}
		return nil
	})
	return ret.Slice()
}

func (q *Query) String() string {
	var ret string
	if len(q.sumby) == 0 {
		ret = q.Expr.String()
	} else {
		ret = fmt.Sprintf("sum(%s)by(%s)", q.Expr.String(), strings.Join(q.sumby, ","))
	}
	log.Debugf("promql: %s", ret)
	return ret
}
