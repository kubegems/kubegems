package forms

import "kubegems.io/pkg/model/client"

type BaseForm struct {
	object client.Object
}

type BaseListForm struct {
	objectlist client.ObjectListIfe
}

type FormInterface interface {
	// 将表单对象转换成模型对象
	AsObject() client.Object
	// 将模型对象转换成表单对象
	Data() FormInterface
}

type FormListInterface interface {
	// 将表单对象转换成模型对象
	AsListObject() client.ObjectListIfe
	// 将模型对象转换成表单对象
	AsListData() FormListInterface
}
