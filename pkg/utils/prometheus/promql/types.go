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
