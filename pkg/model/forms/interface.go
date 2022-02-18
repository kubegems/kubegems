package forms

import "kubegems.io/pkg/model/client"

type BaseForm struct {
	object client.Object `json:"-"`
	data   interface{}   `json:"-"`
}

type BaseListForm struct {
	objectlist client.ObjectListIface
	data       interface{}
}

type FormInterface interface {
	// 将表单对象转换成模型对象
	Object() client.Object
	// 将模型对象转换成表单对象
	DataPtr() interface{}
}

type FormListInterface interface {
	// 将表单对象转换成模型对象
	Object() client.ObjectListIface
	// 将模型对象转换成表单对象
	DataPtr() interface{}
}
