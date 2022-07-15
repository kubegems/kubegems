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

// ref. https://prometheus.io/docs/prometheus/latest/querying/basics/#time-series-selectors
type LabelOperator string

const (
	LabelEqual    LabelOperator = "="
	LabelNotEqual LabelOperator = "!="
	LabelRegex    LabelOperator = "=~"
	LabelNotRegex LabelOperator = "!~"
)

// 比较运算符
type ComparisonOperator string

const (
	Equal          ComparisonOperator = "=="
	NotEqual       ComparisonOperator = "!="
	GreaterThan    ComparisonOperator = ">"
	LessThan       ComparisonOperator = "<"
	GreaterOrEqual ComparisonOperator = ">="
	LessOrEqual    ComparisonOperator = "<="
)

// 二元算术运算符
type BinaryArithmeticOperators string

const (
	Addition       BinaryArithmeticOperators = "+"
	Subtraction    BinaryArithmeticOperators = "-"
	Multiplication BinaryArithmeticOperators = "*"
	Division       BinaryArithmeticOperators = "/"
	Modulo         BinaryArithmeticOperators = "%"
	Power          BinaryArithmeticOperators = "^"
)
