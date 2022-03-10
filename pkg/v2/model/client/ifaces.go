package client

type ObjectTypeIface interface {
	GetKind() *string
	PrimaryKeyField() *string
}

type Object interface {
	ObjectTypeIface
	PrimaryKeyValue() interface{}
	PreloadFields() *[]string
}

type ObjectListIface interface {
	ObjectTypeIface
	GetPageSize() (*int64, *int64)
	GetTotal() *int64
	SetPageSize(int64, int64)
	SetTotal(int64)
	DataPtr() interface{}
}

type Option interface {
	Apply(*Query)
}
