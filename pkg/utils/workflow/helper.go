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

package workflow

import (
	"reflect"
)

func IdentityKeyOfFunction(fun interface{}) string {
	t := reflect.ValueOf(fun).Type()
	if t.Kind() != reflect.Func {
		return "__not_function__"
	}

	str := t.String()
	_ = str

	m := t.Method(0)

	_ = m

	name := t.Name()
	pkg := t.PkgPath()

	return pkg + name
}
