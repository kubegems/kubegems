package client

type ConditionOperator string

var Eq, Gt, Lt, Neq, Gte, Lte, In, Like ConditionOperator = "=", ">", "<", "<>", ">=", "<=", "in", "like"

type PageSizeOption struct {
	page int64
	size int64
}

func (p *PageSizeOption) Apply(q *Query) {
	q.Page = p.page
	q.Size = p.size
}

func PageSize(page, size int64) *PageSizeOption {
	return &PageSizeOption{
		page: page,
		size: size,
	}
}

type WhereOption struct {
	where *Cond
}

func (w *WhereOption) Apply(q *Query) {
	q.Where = append(q.Where, w.where)
}

func Where(field string, op ConditionOperator, value interface{}) *WhereOption {
	return &WhereOption{
		where: &Cond{Field: field, Op: op, Value: value},
	}
}

func WhereEqual(field string, value interface{}) *WhereOption {
	return &WhereOption{
		where: &Cond{Field: field, Op: Eq, Value: value},
	}
}

func WhereNameEqual(value interface{}) *WhereOption {
	return &WhereOption{
		where: &Cond{Field: "name", Op: Eq, Value: value},
	}
}

type PreloadOption struct {
	preloads []string
}

func (p *PreloadOption) Apply(q *Query) {
	q.Preloads = p.preloads
}

// 预加载关联资源
func Preloads(preloads []string) *PreloadOption {
	return &PreloadOption{
		preloads: preloads,
	}
}

type OrderOption struct {
	order string
}

func (o *OrderOption) Apply(q *Query) {
	q.Orders = append(q.Orders, o.order)
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

type SearchOption struct {
	search string
}

func (o *SearchOption) Apply(q *Query) {
	q.Search = o.search
}

// 搜索内容
func Search(value string) *SearchOption {
	return &SearchOption{
		search: value,
	}
}

type BelongToOption struct {
	belongTo Object
}

func (o *BelongToOption) Apply(q *Query) {
	q.Belong = append(q.Belong, o.belongTo)
}

func BelongTo(obj Object) *BelongToOption {
	return &BelongToOption{
		belongTo: obj,
	}
}

type RelationOption struct {
	rel RelationCondition
}

func (o *RelationOption) Apply(q *Query) {
	q.RelationOptions = append(q.RelationOptions, o.rel)
}

func ExistRelation(obj Object) *RelationOption {
	return &RelationOption{
		rel: RelationCondition{
			Target: obj,
		},
	}
}

func ExistRelationWithKeyValue(obj Object, key string, value interface{}) *RelationOption {
	return &RelationOption{
		rel: RelationCondition{
			Key:    key,
			Value:  value,
			Target: obj,
		},
	}
}
