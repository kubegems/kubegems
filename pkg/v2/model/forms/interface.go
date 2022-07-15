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

package forms

import "kubegems.io/kubegems/pkg/v2/model/client"

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
