package client

type ConditionOperator string

var (
	Eq, Gt, Lt, Neq, Gte, Lte, In, Like ConditionOperator = "=", ">", "<", "<>", ">=", "<=", "in", "like"
)

type PageSizeOption struct {
	page int64
	size int64
}

type WhereOption struct {
	where *Cond
}

type PreloadOption struct {
	preloads []string
}

type OrderOption struct {
	order string
}

type SearchOption struct {
	search string
}

type RelationFieldOption struct {
	field string
}

func (p *PageSizeOption) Apply(q *Query) {
	q.Page = p.page
	q.Size = p.size
}

func (w *WhereOption) Apply(q *Query) {
	q.Where = append(q.Where, w.where)
}

func (p *PreloadOption) Apply(q *Query) {
	q.Preloads = p.preloads
}

func (o *OrderOption) Apply(q *Query) {
	q.Orders = append(q.Orders, o.order)
}

func (o *RelationFieldOption) Apply(q *Query) {
	q.RelationFields = append(q.RelationFields, o.field)
}

func (o *SearchOption) Apply(q *Query) {
	q.Search = o.search
}

func PageSize(page, size int64) *PageSizeOption {
	return &PageSizeOption{
		page: page,
		size: size,
	}
}

func Where(field string, op ConditionOperator, value interface{}) *WhereOption {
	return &WhereOption{
		where: &Cond{Field: field, Op: op, Value: value},
	}
}

// 预加载关联资源
func Preloads(preloads []string) *PreloadOption {
	return &PreloadOption{
		preloads: preloads,
	}
}

// 对字段正序排序
func OrderAsc(field string) *OrderOption {
	return &OrderOption{
		order: field + " ASC",
	}
}

// 对字段倒序排序
func OrderDesc(field string) *OrderOption {
	return &OrderOption{
		order: field + " DESC",
	}
}

// 只有对多对多关系生效，选择中间表的字段到结果中
func RelationField(field string) *RelationFieldOption {
	return &RelationFieldOption{
		field: field,
	}
}

// 搜索内容
func Search(value string) *SearchOption {
	return &SearchOption{
		search: value,
	}
}
