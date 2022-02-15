package client

type ObjectTypeIfe interface {
	GetKind() *string
	GetPKField() *string
}

type ObjectKeyIfe interface {
	ObjectTypeIfe
	GetPKValue() interface{}
}

type Object interface {
	ObjectKeyIfe
	ValidPreloads() *[]string
}

type ObjectListIfe interface {
	ObjectTypeIfe
	GetPageSize() (*int64, *int64)
	GetTotal() *int64
	SetPageSize(int64, int64)
	SetTotal(int64)
	DataPtr() interface{}
}

type RelationShip interface {
	Object
	Left() Object
	Right() Object
}

type Option interface {
	Apply(*Query)
}
